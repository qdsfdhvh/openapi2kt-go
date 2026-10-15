package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// operationStructuralType is used only below simple operation containers.
// Plain request/response DTO properties continue through propertyType instead.
func (a *analyzer) operationStructuralType(v *document.Value, name, pointer string, declarations *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if structuredLiteral(v) {
		return a.propertyType(v, name, pointer, declarations, used, depth)
	}
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, err
	}
	if err := object(v, pointer); err != nil {
		return TypeRef{}, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if kind != "object" || has(v, "$ref") || has(v, "allOf") || has(v, "anyOf") || has(v, "oneOf") {
		return a.operationValueType(v, name, pointer, declarations, used, depth)
	}
	ty := TypeRef{Kind: JSONObject}
	if dictionarySchema(v) {
		value, err := a.dictionaryValue(v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		if value != nil {
			item, err := a.operationElementType(value, name+"Value", pointer+"/additionalProperties", declarations, used, depth+1)
			if err != nil {
				return TypeRef{}, err
			}
			if bare(item).Kind != JSON {
				ty = TypeRef{Kind: Map, Element: &item}
			}
		}
	} else {
		if err := a.rejectValidationProjection(v, pointer); err != nil {
			return TypeRef{}, err
		}
		validation := *a
		validation.usage = SharedUsage
		validation.warnings = nil
		// The projection guard rejects enforcing checks before scratch validation.
		// Retain opt-in here so valid uniqueItems:false remains an effect-free no-op.
		var scratch []Declaration
		if _, err := validation.propertyType(v, "$Structural", pointer, &scratch, map[string]bool{}, depth); err != nil {
			return TypeRef{}, err
		}
		// Values are caller-supplied JSON, including unknown keys. DTO-specific
		// field/default/extra-key warnings are replaced by this explicit policy.
		if properties := get(v, "properties"); properties == nil || len(properties.Members()) == 0 {
			ty = TypeRef{Kind: JSON}
			a.warn("W_JSON_ELEMENT_APPROXIMATION", "An empty object schema uses JsonElement and permits non-object values.", pointer)
		} else {
			value := TypeRef{Kind: JSON}.Null()
			ty = TypeRef{Kind: Map, Element: &value}
			a.warn("W_HTTP_VALUE_PROJECTION", "An object inside a simple operation value uses raw JSON map values, not DTO fields, defaults or field validation.", pointer)
		}
	}
	if dictionarySchema(v) {
		ty, err = a.checkedCollection(ty, v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
	}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}

// operationScalarUnion applies shared normalization without importing a model
// property's enum declaration policy into a simple HTTP value. Object folds and
// component references remain on their existing context-aware paths.
func (a *analyzer) operationScalarUnion(union *unionView, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	decision, err := a.normalizeNested(union, depth+1)
	if err != nil {
		return nil, err
	}
	var ty TypeRef
	if decision != nil {
		ty = TypeRef{Kind: Scalar, Name: decision.Scalar}
		if decision.Enum != nil {
			ty.Name = "kotlin.String"
			a.warn("W_ENUM_NATIVE_APPROXIMATION", "An operation enum uses its scalar wire type without enforcing membership.", union.Pointer)
		}
	} else if len(union.Branches) == 1 && union.HasNull {
		branch := union.Branches[0]
		kind, _, err := schemaType(branch.Schema, branch.Pointer)
		if err != nil {
			return nil, err
		}
		if has(branch.Schema, "$ref") || kind != "string" && kind != "boolean" && kind != "integer" && kind != "number" && !(kind == "" && (has(branch.Schema, "enum") || has(branch.Schema, "const"))) {
			return nil, nil
		}
		var scratch []Declaration
		ty, err = a.operationValueType(branch.Schema, "$Scalar", branch.Pointer, &scratch, map[string]bool{}, depth+1)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}
	if union.OverlappingNulls {
		a.warn("W_ONEOF_NULL_OVERLAP", "Nullable folding does not enforce the exact oneOf match count for overlapping null branches.", union.Pointer)
	}
	if union.Nullable {
		ty = ty.Null()
	}
	return &ty, nil
}

// Operation values follow the legacy simple-body projection, not the model
// property's enum declaration policy. Named component identities stay intact.
func (a *analyzer) operationValueType(v *document.Value, name, pointer string, declarations *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if structuredLiteral(v) {
		return a.propertyType(v, name, pointer, declarations, used, depth)
	}
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, err
	}
	if err := object(v, pointer); err != nil {
		return TypeRef{}, err
	}
	union, err := readUnion(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if union != nil {
		projection, err := a.operationScalarUnion(union, depth)
		if err != nil {
			return TypeRef{}, err
		}
		if projection != nil {
			return *projection, nil
		}
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if !has(v, "$ref") && has(v, "enum") && (kind == "string" || kind == "") {
		if _, err := buildStringEnum("$OperationEnum", v, pointer); err != nil {
			return TypeRef{}, err
		}
		a.warn("W_ENUM_NATIVE_APPROXIMATION", "An operation enum uses its scalar wire type without enforcing membership.", naming.Pointer(pointer, "enum"))
		scalar, err := scalarMapping("string", v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		ty := TypeRef{Kind: Scalar, Name: scalar}
		if nullable {
			ty = ty.Null()
		}
		return ty, nil
	}
	if kind == "array" && !has(v, "$ref") {
		decision, err := a.arrayItems(v, pointer, depth)
		if err != nil {
			return TypeRef{}, err
		}
		var item TypeRef
		if decision.Projection != nil {
			item = *decision.Projection
		} else {
			item, err = a.operationElementType(decision.Selected, name+"Inner", decision.Pointer, declarations, used, depth+1)
			if err != nil {
				return TypeRef{}, err
			}
		}
		ty, err := a.checkedCollection(TypeRef{Kind: List, Element: &item}, v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		if nullable {
			ty = ty.Null()
		}
		return ty, nil
	}
	return a.propertyType(v, name, pointer, declarations, used, depth)
}
