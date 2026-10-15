package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func TestBooleanRootExcludedSources(t *testing.T) {
	for _, schemas := range []map[string]any{
		{"Unused": false, "Used": map[string]any{"properties": map[string]any{"v": map[string]any{"type": "string"}}}},
		// Nullable alias intent graduates through boolean_root_nullable_test.go.
		{"A": map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/ZAny"}, map[string]any{"properties": map[string]any{"v": map[string]any{"type": "string"}}}}}, "ZAny": true},
		{"A": map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/ZAny"}, map[string]any{"type": "string"}}}, "ZAny": true},
	} {
		for _, checked := range []bool{false, true} {
			input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": schemas}})
			if e != nil {
				t.Fatal(e)
			}
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("excluded admitted", schemas, checked, got, err)
			}
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) != 1 {
				t.Fatal(err)
			}
			if _, ok := schemas["Unused"]; ok {
				if ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Unused" {
					t.Fatal("unused false original pointer lost", ds)
				}
			}
		}
	}
	r := request([]byte(`{"openapi":"3.0.3","components":{"schemas":{"Any":true}}}`), "")
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("3.0 root admitted", got, err)
	}
}

func TestBooleanRootCatalogMetadataAndCollision(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"properties":{"v":{"type":"string"}},"examples":[{"schema":true,"other":false}]},"UnusedTrue":true}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || len(got.Warnings) != 1 || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/UnusedTrue" {
		t.Fatal("unused true/opaque metadata", got, err)
	}
	collision := []byte(`{"openapi":"3.1.0","components":{"schemas":{"foo_bar":true,"fooBar":{"type":"object"}}}}`)
	got, err = core.GenerateWithDiagnostics(request(collision, "Dto"))
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Code != "E_NAME_COLLISION" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("root bypassed filename reservations", got, err)
	}
}

func booleanRootChain(t *testing.T, edges int, consumer bool) core.GenerationRequest {
	t.Helper()
	schemas := map[string]any{"ZAny": true}
	for i := edges - 1; i >= 0; i-- {
		target := "ZAny"
		if i+1 < edges {
			target = fmt.Sprintf("Alias%03d", i+1)
		}
		schemas[fmt.Sprintf("Alias%03d", i)] = map[string]any{"$ref": "#/components/schemas/" + target}
	}
	if consumer {
		schemas["AHolder"] = map[string]any{"properties": map[string]any{"v": map[string]any{"$ref": "#/components/schemas/Alias000"}}, "required": []string{"v"}}
	}
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": schemas}})
	if err != nil {
		t.Fatal(err)
	}
	return request(input, "")
}
func TestBooleanRootAliasDepthAndSlots(t *testing.T) {
	for _, consumer := range []bool{false, true} {
		low := 48
		if consumer {
			low = 46
		}
		for _, edges := range []int{low, low + 1} {
			for _, checked := range []bool{false, true} {
				r := booleanRootChain(t, edges, consumer)
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if edges == low && err != nil || edges > low && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
					t.Fatal(consumer, edges, checked, got, err)
				}
			}
		}
	}
	for _, width := range []int{240, 241} {
		props := map[string]any{}
		req := []string{}
		for i := 0; i < width; i++ {
			name := fmt.Sprintf("v%03d", i)
			props[name] = map[string]any{"$ref": "#/components/schemas/ZAny"}
			req = append(req, name)
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"ZAny": true, "Holder": map[string]any{"properties": props, "required": req}}}})
		if err != nil {
			t.Fatal(err)
		}
		got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 240 && e != nil || width == 241 && (e == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(width, got, e)
		}
	}
}
