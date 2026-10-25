package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/naming"
)

// Only an actual ordinary field's original collection source grants this leaf
// role. The value and helper use its actual nominal codec, never enum spelling.
func (a *analyzer) falseInlineCollectionDefault(f field, source falseInlineCollectionSource, name string) (TypeRef, string, string, *diagnostic.Diagnostic) {
	validation := *a
	validation.falseCollectionDefaultPlanning = &falseCollectionDefaultSource{Schema: f.Schema, Leaf: source.Leaf, Pointer: f.Pointer, LeafPointer: source.Pointer}
	node, err := validation.nestedDefaultSchema(f.Schema, name, f.Pointer, nil, nil, f.Depth, true)
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
