package core_test

import (
	"bytes"
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"sort"
	"testing"
)

func TestBooleanRootCyclesAndInternalTargets(t *testing.T) {
	for _, input := range []string{
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"},"UnusedTrue":true}}}`,
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"properties":{"v":true}},"B":{"properties":{"v":{"$ref":"#/components/schemas/A/properties/v"}}},"UnusedTrue":true}}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request([]byte(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("cycle/internal source admitted", checked, got, err)
			}
		}
	}
}
func TestBooleanRootCatalogSpellingOrder(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(booleanRootInput(t), "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			before, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			var doc, components, catalog map[string]json.RawMessage
			if err := json.Unmarshal(r.Document, &doc); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(doc["components"], &components); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(components["schemas"], &catalog); err != nil {
				t.Fatal(err)
			}
			names := []string{}
			for name := range catalog {
				names = append(names, name)
			}
			sort.Sort(sort.Reverse(sort.StringSlice(names)))
			var spelling bytes.Buffer
			spelling.WriteByte('{')
			for i, name := range names {
				if i > 0 {
					spelling.WriteByte(',')
				}
				key, _ := json.Marshal(name)
				spelling.Write(key)
				spelling.WriteByte(':')
				spelling.Write(catalog[name])
			}
			spelling.WriteByte('}')
			components["schemas"] = json.RawMessage(spelling.Bytes())
			doc["components"], err = json.Marshal(components)
			if err != nil {
				t.Fatal(err)
			}
			r.Document, err = json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(before, got) {
				t.Fatal("catalog reorder changed full report", style, checked, err)
			}
		}
	}
}
