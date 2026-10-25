package schema

import (
	"math/big"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Literal-only provenance, selected by the existing primitive merge. It never
// enters declaration identity or a name lock. Each supplied value visits at
// most four selected endpoints and one factor, regardless of source width.
type numericDefaultAggregate struct {
	endpoints [4]*numericDefaultEndpoint
	factor    *numericDefaultEndpoint
}

type numericDefaultEndpoint struct {
	source  *document.Value
	pointer string
	exact   *big.Rat
	bound   *NumericBound
}

func (p *numericDefaultAggregate) validate(value *document.Value, scalar, pointer string) *diagnostic.Diagnostic {
	if value.Kind() == document.Null {
		return nil
	}
	_, exact, err := constraintNumber(value, scalar, pointer)
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
	bad := func() *diagnostic.Diagnostic {
		return invalid("Default must satisfy the field's retained validation constraints.", pointer)
	}
	for i, endpoint := range p.endpoints {
		if endpoint == nil {
			continue
		}
		cmp := exact.Cmp(endpoint.exact)
		if floating {
			limit, _ := floatLiteral(endpoint.source)
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
		if i < 2 && (cmp < 0 || endpoint.bound.Exclusive && cmp == 0) || i >= 2 && (cmp > 0 || endpoint.bound.Exclusive && cmp == 0) {
			return bad()
		}
	}
	if p.factor != nil {
		if floating {
			return unsupported(pointer)
		}
		if !new(big.Rat).Quo(exact, p.factor.exact).IsInt() {
			return bad()
		}
	}
	return nil
}
