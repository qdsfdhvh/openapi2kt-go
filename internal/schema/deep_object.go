package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// DeepObject borrows its schema so inherited parameters and fallback method
// names can bind independent Filter declarations without mutating shared IR.
type DeepObject struct {
	Schema  *document.Value
	Pointer string
	Fields  []DeepObjectField
}

type DeepObjectField struct {
	Name, Wire                 string
	Nullable, Enum, ValueClass bool
}

func (a *analyzer) deepObject(v *document.Value, pointer, name string) (*Model, []DeepObjectField, *diagnostic.Diagnostic) {
	if err := a.rejectDTOCountProjection(v, pointer); err != nil {
		return nil, nil, err
	}
	if has(v, "$ref") || has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") {
		return nil, nil, operationUnsupported(pointer)
	}
	context := *a
	context.usage = SharedUsage
	context.fieldUnions = RawOperationFieldUnions
	model, err := context.buildModel(name, v, pointer, 0)
	if err != nil {
		return nil, nil, err
	}
	a.warnings = context.warnings
	var fields []DeepObjectField
	for _, member := range get(v, "properties").Members() {
		var ty TypeRef
		fieldName := ""
		for _, property := range model.Properties {
			if property.WireName == member.Name {
				ty, fieldName = property.Type, property.Name
				break
			}
		}
		for _, value := range model.Constants {
			if value.WireName == member.Name {
				ty, fieldName = value.Type, value.Name
				break
			}
		}
		plain := bare(ty)
		enum := a.parameterEnum(plain)
		for _, d := range model.Nested {
			if value, ok := d.(*StringEnum); ok && plain.Kind == Named && value.Name() == plain.Name {
				enum = true
			}
		}
		if fieldName == "" {
			return nil, nil, operationUnsupported(naming.Pointer(pointer+"/properties", member.Name))
		}
		if plain.Kind != Scalar && plain.Kind != InlineValue && !enum {
			a.warn("W_HTTP_PARAMETER_PROJECTION", "A structured filter field is passed to Ktor as one value using toString, not recursive query encoding.", naming.Pointer(pointer+"/properties", member.Name))
		}
		fields = append(fields, DeepObjectField{Name: fieldName, Wire: member.Name, Nullable: ty.Kind == Nullable, Enum: enum, ValueClass: plain.Kind == InlineValue})
	}
	return model, fields, nil
}
