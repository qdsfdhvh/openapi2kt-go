package schema

import (
	"slices"
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// objectCompositionUnion returns alternatives only when an allOf source graph
// contains an object union. Ordinary allOf and ordinary unions stay on their
// established paths. Every alternative borrows original source fields/shapes.
type compositionAlternatives struct {
	Branches      []unionBranch
	ContainsUnion bool
	Nullable      bool
}

func (a *analyzer) objectCompositionUnion(v *document.Value, pointer string, depth int) (*unionView, *diagnostic.Diagnostic) {
	if !has(v, "allOf") || has(v, "$ref") {
		return nil, nil
	}
	if !a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{}) {
		return nil, nil
	}
	budget := &compositionBudget{remaining: 16384, enabled: true}
	var alternatives func(*document.Value, string, int, []*document.Value) (*compositionAlternatives, *diagnostic.Diagnostic)
	alternatives = func(v *document.Value, p string, d int, stack []*document.Value) (*compositionAlternatives, *diagnostic.Diagnostic) {
		if err := budget.spend(p); err != nil {
			return nil, err
		}
		if err := depthCheck(d, p); err != nil {
			return nil, err
		}
		if slices.Contains(stack, v) {
			return nil, diagnostic.At("E_REF_CYCLE", "Object union/allOf composition graph is cyclic.", p)
		}
		stack = append(stack, v)
		if err := object(v, p); err != nil {
			return nil, err
		}
		_, ownNullable, err := schemaType(v, p)
		if err != nil {
			return nil, err
		}
		if has(v, "$ref") {
			if err := checkKeys(v, []string{"$ref", "nullable"}, p); err != nil {
				return nil, err
			}
			ref, ok := get(v, "$ref").Text()
			if !ok {
				return nil, invalid("$ref must be a string.", p)
			}
			c, steps, intent, err := a.resolveIntersectionMember(ref, p, d)
			if err != nil {
				return nil, err
			}
			if slices.Contains(stack, c.Schema) {
				return nil, diagnostic.At("E_REF_CYCLE", "Object union/allOf composition graph is cyclic.", naming.Pointer(p, "$ref"))
			}
			out, err := alternatives(c.Schema, c.Pointer, d+steps, stack)
			if err != nil {
				return nil, err
			}
			if own := nullableOverride(v); own != nil {
				intent = own
			}
			if intent != nil {
				out.Nullable = *intent
			}
			return out, nil
		}
		union, err := a.readUnion(v, p)
		if err != nil {
			return nil, err
		}
		if union != nil {
			primitiveOnly := true
			for _, branch := range union.Branches {
				kind, _, e := schemaType(branch.Schema, branch.Pointer)
				primitiveOnly = primitiveOnly && e == nil && !has(branch.Schema, "$ref") && slices.Contains([]string{"string", "integer", "number", "boolean"}, kind)
			}
			if primitiveOnly {
				return nil, unsupported(union.Pointer)
			}
			out := &compositionAlternatives{ContainsUnion: true, Nullable: union.Nullable}
			nullSources := 0
			keyword := "anyOf"
			if union.OneOf {
				keyword = "oneOf"
			}
			for _, member := range get(v, keyword).Items() {
				kind, _, _ := schemaType(member, p)
				if kind == "null" {
					nullSources++
				}
			}
			for _, branch := range union.Branches {
				parts, err := alternatives(branch.Schema, branch.Pointer, d+1, stack)
				if err != nil {
					return nil, err
				}
				if len(out.Branches)+len(parts.Branches) > MaxMembers {
					return nil, diagnostic.At("E_UNION_LIMIT", "Composed object union exceeds the supported 240-branch limit.", union.Pointer)
				}
				out.Branches = append(out.Branches, parts.Branches...)
				out.Nullable = out.Nullable || parts.Nullable
				if parts.Nullable {
					nullSources++
				}
			}
			if union.OneOf && nullSources > 1 {
				a.warn("W_ONEOF_NULL_OVERLAP", "Nullable folding does not enforce the exact oneOf match count for overlapping null branches.", union.Pointer)
			}
			return out, nil
		}
		members := get(v, "allOf")
		if members == nil {
			shape, err := a.objectShape(v, p, d)
			if err != nil {
				return nil, err
			}
			nullable := shape.Nullable != nil && *shape.Nullable
			physical := *shape
			physical.Nullable = nil
			return &compositionAlternatives{Branches: []unionBranch{{Schema: v, Pointer: p, Shape: &physical, Depth: d}}, Nullable: nullable}, nil
		}
		if members.Kind() != document.Array || len(members.Items()) == 0 {
			return nil, invalid("allOf must be a non-empty schema array.", naming.Pointer(p, "allOf"))
		}
		own, err := a.objectOwnShape(v, p, d)
		if err != nil {
			return nil, err
		}
		// Null intent belongs to the resulting typed view/carrier. Every
		// physical product alternative retains its non-null constructor.
		own.Nullable = nil
		out := &compositionAlternatives{Branches: []unionBranch{{Schema: v, Pointer: p, Shape: &shape{Required: map[string]bool{}}, Depth: d}}, Nullable: true}
		for i, member := range members.Items() {
			at := naming.Pointer(naming.Pointer(p, "allOf"), strconv.Itoa(i))
			parts, err := alternatives(member, at, d+1, stack)
			if err != nil {
				return nil, err
			}
			out.ContainsUnion = out.ContainsUnion || parts.ContainsUnion
			out.Nullable = out.Nullable && parts.Nullable
			if len(parts.Branches) > 0 && len(out.Branches) > MaxMembers/len(parts.Branches) {
				return nil, diagnostic.At("E_UNION_LIMIT", "Composed object union exceeds the supported 240-branch limit.", at)
			}
			var next []unionBranch
			for _, left := range out.Branches {
				for _, right := range parts.Branches {
					merged, err := mergeCompositionShapes(left.Shape, right.Shape, budget, at)
					if err != nil {
						return nil, err
					}
					next = append(next, unionBranch{Schema: v, Pointer: p, Shape: merged, Depth: d})
				}
			}
			out.Branches = next
		}
		for i := range out.Branches {
			merged, err := mergeCompositionShapes(out.Branches[i].Shape, own, budget, p)
			if err != nil {
				return nil, err
			}
			merged.Compositions = append(merged.Compositions, naming.Pointer(p, "allOf"))
			out.Branches[i].Shape = merged
		}
		if has(v, "nullable") || ownNullable {
			out.Nullable = ownNullable
		}
		return out, nil
	}
	out, err := alternatives(v, pointer, depth, nil)
	if err != nil {
		return nil, err
	}
	if !out.ContainsUnion {
		return nil, nil
	}
	return &unionView{Branches: out.Branches, Pointer: naming.Pointer(pointer, "allOf"), Nullable: out.Nullable}, nil
}

func (a *analyzer) containsObjectCompositionUnion(v *document.Value, depth int, active map[*document.Value]bool) bool {
	if v == nil || depth > MaxSchemaDepth || active[v] || len(active) >= 16384 {
		return false
	}
	if has(v, "oneOf") || has(v, "anyOf") {
		return true
	}
	active[v] = true // Visit shared descendants once; detection emits no shapes.
	if has(v, "$ref") {
		c, err := a.resolveDeclared(text(get(v, "$ref")), "")
		if err != nil || c.InternalModelTarget != nil {
			return false
		}
		if c.BindingTarget != nil {
			c = c.BindingTarget
		}
		return a.containsObjectCompositionUnion(c.Schema, depth+1, active)
	}
	members := get(v, "allOf")
	if members.Kind() != document.Array {
		return false
	}
	for _, member := range members.Items() {
		if a.containsObjectCompositionUnion(member, depth+1, active) {
			return true
		}
	}
	return false
}

func mergeCompositionShapes(left, right *shape, budget *compositionBudget, p string) (*shape, *diagnostic.Diagnostic) {
	out := &shape{Required: map[string]bool{}, Nullable: left.Nullable}
	positions := map[string]int{}
	for _, s := range []*shape{left, right} {
		if out.Nullable == nil {
			out.Nullable = s.Nullable
		}
		for _, wire := range s.RequiredOrder {
			if !out.Required[wire] {
				out.Required[wire] = true
				out.RequiredOrder = append(out.RequiredOrder, wire)
			}
		}
		out.Discarded = append(out.Discarded, s.Discarded...)
		out.Additional = append(out.Additional, s.Additional...)
		out.Compositions = append(out.Compositions, s.Compositions...)
		for _, f := range s.Fields {
			if err := out.insert(positions, f, budget, p); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
