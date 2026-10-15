package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestDirectNullableComponentCountsOriginalPhysicalAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"type":["object","null"],"minProperties":1,"maxProperties":2,"properties":{"value":{"type":"string"}}},"Other":{"nullable":true,"minProperties":0,"properties":{"other":{"type":"boolean"}}},"Alias":{"$ref":"#/components/schemas/Record"},"Narrow":{"$ref":"#/components/schemas/Alias","nullable":false}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.counts", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, failure := a.analyze(root)
	if failure != nil {
		t.Fatal(failure)
	}
	c := a.components["Record"]
	var model *Model
	for _, d := range ds {
		if d.Name() == "Record" {
			model = d.(*Model)
		}
	}
	if model == nil || !model.Nullable || len(model.Properties) != 1 || model.PropertyCount == nil || *model.PropertyCount.Minimum != 1 || *model.PropertyCount.Maximum != 2 || a.modelOrigins[model] != c.Schema || a.modelPointers[model] != c.Pointer || len(a.components) != 4 {
		t.Fatal("original nominal constructor/count authority lost", model)
	}
	for _, name := range []string{"Record", "Alias", "Narrow"} {
		target, failure := a.resolveSchema("#/components/schemas/"+name, "/use")
		if failure != nil {
			t.Fatal(failure)
		}
		ty, failure := a.referenceType(target, nil, 0, "/use")
		if failure != nil || bare(ty).Name != "org.example.counts.models.Record" || (ty.Kind == Nullable) != (name != "Narrow") {
			t.Fatal(name, ty, failure)
		}
	}
	shape, failure := a.modelShape(c.Schema, c.Pointer, 0, true)
	if failure != nil {
		t.Fatal(failure)
	}
	if len(shape.CountSources) != 1 || shape.CountSources[0].Schema != c.Schema || shape.CountSources[0].Pointer != c.Pointer || shape.CountSources[0].Depth != 0 {
		t.Fatal(shape.CountSources)
	}
	other := a.components["Other"]
	// A real unrelated source is still not a direct root's local own count.
	changed := *shape
	changed.CountSources = append([]field{{Schema: other.Schema, Pointer: other.Pointer, Depth: 1}}, shape.CountSources...)
	if _, failure := a.buildModelFromShape("Replacement", c.Schema, c.Pointer, &changed, nil, false, nil, true); failure == nil {
		t.Fatal("borrowed counts authorized by direct nullable syntax")
	}
	copiedRoot, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	copied := get(get(get(copiedRoot, "components"), "schemas"), "Record")
	changed.CountSources = []field{{Schema: copied, Pointer: c.Pointer}}
	if _, failure := a.buildModelFromShape("Replacement", c.Schema, c.Pointer, &changed, nil, false, nil, true); failure == nil {
		t.Fatal("same-byte count source replacement acquired authority")
	}
	changed.CountSources = []field{{Schema: c.Schema, Pointer: c.Pointer, Depth: 49}}
	if _, failure := a.buildModelFromShape("Replacement", c.Schema, c.Pointer, &changed, nil, false, nil, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("count source depth discarded", failure)
	}
	before, _ := root.MarshalJSON()
	if string(before) != string(input) {
		t.Fatal("immutable original tree rewritten")
	}
}
