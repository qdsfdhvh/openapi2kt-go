package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// booleanTrueValue is a value-role predicate, not a relaxation of object slots
// or boolean additionalProperties. The original literal stays in the document.
func (a *analyzer) booleanTrueValue(v *document.Value) bool {
	flag, ok := v.Bool()
	return ok && flag && strings.HasPrefix(text(get(a.root, "openapi")), "3.1.")
}

func (a *analyzer) booleanValueType(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if !a.booleanTrueValue(v) {
		return nil, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	a.warn("W_BOOLEAN_SCHEMA_PROJECTION", "Schema true uses unconstrained JsonElement, including the JsonNull value; absence and Kotlin null follow the existing field presence policy.", pointer)
	ty := TypeRef{Kind: JSON}
	return &ty, nil
}
