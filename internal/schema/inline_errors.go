package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Selected inline JSON errors use the same accepted shared-model types as
// component fields. References retain their declared component identity.
func inlineErrorModelCandidate(v *document.Value, _ string) bool {
	return v != nil && !has(v, "$ref")
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
	// Container roots can emit peer item/value declarations instead of a root
	// classifier. Qualify each actual declaration, never a manufactured wrapper.
	for _, d := range declarations {
		ty = qualifyOperationType(ty, d.Name(), a.options.PackageName+".api.models."+d.Name())
	}
	return ty, declarations, nil
}
