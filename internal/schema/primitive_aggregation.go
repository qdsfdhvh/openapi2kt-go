package schema

import (
	"math/big"
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// ordinaryPrimitiveAggregates borrows the existing shape contributions. It does
// not expand refs, rewrite source nodes, or grant a generic discarded-check
// exemption. Only complete, independently validated groups own an exemption.
func (a *analyzer) ordinaryPrimitiveAggregates(v *document.Value, s *shape, ordinary bool) (map[string]*PrimitiveValidation, map[string]bool, *diagnostic.Diagnostic) {
	plans := map[string]*PrimitiveValidation{}
	preserved := map[string]bool{}
	if !ordinary || !a.options.Validation || !has(v, "allOf") {
		return plans, preserved, nil
	}
	groups := map[string][]field{}
	for _, f := range s.Contributions {
		groups[f.Wire] = append(groups[f.Wire], f)
	}
	for _, winner := range s.Fields {
		fields := groups[winner.Wire]
		if !completePrimitiveContributions(s, winner, fields) {
			continue // Only every real original contribution can own an exemption.
		}
		hasBounds := false
		for _, f := range fields[:len(fields)-1] {
			hasBounds = hasBounds || hasPrimitiveBounds(f.Schema)
		}
		if !hasBounds {
			continue
		}
		scalar, eligible := aggregateSignature(fields)
		if !eligible {
			continue
		} // Existing noDrop diagnostics remain authoritative.
		checks := make([]*PrimitiveValidation, len(fields))
		actualDiscarded := false
		for i, f := range fields {
			var check *PrimitiveValidation
			var err *diagnostic.Diagnostic
			if has(f.Schema, "default") {
				_, check, err = a.modelDefaultType(f, true, "Validation", nil, nil)
			} else {
				_, check, err = a.boundedPrimitiveType(f.Schema, f.Pointer, f.Depth)
			}
			if err != nil {
				return nil, nil, err
			}
			checks[i] = check
			actualDiscarded = actualDiscarded || i < len(fields)-1 && check != nil
		}
		if !actualDiscarded {
			continue
		} // False-only legacy modifiers are not a drop exemption.
		plan, err := a.mergePrimitiveChecks(fields, checks, scalar)
		if err != nil {
			return nil, nil, err
		}
		if value, exists := winner.Schema.Get("default"); exists {
			if scalar == "kotlin.String" {
				// Length and the admitted single pattern need one value scan.
				if err := validateDefaultChecksAt(value, winner.Schema, plan, scalar, winner.Pointer, winner.Pointer+"/default"); err != nil {
					return nil, nil, err
				}
			} else {
				// Keep original numeric nodes for exactness and provenance;
				// only the winning annotation becomes an initializer.
				for i, f := range fields {
					if err := validateDefaultChecksAt(value, f.Schema, checks[i], scalar, f.Pointer, winner.Pointer+"/default"); err != nil {
						return nil, nil, err
					}
				}
			}
		}
		plans[winner.Wire] = plan
		a.emissionHints[plan] = true
		for _, f := range fields[:len(fields)-1] {
			preserved[f.Pointer] = true
		}
	}
	return plans, preserved, nil
}

func aggregateSignature(fields []field) (string, bool) {
	kind, nullable, err := schemaType(fields[0].Schema, fields[0].Pointer)
	if err != nil || kind != "string" && kind != "integer" && kind != "number" {
		return "", false
	}
	format := get(fields[0].Schema, "format")
	scalar, err := scalarMapping(kind, fields[0].Schema, fields[0].Pointer)
	if err != nil || !slices.Contains([]string{"kotlin.String", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double"}, scalar) {
		return "", false
	}
	for _, f := range fields {
		k, n, e := schemaType(f.Schema, f.Pointer)
		if e != nil || k != kind || n != nullable || has(f.Schema, "format") != has(fields[0].Schema, "format") || text(get(f.Schema, "format")) != text(format) {
			return "", false
		}
		for _, m := range f.Schema.Members() {
			if !slices.Contains(metadata, m.Name) && !slices.Contains(primitiveBoundKeys, m.Name) && !slices.Contains([]string{"type", "format", "nullable", "default"}, m.Name) {
				return "", false
			}
		}
	}
	return scalar, true
}

func (a *analyzer) mergePrimitiveChecks(fields []field, checks []*PrimitiveValidation, scalar string) (*PrimitiveValidation, *diagnostic.Diagnostic) {
	result := &PrimitiveValidation{Floating: scalar == "kotlin.Float" || scalar == "kotlin.Double"}
	var endpoints [4]*numericEndpoint // inclusive lower, strict lower, inclusive upper, strict upper
	var factorExact *big.Rat
	factorPointer := ""
	legacy := strings.HasPrefix(text(get(a.root, "openapi")), "3.0.")
	for i, f := range fields {
		check := checks[i]
		if check == nil {
			continue
		}
		if scalar == "kotlin.String" {
			// Fixed min/max order, with attribution to the source closing the interval.
			if check.MinLength != nil && (result.MinLength == nil || *check.MinLength > *result.MinLength) {
				result.MinLength = check.MinLength
			}
			if result.MinLength != nil && result.MaxLength != nil && *result.MinLength > *result.MaxLength {
				return nil, invalid("Minimum length exceeds maximum length.", f.Pointer+"/minLength")
			}
			if check.MaxLength != nil && (result.MaxLength == nil || *check.MaxLength < *result.MaxLength) {
				result.MaxLength = check.MaxLength
			}
			if result.MinLength != nil && result.MaxLength != nil && *result.MinLength > *result.MaxLength {
				return nil, invalid("Minimum length exceeds maximum length.", f.Pointer+"/maxLength")
			}
			if check.Pattern != nil {
				if result.Pattern != nil && *result.Pattern != *check.Pattern {
					return nil, unsupported(f.Pointer + "/pattern")
				}
				result.Pattern = check.Pattern
			}
			continue
		}
		for _, key := range []string{"minimum", "exclusiveMinimum", "maximum", "exclusiveMaximum", "multipleOf"} {
			if !has(f.Schema, key) || legacy && (key == "exclusiveMinimum" || key == "exclusiveMaximum") {
				continue
			}
			// Each source already passed literal/nonzero/finite/resource/domain checks.
			bound, exact, err := constraintNumber(get(f.Schema, key), scalar, f.Pointer+"/"+key)
			if err != nil {
				return nil, err
			}
			if key == "multipleOf" {
				if factorExact != nil && factorExact.Cmp(exact) != 0 {
					return nil, unsupported(f.Pointer + "/multipleOf")
				}
				if factorExact == nil {
					factorExact = exact
					factorPointer = f.Pointer + "/multipleOf"
					result.MultipleOf = check.MultipleOf
				}
				continue
			}
			category := 0
			if key == "maximum" || key == "exclusiveMaximum" {
				category = 2
			}
			p := f.Pointer + "/" + key
			strict := key == "exclusiveMinimum" || key == "exclusiveMaximum"
			if legacy {
				modifier := "exclusiveMinimum"
				if category == 2 {
					modifier = "exclusiveMaximum"
				}
				enabled, _ := get(f.Schema, modifier).Bool()
				if enabled {
					strict = true
					p = f.Pointer + "/" + modifier
				}
			}
			if strict {
				category++
				bound.Exclusive = true
			}
			current := endpoints[category]
			if current == nil || category < 2 && exact.Cmp(current.exact) > 0 || category >= 2 && exact.Cmp(current.exact) < 0 {
				endpoints[category] = &numericEndpoint{key: p, exact: exact, bound: bound}
			}
		}
	}
	var lows, highs []numericEndpoint
	slots := []**NumericBound{&result.Minimum, &result.ExclusiveMinimum, &result.Maximum, &result.ExclusiveMaximum}
	for i, e := range endpoints {
		if e != nil {
			*slots[i] = e.bound
			if i < 2 {
				lows = append(lows, *e)
			} else {
				highs = append(highs, *e)
			}
		}
	}
	// Exact intersection attribution uses the restrictive upper endpoint across
	// categories, without deleting either mapped runtime comparison. The old
	// single-source order stays untouched in numericValidation.
	slices.SortStableFunc(highs, func(x, y numericEndpoint) int {
		if cmp := x.exact.Cmp(y.exact); cmp != 0 {
			return cmp
		}
		if x.bound.Exclusive != y.bound.Exclusive {
			if x.bound.Exclusive {
				return -1
			}
			return 1
		}
		return 0
	})
	var factor *big.Int
	if factorExact != nil && (scalar == "kotlin.Int" || scalar == "kotlin.Long") {
		factor = new(big.Int).Set(factorExact.Num())
	}
	if err := validateNumericDomain(scalar, lows, highs, factor, factorPointer); err != nil {
		return nil, err
	}
	return result, nil
}
