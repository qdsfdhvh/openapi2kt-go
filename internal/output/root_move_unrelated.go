package output

import (
	"openapi-kt/internal/diagnostic"
	"path/filepath"
	"strings"
)

// BindRootMoveConfigPeers permits original target owners to remain declared at
// that exact target. Their inputs are not source preservation dependencies.
// Other peers retain the disjoint-root and stable ancestor-binding contract.
func BindRootMoveConfigPeers(s RootMoveState, roots map[string]string) (RootMoveState, *diagnostic.Diagnostic) {
	for _, owner := range sortedKeys(roots) {
		raw := roots[owner]
		root, err := canonicalRoot(raw)
		if err != nil {
			return RootMoveState{}, err
		}
		if s.MergeTarget != nil && root == s.MergeTarget.root {
			if s.MergeTarget.Owners[owner] == nil {
				return RootMoveState{}, rootMoveInvalid()
			}
			s.bindings = append(s.bindings, rootMoveBinding{raw, root})
			continue
		}
		s, err = BindRootMoveUnrelatedRoots(s, []string{raw})
		if err != nil {
			return RootMoveState{}, err
		}
	}
	return s, nil
}

// BindRootMoveUnrelatedRoots classifies config peers before input reads. Their
// ownership and files are never opened or changed; ancestor bindings remain
// stable through the same read-only/locked source-pair preflight.
func BindRootMoveUnrelatedRoots(s RootMoveState, roots []string) (RootMoveState, *diagnostic.Diagnostic) {
	if len(s.bindings) == 0 {
		return RootMoveState{}, rootMoveInvalid()
	}
	for _, raw := range roots {
		root, err := canonicalRoot(raw)
		if err != nil {
			return RootMoveState{}, err
		}
		for _, moved := range []string{s.root, s.bindings[0].canonical} {
			a, b := strings.ToLower(root), strings.ToLower(moved)
			separator := string(filepath.Separator)
			if a == b || strings.HasPrefix(a, strings.TrimRight(b, separator)+separator) || strings.HasPrefix(b, strings.TrimRight(a, separator)+separator) {
				return RootMoveState{}, rootMoveInvalid()
			}
		}
		s.bindings = append(s.bindings, rootMoveBinding{raw, root})
	}
	return s, nil
}
