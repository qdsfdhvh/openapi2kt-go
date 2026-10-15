package schema

import "openapi-kt/internal/diagnostic"

// Common nominal fields borrow actual typed declarations. The final shared
// ownership barrier checks every registered source; this planner owns no DTO.
func (a *analyzer) commonNominalType(f *field) (TypeRef, bool, *diagnostic.Diagnostic) {
	if !has(f.Schema, "$ref") {
		return TypeRef{}, false, nil
	}
	if err := checkKeys(f.Schema, []string{"$ref", "nullable"}, f.Pointer); err != nil {
		return TypeRef{}, false, err
	}
	if _, _, err := schemaType(f.Schema, f.Pointer); err != nil {
		return TypeRef{}, false, err
	}
	target, err := a.resolveSchema(text(get(f.Schema, "$ref")), f.Pointer)
	if err != nil {
		return TypeRef{}, false, err
	}
	// Complete covers and raw JSON roles keep their existing transparent
	// common-field contract even though their original syntax is a union.
	if target.CanonicalCover != nil || target.CanonicalJSON != nil || target.Catalog != nil && (target.Catalog.RecursiveJSON || target.Catalog.BooleanRoot) {
		return TypeRef{}, false, nil
	}
	if !directInternalModel(target.Schema, target.Pointer) && !composedModelCandidate(target.Schema) && !unionNominalCandidate(target.Schema) && !a.hierarchySources[target.Schema] {
		return TypeRef{}, false, nil
	}
	a.recordBorrowedDTOAliasEdges(text(get(f.Schema, "$ref")), f.Pointer)
	// Reuse original declaration/source authorities, including actual unique
	// Models/unions/hierarchy lineage and retained aliases at the final barrier.
	if !a.admitBorrowedOrdinaryDTO(target, f.Pointer) && !a.admitBorrowedProductUnion(target, f.Pointer) && !a.admitBorrowedAllOfModel(target, f.Pointer) && !a.admitBorrowedUnion(target, f.Pointer) && !a.admitBorrowedHierarchy(target, f.Pointer) {
		return TypeRef{}, false, nil
	}
	ty, err := a.referenceType(target, nullableOverride(f.Schema), f.Depth, f.Pointer)
	if err != nil {
		return TypeRef{}, true, err
	}
	if bare(ty).Kind != Named {
		return TypeRef{}, false, nil
	}
	return ty, true, nil
}
