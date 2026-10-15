package schema

import "openapi-kt/internal/diagnostic"

// FieldUnionPolicy is independent of defaults/presence. An inline filter uses
// ordinary nullable defaults, but has no component primitive payload builder.
type FieldUnionPolicy uint8

const (
	NativeFieldUnions FieldUnionPolicy = iota
	RawOperationFieldUnions
	RawParameterValueUnions
)

// operationFieldUnion validates borrowed branches without emitting their scratch
// declarations. Request-only and filter fields use raw JSON, not root request
// empty variants or component native payload wrappers.
func (a *analyzer) operationFieldUnion(union *unionView, name string, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, union.Pointer); err != nil {
		return TypeRef{}, err
	}
	for _, branch := range union.Branches {
		if err := a.rejectValidationProjection(branch.Schema, branch.Pointer); err != nil {
			return TypeRef{}, err
		}
		context := *a
		context.usage = SharedUsage
		context.fieldUnions = NativeFieldUnions
		var scratch []Declaration
		if _, err := context.propertyType(branch.Schema, name, branch.Pointer, &scratch, map[string]bool{}, depth); err != nil {
			return TypeRef{}, err
		}
		a.warnings = context.warnings
	}
	if a.fieldUnions == RawParameterValueUnions {
		a.warn("W_HTTP_PARAMETER_PROJECTION", "A parameter union uses raw JsonElement and Ktor stringification, without membership validation.", union.Pointer)
	} else if a.usage == SharedUsage {
		a.warn("W_HTTP_FILTER_UNION_PROJECTION", "An inline filter field union uses raw JsonElement without native payload wrappers or membership validation.", union.Pointer)
	} else {
		a.warn("W_HTTP_FIELD_UNION_PROJECTION", "A request-only field union uses raw JsonElement instead of native payload wrappers or union-membership validation.", union.Pointer)
	}
	return TypeRef{Kind: JSON}, nil
}
