package schema

import (
	"maps"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"slices"
)

// Return borrowed original winning fields, never an expanded properties node.
// Both the constructor and key planner use the existing ordinary model shape.
func (a *analyzer) dtoUniqueOriginalFields(v *document.Value, p string) ([]field, map[string]bool, *diagnostic.Diagnostic) {
	if composedModelCandidate(v) {
		if a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{}) {
			return nil, nil, unsupported(p)
		}
		shape, e := a.modelShape(v, p, 0, true)
		if e != nil {
			return nil, nil, e
		}
		if len(shape.CountSources) != 0 {
			return nil, nil, unsupported(p)
		}
		constructor := false
		for _, f := range shape.Fields {
			constructor = constructor || !has(f.Schema, "const")
		}
		if !constructor {
			return nil, nil, unsupported(p)
		}
		return slices.Clone(shape.Fields), maps.Clone(shape.Required), nil
	}
	required, e := objectDefaultRequiredNames(v, p)
	if e != nil {
		return nil, nil, e
	}
	fields := []field{}
	properties := get(v, "properties")
	if properties == nil {
		return fields, required, nil
	}
	for _, member := range properties.Members() {
		fields = append(fields, field{Wire: member.Name, Schema: member.Value, Pointer: naming.Pointer(p+"/properties", member.Name)})
	}
	return fields, required, nil
}

func (a *analyzer) dtoUniqueOriginalShapeMatches(src *dtoUniqueSource) bool {
	fields, required, e := a.dtoUniqueOriginalFields(src.Schema, src.Pointer)
	if e != nil || !slices.Equal(fields, src.Fields) || !maps.Equal(required, src.Required) {
		return false
	}
	nodes, refs, e := a.borrowedAllOfSource(src.Schema, src.Pointer)
	return e == nil && slices.Equal(refs, src.ShapeRefs) && len(nodes) == len(src.Nodes) && sameBorrowedHierarchyNodes(nodes, src.Nodes)
}

type dtoAllOfGetterSnapshot struct {
	Property              *Property
	Name, Wire, Checks    string
	Validation            *PrimitiveValidation
	Types                 []namedCollectionSourceType
	Default               string
	DefaultNull, Deferred bool
}

func (a *analyzer) recordDTOAllOfGetters(model *Model, fields []field) {
	if a.dtoAllOfGetters == nil {
		a.dtoAllOfGetters = map[*Model]map[string]dtoAllOfGetterSnapshot{}
	}
	if a.dtoAllOfGetters[model] != nil {
		return
	}
	a.dtoAllOfGetters[model] = map[string]dtoAllOfGetterSnapshot{}
	byWire := map[string]field{}
	for _, f := range fields {
		byWire[f.Wire] = f
	}
	for i := range model.Properties {
		p := &model.Properties[i]
		f := byWire[p.WireName]
		deferred := has(f.Schema, "default") && a.deferredFieldDefault(f.Schema, f.Pointer)
		a.dtoAllOfGetters[model][p.WireName] = dtoAllOfGetterSnapshot{Property: p, Name: p.Name, Wire: p.WireName, Checks: namedCollectionChecks(TypeRef{Validation: p.Validation}), Validation: p.Validation, Types: dtoOriginalGetterTypes(&p.Type), Default: p.Default, DefaultNull: p.DefaultNull, Deferred: deferred}
	}
}

func (a *analyzer) dtoAllOfOriginalGettersMatch(model *Model) bool {
	originals := a.dtoAllOfGetters[model]
	if len(originals) != len(model.Properties) {
		return false
	}
	for i := range model.Properties {
		p := &model.Properties[i]
		original, ok := originals[p.WireName]
		if !ok || original.Property != p || original.Name != p.Name || original.Wire != p.WireName || original.Validation != p.Validation || original.Checks != namedCollectionChecks(TypeRef{Validation: p.Validation}) || len(original.Types) == 0 {
			return false
		}
		if original.DefaultNull != p.DefaultNull || !original.Deferred && original.Default != p.Default {
			return false
		}
		for _, snapshot := range original.Types {
			value := *snapshot.Node
			value.SchemaSteps = 0
			if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
				return false
			}
		}
	}
	return true
}
