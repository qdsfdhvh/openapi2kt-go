package schema

import (
	"math/big"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// PrimitiveValidation belongs to a nominal wrapper, opt-in DTO property, or
// transparent scalar alias leaf checked by typed consumers. Raw projections
// cannot retain these checks and must not accept/drop them.
// Numeric literals are already validated and typed for deterministic emission.
type PrimitiveValidation struct {
	MinLength, MaxLength               *int64
	Minimum, Maximum                   *NumericBound
	ExclusiveMinimum, ExclusiveMaximum *NumericBound
	MultipleOf                         *NumericBound
	Floating                           bool
	Pattern                            *string
	Enum                               []string // Mapped literal membership for admitted direct native enum fields.
}
type NumericBound struct {
	Literal   string
	Zero      string // Typed remainder comparison for multipleOf only.
	Exclusive bool   // OpenAPI 3.0 modifier on a minimum/maximum.
}

func constraintNumber(v *document.Value, scalar, pointer string) (*NumericBound, *big.Rat, *diagnostic.Diagnostic) {
	number, ok := v.Numeric()
	if !ok {
		return nil, nil, invalid("Numeric bounds must be finite representable numbers.", pointer)
	}
	raw := string(number)
	// Bound exact-order work independently of user-supplied decimal exponents.
	if len(raw) > 256 {
		return nil, nil, invalid("Numeric bound exceeds the supported precision limit.", pointer)
	}
	if i := strings.IndexAny(raw, "eE"); i >= 0 {
		exp, err := strconv.ParseInt(raw[i+1:], 10, 32)
		if err != nil || exp < -512 || exp > 512 {
			return nil, nil, invalid("Numeric bound exponent exceeds the supported limit.", pointer)
		}
	}
	literal, err := scalarLiteral(scalar, v, pointer)
	if err != nil {
		return nil, nil, invalid("Numeric bound must be representable by its mapped Kotlin scalar type.", pointer)
	}
	exact, ok := new(big.Rat).SetString(raw)
	if !ok {
		return nil, nil, invalid("Invalid numeric bound.", pointer)
	}
	// Expression qualifiers can be shadowed by a legitimate field `kotlin`.
	switch literal {
	case "kotlin.Int.MIN_VALUE":
		literal = "(-2147483647 - 1)"
	case "kotlin.Long.MIN_VALUE":
		literal = "(-9223372036854775807L - 1L)"
	}
	return &NumericBound{Literal: literal}, exact, nil
}

var primitiveBoundKeys = []string{"minLength", "maxLength", "minimum", "maximum", "pattern", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"}

func hasPrimitiveBounds(v *document.Value) bool {
	for _, key := range primitiveBoundKeys {
		if has(v, key) {
			return true
		}
	}
	return false
}

func (a *analyzer) primitiveValidation(v *document.Value, scalar, pointer string) (*PrimitiveValidation, *diagnostic.Diagnostic) {
	var keys []string
	for _, key := range primitiveBoundKeys {
		if has(v, key) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	result := &PrimitiveValidation{}
	if scalar == "kotlin.String" || scalar == "kotlin.uuid.Uuid" {
		for _, key := range keys {
			if key == "pattern" {
				pattern, err := validationPattern(get(v, key), pointer+"/"+key)
				if err != nil {
					return nil, err
				}
				result.Pattern = &pattern
				continue
			}
			if key != "minLength" && key != "maxLength" {
				return nil, unsupported(pointer + "/" + key)
			}
			value, ok := integerLiteral(get(v, key))
			if !ok || value < 0 || value > 2147483647 {
				return nil, invalid("Length bound must be a nonnegative Int-range integer literal.", pointer+"/"+key)
			}
			if key == "minLength" {
				result.MinLength = &value
			} else {
				result.MaxLength = &value
			}
		}
		if result.MinLength != nil && result.MaxLength != nil && *result.MinLength > *result.MaxLength {
			return nil, invalid("Minimum length exceeds maximum length.", pointer+"/maxLength")
		}
		return result, nil
	}
	if scalar != "kotlin.Int" && scalar != "kotlin.Long" && scalar != "kotlin.Float" && scalar != "kotlin.Double" {
		return nil, unsupported(pointer + "/" + keys[0])
	}
	return a.numericValidation(v, scalar, pointer, keys, result)
}
