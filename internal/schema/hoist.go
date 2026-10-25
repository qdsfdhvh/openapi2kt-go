package schema

import (
	"openapi-kt/internal/diagnostic"
	"sort"
	"strconv"
	"strings"
)

// HoistOperationModels relocates only repeated eligible operation-root trees.
// Work and diagnostics were fully checked by Analyze. No schema graph is walked.
func HoistOperationModels(plan *Plan, packageName string, qualify bool) {
	if err := hoistOperationModels(plan, packageName, qualify, nil, false); err != nil {
		panic("unexpected unlocked hoisting error")
	}
}

func HoistOperationModelsLocked(plan *Plan, packageName string, qualify bool, state *HoistedNameLock) *diagnostic.Diagnostic {
	return hoistOperationModels(plan, packageName, qualify, state, false)
}

// HoistOperationModelsLockedProjection retains only the original allocation
// occurrences emitted by this explicit migration candidate. Ownership coverage
// is a separate adapter barrier; ordinary generation remains strict.
func HoistOperationModelsLockedProjection(plan *Plan, packageName string, qualify bool, state *HoistedNameLock) *diagnostic.Diagnostic {
	if state == nil {
		return nameLockError("E_NAME_LOCK_INVALID", "Name-lock projection requires explicit previous state.")
	}
	return hoistOperationModels(plan, packageName, qualify, state, true)
}
func hoistOperationModels(plan *Plan, packageName string, qualify bool, state *HoistedNameLock, project bool) *diagnostic.Diagnostic {
	if plan.Client == nil {
		if project && state != nil {
			state.Entries = []NameLockEntry{}
			return nil
		}
		if state != nil && len(state.Entries) != 0 {
			return nameLockError("E_NAME_LOCK_REMOVED", "A locked occurrence is no longer present.")
		}
		return nil
	}
	protected := operationOpaqueOwners(plan, packageName)
	declarations := append([]Declaration(nil), plan.Client.Models...)
	sort.Slice(declarations, func(i, j int) bool { return declarations[i].Name() < declarations[j].Name() })
	groups := map[string][]Declaration{}
	keys := []string{}
	for _, d := range declarations {
		role := plan.operationRoles[d.Name()]
		if protected[d.Name()] || role == "" || state != nil && len(plan.operationOrigins[d.Name()]) == 0 || !dedupEligible(d) || !hoistExpressionsIndependent(d, d.Name()) {
			continue
		}
		key := role + ":" + operationEquivalenceKey(d, packageName)
		if _, exists := groups[key]; !exists {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], d)
	}
	// keys preserve lexical canonical-root order, not unordered map iteration.
	used := map[string]bool{}
	for _, d := range plan.Models {
		used[strings.ToLower(d.Name())] = true
	}
	pkg := packageName + ".models"
	if plan.PatchPackage == pkg {
		used["patch"] = true
		used["patchserializer"] = true
	}
	if plan.BigintPackage == pkg {
		used[strings.ToLower(bigintSerializerName)] = true
	}
	lockedNames, newEntries, err := prepareLockedHoist(plan, packageName, keys, groups, used, state, project)
	if err != nil {
		return err
	}
	aliases := map[Declaration]Declaration{}
	for _, key := range keys {
		peers := groups[key]
		if len(peers) < 2 && lockedNames[key] == "" {
			continue
		}
		retained := peers[0]
		old := retained.Name()
		base := plan.operationRoles[old]
		if qualify {
			base = old
		}
		name := lockedNames[key]
		if name == "" {
			name = base
			for ordinal := 2; used[strings.ToLower(name)]; ordinal++ {
				name = base + strconv.Itoa(ordinal)
			}
			used[strings.ToLower(name)] = true
		}
		if state != nil {
			entry := newEntries[key]
			entry.Name = name
			state.Entries = append(state.Entries, entry)
		}
		for _, d := range peers {
			plan.EmissionHints.transferEquivalent(retained, d)
			aliases[d] = plan.ownedAlias(d.Name(), pkg+"."+name)
		}
		names := map[string]string{}
		dedupOwnedNames(retained, old, pkg+"."+name, packageName+".api.models", names)
		hoistOwnedTypes(retained, names)
		switch d := retained.(type) {
		case *Model:
			d.ClassName = name
		case *StringEnum:
			d.ClassName = name
		default:
			panic("ineligible hoist declaration")
		}
		plan.Models = append(plan.Models, retained)
	}
	for i, d := range plan.Client.Models {
		if alias := aliases[d]; alias != nil {
			plan.Client.Models[i] = alias
		}
	}
	return nil
}

// Opaque code expressions are not symbol graphs. Conservatively skip any tree
// whose expression mentions its old owner instead of rewriting expression text.
func hoistExpressionsIndependent(d Declaration, root string) bool {
	var checkType func(TypeRef) bool
	checkType = func(t TypeRef) bool {
		return !strings.Contains(t.Serializer, root) && (t.Element == nil || checkType(*t.Element))
	}
	switch d := d.(type) {
	case *StringEnum:
		return true
	case *Model:
		for _, p := range d.Properties {
			if !checkType(p.Type) {
				return false
			}
		}
		for _, c := range d.Constants {
			if !checkType(c.Type) || strings.Contains(c.Initializer, root) {
				return false
			}
		}
		for _, n := range d.Nested {
			if !hoistExpressionsIndependent(n, root) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
func hoistOwnedType(t TypeRef, names map[string]string) TypeRef {
	if t.Kind == Named {
		if replacement, ok := names[t.Name]; ok {
			t.Name = replacement
		}
	}
	if t.Element != nil {
		element := hoistOwnedType(*t.Element, names)
		t.Element = &element
	}
	return t // Preserve all original work, nullability, codecs and check pointers.
}
func hoistOwnedTypes(d Declaration, names map[string]string) {
	if m, ok := d.(*Model); ok {
		for i := range m.Properties {
			m.Properties[i].Type = hoistOwnedType(m.Properties[i].Type, names)
		}
		for i := range m.Constants {
			m.Constants[i].Type = hoistOwnedType(m.Constants[i].Type, names)
		}
		for _, n := range m.Nested {
			hoistOwnedTypes(n, names)
		}
	}
}
