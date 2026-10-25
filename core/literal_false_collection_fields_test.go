package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestLiteralFalseCollectionFieldMinimal(t *testing.T) {
	report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"rows":{"type":"array","items":false}}}}`), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, report, "HolderDto")
	if !strings.Contains(text, "List<org.example.models.models.HolderDto.FalseValue>") || strings.Contains(text, "AbsentSerializer") {
		t.Fatal("missing actual non-null zero-domain leaf/ordinary container presence", text)
	}
}

func TestLiteralFalseCollectionFieldsProfilesAndNaming(t *testing.T) {
	input := schemas(`{"Holder":{"required":["required","nullable"],"properties":{"required":{"type":"array","items":false},"nullable":{"type":"array","nullable":true,"items":false},"mapping":{"additionalProperties":{"type":"array","nullable":true,"items":false}},"nested":{"type":"array","items":{"additionalProperties":{"type":"array","items":false}}},"blocked":false,"falseValue":{"type":"string"}}},"OnlyContainers":{"properties":{"rows":{"type":"array","items":false}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for mask := 0; mask < 8; mask++ {
				rq := request(input, "Dto")
				rq.Options.ModelStyle = style
				rq.Options.Validation = checked
				rq.Options.DeduplicateModels = mask&1 != 0
				rq.Options.HoistSharedShapes = mask&2 != 0
				rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
				rq.Options.MergeIdenticalModels = mask&4 != 0
				report, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(style, checked, mask, err)
				}
				holder := source(t, report, "HolderDto")
				only := source(t, report, "OnlyContainersDto")
				if strings.Count(holder, "sealed class FalseValue2") != 1 || !strings.Contains(holder, "List<org.example.models.models.HolderDto.FalseValue2>") || !strings.Contains(holder, "AbsentSerializer") || strings.Contains(only, "AbsentSerializer") {
					t.Fatal("actual shared helper/presence changed", holder, only)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(report, again) {
					t.Fatal("collection generation nondeterministic", e)
				}
			}
		}
	}
}
func TestLiteralFalseCollectionFieldClosedRolesAndKeyword(t *testing.T) {
	for _, role := range []string{
		`"Holder":{"properties":{"rows":{"type":"array","items":false,"default":[]}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":false}},"default":{}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":{"type":"array","items":false,"default":[]}}}}`,
		`"Holder":{"properties":{"nested":{"properties":{"rows":{"type":"array","items":false}}}}}`,
		`"Holder":{"allOf":[{"properties":{"rows":{"type":"array","items":false}}}]}`,
		`"Holder":{"type":"array","items":false}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":false,"uniqueItems":true}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","prefixItems":[false],"items":false}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":false,"readOnly":true}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":false,"unknown":true}}}`,
		`"Target":{"properties":{"rows":{"type":"array","items":false}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(`{`+role+`}`), "Dto"))
		if role == `"Holder":{"properties":{"rows":{"type":"array","items":false,"default":[]}}}` {
			if err != nil || !strings.Contains(source(t, report, "HolderDto"), "openapi-kt default") {
				t.Fatal("original89 ownouterdefault graduation", err)
			}
			continue
		}
		if role == `"Holder":{"properties":{"rows":{"type":"array","items":false}},"default":{}}` {
			if err != nil || !strings.Contains(source(t, report, "HolderDto"), "public fun default()") {
				t.Fatal("original89 ownerrootdefault graduation", err)
			}
			continue
		}
		if role == `"Target":{"properties":{"rows":{"type":"array","items":false}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}` {
			if err != nil || !strings.Contains(source(t, report, "ConsumerDto"), "openapi-kt model default") {
				t.Fatal("original consumingfield default graduation", err)
			}
			continue
		}
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("collection permission leaked", role, err)
		}
	}
	rq := request(schemas(`{"Holder":{"properties":{"rows":{"type":"array","items":false}}}}`), "Dto")
	rq.Document = []byte(strings.Replace(string(rq.Document), "3.1.0", "3.0.3", 1))
	if report, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("3.0 Boolean grammar widened", err)
	}
	report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"closed":{"additionalProperties":false}}}}`), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, report, "HolderDto")
	if !strings.Contains(text, "JsonObject") || strings.Contains(text, "FalseValue") || len(report.Warnings) != 1 {
		t.Fatal("Boolean additionalProperties keyword reinterpreted", text, report.Warnings)
	}
}
func TestLiteralFalseCollectionFieldSourceAndWidthBudgets(t *testing.T) {
	for _, depth := range []int{47, 48} {
		var leaf any = false
		for i := 0; i < depth; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": leaf}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if depth == 47 {
			if err != nil {
				t.Fatal("bounded source depth", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("source48 budget reset", err)
		}
	}
	for _, count := range []int{240, 241} {
		props := map[string]any{}
		for i := 0; i < count; i++ {
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "array", "items": false}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": props}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 240 {
			if err != nil {
				t.Fatal("original240 slots", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("JVM240 cap lost", err)
		}
	}
	for _, count := range []int{8, 20} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "array", "items": false}
		}
		owners := map[string]any{}
		for i := 0; i < count; i++ {
			owners[fmt.Sprintf("H%02d", i)] = map[string]any{"properties": props}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": owners}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 8 {
			if err != nil {
				t.Fatal("bounded sharedsource capture", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("cumulativefalsework reset", err)
		}
	}
	for _, depth := range []int{16, 17} {
		var leaf any = false
		for i := 0; i < depth; i++ {
			leaf = map[string]any{"type": "array", "maxItems": 1, "items": leaf}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": leaf}}}}})
		rq := request(input, "Dto")
		rq.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(rq)
		if depth == 16 {
			if err != nil {
				t.Fatal("bounded checkeddepth", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("checked16 cap lost", err)
		}
	}
}
