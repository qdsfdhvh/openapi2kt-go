package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

func (a *analyzer) tupleMetadataEnumDefaultElement(v *document.Value, pointer string, actual TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if _, err := buildStringEnumKeysFor("TupleMember", v, pointer, false, a); err != nil {
		return nil, err
	}
	ty := defaultBacking(actual)
	if ty.WithoutRootNull().Kind != Named || ty.WithoutRootNull().Element != nil {
		return nil, schemaReferenceUnsupported(pointer)
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, stringEnum: true}, nil
}

func (a *analyzer) tupleMetadataEnumFieldDefault(v *document.Value, pointer, name string, decision *arrayItemsDecision, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	var scratch []Declaration
	if nested == nil {
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
	element, err := a.collectionElementType(decision.Selected, name+"Inner", decision.Pointer, nested, used, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	child, err := a.tupleMetadataEnumDefaultElement(decision.Selected, decision.Pointer, element, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	return a.tupleElementFieldDefault(v, pointer, element, child, decision)
}
