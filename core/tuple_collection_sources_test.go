package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleCollectionOverlappingSourceBudget(t *testing.T) {
	fields := map[string]any{}
	for i := 0; i < 60; i++ {
		fields[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "string"}
	}
	for _, tuple := range []bool{false, true} {
		catalog := map[string]any{"Base": map[string]any{"properties": fields}}
		for i := 0; i < 100; i++ {
			item := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Base"}}
			array := map[string]any{"type": "array", "items": item}
			if tuple {
				array["prefixItems"], array["items"] = []any{item}, true
			}
			catalog[fmt.Sprintf("Rows%03d", i)] = array
		}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if e != nil {
			t.Fatal(e)
		}
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if !tuple && err != nil {
			t.Fatal("tuple accounting changed ordinary collection graphs", err)
		}
		if tuple && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("overlapping collection source work reset its cumulative budget", err)
		}
	}
}

func TestTupleCollectionOverlappingSourceByteBudget(t *testing.T) {
	for _, tuple := range []bool{false, true} {
		catalog := map[string]any{"Base": map[string]any{"description": strings.Repeat("x", 90<<10), "properties": map[string]any{"name": map[string]any{"type": "string"}}}}
		for i := 0; i < 100; i++ {
			item := map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Base"}}
			array := map[string]any{"type": "array", "items": item}
			if tuple {
				array["prefixItems"], array["items"] = []any{item}, true
			}
			catalog[fmt.Sprintf("Rows%03d", i)] = array
		}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if e != nil {
			t.Fatal(e)
		}
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if !tuple && err != nil {
			t.Fatal("tuple byte accounting changed ordinary collection graphs", err)
		}
		if tuple && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("overlapping collection snapshots reset cumulative byte budget", err)
		}
	}
}

func TestTupleCollectionDeferredDefaultWithScalarTail(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, value := range []string{`[7]`, `["bad"]`} {
			input := schemas(`{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Numbers"}],"items":{"type":"integer"},"default":` + value + `}}},"Numbers":{"type":"array","items":{"type":"integer"}}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if value == `[7]` {
				if err != nil || !strings.Contains(source(t, got, "HolderDto"), "List<kotlin.Int>") {
					t.Fatal("discarded forward collection changed selected scalar default", checked, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/rows/default/0" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("deferred validation used discarded collection instead of actual tail", checked, err)
			}
		}
	}
}

func TestTupleCollectionNominalRecursionAndReferenceData(t *testing.T) {
	for _, tail := range []string{"", `,"items":true`} {
		input := schemas(`{"Node":{"properties":{"next":{"$ref":"#/components/schemas/Node"},"payload":{"type":"object","additionalProperties":true,"default":{"$ref":"#/not-a-schema-reference"}}}},"Rows":{"type":"array","prefixItems":[{"type":"array","items":{"$ref":"#/components/schemas/Node"}}]` + tail + `},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"additionalProperties":{"type":"string"}}],"default":[{"$ref":"#/not-a-schema-reference"}]}}}}`)
		rq := request(input, "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("schema graph capture interpreted literal $ref data or rejected nominal recursion", tail, err)
		}
		for i := 0; i < 8; i++ {
			again, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("recursive collection proof is nondeterministic", tail, e)
			}
		}
	}
}

func TestTupleCollectionCheckedDepth(t *testing.T) {
	for _, count := range []int{16, 17} {
		item := `{"type":"array","minItems":1,"items":{"type":"integer"}}`
		for i := 2; i < count; i++ {
			item = `{"type":"array","minItems":1,"items":` + item + `}`
		}
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[`+item+`]}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if count == 16 && err != nil || count == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("collection tuple erased checked container depth", count, err)
		}
	}
}

func TestTupleCollectionDefaultAliasHopsCountOnce(t *testing.T) {
	parts := []string{`"Values":{"type":"array","items":{"type":"integer"}}`}
	next := "Values"
	for i := 0; i < 25; i++ {
		name := fmt.Sprintf("Alias%02d", i)
		parts = append(parts, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/%s"}`, name, next))
		next = name
	}
	parts = append(parts, `"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/`+next+`"}],"default":[[1]]}`)
	parts = append(parts, `"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/`+next+`"}],"default":[[2]]}}}`)
	if _, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")); err != nil {
		t.Fatal("collection default counted the already-resolved alias path twice", err)
	}
}

func TestTupleCollectionSources(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, want string }{
		{`[{"type":"array","items":{"type":"integer"}}]`, "", "List<kotlin.collections.List<kotlin.Int>>"},
		{`[{"additionalProperties":{"type":"string"}}]`, "", "List<kotlin.collections.Map<kotlin.String, kotlin.String>>"},
		{`[{"type":"array","items":{"type":"integer"}},{"type":"array","items":{"type":"integer"}}]`, "", "List<kotlinx.serialization.json.JsonElement>"},
		{`[{"type":"array","items":{"type":"integer"}}]`, `,"items":true`, "List<kotlinx.serialization.json.JsonElement>"},
		{`[{"$ref":"#/components/schemas/Numbers"},{"$ref":"#/components/schemas/Alias"}]`, "", "List<org.example.models.models.NumbersDto>"},
		{`[{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Numbers"}`, "List<org.example.models.models.NumbersDto>"},
		{`[{"type":"integer"}]`, `,"items":{"type":"array","items":{"additionalProperties":{"type":"integer"}}}`, "List<kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.Int>>>"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Numbers":{"type":"array","items":{"type":"integer"}},"Alias":{"$ref":"#/components/schemas/Numbers"},"Rows":{"type":"array","prefixItems":`+tc.prefix+tc.tail+`}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(tc, checked, err)
			}
			if !strings.Contains(source(t, got, "RowsDto"), tc.want) {
				t.Fatal(tc, source(t, got, "RowsDto"))
			}
			again, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("tuple collection source nondeterminism", e)
			}
		}
	}
}

func TestTupleCollectionFormerInferredMapRefusal(t *testing.T) {
	// Preserve the exact former DTO no-drop counterexample. It is now an
	// eligible typed dictionary source, including when a scalar tail wins.
	for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"Item":{"additionalProperties":{"type":"string"}},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}]`+tail+`}}`), "Dto")
				rq.Options.ModelStyle, rq.Options.Validation = style, checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(tail, style, checked, err)
				}
				want := "List<org.example.models.models.ItemDto>"
				if tail != "" {
					want = "List<kotlin.Int>"
				}
				if !strings.Contains(source(t, got, "ItemDto"), "Map<kotlin.String, kotlin.String>") || !strings.Contains(source(t, got, "ProbeDto"), want) || len(got.Warnings) != 1 || got.Warnings[0].Code != "W_TUPLE_APPROXIMATION" {
					t.Fatal("lost original dictionary codec or tail approximation", tail, got)
				}
			}
		}
	}
}

func TestTupleCollectionSourceDepthAndWidth(t *testing.T) {
	for _, count := range []int{48, 49} {
		tuple := `{"type":"array","prefixItems":[{"type":"array","items":{"type":"integer"}}]}`
		for i := 2; i < count; i++ {
			tuple = `{"type":"array","items":` + tuple + `}`
		}
		_, err := core.Generate(request(schemas(`{"Rows":`+tuple+`}`), "Dto"))
		if count == 48 && err != nil || count == 49 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("selected collection source depth was double counted", count, err)
		}
	}
	for _, count := range []int{48, 49} {
		tuple := `{"type":"array","prefixItems":[{"type":"array","items":{"type":"integer"}}],"items":true}`
		for i := 2; i < count; i++ {
			tuple = `{"type":"array","items":` + tuple + `}`
		}
		_, err := core.Generate(request(schemas(`{"Rows":`+tuple+`}`), "Dto"))
		if count == 48 && err != nil || count == 49 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("collection original source depth", count, err)
		}
	}
	for _, count := range []int{46, 47} {
		tuple := `{"type":"array","prefixItems":[{"type":"array","items":{"type":"integer"}}],"items":true}`
		for i := 2; i < count; i++ {
			tuple = `{"type":"array","items":` + tuple + `}`
		}
		for _, forward := range []bool{false, true} {
			parts := []string{`"Rows":` + tuple, `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`}
			if forward {
				parts[0], parts[1] = parts[1], parts[0]
			}
			_, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
			if count == 46 && err != nil || count == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
				t.Fatal("collection cached source depth", count, forward, err)
			}
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%03d":{"type":"array","prefixItems":[{"type":"array","items":{"type":"integer"}}],"default":[[1]]}`, i)
		}
		_, err := core.Generate(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("collection tuple constructor cap", count, err)
		}
	}
}

func TestTupleCollectionSourceDefaultsAndDeferredAnnotations(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"array","items":{"type":"integer"}}`,
		`{"additionalProperties":{"type":"integer"}}`,
		`{"type":"array","items":{"properties":{"name":{"type":"string"}},"required":["name"]}}`,
		`{"type":"array","items":{"enum":["ready","paused"]}}`,
	} {
		for _, tail := range []string{"", `,"items":true`} {
			input := schemas(`{"Rows":{"type":"array","prefixItems":[` + leaf + `]` + tail + `,"default":[]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[` + leaf + `]` + tail + `,"default":[]}}}}`)
			if _, err := core.Generate(request(input, "Dto")); err != nil {
				t.Fatal(leaf, tail, err)
			}
		}
	}
	for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"string"}`} {
		input := schemas(`{"Choice":{"enum":["ready","paused"]},"Rows":{"type":"array","prefixItems":[{"type":"array","items":{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"future"}}}}]` + tail + `,"default":[]}}`)
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Rows/prefixItems/0/items/properties/state/default")
	}
}

func TestTupleCollectionSourceFullValues(t *testing.T) {
	for _, tc := range []struct {
		member, value string
		valid         bool
	}{
		{`{"type":"array","items":{"type":"integer"}}`, `[[1,2],[]]`, true},
		{`{"type":"array","items":{"type":"integer"}}`, `[["bad"]]`, false},
		{`{"additionalProperties":{"type":"integer"}}`, `[{"default":7}]`, true},
		{`{"additionalProperties":{"type":"integer"}}`, `[{"k":"bad"}]`, false},
		{`{"type":"array","items":{"enum":["ready","paused"]}}`, `[["ready"]]`, true},
		{`{"type":"array","items":{"enum":["ready","paused"]}}`, `[["future"]]`, false},
		{`{"type":"array","items":{"properties":{"name":{"type":"string"},"n":{"type":"integer","default":7}},"required":["name"]}}`, `[[{"name":"a"}]]`, true},
		{`{"type":"array","items":{"properties":{"name":{"type":"string"}},"required":["name"]}}`, `[[{}]]`, false},
		{`{"$ref":"#/components/schemas/Numbers"}`, `[[1,2]]`, true},
		{`{"$ref":"#/components/schemas/Numbers"}`, `[["bad"]]`, false},
	} {
		tuple := `{"type":"array","prefixItems":[` + tc.member + `],"default":` + tc.value + `}`
		for _, root := range []bool{false, true} {
			owner := `"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`
			if !root {
				owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
			}
			_, err := core.Generate(request(schemas(`{"Numbers":{"type":"array","items":{"type":"integer"}},`+owner+`}`), "Dto"))
			if (err == nil) != tc.valid {
				if err != nil {
					t.Fatalf("%v root=%v: %v at %s", tc, root, err, *err.(core.Diagnostics)[0].Pointer)
				}
				t.Fatal(tc, root, err)
			}
			if err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" {
				t.Fatal(tc, root, err)
			}
		}
	}
}
