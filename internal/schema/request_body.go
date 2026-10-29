package schema

import (
	"mime"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// RequestBody owns only the selected media's declarations. Shared component
// references retain their model context; inline plain objects are request-only.
type RequestBody struct {
	Type     TypeRef
	Required bool
	Models   []Declaration
	// FormFields are selected-only, flat scalar properties of the retained
	// request DTO. JSON requests leave this nil, including discarded form media.
	// Optional nullable property values omit their form key when absent.
	FormFields []Property
	// RawMedia retains the selected raw scalar media spelling and parameters.
	// A non-null builtin payload delegates byte/charset IO to Ktor; an optional
	// body uses parameter null only for absence, never as a raw wire null.
	RawMedia string
}

func (a *analyzer) requestBody(v *document.Value, pointer, owner string) (*RequestBody, *diagnostic.Diagnostic) {
	if v == nil {
		return nil, nil
	}
	v, pointer, err := a.operationReference(v, pointer)
	if err != nil {
		return nil, err
	}
	if err := operationKeys(v, []string{"description", "required", "content"}, pointer); err != nil {
		return nil, err
	}
	required := false
	if has(v, "required") {
		var ok bool
		required, ok = get(v, "required").Bool()
		if !ok {
			return nil, invalid("required must be boolean.", pointer+"/required")
		}
	}
	content := get(v, "content")
	if err := object(content, pointer+"/content"); err != nil {
		return nil, err
	}
	media := jsonMedia(content)
	form := false
	if media == "" {
		media = formRequestMedia(content)
		form = media != ""
	}
	raw := media == ""
	if raw {
		media = rawValueMedia(content)
	}
	if media == "" {
		return nil, operationUnsupported(pointer + "/content")
	}
	var selected TypeRef
	var selectedModels []Declaration
	var selectedSchema *document.Value
	var selectedPointer string
	for _, entry := range httpMembers(content) {
		p := naming.Pointer(pointer+"/content", entry.Name)
		if err := operationKeys(entry.Value, []string{"schema", "example", "examples"}, p); err != nil {
			return nil, err
		}
		s := get(entry.Value, "schema")
		if entry.Name == media {
			selectedSchema, selectedPointer = s, p+"/schema"
		}
		// A literal true JSON root is a value, not a structural request DTO.
		if ty, err := a.booleanValueType(s, p+"/schema", 0); err != nil {
			return nil, err
		} else if ty != nil {
			if entry.Name == media {
				selected, selectedModels = *ty, nil
			}
			continue
		}
		if err := object(s, p+"/schema"); err != nil {
			return nil, err
		}
		view := s
		union, err := a.readUnion(s, p+"/schema")
		if err != nil {
			return nil, err
		}
		if union != nil && len(union.Branches) == 1 && union.HasNull {
			view = union.Branches[0].Schema
		}
		kind, _, err := schemaType(view, p+"/schema")
		if err != nil {
			return nil, err
		}
		plain := kind == "object" && !has(view, "$ref") && !has(view, "allOf") && !has(view, "anyOf") && !has(view, "oneOf")
		nominal := union != nil && (!union.Nullable || len(union.Branches) > 1)
		local := plain || nominal
		name := "Input"
		if local || tupleFalseContainerCandidate(s) || tupleInlineAllOfContainerCandidate(s) || tupleCollectionContainerCandidate(s) || tupleUnionContainerCandidate(s) {
			name, err = a.operationModelName(owner, "Input", p+"/schema")
			if err != nil {
				return nil, err
			}
		}
		context := *a
		context.usage = SharedUsage
		if plain {
			context.usage = RequestObjectUsage
		}
		var declarations []Declaration
		var ty TypeRef
		if nominal {
			var projection *TypeRef
			projection, err = context.operationScalarUnion(union, 0)
			if err == nil {
				if projection != nil {
					ty = *projection
				} else {
					ty, err = context.unionType(union, name, p+"/schema", &declarations, map[string]bool{}, 0, true)
				}
			}
		} else {
			ty, err = context.operationValueType(s, name, p+"/schema", &declarations, map[string]bool{}, 0)
		}
		if err != nil {
			return nil, err
		}
		a.warnings = context.warnings
		if len(declarations) != 0 && !local && !a.onlyTupleFalsePeers(declarations) && !a.onlyTupleInlineAllOfPeers(declarations) && !a.onlyTupleCollectionPeers(declarations) && !a.onlyTupleUnionPeers(declarations) {
			return nil, operationUnsupported(p + "/schema")
		}
		if nominal && len(declarations) != 0 {
			if _, ok := declarations[0].(*ObjectUnion); !ok || len(declarations) != 1 {
				return nil, operationUnsupported(p + "/schema")
			}
			// A root UnionSealed input keeps its nominal API even when the
			// schema is nullable. Only requestBody.required controls the Kotlin
			// parameter's nullability; Unknown can carry a raw JSON null.
			if union.Nullable {
				ty = bare(ty)
				context.warn("W_HTTP_REQUEST_UNION_PROJECTION", "A nullable nominal request union keeps a non-null declaration; requestBody.required controls parameter nullability and Unknown can carry JSON null.", union.Pointer)
				a.warnings = context.warnings
			}
		}
		if plain && dictionarySchema(view) {
			declarations = append(declarations, &ValueAlias{ClassName: name, Type: ty})
			ty = TypeRef{Kind: Named, Name: name}
		}
		ty = a.qualifyTupleOperationType(ty, name, a.options.PackageName+".api.models."+name)
		if a.onlyTupleFalsePeers(declarations) || a.onlyTupleInlineAllOfPeers(declarations) || a.onlyTupleCollectionPeers(declarations) || a.onlyTupleUnionPeers(declarations) {
			for _, d := range declarations {
				ty = a.qualifyTupleOperationType(ty, d.Name(), a.options.PackageName+".api.models."+d.Name())
			}
		}
		if entry.Name == media {
			selected, selectedModels = ty, declarations
		}
	}
	body := &RequestBody{Type: selected, Required: required, Models: selectedModels}
	if raw {
		if _, _, err := mime.ParseMediaType(media); err != nil {
			return nil, operationUnsupported(naming.Pointer(pointer+"/content", media))
		}
		if selected.Kind == Nullable || !rawValueType(media, selected) {
			return nil, operationUnsupported(selectedPointer)
		}
		body.RawMedia = media
	}
	if form {
		fields, err := formRequestFields(selectedSchema, selectedPointer, selectedModels)
		if err != nil {
			return nil, err
		}
		body.FormFields = fields
	}
	if !required {
		body.Type = body.Type.Null()
	}
	return body, nil
}
