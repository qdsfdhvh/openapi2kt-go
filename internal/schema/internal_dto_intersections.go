package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"strings"
)

// Different original-source roles share the same traversal without sharing
// admission or resetting their separate document-wide capture budgets.
type originalSourceCaptureBudget interface {
	charge(work, bytes int, source string) *diagnostic.Diagnostic
}

type intersectionCaptureBudget struct {
	Seen        map[*document.Value]bool
	Work, Bytes int
}

func (b *intersectionCaptureBudget) charge(work, bytes int, source string) *diagnostic.Diagnostic {
	if work > 16384-b.Work || bytes > (8<<20)-b.Bytes {
		return diagnostic.At("E_MODEL_LIMIT", "Internal intersection source capture exceeds the cumulative work or byte budget.", source)
	}
	b.Work += work
	b.Bytes += bytes
	return nil
}

// The existing ordinary DTO admission records the original lexical owner and
// validates exactly one retained constructor later. This role borrows only its
// fields, never gives an internal node component identity or predicts a name.
func (a *analyzer) internalDTOIntersection(target *resolvedSchema, source string, depth, steps int, intent *bool) (*resolvedSchema, int, *bool, *diagnostic.Diagnostic) {
	if composedModelCandidate(target.Schema) {
		return a.internalAllOfIntersection(target, source, depth, steps, intent)
	}
	if previous := a.borrowedDTOLeaves[target.Schema]; previous != nil && (previous.Owner != target.SourceCatalog || previous.Pointer != target.Pointer) {
		return nil, 0, nil, schemaReferenceUnsupported(source)
	}
	if target.Catalog != nil || target.SourceCatalog == nil || !a.internalIntersectionLexicalOwner(target) ||
		!directInternalModel(target.Schema, target.Pointer) || hasObjectCounts(target.Schema) || has(target.Schema, "default") ||
		a.hierarchySources[target.Schema] || a.hierarchySources[target.SourceCatalog.Schema] &&
		!((target.DirectLexical && !target.ComposedLexical && directInternalOwner(target.SourceCatalog.Schema)) ||
			(target.ComposedLexical && composedModelCandidate(target.SourceCatalog.Schema))) ||
		!a.admitBorrowedOrdinaryDTO(target, source) {
		return nil, 0, nil, schemaReferenceUnsupported(source)
	}
	if err := depthCheck(depth+steps, source); err != nil {
		return nil, 0, nil, err
	}
	budget := a.intersectionCaptures
	if !budget.Seen[target.Schema] {
		if _, _, err := a.borrowedAllOfSourceBudget(target.Schema, target.Pointer, source, budget); err != nil {
			return nil, 0, nil, err
		}
		use := a.borrowedDTOLeaves[target.Schema]
		if err := budget.charge(0, len(use.Bytes), source); err != nil {
			return nil, 0, nil, err
		}
		budget.Seen[target.Schema] = true
	}
	return target, steps, intent, nil
}

// A composed target keeps its own original source graph and final retained
// constructor proof. Lexical admission never replaces the final unique owner
// occurrence check, including for composed owners and named hierarchy leaves.
func (a *analyzer) internalAllOfIntersection(target *resolvedSchema, source string, depth, steps int, intent *bool) (*resolvedSchema, int, *bool, *diagnostic.Diagnostic) {
	owner := target.SourceCatalog
	if target.Catalog != nil || owner == nil || !a.internalIntersectionLexicalOwner(target) ||
		a.hierarchySources[target.Schema] || a.hierarchySources[owner.Schema] &&
		!((target.DirectLexical && directInternalOwner(owner.Schema)) ||
			(target.ComposedLexical && composedModelCandidate(owner.Schema))) ||
		hasObjectCounts(target.Schema) || has(target.Schema, "default") {
		return nil, 0, nil, schemaReferenceUnsupported(source)
	}
	if old := a.borrowedAllOfLeaves[target.Schema]; old != nil && (old.Owner != owner || old.Pointer != target.Pointer) {
		return nil, 0, nil, schemaReferenceUnsupported(source)
	}
	if err := depthCheck(depth+steps, source); err != nil {
		return nil, 0, nil, err
	}
	admitted, err := a.admitBorrowedAllOfModelBudget(target, source, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	if !admitted {
		return nil, 0, nil, schemaReferenceUnsupported(source)
	}
	budget := a.intersectionCaptures
	if !budget.Seen[target.Schema] {
		if _, _, err := a.borrowedAllOfSourceBudget(target.Schema, target.Pointer, source, budget); err != nil {
			return nil, 0, nil, err
		}
		if err := budget.charge(0, len(a.borrowedAllOfLeaves[target.Schema].Bytes), source); err != nil {
			return nil, 0, nil, err
		}
		budget.Seen[target.Schema] = true
	}
	return target, steps, intent, nil
}

func (a *analyzer) internalIntersectionLexicalOwner(target *resolvedSchema) bool {
	owner := target.SourceCatalog
	if owner == nil || a.components[owner.Wire] != owner || !strings.HasPrefix(target.Pointer, owner.Pointer+"/") {
		return false
	}
	if !target.ComposedLexical {
		return target.DirectLexical && directInternalOwner(owner.Schema)
	}
	if !directInternalOwner(owner.Schema) && !composedModelCandidate(owner.Schema) && !unionNominalCandidate(owner.Schema) {
		return false
	}
	// A composed owner's ordinary nested property/collection model has its own
	// source. Do not borrow a variant root whose physical API may also contain
	// owner common fields. Existing final source/occurrence proof decides the
	// unique retained owner; syntax alone cannot select a product occurrence.
	path := strings.TrimPrefix(target.Pointer, owner.Pointer)
	return strings.Contains(path, "/properties/") || strings.Contains(path, "/items") || strings.Contains(path, "/additionalProperties")
}
