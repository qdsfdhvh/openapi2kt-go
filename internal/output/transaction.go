package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"openapi-kt/internal/diagnostic"
)

type operation struct {
	Attributes *fileAttributes
	Path       string
	Data       []byte
	Write      bool
}
type journalEntry struct {
	Path   string  `json:"path"`
	Backup string  `json:"backup"`
	Staged *string `json:"staged"`
	HadOld bool    `json:"had_old"`
}
type journal struct {
	Version    int            `json:"version"`
	Operations []journalEntry `json:"operations"`
}
type applied struct {
	Destination, Backup string
	MovedOld, WroteNew  bool
}

func removeFile(path string) error {
	meta, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if meta.IsDir() {
		return fmt.Errorf("not a file")
	}
	return os.Remove(path)
}
func removeDirectory(path string) error {
	meta, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !meta.IsDir() || meta.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("not a directory")
	}
	return os.Remove(path)
}

func ensureParents(path, root string, created *[]string, checked map[string]bool) error {
	parent := filepath.Dir(path)
	relative, err := filepath.Rel(root, parent)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("outside root")
	}
	if relative == "." {
		return nil
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		if checked[current] {
			continue
		}
		meta, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0777); err != nil {
				return err
			}
			*created = append(*created, current)
		} else if err != nil {
			return err
		} else if !meta.IsDir() || meta.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("invalid parent")
		}
		checked[current] = true
	}
	return nil
}

// Staging and application are separate so batches retain every backup until
// all roots have committed. Cleanup may recurse only inside the transaction's
// newly created stage; it never recursively clears an output root.
type transaction struct {
	root, stage string
	operations  []operation
	changes     []applied
	created     []string
	retain      bool
}

func prepare(root string, plan *plan) (*transaction, *diagnostic.Diagnostic) {
	return prepareWithAttributes(root, plan, nil)
}

// beforeAttribute is a package-private staging fault seam, never a CLI/env hook.
func prepareWithAttributes(root string, plan *plan, beforeAttribute func(string, string) error) (result *transaction, failure *diagnostic.Diagnostic) {
	stage, err := os.MkdirTemp(root, stagePrefix)
	if err != nil {
		return nil, ioFailure()
	}
	tx := &transaction{root: root, stage: stage}
	defer func() {
		if failure != nil {
			tx.cleanup()
		}
	}()
	operations := []operation{}
	for _, path := range sortedKeys(plan.Writes) {
		operations = append(operations, operation{Path: path, Data: plan.Writes[path], Write: true})
	}
	for _, path := range plan.Removals {
		operations = append(operations, operation{Path: path})
	}
	manifestBytes := plan.ManifestBytes
	if !plan.RemoveManifest && manifestBytes == nil {
		manifestBytes, err = json.MarshalIndent(plan.Manifest, "", "  ")
		if err != nil {
			return nil, ioFailure()
		}
	}
	operations = append(operations, operation{Path: manifestName, Data: manifestBytes, Write: !plan.RemoveManifest})
	for i := range operations {
		if attributes, ok := plan.Attributes[operations[i].Path]; ok {
			operations[i].Attributes = &attributes
		}
	}
	record := journal{Version: 1, Operations: make([]journalEntry, 0, len(operations))}
	for index, op := range operations {
		var staged *string
		if op.Write {
			name := fmt.Sprintf("new-%d", index)
			staged = &name
		}
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(op.Path)))
		record.Operations = append(record.Operations, journalEntry{Path: op.Path, Backup: fmt.Sprintf("old-%d", index), Staged: staged, HadOld: err == nil})
	}
	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, ioFailure()
	}
	if err := os.WriteFile(filepath.Join(stage, "transaction.json"), encoded, 0666); err != nil {
		return nil, ioFailure()
	}
	if err := stageFilesWithAttributes(stage, operations, beforeAttribute); err != nil {
		return nil, ioFailure()
	}
	tx.operations = operations
	return tx, nil
}

// Staged files have unique names inside a new private transaction directory.
// Complete every worker before cleanup or application; destination mutation
// and the manifest commit remain serial and keep their original journal order.
func stageFiles(stage string, operations []operation) error {
	return stageFilesWithAttributes(stage, operations, nil)
}

func stageFilesWithAttributes(stage string, operations []operation, beforeAttribute func(string, string) error) error {
	workers := min(runtime.GOMAXPROCS(0), 8, max(1, len(operations)/32))
	failures := make([]error, workers)
	write := func(worker int) {
		for index := worker; index < len(operations); index += workers {
			op := operations[index]
			if op.Write {
				path := filepath.Join(stage, fmt.Sprintf("new-%d", index))
				if err := os.WriteFile(path, op.Data, 0666); err != nil {
					failures[worker] = err
					return
				}
				if a := op.Attributes; a != nil {
					if beforeAttribute != nil {
						if err := beforeAttribute("chmod", path); err != nil {
							failures[worker] = err
							return
						}
					}
					if err := os.Chmod(path, a.Mode); err != nil {
						failures[worker] = err
						return
					}
					if beforeAttribute != nil {
						if err := beforeAttribute("chtimes", path); err != nil {
							failures[worker] = err
							return
						}
					}
					if err := os.Chtimes(path, a.Modified, a.Modified); err != nil {
						failures[worker] = err
						return
					}
				}
			}
		}
	}
	var group sync.WaitGroup
	for worker := 1; worker < workers; worker++ {
		group.Go(func() { write(worker) })
	}
	write(0)
	group.Wait()
	for _, err := range failures {
		if err != nil {
			return err
		}
	}
	return nil
}

func (tx *transaction) apply(before func(index int, stage string) error) error {
	root, stage := tx.root, tx.stage
	// This locked application only creates real directories and moves regular
	// files. Share parent checks within it, never with the earlier planner.
	checked := map[string]bool{}
	for index, op := range tx.operations {
		if before != nil {
			if err := before(index, stage); err != nil {
				return err
			}
			// Fault injection may change the filesystem between operations.
			clear(checked)
		}
		destination := filepath.Join(root, filepath.FromSlash(op.Path))
		if err := ensureParents(destination, root, &tx.created, checked); err != nil {
			return err
		}
		tx.changes = append(tx.changes, applied{Destination: destination, Backup: filepath.Join(stage, fmt.Sprintf("old-%d", index))})
		change := &tx.changes[len(tx.changes)-1]
		_, err := os.Stat(destination)
		if err == nil {
			if err := os.Rename(destination, change.Backup); err != nil {
				return err
			}
			change.MovedOld = true
		} else if !os.IsNotExist(err) {
			return err
		}
		if op.Write {
			if err := os.Rename(filepath.Join(stage, fmt.Sprintf("new-%d", index)), destination); err != nil {
				return err
			}
			change.WroteNew = true
		}
	}
	return nil
}

func (tx *transaction) rollback() bool {
	rollbackFailed := false
	for index := len(tx.changes) - 1; index >= 0; index-- {
		change := tx.changes[index]
		if change.WroteNew {
			if err := removeFile(change.Destination); err != nil {
				rollbackFailed = true
			}
		}
		if change.MovedOld {
			if err := os.Rename(change.Backup, change.Destination); err != nil {
				rollbackFailed = true
			}
		}
	}
	for index := len(tx.created) - 1; index >= 0; index-- {
		if err := removeDirectory(tx.created[index]); err != nil {
			rollbackFailed = true
		}
	}
	tx.retain = rollbackFailed
	return rollbackFailed
}

func (tx *transaction) cleanup() {
	if !tx.retain {
		_ = os.RemoveAll(tx.stage)
	}
}

// before is a package-private fault-injection seam, never a CLI/env hook.
func commit(root string, plan *plan, before func(index int, stage string) error) *diagnostic.Diagnostic {
	tx, err := prepare(root, plan)
	if err != nil {
		return err
	}
	defer tx.cleanup()
	if tx.apply(before) == nil {
		return nil
	}
	if tx.rollback() {
		return fail("E_RECOVERY_REQUIRED", "Rollback could not finish; transaction backups were retained in the output directory.")
	}
	return fail("E_OUTPUT_COMMIT", "Output commit failed; previous generated files and manifest were restored.")
}
