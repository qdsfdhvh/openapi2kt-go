package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestMergeNestedEnumsAndSemanticDistinctions(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/merge-probes.json")
	if err != nil {
		t.Fatal(err)
	}
	r := request(input, "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"LooseZDto": "LooseADto", "StatusZDto": "StatusADto"} {
		if !strings.Contains(source(t, got, "models/"+name), "typealias "+name+" = org.example.models.models."+target) {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"StatusDifferentDto", "CountADto", "CountZDto", "ConstADto", "ConstZDto"} {
		if strings.Contains(source(t, got, "models/"+name), "typealias "+name) {
			t.Fatal("lost distinct codec/order/default", name)
		}
	}
}
func TestMergeResolvedMethodErasure(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/merge-probes.json")
	if err != nil {
		t.Fatal(err)
	}
	r := request(input, "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	code := source(t, got, "api/DefaultApi")
	if strings.Count(code, "public suspend fun submit(") != 1 || !strings.Contains(code, "public suspend fun submit3(") || !strings.Contains(code, "public suspend fun submit2(") {
		t.Fatal("post-merge nominal erasure must preserve natural submit2 and allocate submit3", code)
	}
}

func TestMergeSchemaErrorsStillHaveNoPartialReport(t *testing.T) {
	properties := map[string]any{}
	required := []string{}
	for i := 0; i < 241; i++ {
		name := fmt.Sprintf("f%d", i)
		properties[name] = map[string]any{"type": "string"}
		required = append(required, name)
	}
	inputs := []map[string]any{
		{"A": map[string]any{"properties": properties, "required": required}},
		{"A": map[string]any{"properties": map[string]any{"x": map[string]any{"type": "string", "unsupportedWireKeyword": true}}}},
	}
	for _, catalog := range inputs {
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		r := request(input, "")
		before, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(before, core.GenerationReport{}) {
			t.Fatal("missing precondition", err)
		}
		r.Options.MergeIdenticalModels = true
		got, again := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(before, got) || !reflect.DeepEqual(err, again) {
			t.Fatal("optimizer changed failure", err, again)
		}
	}
}
func TestMergeLongNominalGraphsSkipWithoutNewRejection(t *testing.T) {
	catalog := map[string]any{}
	for _, prefix := range []string{"A", "Z"} {
		for i := 0; i < 50; i++ {
			field := map[string]any{"type": "string"}
			if i < 49 {
				field = map[string]any{"$ref": fmt.Sprintf("#/components/schemas/%s%d", prefix, i+1)}
			}
			catalog[fmt.Sprintf("%s%d", prefix, i)] = map[string]any{"properties": map[string]any{"value": field}, "required": []string{"value"}}
		}
	}
	input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
	r := request(input, "")
	r.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"A0", "Z0", "A1", "Z1"} {
		if !strings.Contains(source(t, got, "models/"+name), "data class "+name) {
			t.Fatal("over-height optimized", name)
		}
	}
	if !strings.Contains(source(t, got, "models/Z49"), "typealias Z49 =") {
		t.Fatal("acyclic leaves not merged")
	}
}
func FuzzGenerateMerge(f *testing.F) {
	f.Add([]byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"properties":{"next":{"$ref":"#/components/schemas/A"}}},"Z":{"properties":{"value":{"type":"string"}}}}}}`))
	f.Add([]byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"string","enum":["a","b"]},"Z":{"type":"string","enum":["a","b"]}}}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 100000 {
			t.Skip()
		}
		r := request(input, "")
		r.Options.MergeIdenticalModels = true
		r.Options.Validation = true
		a, e := core.GenerateWithDiagnostics(r)
		b, again := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(e, again) {
			t.Fatal("nondeterministic")
		}
		if e != nil && !reflect.DeepEqual(a, core.GenerationReport{}) {
			t.Fatal("partial failure")
		}
	})
}
