package schema

import (
	"slices"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Closed subtype defaults use Json.Default's object discriminator. The final
// named root owns the subtype inventory; ordinary inline content dispatch and
// hierarchy/internal-owned classifiers retain their separate source roles.
func (a *analyzer) closedObjectUnionEncoded(value *document.Value, at string, source *document.Value, sourcePointer string, depth int, union *ObjectUnion) (string, *diagnostic.Diagnostic) {
	owner := a.componentsByOriginalSchema(source)
	binding := a.deferredNominals[source]
	bound := a.nominalsLinked && owner != nil && binding != nil && binding.Schema == source && binding.Owner == owner && binding.Pointer == sourcePointer && binding.Name == a.options.PackageName+".models."+owner.Name
	if !bound {
		bound = a.tupleClosedUnionDefaultAuthorized(source, sourcePointer, union)
	}
	if !bound || owner == nil || a.hierarchySources[source] || owner.Pointer != sourcePointer || union.ClassName != owner.Name ||
		len(union.SerialNames) != len(union.Variants) || has(source, "allOf") {
		return "", unsupported(at)
	}
	// A business field cannot share Json.Default's class discriminator. Check
	// the complete closed inventory used by the generated sealed serializer.
	for _, variant := range union.Variants {
		for _, property := range variant.Properties {
			if property.WireName == "type" {
				return "", unsupported(at)
			}
		}
		for _, constant := range variant.Constants {
			if constant.WireName == "type" {
				return "", unsupported(at)
			}
		}
	}
	if value.Kind() != document.Object {
		return "", invalid("A closed union default must be a JSON object.", at)
	}
	if !has(value, "type") {
		return "", invalid("A closed union default must provide its class discriminator.", at)
	}
	tag, ok := get(value, "type").Text()
	index := slices.Index(union.SerialNames, tag)
	if !ok || index < 0 {
		return "", invalid("A closed union default must select a known subtype.", naming.Pointer(at, "type"))
	}
	variant := union.Variants[index]
	variantSource := a.modelOrigins[variant]
	if variantSource == nil || len(variant.Properties) == 0 {
		return "", unsupported(at)
	}
	node, err := a.objectDefaultNode(variantSource, a.modelPointers[variant], depth, false)
	if err != nil {
		return "", err
	}
	// Treat the already checked backend discriminator as a value-only member.
	// Schema fields and every supplied value still use their original nodes;
	// neither the source tree nor the default object is copied or rewritten.
	node.objectFields = append(node.objectFields, objectDefaultProperty{
		wire: "type", required: true,
		node: &nestedDefaultNode{ty: TypeRef{Kind: Scalar, Name: "kotlin.String"}},
	})
	encoded, err := node.value(value, at)
	if err != nil {
		return "", err
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical union default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	return encoded, nil
}
