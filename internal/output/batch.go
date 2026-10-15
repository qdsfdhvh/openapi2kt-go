package output

import (
	"openapi-kt/internal/diagnostic"
	"os"
	"path/filepath"
	"strings"
)

// canonicalRoot resolves existing ancestor aliases without creating directories.
func canonicalRoot(root string) (string, *diagnostic.Diagnostic) {
	if err := validateRoot(root); err != nil {
		return "", err
	}
	absolute, e := filepath.Abs(root)
	if e != nil {
		return "", ioFailure()
	}
	ancestor := absolute
	suffix := ""
	for {
		_, e := os.Lstat(ancestor)
		if e == nil {
			break
		}
		if !os.IsNotExist(e) {
			return "", ioFailure()
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return "", ioFailure()
		}
		suffix = filepath.Join(filepath.Base(ancestor), suffix)
		ancestor = parent
	}
	resolved, e := filepath.EvalSymlinks(ancestor)
	if e != nil {
		return "", ioFailure()
	}
	return filepath.Join(resolved, suffix), nil
}

// WriteOwnersBatch holds all cooperative root locks and stages every changed
// root before applying any. Normal failures roll back all applied roots. This
// is not cross-root crash atomicity. Caller data must remain immutable.
func WriteOwnersBatch(updates map[string]OwnerUpdate, check bool) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatch(updates, check, nil)
}

// before is test-only: prelock, prepare or apply, never a CLI/environment hook.
func writeOwnersBatch(updates map[string]OwnerUpdate, check bool, before func(rootIndex int, phase string, opIndex int, stage string) error) (bool, *diagnostic.Diagnostic) {
	return writeOwnersBatchWith(updates, check, before, makeOwnersPlan)
}
func writeOwnersBatchWith[U any](updates map[string]U, check bool, before func(rootIndex int, phase string, opIndex int, stage string) error, planner func(string, U) (*plan, *diagnostic.Diagnostic)) (bool, *diagnostic.Diagnostic) {
	if len(updates) == 0 {
		return false, fail("E_OUTPUT_EMPTY", "Select output roots or explicitly reconcile a configured root.")
	}
	canonical := map[string]U{}
	for _, raw := range sortedKeys(updates) {
		root, e := canonicalRoot(raw)
		if e != nil {
			return false, e
		}
		if _, duplicate := canonical[root]; duplicate {
			return false, fail("E_OUTPUT_ROOTS", "Output roots alias or overlap; use one canonical inventory per root.")
		}
		canonical[root] = updates[raw]
	}
	roots := sortedKeys(canonical)
	for i, a := range roots {
		for _, b := range roots[i+1:] {
			a, b := strings.ToLower(a), strings.ToLower(b)
			if a == b || strings.HasPrefix(b, strings.TrimRight(a, string(filepath.Separator))+string(filepath.Separator)) || strings.HasPrefix(a, strings.TrimRight(b, string(filepath.Separator))+string(filepath.Separator)) {
				return false, fail("E_OUTPUT_ROOTS", "Output roots alias or overlap; use disjoint canonical roots.")
			}
		}
	}
	changed := false
	for _, root := range roots {
		p, e := planner(root, canonical[root])
		if e != nil {
			return false, e
		}
		changed = changed || p.Changed
	}
	if check {
		return changed, nil
	}
	if before != nil {
		if e := before(-1, "prelock", -1, ""); e != nil {
			return false, ioFailure()
		}
	}
	locks := []*os.File{}
	defer func() {
		for i := len(locks) - 1; i >= 0; i-- {
			unlock(locks[i])
			_ = locks[i].Close()
		}
	}()
	for _, root := range roots {
		if e := os.MkdirAll(root, 0777); e != nil {
			return false, ioFailure()
		}
		if e := validateRoot(root); e != nil {
			return false, e
		}
		resolved, e := filepath.EvalSymlinks(root)
		if e != nil || resolved != root {
			return false, fail("E_OUTPUT_PATH", "Output root changed while acquiring locks.")
		}
		path := filepath.Join(root, lockName)
		meta, e := os.Lstat(path)
		if e == nil && !meta.Mode().IsRegular() {
			return false, fail("E_OUTPUT_PATH", "Output lock must be a regular file.")
		}
		if e != nil && !os.IsNotExist(e) {
			return false, ioFailure()
		}
		f, e := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		if e != nil {
			return false, ioFailure()
		}
		if e := tryLock(f); e != nil {
			_ = f.Close()
			return false, fail("E_OUTPUT_BUSY", "Another generator owns this output directory.")
		}
		locks = append(locks, f)
	}
	// Replan all roots under the complete lock set before staging any root.
	plans := map[string]*plan{}
	changed = false
	for _, root := range roots {
		p, e := planner(root, canonical[root])
		if e != nil {
			return false, e
		}
		plans[root] = p
		changed = changed || p.Changed
	}
	if !changed {
		return false, nil
	}
	transactions := []*transaction{}
	defer func() {
		for _, tx := range transactions {
			tx.cleanup()
		}
	}()
	for i, root := range roots {
		if !plans[root].Changed {
			continue
		}
		if before != nil {
			if e := before(i, "prepare", -1, ""); e != nil {
				return false, ioFailure()
			}
		}
		tx, e := prepare(root, plans[root])
		if e != nil {
			return false, e
		}
		transactions = append(transactions, tx)
	}
	for _, tx := range transactions {
		rootIndex := 0
		for roots[rootIndex] != tx.root {
			rootIndex++
		}
		var inject func(int, string) error
		if before != nil {
			inject = func(index int, stage string) error { return before(rootIndex, "apply", index, stage) }
		}
		if tx.apply(inject) != nil {
			recovery := false
			for i := len(transactions) - 1; i >= 0; i-- {
				if transactions[i].rollback() {
					recovery = true
				}
			}
			if recovery {
				return false, fail("E_RECOVERY_REQUIRED", "Rollback could not finish; transaction backups were retained in affected output directories.")
			}
			return false, fail("E_OUTPUT_COMMIT", "Batch commit failed; previous generated files and manifests were restored in all output roots.")
		}
	}
	return true, nil
}
