package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestLiteralFalseCollectionDTOFieldOwnDefaults(t *testing.T) {
	for _, input := range []string{
		`{"Target":{"properties":{"rows":{"type":"array","items":false}}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"rows":[]}}}}}`,
		`{"Target":{"properties":{"rows":{"type":"array","items":false}},"default":{"rows":[]}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{}}}}}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err != nil {
			t.Errorf("actual typed collection DTO default rejected: %v", err)
			continue
		}
		if !strings.Contains(source(t, report, "ConsumerDto"), "openapi-kt model default") {
			t.Fatal("missing actual consuming field initializer")
		}
	}
}

func TestLiteralFalseCollectionDTOOwnDefaultsTargetCountsAndFactories(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		for _, own := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, checked := range []bool{false, true} {
					for mask := 0; mask < 8; mask++ {
						ref := func(name string, def any, annotated bool) map[string]any {
							v := map[string]any{"$ref": "#/components/schemas/" + name}
							if annotated {
								v["default"] = def
							}
							return v
						}
						props := map[string]any{"rows": map[string]any{"type": "array", "items": false}, "labels": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}
						target := map[string]any{"properties": props}
						if checked {
							target["minProperties"] = 0
							target["maxProperties"] = 1
						}
						if mixed {
							props["reference"] = ref("Blocked", nil, false)
						}
						if own {
							target["default"] = map[string]any{"labels": []any{"target"}}
						}
						literal := map[string]any{"labels": []any{"field"}}
						schemas := map[string]any{"Blocked": false, "Target": target, "Alias": ref("Target", nil, false), "Nullable": map[string]any{"$ref": "#/components/schemas/Target", "nullable": true}, "Consumer": map[string]any{"required": []string{"required"}, "properties": map[string]any{"optional": ref("Alias", literal, true), "required": ref("Target", literal, true), "nullable": ref("Nullable", nil, true), "plain": ref("Target", nil, false)}}}
						input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": schemas}})
						rq := request(input, "Dto")
						rq.Options.ModelStyle = style
						rq.Options.Validation = checked
						rq.Options.DeduplicateModels = mask&1 != 0
						rq.Options.HoistSharedShapes = mask&2 != 0
						rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
						rq.Options.MergeIdenticalModels = mask&4 != 0
						report, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal(mixed, own, style, checked, mask, err)
						}
						text := source(t, report, "ConsumerDto")
						if strings.Contains(text, "public val required: org.example.models.models.TargetDto =") || !strings.Contains(text, "public val required: org.example.models.models.TargetDto,") || !strings.Contains(text, "public val nullable: org.example.models.models.TargetDto? = null") {
							t.Fatal("required suppression/nullable policy changed", text)
						}
						if strings.Contains(source(t, report, "TargetDto"), "public fun default()") != own {
							t.Fatal("target factory inherited/removed")
						}
						again, e := core.GenerateWithDiagnostics(rq)
						if e != nil || !reflect.DeepEqual(report, again) {
							t.Fatal("nondeterministic field factory", e)
						}
					}
				}
			}
		}
	}
}

func TestLiteralFalseCollectionDTOFieldDefaultInvalidValuesAndRoles(t *testing.T) {
	for _, tc := range []struct{ required, value, pointer string }{{`[]`, `{"rows":null}`, "/default/rows"}, {`[]`, `{"rows":false}`, "/default/rows"}, {`["rows"]`, `{}`, "/default"}, {`[]`, `null`, "/default"}} {
		input := schemas(`{"Target":{"required":` + tc.required + `,"properties":{"rows":{"type":"array","items":false}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":` + tc.value + `}}}}`)
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("bad field own literal admitted", tc, err)
		}
		d := err.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Consumer/properties/value"+tc.pointer {
			t.Fatal("lost consuming literal Pointer", tc, d)
		}
	}
	for _, role := range []string{
		`"Target":{"properties":{"rows":{"type":"array","items":false}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}},"default":{}}`,
		`"Target":{"properties":{"rows":{"type":"array","items":false}}},"Consumer":{"properties":{"nested":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}}}`,
		`"Target":{"properties":{"rows":{"type":"array","items":false}}},"Consumer":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}]}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(`{`+role+`}`), "Dto"))
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("unscoped field literal admitted", role, err)
		}
	}
}

func TestLiteralFalseCollectionDTOFieldDefaultBudgets(t *testing.T) {
	for _, count := range []int{8, 20} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"$ref": "#/components/schemas/Target", "default": map[string]any{}}
		}
		owners := map[string]any{"Target": map[string]any{"properties": map[string]any{"rows": map[string]any{"type": "array", "items": false}}}}
		for i := 0; i < count; i++ {
			owners[fmt.Sprintf("C%02d", i)] = map[string]any{"properties": props}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": owners}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 8 {
			if err != nil {
				t.Fatal("bounded cumulative consumer capture", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("cumulative consumer source budget reset", err)
		}
	}
	for _, count := range []int{6551, 6552} {
		values := make([]any, count)
		target := map[string]any{"properties": map[string]any{"rows": map[string]any{"type": "array", "items": false}, "labels": map[string]any{"type": "array", "items": map[string]any{"type": "array", "nullable": true, "items": false}}}}
		field := map[string]any{"$ref": "#/components/schemas/Target", "default": map[string]any{"labels": values}}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Target": target, "Consumer": map[string]any{"properties": map[string]any{"value": field}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 6551 {
			if err != nil {
				t.Fatal("bounded32767bytefieldliteral", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("32KiB field literal cap lost", err)
		}
	}
	for _, depth := range []int{43, 49} {
		var leaf any = false
		for i := 0; i < depth; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Target": map[string]any{"properties": map[string]any{"blocked": false, "rows": leaf}}, "Consumer": map[string]any{"properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/Target", "default": map[string]any{}}}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if depth == 43 {
			if err != nil {
				t.Fatal("bounded original typedsource depth", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("source48 reset byfieldliteralrole", err)
		}
	}
}
