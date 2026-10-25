package schema

import (
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Catalog classification needs null intent before forward aliases are bound.
// Walk only original composition/ref edges; nominal property interiors remain
// recursion barriers and actual shape planning still validates their fields.
func (a *analyzer) compositionNullable(v *document.Value, pointer string, depth int, budget *compositionBudget, active map[*document.Value]bool) (bool, *diagnostic.Diagnostic) {
	if err := budget.spend(pointer); err != nil {
		return false, err
	}
	if err := depthCheck(depth, pointer); err != nil {
		return false, err
	}
	if err := object(v, pointer); err != nil {
		return false, err
	}
	if active[v] {
		return false, diagnostic.At("E_REF_CYCLE", "Object composition reference expansion is cyclic.", pointer)
	}
	active[v] = true
	defer delete(active, v)
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return false, err
	}
	if has(v, "$ref") {
		reference := text(get(v, "$ref"))
		location := naming.Pointer(pointer, "$ref")
		declared, err := a.resolveDeclared(reference, location)
		var node *document.Value
		var nodePointer string
		if err == nil {
			// Keep the original named-edge walk, including each alias node's
			// source/work/depth charge and nearest annotation precedence.
			node, nodePointer = declared.Schema, declared.Pointer
		} else {
			// Discovery may additionally follow an original internal edge.
			// Lookup does not admit a type, fabricate a catalog, bind an owner
			// or skip any transparent hops: each original ref node recurses.
			target, failure := a.lookupSchemaReference(reference, location)
			if failure != nil {
				return false, failure
			}
			if target.Catalog != nil {
				return false, err
			}
			node, nodePointer = target.Schema, target.Pointer
		}
		value, err := a.compositionNullable(node, nodePointer, depth+1, budget, active)
		if err != nil {
			return false, err
		}
		if has(v, "nullable") {
			value = nullable
		}
		return value, nil
	}
	if members := get(v, "allOf"); members != nil {
		if members.Kind() != document.Array || len(members.Items()) == 0 {
			return false, invalid("allOf must be a non-empty schema array.", naming.Pointer(pointer, "allOf"))
		}
		value := true
		for i, member := range members.Items() {
			n, err := a.compositionNullable(member, naming.Pointer(naming.Pointer(pointer, "allOf"), strconv.Itoa(i)), depth+1, budget, active)
			if err != nil {
				return false, err
			}
			value = value && n
		}
		if has(v, "nullable") || nullable {
			value = nullable // Existing explicit-root widening/narrowing policy.
		}
		return value, nil
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if members := get(v, key); members != nil {
			if members.Kind() != document.Array || len(members.Items()) == 0 {
				return false, invalid("Composition must be a non-empty schema array.", naming.Pointer(pointer, key))
			}
			for i, member := range members.Items() {
				n, err := a.compositionNullable(member, naming.Pointer(naming.Pointer(pointer, key), strconv.Itoa(i)), depth+1, budget, active)
				if err != nil {
					return false, err
				}
				nullable = nullable || n
			}
			return nullable, nil
		}
	}
	return nullable || kind == "null", nil
}

// Shape borrowing retains the canonical component and complete alias height.
// Nullable ordinary DTOs are legal operands; other nullable scalar/raw roles
// keep their old exclusion. The shape parser validates object-only grammar.
func (a *analyzer) resolveIntersectionMember(reference, source string, depth int) (*resolvedSchema, int, *bool, *diagnostic.Diagnostic) {
	declared, err := a.resolveDeclared(reference, source)
	if err != nil {
		target, failure := a.resolveSchema(reference, source)
		if failure != nil {
			return nil, 0, nil, failure
		}
		return a.internalDTOIntersection(target, source, depth, max(1, target.Steps), target.Nullable)
	}
	if declared.InternalModelTarget != nil {
		target := declared.InternalModelTarget
		c := target.Catalog
		if c == nil {
			// Copy only the transient resolved carrier; admission may fill its
			// physical binding but cannot rewrite the root bridge's own source.
			view := *target
			a.recordBorrowedDTOAliasEdges(reference, source)
			return a.internalDTOIntersection(&view, source, depth, 1+declared.AliasHeight, declared.BindingNullable)
		}
		// An already-bound root bridge can end at an original direct named
		// ordinary DTO. Borrow that real catalog/source, never fabricate one for
		// an internal lexical node. Nearest bridge/member null intent remains
		// a typed view of the original ordinary DTO; other terminal roles stay separate; final nominal validation still requires its actual owner.
		if c == nil || a.components[c.Wire] != c || c.Schema != target.Schema || c.InternalModelTarget != nil || c.BindingTarget != nil ||
			!directInternalModel(c.Schema, c.Pointer) || has(c.Schema, "allOf") || hasObjectCounts(c.Schema) || a.hierarchySources[c.Schema] {
			return nil, 0, nil, schemaReferenceUnsupported(source)
		}
		if _, _, failure := schemaType(c.Schema, c.Pointer); failure != nil {
			return nil, 0, nil, schemaReferenceUnsupported(source)
		}
		steps := 1 + declared.AliasHeight
		if err := depthCheck(depth+steps, source); err != nil {
			return nil, 0, nil, err
		}
		return &resolvedSchema{Catalog: c, Schema: c.Schema, Pointer: c.Pointer}, steps, declared.BindingNullable, nil
	}
	c, steps := declared, 1
	if declared.BindingTarget != nil {
		c, steps = declared.BindingTarget, 1+declared.AliasHeight
	}
	if c.NullableEnum || c.NullableValueClass || c.NullableTemporal || c.NullableBigint || c.NullableBytes || c.NullableCallerNative || c.NullableRaw || c.NullablePrimitive || c.NullableNativeEnum {
		return nil, 0, nil, unsupported(source)
	}
	if c.NullableDTO && hasObjectCounts(c.Schema) {
		return nil, 0, nil, unsupported(source)
	}
	if declared.BindingNullable != nil && *declared.BindingNullable && !nullableDTOCandidate(c.Schema) && !a.containsObjectCompositionUnion(c.Schema, 0, map[*document.Value]bool{}) {
		return nil, 0, nil, unsupported(source)
	}
	if err := depthCheck(depth+steps, source); err != nil {
		return nil, 0, nil, err
	}
	return &resolvedSchema{Catalog: c, Schema: c.Schema, Pointer: c.Pointer}, steps, declared.BindingNullable, nil
}
