package schema

import (
	"openapi-kt/internal/diagnostic"
	"sort"
	"strings"
)

// Validate every retained name/occurrence before mutating any declaration.
func prepareLockedHoist(plan *Plan, pkg string, keys []string, groups map[string][]Declaration, used map[string]bool, state *HoistedNameLock) (map[string]string, map[string]NameLockEntry, *diagnostic.Diagnostic) {
	names, entries := map[string]string{}, map[string]NameLockEntry{}
	if state == nil {
		return names, entries, nil
	}
	current, byShape, present := map[string]string{}, map[string]string{}, map[string]bool{}
	for _, origins := range plan.operationOrigins {
		for _, origin := range origins {
			present[origin] = true
		}
	}
	for _, key := range keys {
		peers := groups[key]
		if len(peers) < 2 {
			continue
		}
		role, shape := plan.operationRoles[peers[0].Name()], nameLockShapeV1(peers[0], pkg)
		identity := role + ":" + shape
		if _, ok := byShape[identity]; ok {
			return nil, nil, nameLockError("E_NAME_LOCK_CONFLICT", "Locked shape identity is ambiguous.")
		}
		byShape[identity] = key
		origins := map[string]bool{}
		for _, d := range peers {
			for _, origin := range plan.operationOrigins[d.Name()] {
				if previous, ok := current[role+":"+origin]; ok && previous != identity {
					return nil, nil, nameLockError("E_NAME_LOCK_CONFLICT", "Locked occurrence identity is ambiguous.")
				}
				origins[origin] = true
				current[role+":"+origin] = identity
			}
		}
		sorted := make([]string, 0, len(origins))
		for origin := range origins {
			sorted = append(sorted, origin)
		}
		sort.Strings(sorted)
		entries[key] = NameLockEntry{Role: role, Shape: shape, Occurrences: sorted}
	}
	for _, entry := range state.Entries {
		identity := entry.Role + ":" + entry.Shape
		for _, origin := range entry.Occurrences {
			if !present[origin] {
				return nil, nil, nameLockError("E_NAME_LOCK_REMOVED", "A locked occurrence is no longer present.")
			}
		}
		for _, origin := range entry.Occurrences {
			if current[entry.Role+":"+origin] != identity {
				return nil, nil, nameLockError("E_NAME_LOCK_CHANGED", "A locked occurrence changed shape, eligibility or role.")
			}
		}
		key := byShape[identity]
		if key == "" || used[strings.ToLower(entry.Name)] {
			return nil, nil, nameLockError("E_NAME_LOCK_CONFLICT", "A retained hoisted name conflicts with an existing declaration or helper.")
		}
		names[key] = entry.Name
		used[strings.ToLower(entry.Name)] = true
	}
	state.Entries = []NameLockEntry{}
	return names, entries, nil
}
