package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Detect only collection references within an object-default property role.
// No new consuming-field/component/child-annotation role is inferred here.
func (a *analyzer) objectDefaultCollectionGraph(v *document.Value, depth int) bool {
	if v == nil || depth > MaxSchemaDepth {
		return false
	}
	if has(v, "$ref") {
		for steps := depth; steps <= MaxSchemaDepth && has(v, "$ref"); steps++ {
			source, _, err := a.defaultSource(text(get(v, "$ref")), "")
			if err != nil {
				return false
			}
			v = source
		}
		return collectionAliasSchema(v)
	}
	kind, _, _ := schemaType(v, "")
	if kind == "array" {
		return a.objectDefaultCollectionGraph(get(v, "items"), depth+1)
	}
	if dictionarySchema(v) {
		return a.objectDefaultCollectionGraph(get(v, "additionalProperties"), depth+1)
	}
	return false
}

type objectDefaultGraphUse struct {
	Schema  *document.Value
	Pointer string
	Type    TypeRef
	Depth   int
	Own     bool
}

// Graph nodes borrow source fields and scalar checks. Real typed reference
// binding stays with the existing resolver, including forward alias backing.
// A final pass pairs each reference with that linked backing before emission.
func (a *analyzer) objectDefaultGraphNode(v *document.Value, pointer string, depth int, own bool, active map[*document.Value]bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	if active[v] {
		return nil, diagnostic.At("E_REF_CYCLE", "Object default collection source graph is cyclic.", pointer)
	}
	active[v] = true
	defer delete(active, v)
	a, err := a.objectDefaultStep(pointer)
	if err != nil {
		return nil, err
	}
	// A marked original collection/alias factory keeps its annotation while
	// consumers borrow its shape. The final graph-use pass still proves each
	// actual linked backing and retained declaration/check object.
	own = own || a.independentLexicalEnumCollectionDefault(v, pointer) || a.independentReferencedEnumCollectionDefault(v, pointer) || a.independentEnumComponentDefault(v, pointer)
	if has(v, "$ref") {
		if !a.objectDefaultCollectionGraph(v, depth) {
			// Existing leaf/DTO/allOf member roles retain their original authority,
			// chain grammar, declaration ownership and nearest-nullable checks.
			keys := []string{"$ref", "nullable"}
			if own {
				keys = append(keys, "default")
			}
			if err := checkKeys(v, keys, pointer); err != nil {
				return nil, err
			}
			return a.objectDefaultPropertyReference(v, pointer, depth)
		}
		keys := []string{"$ref", "nullable"}
		if own {
			keys = append(keys, "default")
		}
		if err := checkKeys(v, keys, pointer); err != nil {
			return nil, err
		}
		reference := text(get(v, "$ref"))
		target, err := a.resolveSchema(reference, pointer)
		if err != nil {
			return nil, err
		}
		ty, err := a.referenceType(target, nullableOverride(v), depth, pointer)
		if err != nil {
			return nil, err
		}
		source, p, err := a.defaultSource(reference, pointer)
		if err != nil {
			return nil, err
		}
		owner := a.componentsByOriginalSchema(source)
		if owner == nil {
			if err := a.internalDefaultMemberSource(source, p, reference); err != nil {
				return nil, err
			}
		} else if owner.Pointer != p || reference != "#"+p || a.components[owner.Wire] != owner || a.hierarchySources[source] {
			return nil, unsupported(naming.Pointer(pointer, "$ref"))
		}
		node, err := a.objectDefaultGraphNode(source, p, depth+1, false, active)
		if err != nil {
			return nil, err
		}
		// Preserve the parsed element tree for default equality while the shared
		// alias backing links forward; only the actual root nullable view applies.
		node.ty = node.ty.WithoutRootNull()
		if ty.Kind == Nullable {
			node.ty = node.ty.Null()
		}
		if a.objectDefaultGraphUses == nil {
			a.objectDefaultGraphUses = &[]objectDefaultGraphUse{}
		}
		*a.objectDefaultGraphUses = append(*a.objectDefaultGraphUses, objectDefaultGraphUse{v, pointer, ty, depth, own})
		return node, nil
	}
	if leaf := inlineStringEnumBoundsLeaf(v, 1); a.options.Validation && leaf != nil && a.componentRoot(v, pointer) {
		return a.nestedDefaultSchemaWithEnumBounds(v, "Validation", pointer, nil, nil, depth, own, leaf)
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	edge, typeKind := "", List
	keys := []string{"type", "nullable", "items"}
	if kind == "array" {
		edge = "items"
	} else if dictionarySchema(v) {
		edge, typeKind = "additionalProperties", Map
		keys = []string{"type", "nullable", "properties", "required", edge}
	}
	if edge == "" {
		if a.checkedEnumReferenceCandidate(v) {
			return a.nestedDefaultSchemaWithEnumBounds(v, "Validation", pointer, nil, nil, depth, own || a.independentCheckedEnumFieldDefault(v, pointer) || a.independentEnumComponentDefault(v, pointer), v)
		}
		if has(v, "allOf") {
			return a.inlineAllOfObjectDefaultMember(v, pointer, depth, own)
		}
		if objectDefaultCandidateSchema(v) {
			return a.objectDefaultNode(v, pointer, depth, own)
		}
		return a.nestedDefaultSchema(v, "Validation", pointer, nil, nil, depth, own)
	}
	keys = a.collectionKeys(keys)
	if own {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	if typeKind == Map {
		required, err := objectDefaultRequiredNames(v, pointer)
		if err != nil {
			return nil, err
		}
		if len(required) != 0 {
			return nil, invalid("Dictionary schemas cannot require undeclared fixed properties.", naming.Pointer(pointer, "required"))
		}
	}
	child, err := a.objectDefaultGraphNode(get(v, edge), naming.Pointer(pointer, edge), depth+1, false, active)
	if err != nil {
		return nil, err
	}
	ty, err := a.checkedCollectionRole(TypeRef{Kind: typeKind, Element: &child.ty}, v, pointer, child.object)
	if err != nil {
		return nil, err
	}
	if _, nullable, _ := schemaType(v, pointer); nullable {
		ty = ty.Null()
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, element: child, ty: ty, dtoUniqueKeys: a.dtoUniqueKeys}, nil
}

func (a *analyzer) validateObjectDefaultGraphUses() *diagnostic.Diagnostic {
	if a.objectDefaultGraphUses == nil {
		return nil
	}
	for _, use := range *a.objectDefaultGraphUses {
		if _, err := a.referencedDefaultSchemaRole(use.Schema, use.Pointer, use.Type, use.Depth, use.Own, referencedCollectionOwnDefault); err != nil {
			return err
		}
	}
	return nil
}
