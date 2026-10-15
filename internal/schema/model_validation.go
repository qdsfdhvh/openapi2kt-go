package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// modelPropertyType keeps the existing direct-DTO validation representation and
// emitted scopes. Collection leaves use TypeRef.Validation instead, so every
// typed consumer (not only DTO properties) can preserve recursive checks.
func (a *analyzer) modelPropertyType(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *PrimitiveValidation, *diagnostic.Diagnostic) {
	if !a.options.Validation || !hasPrimitiveBounds(v) {
		ty, err := a.propertyType(v, name, pointer, nested, used, depth)
		return ty, nil, err
	}
	if has(v, "enum") {
		kind, _, _ := schemaType(v, pointer)
		if kind == "integer" || kind == "number" || kind == "boolean" || kind == "null" {
			ty, err := a.propertyType(v, name, pointer, nested, used, depth)
			return ty, nil, err
		}
	}
	return a.boundedPrimitiveType(v, pointer, depth)
}

func (a *analyzer) boundedPrimitiveType(v *document.Value, pointer string, depth int) (TypeRef, *PrimitiveValidation, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, nil, err
	}
	if err := object(v, pointer); err != nil {
		return TypeRef{}, nil, err
	}
	if err := checkKeys(v, append([]string{"type", "nullable", "format"}, primitiveBoundKeys...), pointer); err != nil {
		return TypeRef{}, nil, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	if kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" {
		for _, key := range primitiveBoundKeys {
			if has(v, key) {
				return TypeRef{}, nil, unsupported(pointer + "/" + key)
			}
		}
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	validation, err := a.primitiveValidation(v, scalar, pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	ty := TypeRef{Kind: Scalar, Name: scalar}
	if nullable {
		ty = ty.Null()
	}
	return ty, validation, nil
}

// primitiveElementType grants inline bounds only below typed collection edges.
// Direct HTTP scalar slots retain their existing policy. A nullable single-leaf
// fold preserves checks; general union aggregation remains a separate feature.
func (a *analyzer) primitiveElementType(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if !a.options.Validation {
		return nil, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if has(v, "enum") {
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return nil, err
		}
		if kind == "integer" || kind == "number" || kind == "boolean" {
			ty, err := a.nativeEnumValueType(v, pointer, depth)
			return &ty, err
		}
	}
	leaf, source := v, pointer
	var union *unionView
	if !hasPrimitiveBounds(v) {
		var err *diagnostic.Diagnostic
		union, err = readUnion(v, pointer)
		if err != nil {
			return nil, err
		}
		if union == nil || len(union.Branches) != 1 || !union.HasNull || !hasPrimitiveBounds(union.Branches[0].Schema) {
			return nil, nil
		}
		leaf, source = union.Branches[0].Schema, union.Branches[0].Pointer
		depth++
	}
	ty, validation, err := a.boundedPrimitiveType(leaf, source, depth)
	if err != nil {
		return nil, err
	}
	if validation != nil {
		a.validationSources[validation] = source
	}
	if ty.Kind == Nullable {
		inner := *ty.Element
		inner.Validation = validation
		ty.Element = &inner
	} else {
		ty.Validation = validation
	}
	if union != nil {
		if union.OverlappingNulls {
			a.warn("W_ONEOF_NULL_OVERLAP", "Nullable folding does not enforce the exact oneOf match count for overlapping null branches.", union.Pointer)
		}
		if union.Nullable {
			ty = ty.Null()
		}
	}
	return &ty, nil
}

func (a *analyzer) collectionElementType(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	// A borrowed container retains raw/true source authority before planning.
	// Use its transparent backing here so dictionaries preserve JsonObject
	// projection and nullable/raw aliases do not become nominal barriers.
	if a.aliasPlanning && has(v, "$ref") {
		target, err := a.resolveSchema(text(get(v, "$ref")), pointer)
		if err != nil {
			return TypeRef{}, err
		}
		canonical := target.CanonicalJSON
		if canonical == nil {
			canonical = target.CanonicalCover
		}
		if canonical == nil && target.Catalog != nil {
			canonical = target.Catalog
			if canonical.BindingTarget != nil {
				canonical = canonical.BindingTarget
			}
		}
		if a.borrowedRawTrueLeaves[canonical] != nil && a.admitBorrowedRawTrue(target, pointer) || a.borrowedRecursiveCoverLeaves[canonical] != nil && a.admitBorrowedRecursiveCover(target, pointer) {
			if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
				return TypeRef{}, err
			}
			if _, _, err := schemaType(v, pointer); err != nil {
				return TypeRef{}, err
			}
			return a.referenceType(target, nullableOverride(v), depth, pointer)
		}
	}
	if ty, err := a.primitiveElementType(v, pointer, depth); err != nil {
		return TypeRef{}, err
	} else if ty != nil {
		return *ty, nil
	}
	return a.propertyType(v, name, pointer, nested, used, depth)
}

func (a *analyzer) operationElementType(v *document.Value, name, pointer string, declarations *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if ty, err := a.booleanValueType(v, pointer, depth); err != nil {
		return TypeRef{}, err
	} else if ty != nil {
		return *ty, nil
	}
	if ty, err := a.primitiveElementType(v, pointer, depth); err != nil {
		return TypeRef{}, err
	} else if ty != nil {
		return *ty, nil
	}
	return a.operationStructuralType(v, name, pointer, declarations, used, depth)
}
