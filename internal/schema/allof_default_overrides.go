package schema

import "openapi-kt/internal/diagnostic"

// Literal-only plans for complete homogeneous override groups whose discarded
// contributions carry no bounds. Actual model validation remains the winning
// field's original plan; this path adds no emitted aggregation or hints.
func (a *analyzer) allOfDefaultOverridePlans(s *shape, plans map[string]*PrimitiveValidation, preserved map[string]bool, numeric map[string]*numericDefaultAggregate) *diagnostic.Diagnostic {
	if !a.options.Validation {
		return nil
	}
	groups := map[string][]field{}
	for _, f := range s.Contributions {
		groups[f.Wire] = append(groups[f.Wire], f)
	}
	for _, winner := range s.Fields {
		if plans[winner.Wire] != nil {
			continue
		}
		fields := groups[winner.Wire]
		if !completePrimitiveContributions(s, winner, fields) {
			continue
		}
		eligible := true
		for _, f := range fields[:len(fields)-1] {
			eligible = eligible && !hasPrimitiveBounds(f.Schema)
		}
		if !eligible {
			continue
		}
		scalar, eligible := aggregateSignature(fields)
		if !eligible {
			continue
		}
		checks := make([]*PrimitiveValidation, len(fields))
		for i, f := range fields {
			var e *diagnostic.Diagnostic
			if has(f.Schema, "default") {
				_, checks[i], e = a.modelDefaultType(f, true, "Validation", nil, nil)
			} else {
				_, checks[i], e = a.boundedPrimitiveType(f.Schema, f.Pointer, f.Depth)
			}
			if e != nil {
				return e
			}
		}
		plan, literals, e := a.mergePrimitiveChecks(fields, checks, scalar)
		if e != nil {
			return e
		}
		plans[winner.Wire] = plan
		if literals != nil {
			numeric[winner.Wire] = literals
		}
		for _, f := range fields[:len(fields)-1] {
			preserved[f.Pointer] = true
		}
	}
	return nil
}
