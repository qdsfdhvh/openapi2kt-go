package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// A consuming reference's own default may use a disjoint ordinary allOf DTO
// only after the existing final nominal barrier has retained its named Model.
// Source shape alone, a coincident Named spelling or a hierarchy leaf is not
// authority for this role. Pure named aliases are traversed by the caller.
func (a *analyzer) referencedAllOfObjectDefaultNode(v *document.Value, pointer string, ty TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if has(v, "default") {
		return nil, unsupported(naming.Pointer(pointer, "default"))
	}
	target := ty.WithoutRootNull()
	owner := a.componentsByOriginalSchema(v)
	binding := a.deferredNominals[v]
	if !a.nominalsLinked || !composedModelCandidate(v) || a.hierarchySources[v] ||
		a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{}) ||
		target.Kind != Named || target.Element != nil || owner == nil || owner.InternalModelTarget != nil ||
		owner.Pointer != pointer || binding == nil || binding.Schema != v || binding.Owner != owner ||
		binding.Pointer != pointer || binding.Name == "" || binding.Name != target.Name {
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
