package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Route collection roots with an inline tree and an eligible allOf element.
// A named/internal element reference keeps its original hop. A bare inline allOf
// leaf owns its own original node, matching the consuming-field and inline
// collection-member roles; the enclosing shared type plan still owns its
// constructor. Internal element-reference binding can be pending when the
// enclosing object is planned, so that role stays with the shared final linker.
func (a *analyzer) namedAllOfCollectionMemberElement(v *document.Value) *document.Value {
	leaf := a.inlineReferencedAllOfElement(v)
	if leaf == nil {
		// A bare inline allOf leaf has no reference edge of its own. The shared
		// type plan must already retain a unique nonempty constructor under this
		// original element; this detector only scopes the source permission.
		if bare := inlineAllOfDefaultElement(v); bare != nil && composedModelCandidate(bare) {
			return bare
		}
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

// Admission remains with the shared type planner. Original named/internal ref
// hops may precede an original named or internal collection entity.
func (a *analyzer) referencedCollectionAllOfMember(v *document.Value) bool {
	for edges := 0; edges <= MaxSchemaDepth && has(v, "$ref"); edges++ {
		source, p, err := a.defaultSource(text(get(v, "$ref")), "")
		if err != nil {
			return false
		}
		owner := a.componentsByOriginalSchema(source)
		if owner != nil && owner.Pointer != p {
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
	// Prefer the first named view; without one, use the original internal
	// container's nullable intent. Nearer ref/consumer overrides win. This reads
	// existing metadata only; the source walk still checks every hop.
	if _, _, err := schemaType(v, pointer); err != nil {
		return nil, err
	}
	intent := nullableOverride(v)
	nullable, namedView, internalPrefix := false, false, false
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
		if owner == nil {
			if err := a.internalDefaultMemberSource(v, pointer, reference); err != nil {
				return nil, err
			}
		} else if owner.Pointer != pointer || reference != "#"+pointer || a.components[owner.Wire] != owner || a.hierarchySources[v] {
			return nil, unsupported(naming.Pointer(pointer, "$ref"))
		}
		if !namedView {
			if owner == nil {
				internalPrefix = true
				if !has(v, "$ref") {
					_, nullable, err = schemaType(v, pointer)
					if err != nil {
						return nil, err
					}
					if intent != nil {
						nullable = *intent
					}
				} else if intent == nil {
					intent = nullableOverride(v)
				}
			} else {
				namedView = true
				nullable = a.componentType(owner).Kind == Nullable
				bridged := owner.InternalModelTarget != nil && (owner.InternalModelTarget.CanonicalCollection != nil || owner.InternalModelTarget.BorrowedCollection != nil)
				// An internal prefix provides the existing narrowable collection
				// binding. Direct named Maps retain their historical false rule.
				if intent != nil && (*intent || internalPrefix || owner.BindingTarget != nil || bridged || nullableArraySchema(owner.Schema)) {
					nullable = *intent
				}
			}
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
