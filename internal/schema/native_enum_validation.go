package schema

import (
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Direct native enum schemas build a typed membership plan. Consumers retain
// its validation and source metadata; transparent aliases add no runtime barrier.
func (a *analyzer) nativeEnumFieldType(f field, fieldDefault bool) (TypeRef, *PrimitiveValidation, *diagnostic.Diagnostic) {
	if err := depthCheck(f.Depth, f.Pointer); err != nil {
		return TypeRef{}, nil, err
	}
	kind, nullable, err := schemaType(f.Schema, f.Pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	if kind != "integer" && kind != "number" && kind != "boolean" {
		return TypeRef{}, nil, unsupported(naming.Pointer(f.Pointer, "enum"))
	}
	if fieldDefault {
		if err := a.enumDefaultValue(f.Schema, f.Pointer); err != nil {
			return TypeRef{}, nil, err
		}
	}
	validation := *a
	validation.warnings = nil
	scalar, err := validation.nativeEnumProfile(kind, f.Schema, f.Pointer, fieldDefault, true)
	if err != nil {
		return TypeRef{}, nil, err
	}
	values := get(f.Schema, "enum").Items()
	if len(values) > MaxMembers {
		return TypeRef{}, nil, diagnostic.At("E_MODEL_LIMIT", "Checked native enum exceeds the supported 240-member limit.", naming.Pointer(f.Pointer, "enum"))
	}
	checks, err := a.primitiveValidation(f.Schema, scalar, f.Pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	if checks == nil {
		checks = &PrimitiveValidation{}
	}
	checks.Floating = scalar == "kotlin.Float" || scalar == "kotlin.Double"
	seen := map[string]bool{}
	for i, value := range values {
		if value.Kind() == document.Null {
			continue // The vocabulary parser admitted codec null intent separately.
		}
		at := naming.Pointer(naming.Pointer(f.Pointer, "enum"), strconv.Itoa(i))
		literal := ""
		if kind == "boolean" {
			literal, err = scalarLiteral(scalar, value, at)
		} else {
			var bound *NumericBound
			bound, _, err = constraintNumber(value, scalar, at)
			if bound != nil {
				literal = bound.Literal
			}
		}
		if err != nil {
			return TypeRef{}, nil, err
		}
		if !seen[literal] {
			checks.Enum = append(checks.Enum, literal)
			seen[literal] = true
		}
	}
	if err := validateNativeEnumDomain(f.Schema, checks, scalar, f.Pointer, fieldDefault); err != nil {
		return TypeRef{}, nil, err
	}
	a.validationSources[checks] = f.Pointer
	a.emissionHints[checks] = true // Non-inline helper preserves wide constructor budget.
	ty := TypeRef{Kind: Scalar, Name: scalar}
	if nullable {
		ty = ty.Null()
	}
	return ty, checks, nil
}

func (a *analyzer) nativeEnumValueType(v *document.Value, pointer string, depth int) (TypeRef, *diagnostic.Diagnostic) {
	return a.nativeEnumValueTypeOwnDefault(v, pointer, depth, false)
}

func (a *analyzer) nativeEnumValueTypeOwnDefault(v *document.Value, pointer string, depth int, ownDefault bool) (TypeRef, *diagnostic.Diagnostic) {
	ty, checks, err := a.nativeEnumFieldType(field{Schema: v, Pointer: pointer, Depth: depth}, ownDefault)
	if err != nil {
		return TypeRef{}, err
	}
	if ty.Kind == Nullable {
		inner := *ty.Element
		inner.Validation = checks
		ty.Element = &inner
	} else {
		ty.Validation = checks
	}
	return ty, nil
}
