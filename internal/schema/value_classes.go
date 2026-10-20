package schema

import (
	"slices"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// PrimitiveValueClass is opt-in and nominal. Its underlying primitive is only
// used for JVM width and parameter wire emission, never reference inlining.
type PrimitiveValueClass struct {
	ClassName, Scalar string
	Validation        *PrimitiveValidation
}

func (m *PrimitiveValueClass) Name() string { return m.ClassName }
func (*PrimitiveValueClass) declaration()   {}

// Only an already valid primitive wrapper may acquire intrinsic root intent.
// Its physical classifier and underlying constructor scalar remain non-null.
func nullableValueClassCandidate(v *document.Value, kind, pointer string) bool {
	_, enabled, err := valueClassScalar(v, kind, pointer)
	return err == nil && enabled
}

func valueClassScalar(v *document.Value, kind, pointer string) (string, bool, *diagnostic.Diagnostic) {
	enabled, valid := get(v, "x-kotlin-value-class").Bool()
	if !valid {
		return "", false, invalid("x-kotlin-value-class must be boolean.", pointer+"/x-kotlin-value-class")
	}
	if !slices.Contains([]string{"string", "boolean", "integer", "number"}, kind) {
		return "", false, unsupported(pointer + "/x-kotlin-value-class")
	}
	allowed := []string{"type", "nullable", "format", "x-kotlin-value-class"}
	if enabled {
		allowed = append(allowed, primitiveBoundKeys...)
	}
	if err := checkKeys(v, allowed, pointer); err != nil {
		return "", false, err
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return "", false, err
	}
	if !slices.Contains([]string{"kotlin.String", "kotlin.uuid.Uuid", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double"}, scalar) {
		return "", false, unsupported(pointer + "/format")
	}
	return scalar, enabled, nil
}
