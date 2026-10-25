package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Additional tuple targets share one cumulative capture budget across analyzer
// contexts, arrays and discarded prefixes. Charge overlapping snapshots rather
// than resetting a per-target allowance. Existing non-tuple roles keep their
// historical per-source contract and pass a nil budget.
type tupleAllOfSourceBudget struct {
	Seen        map[*document.Value]bool
	Work, Bytes int
}

func (b *tupleAllOfSourceBudget) charge(work, bytes int, source string) *diagnostic.Diagnostic {
	if b == nil {
		return nil
	}
	if work > 16384-b.Work || bytes > (8<<20)-b.Bytes {
		return diagnostic.At("E_MODEL_LIMIT", "Tuple allOf source snapshots exceed the cumulative work or byte budget.", source)
	}
	b.Work += work
	b.Bytes += bytes
	return nil
}

func (a *analyzer) tupleAllOf(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := checkKeys(terminal, a.tupleSourceKeys(terminal, pointer, []string{"type", "nullable", "allOf", "properties", "required", "minProperties", "maxProperties"}), pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	ty, err := a.tupleNominalType(original, source, terminal, pointer, depth, true)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	return tupleScalarSignature{Kind: "object", Nullable: ty.Kind == Nullable, ModelSource: terminal}, nil
}
