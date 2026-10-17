package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Route collection roots with an inline tree and an eligible allOf element.
// A bare inline allOf leaf owns its own original node, matching the consuming
// field and inline collection-member roles. A named element reference keeps its
// original hop. An internal allOf terminal has no lexical component of its own;
// the shared resolver seam binds that original node on demand so the eager
// member walk can see its identity, and the final retained-declaration linker
// still authorizes the declaration before any artifact emits.
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
		reference := text(get(v, "$ref"))
		source, pointer, err := a.defaultSource(reference, "")
		if err != nil {
			return nil
		}
		v = source
		if !has(v, "$ref") {
			owner := a.componentsByOriginalSchema(v)
			if owner != nil && owner.Pointer == pointer && has(v, "allOf") {
				return leaf
			}
			// An original internal allOf terminal has no lexical component. Bind
			// that original node through the shared resolver seam, then require the
			// same node, Pointer and lexical SourceCatalog the deferred member walk
			// checks. A missing, copied or ambiguous declaration still fails there.
			if owner == nil && composedModelCandidate(v) {
				if _, err := a.resolveSchema(reference, ""); err != nil {
					return nil
				}
				if a.internalAllOfDefaultMemberSource(v, pointer, reference) == nil {
					return leaf
				}
			}
			return nil
		}
	}
	return nil
}

// plainDTOCollectionMember reports whether an original collection entity's
// inline element tree resolves to a plain ordinary DTO rather than an allOf
// constructor. Such a collection uses the same member profile as a direct
// plain-DTO collection, so an alias or internal bridge to it does not fall
// through to the allOf terminal role.
func (a *analyzer) plainDTOCollectionMember(v *document.Value) bool {
	if v == nil || has(v, "$ref") {
		return false
	}
	kind, _, err := schemaType(v, "")
	if err != nil {
		return false
	}
	if kind != "array" && !dictionarySchema(v) {
		return false
	}
	if a.namedAllOfCollectionMemberElement(v) != nil {
		return false
	}
	return a.objectDefaultCandidate(v)
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
			return a.namedAllOfCollectionMemberElement(v) != nil || a.plainDTOCollectionMember(v)
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
			validation := *a
			if leaf != nil {
				validation.objectDefaultAllOfMemberElement = leaf
			} else if !a.plainDTOCollectionMember(v) {
				return nil, unsupported(pointer)
			}
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
