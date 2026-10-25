package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestLiteralFalseRootOwnDefaults(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"blocked":false,"label":{"type":"string"}},"default":{"label":"seed"}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(input, "Dto")
		rq.Options.ModelStyle = style
		report, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		var text string
		for _, file := range report.Artifacts.Files {
			text += file.Content
		}
		if !strings.Contains(text, "fun default()") || !strings.Contains(text, "sealed class FalseValue private constructor()") {
			t.Fatal("literalfalse ownerfactory missing", text)
		}
	}
}

func TestLiteralFalseRootDefaultsProfilesAndValues(t *testing.T) {
	input := schemas(`{"OptionalOnly":{"properties":{"blocked":false},"default":{}},"Fresh":{"properties":{"blocked":false,"labels":{"type":"array","items":{"type":"string"}}},"default":{"labels":["seed"]}},"RootNullable":{"type":"object","nullable":true,"required":["blocked"],"properties":{"blocked":false},"default":null},"Alias":{"$ref":"#/components/schemas/Fresh"}}`)
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
				for _, name := range []string{"OptionalOnlyDto", "FreshDto", "RootNullableDto"} {
					if !strings.Contains(source(t, report, name), "public fun default()") {
						t.Fatal("missing actual own factory", name)
					}
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(report, again) {
					t.Fatal("factory nondeterministic", e)
				}
			}
		}
	}
	for _, tc := range []struct{ required, value, extra, pointer string }{
		{`[]`, `{"blocked":null}`, ``, "/default/blocked"},
		{`[]`, `{"blocked":false}`, ``, "/default/blocked"},
		{`[]`, `{"blocked":{}}`, ``, "/default/blocked"},
		{`["blocked"]`, `{}`, ``, "/default"},
		{`["blocked"]`, `{"blocked":null}`, ``, "/default/blocked"},
		{`[]`, `null`, ``, "/default"},
		{`[]`, `{}`, `,"minProperties":1`, "/default"},
	} {
		input := schemas(fmt.Sprintf(`{"Holder":{"required":%s,"properties":{"blocked":false},"default":%s%s}}`, tc.required, tc.value, tc.extra))
		rq := request(input, "Dto")
		rq.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalid literalfalse default admitted", tc, err)
		}
		d := err.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Holder"+tc.pointer {
			t.Fatal("lost source default Pointer", tc, d)
		}
	}
	for _, role := range []string{
		`"Holder":{"properties":{"nested":{"properties":{"blocked":false}}},"default":{}}`,
		`"Holder":{"allOf":[{"properties":{"blocked":false}}],"default":{}}`,
		`"Target":{"properties":{"blocked":false}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}`,
		`"Target":{"properties":{"blocked":false}},"Alias":{"$ref":"#/components/schemas/Target","default":{}}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(`{`+role+`}`), "Dto"))
		if role == `"Target":{"properties":{"blocked":false}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{}}}}` {
			if err != nil || !strings.Contains(source(t, report, "HolderDto"), "openapi-kt model default") {
				t.Fatal("original87 fielddefault graduation", err)
			}
			continue
		}
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("unscoped literalfalse default admitted", role, err)
		}
	}
}

func TestLiteralFalseRootDefaultWorkAndBytes(t *testing.T) {
	for _, count := range []int{33, 34} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = false
		}
		owners := map[string]any{}
		for i := 0; i < count; i++ {
			owners[fmt.Sprintf("H%02d", i)] = map[string]any{"properties": props, "default": map[string]any{}}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": owners}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 33 {
			if err != nil {
				t.Fatal("bounded ownfactory capture", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("ownfactory cumulative budget reset", err)
		}
	}
	for _, count := range []int{6551, 6552} {
		values := make([]any, count)
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"blocked": false, "labels": map[string]any{"type": "array", "items": map[string]any{"type": "string", "nullable": true}}}, "default": map[string]any{"labels": values}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 6551 {
			if err != nil {
				t.Fatal("bounded32767byteownliteral", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("32KiB ownliteral cap lost", err)
		}
	}
}
