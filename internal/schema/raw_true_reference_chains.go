package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"sort"
)

// JSON spelling does not establish a source role. Raw requires a retained
// ordinary empty Model; literal true requires its original JSON ValueAlias.
type rawTrueOrigin struct {
	Owner                *component
	Schema               *document.Value
	Pointer, SourceBytes string
	True                 bool
}

func (a *analyzer) rawTrueCandidate(c *component) bool {
	if c == nil || c.Reference != "" || c.RecursiveJSON {
		return false
	}
	if c.BooleanRoot {
		return a.booleanTrueValue(c.Schema)
	}
	kind, _, err := schemaType(c.Schema, c.Pointer)
	if err != nil || c.Scalar != "" || c.ValueClass {
		return false
	}
	if kind == "object" && nullableRawCandidate(c.Schema) {
		return true
	}
	if composedModelCandidate(c.Schema) && !a.containsObjectCompositionUnion(c.Schema, 0, map[*document.Value]bool{}) {
		shape, err := a.objectShape(c.Schema, c.Pointer, 0)
		return err == nil && len(shape.Fields) == 0
	}
	return false
}

func (a *analyzer) recordRawTrueOrigin(d Declaration, c *component) {
	if !a.rawTrueCandidate(c) {
		return
	}
	bytes, _ := c.Schema.MarshalJSON()
	a.rawTrueOrigins[d] = rawTrueOrigin{c, c.Schema, c.Pointer, string(bytes), c.BooleanRoot}
}

func (a *analyzer) rawTrueSourceMatches(origin rawTrueOrigin) bool {
	c := origin.Owner
	if a.components[c.Wire] != c || c.Schema != origin.Schema || c.Pointer != origin.Pointer || c.BooleanRoot != origin.True || !a.rawTrueCandidate(c) {
		return false
	}
	bytes, _ := c.Schema.MarshalJSON()
	return string(bytes) == origin.SourceBytes
}

func (a *analyzer) rawTrueDeclaration(d Declaration, origin rawTrueOrigin) bool {
	if d.Name() != origin.Owner.Name {
		return false
	}
	if origin.True {
		alias, ok := d.(*ValueAlias)
		return ok && alias.Type == (TypeRef{Kind: JSON}) && a.booleanTrueValue(origin.Schema)
	}
	model, ok := d.(*Model)
	if !ok {
		return false
	}
	nullable, err := a.nominalSourceNullable(&resolvedSchema{Catalog: origin.Owner, Schema: origin.Schema, Pointer: origin.Pointer})
	return err == nil && model.Nullable == nullable && len(model.Properties) == 0 && len(model.Constants) == 0 && len(model.Nested) == 0 && model.PropertyCount == nil && a.modelOrigins[model] == origin.Schema && a.modelPointers[model] == origin.Pointer
}

func (a *analyzer) bindNamedChainRawTrue(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	c := target.CanonicalJSON
	if c == nil {
		c = target.Catalog
		if c.BindingTarget != nil {
			c = c.BindingTarget
		}
	}
	if target.Schema != c.Schema || target.Pointer != c.Pointer || !a.rawTrueCandidate(c) {
		return false
	}
	if a.nominalsLinked {
		found := false
		for d, origin := range a.rawTrueOrigins {
			if origin.Owner == c && a.rawTrueSourceMatches(origin) && a.rawTrueDeclaration(d, origin) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	target.CanonicalJSON = c
	if previous, ok := a.rawTrueUses[c]; !ok || source < previous {
		a.rawTrueUses[c] = source
	}
	return true
}

func (a *analyzer) validateRawTrueBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	used := []*component{}
	for c := range a.rawTrueUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.rawTrueUses[used[i]] < a.rawTrueUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.rawTrueOrigins {
			if actual[d] != 0 && origin.Owner == c && a.rawTrueSourceMatches(origin) && a.rawTrueDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.rawTrueUses[c])
		}
	}
	return nil
}

func (a *analyzer) rawTrueType(c *component) TypeRef {
	ty := TypeRef{Kind: JSON}
	if !c.BooleanRoot {
		target := &resolvedSchema{Catalog: c, Schema: c.Schema, Pointer: c.Pointer}
		nullable, _ := a.nominalSourceNullable(target)
		if nullable {
			ty = ty.Null()
		}
	}
	return ty
}
