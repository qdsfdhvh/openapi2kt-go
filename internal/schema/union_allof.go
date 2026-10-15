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
func (a *analyzer) objectCompositionUnion(v *document.Value, pointer string, depth int) (*unionView, *diagnostic.Diagnostic) {
	if !has(v, "allOf") || has(v, "$ref") {
		return nil, nil
	}
	// Do not change ordinary allOf diagnostics, traversal order or budgets.
	// Detection borrows the original catalog without expanding its fields.
	if !a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{}) {
		return nil, nil
	}
	_, rootNullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	root := v
	budget := &compositionBudget{remaining: 16384, enabled: true}
	var alternatives func(*document.Value, string, int, []*document.Value) ([]unionBranch, bool, *diagnostic.Diagnostic)
	alternatives = func(v *document.Value, p string, d int, stack []*document.Value) ([]unionBranch, bool, *diagnostic.Diagnostic) {
		if err := budget.spend(p); err != nil {
			return nil, false, err
		}
		if err := depthCheck(d, p); err != nil {
			return nil, false, err
		}
		if slices.Contains(stack, v) {
			return nil, false, diagnostic.At("E_REF_CYCLE", "Object union/allOf composition graph is cyclic.", p)
		}
		stack = append(stack, v)
		if err := object(v, p); err != nil {
			return nil, false, err
		}
		if has(v, "$ref") {
			if err := checkKeys(v, []string{"$ref", "nullable"}, p); err != nil {
				return nil, false, err
			}
			_, nullable, err := schemaType(v, p)
			if err != nil || nullable {
				if err == nil {
					err = unsupported(p)
				}
				return nil, false, err
			}
			ref, ok := get(v, "$ref").Text()
			if !ok {
				return nil, false, invalid("$ref must be a string.", p)
			}
			c, steps, err := a.resolveNominal(ref, p, d)
			if err != nil {
				return nil, false, err
			}
			if slices.Contains(stack, c.Schema) {
				return nil, false, diagnostic.At("E_REF_CYCLE", "Object union/allOf composition graph is cyclic.", naming.Pointer(p, "$ref"))
			}
			return alternatives(c.Schema, c.Pointer, d+steps, stack)
		}
		union, err := readUnion(v, p)
		if err != nil {
			return nil, false, err
		}
		if union != nil {
			if union.Nullable {
				return nil, false, unsupported(union.Pointer)
			}
			// Preserve the established location for a wholly primitive union
			// inside allOf; this object profile does not widen that grammar.
			primitiveOnly := true
			for _, branch := range union.Branches {
				kind, _, e := schemaType(branch.Schema, branch.Pointer)
				primitiveOnly = primitiveOnly && e == nil && !has(branch.Schema, "$ref") && slices.Contains([]string{"string", "integer", "number", "boolean"}, kind)
			}
			if primitiveOnly {
				return nil, false, unsupported(union.Pointer)
			}
			var out []unionBranch
			for _, branch := range union.Branches {
				parts, _, err := alternatives(branch.Schema, branch.Pointer, d+1, stack)
				if err != nil {
					return nil, false, err
				}
				if len(out)+len(parts) > MaxMembers {
					return nil, false, diagnostic.At("E_UNION_LIMIT", "Composed object union exceeds the supported 240-branch limit.", union.Pointer)
				}
				out = append(out, parts...)
			}
			return out, true, nil
		}
		members := get(v, "allOf")
		if members == nil {
			s, err := a.objectShape(v, p, d)
			if err != nil {
				return nil, false, err
			}
			if s.Nullable != nil && *s.Nullable {
				return nil, false, unsupported(p)
			}
			return []unionBranch{{Schema: v, Pointer: p, Shape: s, Depth: d}}, false, nil
		}
		if members.Kind() != document.Array || len(members.Items()) == 0 {
			return nil, false, invalid("allOf must be a non-empty schema array.", naming.Pointer(p, "allOf"))
		}
		// Validate original root grammar/own fields through the shape parser;
		// member expansion is performed below without rewriting the source.
		own, err := a.objectOwnShape(v, p, d)
		if err != nil {
			return nil, false, err
		}
		if own.Nullable != nil && *own.Nullable {
			if v != root || len(stack) != 1 {
				return nil, false, unsupported(p)
			}
			// Root null intent belongs to the union's typed view/carrier,
			// never to the physical constructor of every product variant.
			own.Nullable = nil
		}
		product := []unionBranch{{Schema: v, Pointer: p, Shape: &shape{Required: map[string]bool{}}, Depth: d}}
		found := false
		for i, member := range members.Items() {
			at := naming.Pointer(naming.Pointer(p, "allOf"), strconv.Itoa(i))
			parts, containsUnion, err := alternatives(member, at, d+1, stack)
			if err != nil {
				return nil, false, err
			}
			found = found || containsUnion
			if len(parts) > 0 && len(product) > MaxMembers/len(parts) {
				return nil, false, diagnostic.At("E_UNION_LIMIT", "Composed object union exceeds the supported 240-branch limit.", at)
			}
			var next []unionBranch
			for _, left := range product {
				for _, right := range parts {
					merged, err := mergeCompositionShapes(left.Shape, right.Shape, budget, at)
					if err != nil {
						return nil, false, err
					}
					next = append(next, unionBranch{Schema: v, Pointer: p, Shape: merged, Depth: d})
				}
			}
			product = next
		}
		for i := range product {
			merged, err := mergeCompositionShapes(product[i].Shape, own, budget, p)
			if err != nil {
				return nil, false, err
			}
			merged.Compositions = append(merged.Compositions, naming.Pointer(p, "allOf"))
			product[i].Shape = merged
		}
		return product, found, nil
	}
	branches, found, err := alternatives(v, pointer, depth, nil)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &unionView{Branches: branches, Pointer: naming.Pointer(pointer, "allOf"), Nullable: rootNullable}, nil
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
