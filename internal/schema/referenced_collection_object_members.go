package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Route only named collection roots/aliases with an inline container tree and
// a reference to a named allOf terminal. Internal terminal binding can still be
// pending when the enclosing object is planned, so that role stays separate.
func (a *analyzer) namedAllOfCollectionMemberElement(v *document.Value) *document.Value {
	leaf := a.inlineReferencedAllOfElement(v)
	if leaf == nil {
		return nil
	}
	v = leaf
	for edges := 0; edges <= MaxSchemaDepth && has(v, "$ref"); edges++ {
		source, pointer, err := a.defaultSource(text(get(v, "$ref")), "")
		if err != nil {
			return nil
		}
		v = source
		if !has(v, "$ref") {
			owner := a.componentsByOriginalSchema(v)
			if owner != nil && owner.Pointer == pointer && has(v, "allOf") {
				return leaf
			}
			return nil
		}
	}
	return nil
}

// Admission remains with the shared type planner.
func (a *analyzer) referencedCollectionAllOfMember(v *document.Value) bool {
	for edges := 0; edges <= MaxSchemaDepth && has(v, "$ref"); edges++ {
		source, p, err := a.defaultSource(text(get(v, "$ref")), "")
		if err != nil {
			return false
		}
		owner := a.componentsByOriginalSchema(source)
		if owner == nil || owner.Pointer != p {
			return false
		}
		v = source
		if !has(v, "$ref") {
			return a.namedAllOfCollectionMemberElement(v) != nil
		}
	}
	return false
}

func (a *analyzer) referencedCollectionAllOfMemberNode(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	// Root nullability comes from the existing named component view, including
	// alias intent and the historical direct nullable Map false-override rule.
	if _, _, err := schemaType(v, pointer); err != nil {
		return nil, err
	}
	owner, err := a.resolveDeclared(text(get(v, "$ref")), pointer)
	if err != nil {
		return nil, err
	}
	nullable := a.componentType(owner).Kind == Nullable
	if intent := nullableOverride(v); intent != nil && (*intent || owner.BindingTarget != nil || nullableArraySchema(owner.Schema)) {
		nullable = *intent
	}
	reference := text(get(v, "$ref"))
	for {
		source, p, err := a.defaultSource(reference, pointer)
		if err != nil {
			return nil, err
		}
		v, pointer = source, p
		depth++
		if err := depthCheck(depth, pointer); err != nil {
			return nil, err
		}
		owner := a.componentsByOriginalSchema(v)
		if owner == nil || owner.Pointer != pointer || reference != "#"+pointer || a.components[owner.Wire] != owner || a.hierarchySources[v] {
			return nil, unsupported(naming.Pointer(pointer, "$ref"))
		}
		if !has(v, "$ref") {
			leaf := a.namedAllOfCollectionMemberElement(v)
			if leaf == nil {
				return nil, unsupported(pointer)
			}
			validation := *a
			validation.objectDefaultAllOfMemberElement = leaf
			node, err := validation.objectDefaultNode(v, pointer, depth, false)
			if err != nil {
				return nil, err
			}
			node.ty = node.ty.WithoutRootNull()
			if nullable {
				node.ty = node.ty.Null()
			}
			return node, nil
		}
		a, err = a.objectDefaultStep(pointer)
		if err != nil {
			return nil, err
		}
		if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
			return nil, err
		}
		if _, _, err := schemaType(v, pointer); err != nil {
			return nil, err
		}
		reference = text(get(v, "$ref"))
	}
}
