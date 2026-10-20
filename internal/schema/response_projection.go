package schema

import (
	"slices"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// emptyResponseBranch recognizes only the bounded response caller's empty
// projection. Exact not:{} is intentionally not enabled on model/request paths.
func emptyResponseBranch(v *document.Value, pointer string) (bool, *diagnostic.Diagnostic) {
	for _, member := range v.Members() {
		if member.Name != "not" && member.Name != "nullable" && !slices.Contains(metadata, member.Name) {
			return false, nil
		}
	}
	if err := checkKeys(v, []string{"not", "nullable"}, pointer); err != nil {
		return false, err
	}
	if _, _, err := schemaType(v, pointer); err != nil {
		return false, err
	}
	if has(v, "not") {
		p := naming.Pointer(pointer, "not")
		if err := object(get(v, "not"), p); err != nil {
			return false, err
		}
		if len(get(v, "not").Members()) != 0 {
			return false, unsupported(p)
		}
	}
	return true, nil
}

// responseProjection returns either a simple type or a borrowed branch to plan
// as a nullable response model. This keeps its source and ownership intact.
func (a *analyzer) responseProjection(v *document.Value, pointer string) (*TypeRef, *unionBranch, *diagnostic.Diagnostic) {
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return nil, nil, err
	}
	if kind == "null" {
		switch {
		case has(v, "enum"):
			if _, err := nullOnlyEnumType(v, pointer); err != nil {
				return nil, nil, err
			}
		case has(v, "const"):
			if _, err := constant("validation", "value", v, pointer, 0); err != nil {
				return nil, nil, err
			}
		default:
			if err := checkKeys(v, []string{"type", "nullable"}, pointer); err != nil {
				return nil, nil, err
			}
		}
		return &TypeRef{Kind: Scalar, Name: "kotlin.Unit"}, nil, nil
	}
	union, err := a.readUnion(v, pointer)
	if err != nil || union == nil {
		return nil, nil, err
	}
	var real []unionBranch
	for _, branch := range union.Branches {
		empty, err := emptyResponseBranch(branch.Schema, branch.Pointer)
		if err != nil {
			return nil, nil, err
		}
		if !empty && !(len(union.Branches) == 1 && union.HasNull) {
			if err := rejectObjectCounts(branch.Schema, branch.Pointer); err != nil {
				return nil, nil, err
			}
		}
		if empty {
			a.warn("W_HTTP_RESPONSE_PROJECTION", "An empty response union branch uses the compatibility Unit/nullable/raw JSON projection, not schema-membership validation.", branch.Pointer)
		} else {
			real = append(real, branch)
		}
	}
	if len(real) != len(union.Branches) {
		if len(real) == 0 {
			return &TypeRef{Kind: Scalar, Name: "kotlin.Unit"}, nil, nil
		}
		if len(real) == 1 {
			branch := real[0]
			kind, _, err := schemaType(branch.Schema, branch.Pointer)
			if err != nil {
				return nil, nil, err
			}
			properties := get(branch.Schema, "properties")
			if has(branch.Schema, "$ref") || kind == "object" && properties != nil && len(properties.Members()) != 0 {
				return nil, &branch, nil
			}
		}
	} else {
		projection, err := a.operationScalarUnion(union, 0)
		if err != nil || projection != nil {
			return projection, nil, err
		}
		if len(union.Branches) == 1 {
			return nil, nil, nil
		}
	}
	if err := a.rejectWriteOnlyProjection(v, pointer); err != nil {
		return nil, nil, err
	}
	// A raw result must still validate all non-empty borrowed branches. Empty
	// branches were checked above; do not validate a projected not:{} as a model.
	for _, branch := range real {
		if err := a.rejectValidationProjection(branch.Schema, branch.Pointer); err != nil {
			return nil, nil, err
		}
		context := *a
		context.usage = SharedUsage
		var scratch []Declaration
		if _, err := context.operationValueType(branch.Schema, "$Response", branch.Pointer, &scratch, map[string]bool{}, 1); err != nil {
			return nil, nil, err
		}
		a.warnings = context.warnings
	}
	a.warn("W_HTTP_RESPONSE_PROJECTION", "An operation response union uses raw JsonElement rather than generated variants or union validation.", union.Pointer)
	// JsonNull is already a JsonElement; this fallback is not a nullable container.
	return &TypeRef{Kind: JSON}, nil, nil
}
