package schema

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

type numericEndpoint struct {
	key   string
	exact *big.Rat
	bound *NumericBound
}

func (a *analyzer) numericValidation(v *document.Value, scalar, pointer string, keys []string, result *PrimitiveValidation) (*PrimitiveValidation, *diagnostic.Diagnostic) {
	var lows, highs []numericEndpoint
	var factor *big.Int
	legacy := strings.HasPrefix(text(get(a.root, "openapi")), "3.0.")
	for _, key := range keys {
		switch key {
		case "exclusiveMinimum", "exclusiveMaximum":
			if legacy {
				enabled, valid := get(v, key).Bool()
				if !valid {
					return nil, invalid("OpenAPI 3.0 exclusive bounds must be boolean.", pointer+"/"+key)
				}
				if !enabled {
					continue
				}
				base := "minimum"
				bound := result.Minimum
				endpoints := lows
				if key == "exclusiveMaximum" {
					base = "maximum"
					bound = result.Maximum
					endpoints = highs
				}
				if bound == nil {
					return nil, invalid("An enabled exclusive bound requires its corresponding numeric bound.", pointer+"/"+key)
				}
				bound.Exclusive = true
				// Retain the modifier's Pointer for empty-domain diagnostics.
				for i := range endpoints {
					if endpoints[i].key == pointer+"/"+base {
						endpoints[i].key = pointer + "/" + key
					}
				}
				continue
			}
			if get(v, key).Kind() == document.Boolean {
				return nil, invalid("OpenAPI 3.1 exclusive bounds must be numeric.", pointer+"/"+key)
			}
		case "minimum", "maximum", "multipleOf":
		default:
			return nil, unsupported(pointer + "/" + key)
		}
		bound, exact, err := constraintNumber(get(v, key), scalar, pointer+"/"+key)
		if err != nil {
			return nil, err
		}
		endpoint := numericEndpoint{pointer + "/" + key, exact, bound}
		switch key {
		case "minimum":
			result.Minimum = bound
			lows = append(lows, endpoint)
		case "maximum":
			result.Maximum = bound
			highs = append(highs, endpoint)
		case "exclusiveMinimum":
			result.ExclusiveMinimum = bound
			bound.Exclusive = true
			lows = append(lows, endpoint)
		case "exclusiveMaximum":
			result.ExclusiveMaximum = bound
			bound.Exclusive = true
			highs = append(highs, endpoint)
		case "multipleOf":
			if exact.Sign() <= 0 {
				return nil, invalid("multipleOf must be positive.", pointer+"/"+key)
			}
			if scalar == "kotlin.Float" || scalar == "kotlin.Double" {
				mapped, _ := strconv.ParseFloat(strings.TrimRight(bound.Literal, "fF"), 64)
				if mapped == 0 {
					return nil, invalid("multipleOf must remain nonzero in its mapped scalar type.", pointer+"/"+key)
				}
			}
			bound.Zero = map[string]string{"kotlin.Int": "0", "kotlin.Long": "0L", "kotlin.Float": "0.0f", "kotlin.Double": "0.0"}[scalar]
			result.MultipleOf = bound
			if scalar == "kotlin.Int" || scalar == "kotlin.Long" {
				factor = new(big.Int).Set(exact.Num())
			}
		}
	}
	if err := validateNumericDomain(scalar, lows, highs, factor, pointer+"/multipleOf"); err != nil {
		return nil, err
	}
	if result.Minimum == nil && result.Maximum == nil && result.ExclusiveMinimum == nil && result.ExclusiveMaximum == nil && result.MultipleOf == nil {
		return nil, nil
	}
	result.Floating = scalar == "kotlin.Float" || scalar == "kotlin.Double"
	return result, nil
}

// validateNumericDomain consumes exact borrowed-source endpoints, never Kotlin literals.
// Provenance stays ephemeral and does not participate in declaration identity.
func validateNumericDomain(scalar string, lows, highs []numericEndpoint, factor *big.Int, factorPointer string) *diagnostic.Diagnostic {
	for _, low := range lows {
		for _, high := range highs {
			cmp := low.exact.Cmp(high.exact)
			if cmp > 0 || cmp == 0 && (low.bound.Exclusive || high.bound.Exclusive) {
				return invalid("Numeric bounds have an empty intersection.", high.key)
			}
		}
	}
	if scalar == "kotlin.Int" || scalar == "kotlin.Long" {
		min, max := int64(math.MinInt64), int64(math.MaxInt64)
		if scalar == "kotlin.Int" {
			min, max = math.MinInt32, math.MaxInt32
		}
		low, high := big.NewInt(min), big.NewInt(max)
		problem := ""
		for _, e := range lows {
			n := new(big.Int).Set(e.exact.Num())
			if e.bound.Exclusive {
				n.Add(n, big.NewInt(1))
			}
			if n.Cmp(low) > 0 {
				low = n
				problem = e.key
			}
		}
		for _, e := range highs {
			n := new(big.Int).Set(e.exact.Num())
			if e.bound.Exclusive {
				n.Sub(n, big.NewInt(1))
			}
			if n.Cmp(high) < 0 {
				high = n
				problem = e.key
			}
		}
		if low.Cmp(high) > 0 {
			return invalid("Numeric bounds have no representable integer value.", problem)
		}
		if factor != nil {
			// Euclidean modulus handles negative lower endpoints without abs or
			// signed-width overflow. All operands remain bounded scalar integers.
			rem := new(big.Int).Mod(low, factor)
			first := new(big.Int).Set(low)
			if rem.Sign() != 0 {
				first.Add(first, new(big.Int).Sub(factor, rem))
			}
			if first.Cmp(high) > 0 {
				return invalid("The integer domain contains no multipleOf value.", factorPointer)
			}
		}
	}
	return nil
}
