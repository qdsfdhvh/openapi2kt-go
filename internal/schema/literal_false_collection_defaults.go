package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func (a *analyzer) literalFalseCollectionDefaultLeaf(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if a.literalFalseCollectionPlanning == nil || !a.literalFalseCollectionPlanning.Source.OwnDefault {
		return a.literalFalseObjectDefaultNode(v, pointer, depth)
	}
	ty, err := a.literalFalseCollectionElementType(v, pointer, depth)
	if err != nil || ty == nil {
		return nil, err
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: *ty, uninhabited: true}, nil
}
func (a *analyzer) literalFalseCollectionDefault(f field, name string) (TypeRef, string, string, *diagnostic.Diagnostic) {
	node, err := a.nestedDefaultSchema(f.Schema, name, f.Pointer, nil, nil, f.Depth, true)
	if err != nil {
		return TypeRef{}, "", "", err
	}
	value := get(f.Schema, "default")
	at := naming.Pointer(f.Pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return TypeRef{}, "", "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return TypeRef{}, "", "", err
	}
	if len(encoded) > maxNestedDefaultBytes {
		return TypeRef{}, "", "", diagnostic.At("E_MODEL_LIMIT", "Canonical nested default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	initializer, err := referencedDefaultInitializer(node, node.ty, value, encoded, at)
	return node.ty, initializer, encoded, err
}
