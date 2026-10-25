package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Reuse complete original ordinary primitive groups, not a generic discarded
// permission. Numeric checks retain selected original bound nodes separately
// from the emitted plan, without synthesizing schemas or changing precision.
type allOfDefaultAggregate struct {
	checks  *PrimitiveValidation
	numeric *numericDefaultAggregate
}

func (a *analyzer) allOfDefaultPrimitiveAggregates(v *document.Value, pointer string, s *shape) (map[string]allOfDefaultAggregate, *diagnostic.Diagnostic) {
	validation := *a
	validation.emissionHints = EmissionHints{} // Literal-only plans own no emitted hints.
	numeric := map[string]*numericDefaultAggregate{}
	plans, preserved, err := validation.ordinaryPrimitiveAggregatesWithDefaults(v, s, true, numeric)
	if err != nil {
		return nil, err
	}
	if err := validation.allOfDefaultOverridePlans(s, plans, preserved, numeric); err != nil {
		return nil, err
	}
	for _, f := range s.Discarded {
		kind, _, failure := schemaType(f.Schema, f.Pointer)
		if failure != nil {
			return nil, failure
		}
		if !preserved[f.Pointer] || plans[f.Wire] == nil || kind == "string" && has(f.Schema, "format") && text(get(f.Schema, "format")) != "uuid" || kind != "string" && numeric[f.Wire] == nil {
			return nil, unsupported(naming.Pointer(pointer, "default"))
		}
		// Validate and charge every original discarded field as well as each
		// winning field, even for empty/null/suppressed enclosing annotations.
		if _, err := a.objectDefaultPropertyNode(f.Schema, f.Pointer, f.Depth, true); err != nil {
			return nil, err
		}
	}
	result := map[string]allOfDefaultAggregate{}
	for wire, checks := range plans {
		result[wire] = allOfDefaultAggregate{checks: checks, numeric: numeric[wire]}
	}
	return result, nil
}
