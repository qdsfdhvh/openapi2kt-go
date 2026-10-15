package schema

import (
	"slices"
	"testing"

	"openapi-kt/internal/document"
)

func TestOriginalObjectCountSourcesRetainPostorderNodesAndDepth(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"type":"object","allOf":[{"properties":{"value":{"type":"string"}},"minProperties":1},{"type":"object","allOf":[{"properties":{"value":{"type":"string"}},"maxProperties":4}],"minProperties":2}],"maxProperties":3}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	v := get(get(get(root, "components"), "schemas"), "Record")
	a := analyzer{root: root, options: Options{Validation: true}, collectionSources: map[*CollectionValidation]string{}}
	s, failure := a.modelShape(v, "/components/schemas/Record", 0, true)
	if failure != nil || len(s.CountSources) != 4 {
		t.Fatal(s, failure)
	}
	members := get(v, "allOf").Items()
	want := []field{
		{Schema: members[0], Pointer: "/components/schemas/Record/allOf/0", Depth: 1},
		{Schema: get(members[1], "allOf").Items()[0], Pointer: "/components/schemas/Record/allOf/1/allOf/0", Depth: 2},
		{Schema: members[1], Pointer: "/components/schemas/Record/allOf/1", Depth: 1},
		{Schema: v, Pointer: "/components/schemas/Record", Depth: 0},
	}
	if !slices.Equal(s.CountSources, want) {
		t.Fatal(s.CountSources, want)
	}
	plan, failure := a.ordinaryObjectCounts(v, "/components/schemas/Record", s, true, true)
	if failure != nil || *plan.Minimum != 2 || *plan.Maximum != 3 {
		t.Fatal(plan, failure)
	}
	bytes, marshalErr := root.MarshalJSON()
	if marshalErr != nil || string(bytes) != string(input) {
		t.Fatal("Original tree rewritten", marshalErr)
	}
	copied, e := document.Load([]byte(`{"properties":{"value":{"type":"string"}},"minProperties":1}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	altered := slices.Clone(s.CountSources)
	altered[0].Schema = copied
	changed := *s
	changed.CountSources = altered
	if _, failure := a.ordinaryObjectCounts(v, "/components/schemas/Record", &changed, true, true); failure == nil {
		t.Fatal("Same-byte replacement count source authorized")
	}
	altered = slices.Clone(s.CountSources)
	altered[0].Pointer += "/missing"
	changed.CountSources = altered
	if _, failure := a.ordinaryObjectCounts(v, "/components/schemas/Record", &changed, true, true); failure == nil {
		t.Fatal("Changed count source location authorized")
	}
	altered = slices.Clone(s.CountSources)
	altered[0].Depth = 49
	changed.CountSources = altered
	if _, failure := a.ordinaryObjectCounts(v, "/components/schemas/Record", &changed, true, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("Cached original source work lost", failure)
	}
	changed.CountSources = s.CountSources[:3]
	if _, failure := a.ordinaryObjectCounts(v, "/components/schemas/Record", &changed, true, true); failure == nil {
		t.Fatal("Missing own root count source authorized")
	}
}
