package schema

import (
	"slices"
	"testing"

	"openapi-kt/internal/document"
)

func TestPrimitiveContributionsKeepTraversalOrderAndCompleteOriginalOccurrences(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"allOf":[{"properties":{"value":{"type":"string","minLength":1}}},{"allOf":[{"properties":{"value":{"type":"string","maxLength":4}}},{"properties":{"value":{"type":"string","minLength":2}}}]}]}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	v := get(get(get(root, "components"), "schemas"), "Record")
	a := analyzer{root: root, options: Options{Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}}
	s, err := a.modelShape(v, "/components/schemas/Record", 0, true)
	if err != nil || len(s.Contributions) != 3 || len(s.Fields) != 1 || len(s.Discarded) != 2 {
		t.Fatal(s, err)
	}
	want := []string{"/components/schemas/Record/allOf/0/properties/value", "/components/schemas/Record/allOf/1/allOf/0/properties/value", "/components/schemas/Record/allOf/1/allOf/1/properties/value"}
	members := get(v, "allOf").Items()
	nested := get(members[1], "allOf").Items()
	originals := []*document.Value{get(get(members[0], "properties"), "value"), get(get(nested[0], "properties"), "value"), get(get(nested[1], "properties"), "value")}
	for i, f := range s.Contributions {
		if f.Pointer != want[i] || f.Schema != originals[i] {
			t.Fatal(i, f)
		}
	}
	fields := slices.Clone(s.Contributions)
	winner := s.Fields[0]
	if !completePrimitiveContributions(s, winner, fields) {
		t.Fatal("Complete original contributions rejected")
	}
	for _, altered := range [][]field{fields[1:], append(slices.Clone(fields), fields[0]), {fields[0], fields[0], fields[2]}} {
		if completePrimitiveContributions(s, winner, altered) {
			t.Fatal("Missing/repeated source contribution authorized")
		}
	}
	copied, e := document.Load([]byte(`{"type":"string","minLength":1}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	changed := slices.Clone(fields)
	changed[0].Schema = copied
	if completePrimitiveContributions(s, winner, changed) {
		t.Fatal("Same-byte source node replacement authorized")
	}
	changed = slices.Clone(fields)
	changed[0].Depth++
	if completePrimitiveContributions(s, winner, changed) {
		t.Fatal("Changed retained source depth authorized")
	}
	plans, preserved, failure := a.ordinaryPrimitiveAggregates(v, s, true)
	if failure != nil || plans["value"].MinLength == nil || *plans["value"].MinLength != 2 || !preserved[fields[0].Pointer] || !preserved[fields[1].Pointer] {
		t.Fatal(plans, preserved, failure)
	}
	bytes, _ := root.MarshalJSON()
	if string(bytes) != string(input) {
		t.Fatal("Source tree rewritten")
	}
}
