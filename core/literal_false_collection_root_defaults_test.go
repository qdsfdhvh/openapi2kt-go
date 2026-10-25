package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestLiteralFalseCollectionRootOwnDefaults(t *testing.T) {
	for _, input := range []string{
		`{"Owner":{"properties":{"values":{"type":"array","items":false}},"default":{"values":[]}}}`,
		`{"Owner":{"properties":{"values":{"type":"object","additionalProperties":{"type":"array","items":false}}},"default":{"values":{}}}}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err != nil {
			t.Errorf("already typed false collection rejected in own root default: %v", err)
			continue
		}
		if !strings.Contains(source(t, report, "OwnerDto"), "public fun default()") {
			t.Fatal("missing actual owner default factory")
		}
	}
}

func TestLiteralFalseCollectionRootDefaultProfilesAndValues(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/literal-false-collection-root-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), input...)
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
				for _, name := range []string{"HolderDto", "OnlyDto", "NoFieldDefaultDto", "NullableFieldDto", "RootNullableDto"} {
					if !strings.Contains(source(t, report, name), "public fun default()") {
						t.Fatal("actual owner factory missing", name)
					}
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(report, again) || !bytes.Equal(input, before) {
					t.Fatal("factory nondeterminism or input mutation", e)
				}
			}
		}
	}
	for _, tc := range []struct{ field, value, extra, pointer string }{
		{`{"type":"array","items":false}`, `{"rows":[null]}`, ``, "/default/rows/0"},
		{`{"type":"array","items":false}`, `{"rows":[false]}`, ``, "/default/rows/0"},
		{`{"type":"array","items":false}`, `{"rows":null}`, ``, "/default/rows"},
		{`{"type":"array","items":false}`, `null`, ``, "/default"},
		{`{"type":"array","items":false,"default":[]}`, `{}`, `,"required":["rows"]`, "/default"},
		{`{"type":"array","items":false,"default":[null]}`, `{"rows":[]}`, `,"required":["rows"]`, "/properties/rows/default/0"},
		{`{"type":"array","minItems":1,"items":false}`, `{"rows":[]}`, ``, "/default/rows"},
		{`{"additionalProperties":{"type":"array","nullable":true,"items":false}}`, `{"rows":{"description":[null]}}`, ``, "/default/rows/description/0"},
		{`{"type":"array","items":false}`, `{}`, `,"minProperties":1`, "/default"},
	} {
		rq := request(schemas(fmt.Sprintf(`{"Holder":{"properties":{"rows":%s},"default":%s%s}}`, tc.field, tc.value, tc.extra)), "Dto")
		rq.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalid owner default accepted", tc, err)
		}
		d := err.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Holder"+tc.pointer {
			t.Fatal("root/default source Pointer lost", tc, d)
		}
	}
}

func TestLiteralFalseCollectionRootDefaultBudgets(t *testing.T) {
	for _, depth := range []int{47, 48} {
		var leaf any = false
		for i := 0; i < depth; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": leaf}, "default": map[string]any{}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if depth == 47 {
			if err != nil {
				t.Fatal("bounded original source depth", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("source48 budget reset", err)
		}
	}
	for _, count := range []int{8, 20} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "array", "items": false}
		}
		owners := map[string]any{}
		for i := 0; i < count; i++ {
			owners[fmt.Sprintf("H%02d", i)] = map[string]any{"properties": props, "default": map[string]any{}}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": owners}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 8 {
			if err != nil {
				t.Fatal("bounded cumulative source+factory capture", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("shared falsework budget reset", err)
		}
	}
	for _, count := range []int{6551, 6552} {
		field := map[string]any{"type": "array", "items": map[string]any{"type": "array", "nullable": true, "items": false}}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Holder": map[string]any{"properties": map[string]any{"rows": field}, "default": map[string]any{"rows": make([]any, count)}}}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 6551 {
			if err != nil {
				t.Fatal("bounded32765byte root literal", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("32KiB root literal cap reset", err)
		}
	}
}
