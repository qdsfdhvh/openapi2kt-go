package schema

import "openapi-kt/internal/diagnostic"

// Nullable ordinary DTO union members use their existing source/declaration
// authority. This is field-like physical borrowing, not an allOf intersection.
func (a *analyzer) nullableObjectReferenceMember(branch unionBranch, depth int) (*resolvedSchema, bool, *diagnostic.Diagnostic) {
	if !has(branch.Schema, "$ref") {
		return nil, false, nil
	}
	if err := checkKeys(branch.Schema, []string{"$ref", "nullable"}, branch.Pointer); err != nil {
		return nil, false, err
	}
	if _, _, err := schemaType(branch.Schema, branch.Pointer); err != nil {
		return nil, false, err
	}
	target, err := a.resolveSchema(text(get(branch.Schema, "$ref")), branch.Pointer)
	if err != nil {
		return nil, false, err
	}
	if !directInternalModel(target.Schema, target.Pointer) || target.Catalog == nil || a.hierarchySources[target.Schema] {
		return nil, false, nil
	}
	ty, err := a.referenceType(target, nullableOverride(branch.Schema), depth, branch.Pointer)
	if err != nil {
		return nil, false, err
	}
	_, sourceNullable, err := schemaType(target.Schema, target.Pointer)
	if err != nil {
		return nil, false, err
	}
	if ty.Kind != Nullable && !sourceNullable {
		return nil, false, nil
	}
	if !a.admitBorrowedOrdinaryDTO(target, branch.Pointer) {
		return nil, false, schemaReferenceUnsupported(branch.Pointer)
	}
	a.recordBorrowedDTOAliasEdges(text(get(branch.Schema, "$ref")), branch.Pointer)
	return target, true, nil
}

func (a *analyzer) deriveObjectReferenceNullability(union *unionView, depth int) *diagnostic.Diagnostic {
	// Product intersections keep their separate original-shape contract.
	if len(union.Branches) == 0 || union.Branches[0].Shape != nil {
		return nil
	}
	count := 0
	for _, branch := range union.Branches {
		target, admitted, err := a.nullableObjectReferenceMember(branch, depth)
		if err != nil {
			return err
		}
		if admitted {
			ty, err := a.referenceType(target, nullableOverride(branch.Schema), depth, branch.Pointer)
			if err != nil {
				return err
			}
			if ty.Kind == Nullable {
				count++
				union.Nullable = true
			}
		}
	}
	if count > 0 && union.OneOf && count+union.NullSources > 1 {
		union.OverlappingNulls = true
	}
	return nil
}
