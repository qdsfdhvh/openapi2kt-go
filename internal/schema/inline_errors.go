package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Retain direct constructor-bearing ordinary inline error objects only. Every
// other accepted error schema keeps its existing validated scratch projection.
func inlineErrorModelCandidate(v *document.Value, pointer string) bool {
	if v == nil || v.Kind() != document.Object {
		return false
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "enum", "const", "x-native-type", "x-kotlin-value-class", "discriminator"} {
		if has(v, key) {
			return false
		}
	}
	kind, _, err := schemaType(v, pointer)
	properties := get(v, "properties")
	if err != nil || kind != "object" || properties.Kind() != document.Object {
		return false
	}
	for _, field := range properties.Members() {
		if !has(field.Value, "const") {
			return true
		}
	}
	return false
}

func (a *analyzer) inlineErrorType(v *document.Value, owner, status, pointer string) (TypeRef, []Declaration, *diagnostic.Diagnostic) {
	role := "Error" + strings.ToUpper(status)
	if status == "default" {
		role = "ErrorDefault"
	}
	name, err := a.operationModelName(owner, role, pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	var declarations []Declaration
	ty, err := a.propertyType(v, name, pointer, &declarations, map[string]bool{}, 0)
	if err != nil {
		return TypeRef{}, nil, err
	}
	ty = qualifyOperationType(ty, name, a.options.PackageName+".api.models."+name)
	return ty, declarations, nil
}
