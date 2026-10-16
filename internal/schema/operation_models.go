package schema

import (
	"reflect"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func (a *analyzer) operationModelName(owner, role, pointer string) (string, *diagnostic.Diagnostic) {
	name, err := naming.APIClassName(owner, pointer)
	if err != nil {
		return "", err
	}
	allocated, short := name+role+a.options.Suffix, role+a.options.Suffix
	if a.operationRoles == nil {
		a.operationRoles = map[string]string{}
	}
	if previous, ok := a.operationRoles[allocated]; ok && previous != short {
		a.operationRoles[allocated] = "" // Provenance ambiguity cannot authorize relocation.
	} else if !ok {
		a.operationRoles[allocated] = short
	}
	if a.operationOrigins == nil {
		a.operationOrigins = map[string][]string{}
	}
	found := false
	for _, origin := range a.operationOrigins[allocated] {
		found = found || origin == pointer
	}
	if !found {
		a.operationOrigins[allocated] = append(a.operationOrigins[allocated], pointer)
	}
	return allocated, nil
}

// Local declaration names are relative inside their Kotlin package. Only the
// exposed operation signature is qualified; component references already carry
// their original models-package identity. Never mutate borrowed element trees.
func qualifyOperationType(t TypeRef, local, qualified string) TypeRef {
	if t.Kind == Named && t.Name == local {
		t.Name = qualified
	}
	if t.Element != nil {
		element := qualifyOperationType(*t.Element, local, qualified)
		t.Element = &element
	}
	return t
}

func (a *analyzer) bindParameterModels(params []Parameter, owner, pointer string) ([]Declaration, *diagnostic.Diagnostic) {
	var declarations []Declaration
	for i := range params {
		p := &params[i]
		if p.DeepObject != nil {
			name, err := a.operationModelName(owner, "Filter", p.DeepObject.Pointer)
			if err != nil {
				return nil, err
			}
			model, fields, err := a.deepObject(p.DeepObject.Schema, p.DeepObject.Pointer, name)
			if err != nil {
				return nil, err
			}
			bound := *p.DeepObject
			bound.Fields = fields
			p.DeepObject = &bound
			p.Type = TypeRef{Kind: Named, Name: a.options.PackageName + ".api.models." + name}
			if !p.Required {
				p.Type = p.Type.Null()
			}
			declarations = append(declarations, model)
			continue
		}
		if p.InlineEnum == nil {
			continue
		}
		role, err := naming.APIClassName(p.Wire, pointer)
		if err != nil {
			return nil, err
		}
		switch role {
		case "Input", "Response", "Filter", "Event", "Data", "Item":
			role += "Param"
		}
		name, err := a.operationModelName(owner, role, pointer)
		if err != nil {
			return nil, err
		}
		model := *p.InlineEnum
		qualified := a.options.PackageName + ".api.models." + name
		p.Type = qualifyOperationType(p.Type, model.ClassName, qualified)
		p.Type = qualifyOperationType(p.Type, a.options.PackageName+".api.models."+model.ClassName, qualified)
		model.ClassName = name
		p.InlineEnum = &model
		declarations = append(declarations, &model)
	}
	return declarations, nil
}

func (a *analyzer) operationResponseSchema(v *document.Value, owner, pointer string) (TypeRef, []Declaration, *diagnostic.Diagnostic) {
	// JsonNull is a value of this non-null JSON carrier, never a Unit response.
	if ty, err := a.booleanValueType(v, pointer, 0); err != nil {
		return TypeRef{}, nil, err
	} else if ty != nil {
		return *ty, nil, nil
	}
	if err := object(v, pointer); err != nil {
		return TypeRef{}, nil, err
	}
	projection, folded, err := a.responseProjection(v, pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	if projection != nil {
		return *projection, nil, nil
	}
	if folded != nil {
		v, pointer = folded.Schema, folded.Pointer
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	view := v
	foldedObject := folded != nil
	union, err := a.readUnion(v, pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	if union != nil && len(union.Branches) == 1 && union.HasNull {
		branch := union.Branches[0]
		branchKind, _, err := schemaType(branch.Schema, branch.Pointer)
		if err != nil {
			return TypeRef{}, nil, err
		}
		if branchKind == "object" && !has(branch.Schema, "$ref") {
			view, kind, foldedObject = branch.Schema, "object", true
		}
	}
	role := "Response"
	local := kind == "object" && !has(v, "$ref")
	itemObject := false
	if kind == "array" && has(v, "items") {
		item := get(v, "items")
		itemKind, _, e := schemaType(item, pointer+"/items")
		if e != nil {
			return TypeRef{}, nil, e
		}
		itemObject = itemKind == "object" && !has(item, "$ref")
		if itemObject {
			role = "Item"
			local = true
		}
	}
	name := "Response"
	if local {
		name, err = a.operationModelName(owner, role, pointer)
		if err != nil {
			return TypeRef{}, nil, err
		}
	}
	var declarations []Declaration
	schema, source := v, pointer
	depth := 0
	if folded != nil {
		depth = 1
	}
	if itemObject {
		if err := checkKeys(v, a.collectionKeys([]string{"type", "nullable", "items"}), pointer); err != nil {
			return TypeRef{}, nil, err
		}
		schema, source = get(v, "items"), pointer+"/items"
		depth = 1
	}
	ty, err := a.operationValueType(schema, name, source, &declarations, map[string]bool{}, depth)
	if err != nil {
		return TypeRef{}, nil, err
	}
	// Top-level unions/compositions and enum response declarations have separate
	// legacy projection rules. Do not accidentally expose a new sealed API.
	if len(declarations) > 0 && (!local || !foldedObject && (has(schema, "anyOf") || has(schema, "oneOf") || has(schema, "allOf"))) {
		return TypeRef{}, nil, operationUnsupported(source)
	}
	if itemObject {
		view = schema
	}
	if local && dictionarySchema(view) {
		// Value declarations are package peers: aliases cannot own nested types.
		// Shared dictionary planning already uses the same Value/Inner identity
		// as component aliases, while preserving component reference packages.
		declarations = append(declarations, &ValueAlias{ClassName: name, Type: ty})
		target := ty
		ty = TypeRef{Kind: Named, Name: name, Element: &target}
	}
	ty = qualifyOperationType(ty, name, a.options.PackageName+".api.models."+name)
	if folded != nil {
		ty = ty.Null()
	}
	if itemObject {
		item := ty
		ty, err = a.checkedCollection(TypeRef{Kind: List, Element: &item}, v, pointer)
		if err != nil {
			return TypeRef{}, nil, err
		}
		if nullable {
			ty = ty.Null()
		}
	}
	return ty, declarations, nil
}

func (a *analyzer) operationDeclarations(client *Client) *diagnostic.Diagnostic {
	used := map[string]Declaration{}
	for _, group := range client.Groups {
		for _, op := range group.Operations {
			for _, decl := range op.Models {
				key := strings.ToLower(decl.Name())
				if previous := used[key]; previous != nil {
					if reflect.DeepEqual(previous, decl) {
						a.emissionHints.transferEquivalent(previous, decl)
						continue
					}
					return diagnostic.At("E_NAME_COLLISION", "Operation model filenames collide case-insensitively.", op.Pointer)
				}
				used[key] = decl
				client.Models = append(client.Models, decl)
			}
		}
	}
	return nil
}
