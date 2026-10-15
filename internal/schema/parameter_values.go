package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// parameterType keeps direct/item inline enums owned by the operation. Other
// values follow TypeMapper's structural projection, not a DTO property builder.
func (a *analyzer) parameterType(v *document.Value, pointer string, declarations *[]Declaration) (TypeRef, *diagnostic.Diagnostic) {
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	leaf, leafPointer := v, pointer
	if kind == "array" && !has(v, "$ref") {
		leaf, leafPointer = get(v, "items"), pointer+"/items"
	}
	leafKind := ""
	if leaf != nil && leaf.Kind() == document.Object {
		leafKind, _, err = schemaType(leaf, leafPointer)
		if err != nil {
			return TypeRef{}, err
		}
	}
	context := *a
	context.fieldUnions = RawParameterValueUnions
	var ty TypeRef
	if has(leaf, "enum") && !has(leaf, "$ref") && (leafKind == "string" || leafKind == "") {
		ty, err = context.propertyType(v, "Parameter", pointer, declarations, map[string]bool{}, 0)
	} else {
		ty, err = context.operationStructuralType(v, "Parameter", pointer, declarations, map[string]bool{}, 0)
	}
	if err == nil {
		if failure := a.rejectDTOCountProjectionRole(v, pointer, true); failure != nil {
			return TypeRef{}, failure
		}
		a.warnings = context.warnings
	}
	return ty, err
}
