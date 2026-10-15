package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func validateNativeEnumDomain(v *document.Value, checks *PrimitiveValidation, scalar, pointer string, fieldDefault bool) *diagnostic.Diagnostic {
	if !hasPrimitiveBounds(v) {
		return nil
	}
	if checks.MultipleOf != nil && (scalar == "kotlin.Float" || scalar == "kotlin.Double") {
		// Preserve the existing runtime remainder contract, but do not approximate a
		// compile-time floating remainder intersection/default via a different host.
		return unsupported(naming.Pointer(pointer, "multipleOf"))
	}
	possible := false
	for _, member := range get(v, "enum").Items() {
		if member.Kind() == document.Null {
			continue // Null cannot make an empty non-null numeric domain usable.
		}
		err := validateDefaultChecks(member, v, checks, scalar, pointer)
		if err == nil {
			possible = true
			continue
		}
		if err.Code != "E_SCHEMA_INVALID" || err.Pointer == nil || *err.Pointer != naming.Pointer(pointer, "default") {
			return err
		}
	}
	if !possible {
		return invalid("Enum and retained validation constraints have an empty mapped intersection.", naming.Pointer(pointer, "enum"))
	}
	if fieldDefault {
		if err := validateDefaultChecks(get(v, "default"), v, checks, scalar, pointer); err != nil {
			return err
		}
	}
	return nil
}
