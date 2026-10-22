package schema

import (
	"slices"
	"testing"
)

func TestNullableAllOfCountsCannotSubstituteOrDropContributions(t *testing.T) {
	root := loadComponentDocument(t, `{"Base":{"properties":{"value":{"type":"string"}},"minProperties":1},"Other":{"properties":{"other":{"type":"string"}},"minProperties":0},"Record":{"nullable":true,"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"note":{"type":"string"}}}]}}`)
	a := objectDefaultTestAnalyzer(true)
	a.root = root
	a.components = map[string]*component{}
	a.emissionHints = EmissionHints{}
	if _, err := a.analyze(root); err != nil {
		t.Fatal(err)
	}
	c := a.components["Record"]
	original, err := a.modelShape(c.Schema, c.Pointer, 0, true)
	if err != nil || len(original.CountSources) != 1 {
		t.Fatal(original, err)
	}
	other := a.components["Other"]
	for _, sources := range [][]field{
		nil,
		{{Schema: other.Schema, Pointer: other.Pointer, Depth: 1}},
		{{Schema: original.CountSources[0].Schema, Pointer: original.CountSources[0].Pointer, Depth: 0}},
		append(slices.Clone(original.CountSources), original.CountSources[0]),
	} {
		changed := *original
		changed.CountSources = sources
		if _, err := a.buildModelFromShape("Replacement", c.Schema, c.Pointer, &changed, nil, false, nil, true); err == nil {
			t.Fatal("altered nullable composition count inventory accepted", sources)
		}
	}
}
