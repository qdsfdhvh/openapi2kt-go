package schema

import (
	"math/big"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Validate each original direct DTO field default, including discarded fields.
// Scalar roots, references and collection leaves keep their strict planners.
func scalarSchemaDefault(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	value, exists := v.Get("default")
	if !exists {
		return "", nil
	}
	at := naming.Pointer(pointer, "default")
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return "", err
	}
	if !slices.Contains([]string{"string", "boolean", "integer", "number", "null"}, kind) {
		return "", unsupported(at)
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "enum", "const", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return "", unsupported(at)
		}
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return "", err
	}
	if !slices.Contains([]string{"kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double", "kotlin.time.Instant", "kotlinx.datetime.LocalDate", "kotlinx.serialization.json.JsonNull"}, scalar) {
		return "", unsupported(at)
	}
	if kind == "null" {
		if value.Kind() != document.Null {
			return "", invalid("A null-only field default must be JSON null.", at)
		}
		return "null", nil
	}
	if value.Kind() == document.Null {
		if !nullable {
			return "", invalid("A null default requires a nullable schema.", at)
		}
		return "null", nil
	}
	literal, err := scalarLiteral(scalar, value, at)
	if err != nil && err.Code == "E_SCHEMA_INVALID" {
		return "", invalid("Default must be representable by its mapped Kotlin scalar type.", at)
	}
	return literal, err
}

func (a *analyzer) modelDefaultType(f field, ordinary bool, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *PrimitiveValidation, *diagnostic.Diagnostic) {
	if has(f.Schema, "$ref") {
		ty, err := a.referencedDefaultType(f, ordinary)
		return ty, nil, err
	}
	if objectDefaultCandidateSchema(f.Schema) {
		ty, err := a.objectDefaultField(f, ordinary, name, nested, used)
		return ty, nil, err
	}
	if nestedCollectionDefaultSchema(f.Schema) {
		if !ordinary {
			return TypeRef{}, nil, unsupported(naming.Pointer(f.Pointer, "default"))
		}
		ty, _, err := a.nestedCollectionDefaultNamed(f.Schema, name, f.Pointer, nested, used, f.Depth)
		return ty, nil, err
	}
	if dictionarySchema(f.Schema) {
		if !ordinary {
			return TypeRef{}, nil, unsupported(naming.Pointer(f.Pointer, "default"))
		}
		ty, _, err := a.primitiveMapDefault(f.Schema, name, f.Pointer, nested, used, f.Depth)
		return ty, nil, err
	}
	if text(get(f.Schema, "type")) == "array" || bareArrayDefaultSchema(f.Schema) {
		if !ordinary {
			return TypeRef{}, nil, unsupported(naming.Pointer(f.Pointer, "default"))
		}
		ty, _, err := a.primitiveArrayDefault(f.Schema, name, f.Pointer, nested, used, f.Depth)
		return ty, nil, err
	}
	if objectDefaultCandidateSchema(f.Schema) {
		ty, err := a.objectDefaultField(f, ordinary, name, nested, used)
		return ty, nil, err
	}
	if err := depthCheck(f.Depth, f.Pointer); err != nil {
		return TypeRef{}, nil, err
	}
	if _, err := scalarSchemaDefault(f.Schema, f.Pointer); err != nil {
		return TypeRef{}, nil, err
	}
	keys := []string{"type", "nullable", "format", "default"}
	if a.options.Validation {
		keys = append(keys, primitiveBoundKeys...)
	}
	if err := checkKeys(f.Schema, keys, f.Pointer); err != nil {
		return TypeRef{}, nil, err
	}
	kind, nullable, err := schemaType(f.Schema, f.Pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	scalar, err := scalarMapping(kind, f.Schema, f.Pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	var checks *PrimitiveValidation
	if a.options.Validation && hasPrimitiveBounds(f.Schema) {
		checks, err = a.primitiveValidation(f.Schema, scalar, f.Pointer)
		if err != nil {
			return TypeRef{}, nil, err
		}
	}
	ty := TypeRef{Kind: Scalar, Name: scalar}
	if nullable {
		ty = ty.Null()
	}
	if err := validateDefaultChecks(get(f.Schema, "default"), f.Schema, checks, scalar, f.Pointer); err != nil {
		return TypeRef{}, nil, err
	}
	return ty, checks, nil
}

func validateDefaultChecks(value, schema *document.Value, checks *PrimitiveValidation, scalar, pointer string) *diagnostic.Diagnostic {
	return validateDefaultChecksAt(value, schema, checks, scalar, pointer, naming.Pointer(pointer, "default"))
}

// Keep constraint provenance separate from the original default member location.
func validateDefaultChecksAt(value, schema *document.Value, checks *PrimitiveValidation, scalar, pointer, valuePointer string) *diagnostic.Diagnostic {
	if value.Kind() == document.Null || checks == nil {
		return nil
	}
	bad := func() *diagnostic.Diagnostic {
		return invalid("Default must satisfy the field's retained validation constraints.", valuePointer)
	}
	if text, ok := value.Text(); ok {
		length := int64(utf8.RuneCountInString(text))
		if checks.MinLength != nil && length < *checks.MinLength || checks.MaxLength != nil && length > *checks.MaxLength {
			return bad()
		}
		if checks.Pattern != nil {
			// The general runtime grammar stays unchanged. Compile-time default
			// matching is limited to printable ASCII to avoid Java/Go Unicode
			// and line-terminator differences.
			for _, r := range text + *checks.Pattern {
				if r < 32 || r > 126 {
					return unsupported(valuePointer)
				}
			}
			pattern := *checks.Pattern
			if matched, err := regexp.MatchString(pattern, text); err != nil || !matched {
				return bad()
			}
		}
		return nil
	}
	// Reuse numeric-bound precision/exponent limits before exact arithmetic.
	_, exact, err := constraintNumber(value, scalar, valuePointer)
	if err != nil {
		return err
	}
	floating := scalar == "kotlin.Float" || scalar == "kotlin.Double"
	mapped := 0.0
	if floating {
		mapped, _ = floatLiteral(value)
		if scalar == "kotlin.Float" {
			mapped = float64(float32(mapped))
		}
	}
	for _, endpoint := range []struct {
		key   string
		bound *NumericBound
		lower bool
	}{
		{"minimum", checks.Minimum, true}, {"maximum", checks.Maximum, false}, {"exclusiveMinimum", checks.ExclusiveMinimum, true}, {"exclusiveMaximum", checks.ExclusiveMaximum, false},
	} {
		if endpoint.bound == nil {
			continue
		}
		_, bound, err := constraintNumber(get(schema, endpoint.key), scalar, naming.Pointer(pointer, endpoint.key))
		if err != nil {
			return err
		}
		cmp := exact.Cmp(bound)
		if floating {
			limit, _ := floatLiteral(get(schema, endpoint.key))
			if scalar == "kotlin.Float" {
				limit = float64(float32(limit))
			}
			cmp = 0
			if mapped < limit {
				cmp = -1
			} else if mapped > limit {
				cmp = 1
			}
		}
		exclusive := endpoint.bound.Exclusive || strings.HasPrefix(endpoint.key, "exclusive")
		if endpoint.lower && (cmp < 0 || exclusive && cmp == 0) || !endpoint.lower && (cmp > 0 || exclusive && cmp == 0) {
			return bad()
		}
	}
	if checks.MultipleOf != nil {
		if floating {
			return unsupported(valuePointer)
		}
		_, factor, err := constraintNumber(get(schema, "multipleOf"), scalar, naming.Pointer(pointer, "multipleOf"))
		if err != nil {
			return err
		}
		if !new(big.Rat).Quo(exact, factor).IsInt() {
			return bad()
		}
	}
	return nil
}

// Defaults are annotations, never a replacement for required request presence.
// The bounded literal profile uses the mapped scalar and original nullable view.
func (a *analyzer) modelDefault(f field, ty TypeRef, required bool) (string, *diagnostic.Diagnostic) {
	value, exists := f.Schema.Get("default")
	if !exists {
		return "", nil
	}
	if has(f.Schema, "$ref") {
		return "", nil // Final linked backing supplies the typed default plan.
	}
	pointer := naming.Pointer(f.Pointer, "default")
	if objectDefaultCandidateSchema(f.Schema) {
		validation := *a
		validation.warnings = nil // Type planning owns source warnings.
		initializer, err := validation.objectDefaultInitializer(f)
		if err != nil {
			return "", err
		}
		if required || ty.Kind == PatchValue {
			return "", nil
		}
		return initializer, nil
	}
	if nestedCollectionDefaultSchema(f.Schema) {
		validation := *a
		validation.warnings = nil // Type planning owns source warnings.
		_, initializer, err := validation.nestedCollectionDefault(f.Schema, f.Pointer, f.Depth)
		if err != nil {
			return "", err
		}
		if required || ty.Kind == PatchValue {
			return "", nil
		}
		return initializer, nil
	}
	if dictionarySchema(f.Schema) {
		validation := *a
		validation.warnings = nil // Type planning owns source warnings.
		_, initializer, err := validation.primitiveMapDefault(f.Schema, "Validation", f.Pointer, nil, nil, f.Depth)
		if err != nil {
			return "", err
		}
		if required || ty.Kind == PatchValue {
			return "", nil
		}
		return initializer, nil
	}
	if bareArrayDefaultSchema(f.Schema) {
		validation := *a
		validation.warnings = nil // Type planning owns source warnings.
		_, initializer, err := validation.primitiveArrayDefault(f.Schema, "Validation", f.Pointer, nil, nil, f.Depth)
		if err != nil {
			return "", err
		}
		if required || ty.Kind == PatchValue {
			return "", nil
		}
		return initializer, nil
	}
	if has(f.Schema, "enum") {
		if err := a.enumDefaultValue(f.Schema, f.Pointer); err != nil {
			return "", err
		}
	} else if _, err := scalarSchemaDefault(f.Schema, f.Pointer); err != nil {
		return "", err
	}
	if ty.Kind == PatchValue {
		return "", nil // Missing request keys always retain Patch.Absent.
	}
	if value.Kind() == document.Null {
		target := bare(ty)
		if required && target.Kind == Scalar && target.Name == "kotlinx.serialization.json.JsonNull" {
			return "", nil // JsonNull represents the value, not Kotlin nullable intent.
		}
		if ty.Kind != Nullable {
			return "", invalid("A null default requires a nullable field.", pointer)
		}
		if required {
			return "", nil
		}
		return "null", nil
	}
	target := bare(ty)
	if has(f.Schema, "enum") && target.Kind == Named {
		if required {
			return "", nil
		}
		text, _ := value.Text()
		return target.Name + ".fromSerialName(" + naming.Literal(text) + ")", nil
	}
	if target.Kind != Scalar {
		return "", unsupported(pointer)
	}
	literal, err := scalarLiteral(target.Name, value, pointer)
	if err != nil {
		return "", err
	}
	if required {
		return "", nil
	}
	return scalarDefaultInitializer(target.Name, literal), nil
}

// Keep literal initializers safe when constructor properties shadow packages.
func scalarDefaultInitializer(scalar, literal string) string {
	switch literal {
	case "kotlin.Int.MIN_VALUE":
		literal = "(-2147483647 - 1)"
	case "kotlin.Long.MIN_VALUE":
		literal = "(-9223372036854775807L - 1L)"
	}
	switch scalar {
	case "kotlin.time.Instant":
		literal = strings.Replace(literal, "kotlin.time.Instant.parse(", "`openapi-kt default instant`(", 1)
	case "kotlinx.datetime.LocalDate":
		literal = strings.Replace(literal, "kotlinx.datetime.LocalDate.parse(", "`openapi-kt default date`(", 1)
	}
	return literal
}
