package schema

import (
	"openapi-kt/internal/diagnostic"
	"sort"
	"strings"
)

// Validate every retained name/occurrence before mutating any declaration.
func prepareLockedHoist(plan *Plan, pkg string, keys []string, groups map[string][]Declaration, used map[string]bool, state *HoistedNameLock, project bool) (map[string]string, map[string]NameLockEntry, *diagnostic.Diagnostic) {
	names, entries := map[string]string{}, map[string]NameLockEntry{}
	if state == nil {
		return names, entries, nil
	}
	current, byShape, present := map[string]string{}, map[string]string{}, map[string]bool{}
	prior := map[string]NameLockEntry{}
	for _, entry := range state.Entries {
		prior[entry.Role+":"+entry.Shape] = entry
	}
	emitted := map[string]bool{}
	for _, d := range plan.Client.Models {
		for _, origin := range plan.operationOrigins[d.Name()] {
			emitted[plan.operationRoles[d.Name()]+":"+origin] = true
		}
	}
	for _, origins := range plan.operationOrigins {
		for _, origin := range origins {
			present[origin] = true
		}
	}
	for _, key := range keys {
		peers := groups[key]
		role, shape := plan.operationRoles[peers[0].Name()], nameLockShapeV1(peers[0], pkg)
		identity := role + ":" + shape
		if len(peers) < 2 {
			if _, retained := prior[identity]; !retained {
				continue
			}
		}
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
		retained := []string{}
		for _, origin := range entry.Occurrences {
			actual := current[entry.Role+":"+origin]
			if project {
				if actual == identity {
					retained = append(retained, origin)
				} else if actual != "" || emitted[entry.Role+":"+origin] {
					return nil, nil, nameLockError("E_NAME_LOCK_CHANGED", "A projected occurrence changed shape, eligibility or role.")
				}
			} else {
				if !present[origin] {
					return nil, nil, nameLockError("E_NAME_LOCK_REMOVED", "A locked occurrence is no longer present.")
				}
				if actual != identity {
					return nil, nil, nameLockError("E_NAME_LOCK_CHANGED", "A locked occurrence changed shape, eligibility or role.")
				}
				retained = append(retained, origin)
			}
		}
		if len(retained) == 0 {
			continue
		}
		key := byShape[identity]
		if key == "" || used[strings.ToLower(entry.Name)] {
			return nil, nil, nameLockError("E_NAME_LOCK_CONFLICT", "A retained hoisted name conflicts with an existing declaration or helper.")
		}
		if project {
			allowed := map[string]bool{}
			for _, origin := range entry.Occurrences {
				allowed[origin] = true
			}
			for _, origin := range entries[key].Occurrences {
				if !allowed[origin] {
					return nil, nil, nameLockError("E_NAME_LOCK_CHANGED", "Projection cannot add a locked occurrence.")
				}
			}
		}
		names[key] = entry.Name
		used[strings.ToLower(entry.Name)] = true
	}
	if project {
		for key, entry := range entries {
			if _, ok := prior[entry.Role+":"+entry.Shape]; !ok || names[key] == "" {
				return nil, nil, nameLockError("E_NAME_LOCK_CHANGED", "Projection cannot allocate new locked shapes.")
			}
		}
	}

	state.Entries = []NameLockEntry{}
	return names, entries, nil
}
