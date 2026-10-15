package schema

import (
	"sort"
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Only aliases used as members propagate their target's nullable intent. A
// direct child remains a physical inheritance edge under the intrinsic contract.
// Build reverse dependencies from original named nodes, then propagate each true
// fact once. No JSON expansion, repeated fixed-point graph scan or guessed owner.
func (a *analyzer) discoverHierarchyNullability() *diagnostic.Diagnostic {
	ordered := []string{}
	relevant := false
	for key, c := range a.components {
		if a.hierarchySources[c.Schema] {
			ordered = append(ordered, key)
			_, nullable, err := schemaType(c.Schema, c.Pointer)
			if err != nil {
				return err
			}
			c.NullableHierarchy = nullable
			relevant = relevant || nullable
			branches := get(c.Schema, "oneOf")
			if branches.Kind() != document.Array {
				branches = get(c.Schema, "anyOf")
			}
			if branches.Kind() == document.Array {
				for _, branch := range branches.Items() {
					relevant = relevant || text(get(branch, "type")) == "null"
					if flag := nullableOverride(branch); flag != nil {
						relevant = relevant || *flag
					}
				}
			}
		}
		if c.Reference != "" {
			if flag := nullableOverride(c.Schema); flag != nil {
				relevant = relevant || *flag
			}
		}
	}
	if !relevant {
		return nil
	} // Preserve the established non-null graph gate.
	sort.Strings(ordered)
	reverse := map[*component][]*component{}
	work := 0
	consume := func(pointer string) *diagnostic.Diagnostic {
		work++
		if work > 16384 {
			return hierarchyError("E_SCHEMA_LIMIT", pointer)
		}
		return nil
	}
	for _, key := range ordered {
		c := a.components[key]
		keyword := "oneOf"
		branches := get(c.Schema, keyword)
		if branches.Kind() != document.Array {
			keyword = "anyOf"
			branches = get(c.Schema, keyword)
		}
		if branches.Kind() != document.Array {
			continue
		}
		for i, branch := range branches.Items() {
			pointer := naming.Pointer(naming.Pointer(c.Pointer, keyword), strconv.Itoa(i))
			if err := consume(pointer); err != nil {
				return err
			}
			if text(get(branch, "type")) == "null" {
				c.NullableHierarchy, c.MemberNullableHierarchy = true, true
				continue
			}
			if flag := nullableOverride(branch); flag != nil {
				if *flag {
					c.NullableHierarchy, c.MemberNullableHierarchy = true, true
				}
				continue
			}
			ref, ok := get(branch, "$ref").Text()
			if !ok {
				continue
			}
			target, err := a.resolveDeclared(ref, pointer+"/$ref")
			if err != nil || target.Reference == "" {
				continue
			}
			seen := map[*component]bool{}
			for !seen[target] && len(seen) <= MaxSchemaDepth {
				if err := consume(pointer + "/$ref"); err != nil {
					return err
				}
				seen[target] = true
				if target.Reference == "" {
					if a.hierarchySources[target.Schema] {
						reverse[target] = append(reverse[target], c)
					}
					break
				}
				if flag := nullableOverride(target.Schema); flag != nil {
					if *flag {
						c.NullableHierarchy, c.MemberNullableHierarchy = true, true
					}
					break
				}
				next, err := a.resolveDeclared(target.Reference, target.Pointer+"/$ref")
				if err != nil {
					break
				}
				target = next
			}
		}
	}
	pending := []*component{}
	for _, key := range ordered {
		if c := a.components[key]; c.NullableHierarchy {
			pending = append(pending, c)
		}
	}
	for len(pending) > 0 {
		c := pending[0]
		pending = pending[1:]
		for _, parent := range reverse[c] {
			if err := consume(parent.Pointer); err != nil {
				return err
			}
			parent.MemberNullableHierarchy = true
			if !parent.NullableHierarchy {
				parent.NullableHierarchy = true
				pending = append(pending, parent)
			}
		}
	}
	return nil
}
