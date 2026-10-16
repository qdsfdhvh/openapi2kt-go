package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// A consuming reference's own default may use a disjoint ordinary allOf DTO
// only after the existing final nominal barrier has retained its actual Model
// under the original lexical component. Source shape alone, a coincident Named
// spelling or a hierarchy owner is not authority. Aliases are walked by caller.
func (a *analyzer) referencedAllOfObjectDefaultNode(v *document.Value, pointer string, ty TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if has(v, "default") {
		return nil, unsupported(naming.Pointer(pointer, "default"))
	}
	target := ty.WithoutRootNull()
	binding := a.deferredNominals[v]
	owner := a.componentsByOriginalSchema(v)
	if owner == nil && binding != nil {
		owner = binding.Owner
	}
	if !a.nominalsLinked || !composedModelCandidate(v) || a.hierarchySources[v] ||
		a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{}) ||
		target.Kind != Named || target.Element != nil || owner == nil || owner.InternalModelTarget != nil ||
		a.components[owner.Wire] != owner || a.hierarchySources[owner.Schema] ||
		binding == nil || binding.Schema != v || binding.Owner != owner ||
		binding.Pointer != pointer || binding.Name == "" || binding.Name != target.Name {
		return nil, unsupported(naming.Pointer(pointer, "allOf"))
	}
	if pointer != owner.Pointer {
		// Recheck an original schema path, not a generated classifier spelling.
		// Lookup admits no new types; the final binding already owns the Model.
		source, err := a.lookupSchemaReference("#"+pointer, pointer)
		if err != nil || source.Schema != v || source.Pointer != pointer || source.SourceCatalog != owner {
			return nil, unsupported(naming.Pointer(pointer, "allOf"))
		}
	} else if owner.Schema != v {
		return nil, unsupported(naming.Pointer(pointer, "allOf"))
	}
	validation := *a
	validation.warnings = nil // Type planning owns original source warnings.
	node, err := validation.ordinaryAllOfDefaultNode(v, pointer, depth)
	if err != nil {
		return nil, err
	}
	// The shared reference planner, before optional presence, is the nullable
	// authority. A consumer or nearest alias can independently narrow the root.
	node.ty = ty
	return node, nil
}
