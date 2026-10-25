package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestLiteralFalseCollectionOwnDefaultMinimal(t *testing.T) {
	report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"rows":{"type":"array","items":false,"default":[]}}}}`), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, report, "HolderDto")
	if !strings.Contains(text, "openapi-kt default") || !strings.Contains(text, "List<org.example.models.models.HolderDto.FalseValue>") {
		t.Fatal("missing actual typed literal default", text)
	}
}

func TestLiteralFalseCollectionOwnDefaultsProfilesAndValues(t *testing.T) {
	input := schemas(`{"Holder":{"required":["required","requiredNull"],"properties":{"optional":{"type":"array","items":false,"default":[]},"required":{"type":"array","items":false,"default":[]},"nullable":{"type":"array","nullable":true,"items":false,"default":null},"requiredNull":{"type":"array","nullable":true,"items":false,"default":null},"mapping":{"additionalProperties":{"type":"array","nullable":true,"items":false},"default":{"description":null,"empty":[]}},"nested":{"type":"array","items":{"type":"array","items":false},"default":[[]]},"blocked":false}}}`)
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
				text := source(t, report, "HolderDto")
				if !strings.Contains(text, "public val required: kotlin.collections.List<org.example.models.models.HolderDto.FalseValue>,") || !strings.Contains(text, "public val requiredNull: kotlin.collections.List<org.example.models.models.HolderDto.FalseValue>?,") || strings.Contains(text, "public val requiredNull: kotlin.collections.List<org.example.models.models.HolderDto.FalseValue>? =") {
					t.Fatal("required annotation initializer not suppressed", text)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(report, again) {
					t.Fatal("ownliteral nondeterminism", e)
				}
			}
		}
	}
	for _, tc := range []struct{ schema, value, pointer string }{
		{`{"type":"array","items":false}`, `[null]`, "/default/0"},
		{`{"type":"array","items":false}`, `[false]`, "/default/0"},
		{`{"type":"array","items":false}`, `null`, "/default"},
		{`{"type":"array","maxItems":0,"items":false}`, `[null]`, "/default"},
		{`{"type":"array","minItems":1,"items":false}`, `[]`, "/default"},
	} {
		var schema map[string]any
		if e := json.Unmarshal([]byte(tc.schema), &schema); e != nil {
			t.Fatal(e)
		}
		var value any
		if e := json.Unmarshal([]byte(tc.value), &value); e != nil {
			t.Fatal(e)
		}
		schema["default"] = value
		data, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": schema}}}}})
		rq := request(data, "Dto")
		rq.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalidliteral admitted", tc, err)
		}
		d := err.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Holder/properties/rows"+tc.pointer {
			t.Fatal("literal originalPointer lost", tc, d)
		}
	}
	for _, role := range []string{
		`"Holder":{"properties":{"rows":{"type":"array","items":false,"default":[]}},"default":{}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":{"type":"array","items":false,"default":[]},"default":[]}}}`,
		`"Holder":{"properties":{"nested":{"properties":{"rows":{"type":"array","items":false,"default":[]}}}}}`,
		`"Holder":{"allOf":[{"properties":{"rows":{"type":"array","items":false,"default":[]}}}]}`,
		`"Target":{"properties":{"rows":{"type":"array","items":false,"default":[]}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(`{`+role+`}`), "Dto"))
		if role == `"Holder":{"properties":{"rows":{"type":"array","items":false,"default":[]}},"default":{}}` {
			if err != nil || !strings.Contains(source(t, report, "HolderDto"), "public fun default()") || !strings.Contains(source(t, report, "HolderDto"), "openapi-kt default") {
				t.Fatal("original90 independent field and owner defaults graduation", err)
			}
			continue
		}
		if role == `"Target":{"properties":{"rows":{"type":"array","items":false,"default":[]}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}` {
			if err != nil || !strings.Contains(source(t, report, "ConsumerDto"), "openapi-kt model default") {
				t.Fatal("original consumingfield default graduation", err)
			}
			continue
		}
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("unscopedliteral role admitted", role, err)
		}
	}
}

func TestLiteralFalseCollectionOwnDefaultBudgets(t *testing.T) {
	for _, depth := range []int{47, 48} {
		var leaf any = false
		for i := 0; i < depth; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		leaf.(map[string]any)["default"] = []any{}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": leaf}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if depth == 47 {
			if err != nil {
				t.Fatal("bounded original ownschema depth", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("source48 cap reset", err)
		}
	}
	for _, count := range []int{8, 20} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "array", "items": false, "default": []any{}}
		}
		owners := map[string]any{}
		for i := 0; i < count; i++ {
			owners[fmt.Sprintf("H%02d", i)] = map[string]any{"properties": props}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": owners}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 8 {
			if err != nil {
				t.Fatal("bounded source+literal capture", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("shared falsework cap reset", err)
		}
	}
	for _, count := range []int{6553, 6554} {
		values := make([]any, count)
		field := map[string]any{"type": "array", "items": map[string]any{"type": "array", "nullable": true, "items": false}, "default": values}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": field}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 6553 {
			if err != nil {
				t.Fatal("bounded32766byte literal", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("32KiB literal cap reset", err)
		}
	}
}
