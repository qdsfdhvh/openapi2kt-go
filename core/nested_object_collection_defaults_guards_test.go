package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedObjectCollectionDefaultsGuards(t *testing.T) {
	object := `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`
	for name, tc := range map[string]struct {
		field   string
		code    string
		pointer string
	}{
		"list missing required": {`{"type":"array","items":` + object + `,"default":[{}]}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/0"},
		"list unknown key":      {`{"type":"array","items":` + object + `,"default":[{"name":"a","zz":1}]}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/0/zz"},
		"element own default":   {`{"type":"array","items":{"type":"object","properties":{"n":{"type":"integer"}},"default":{"n":1}},"default":[{"n":1}]}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/value/items/default"},
		"list wrong kind":       {`{"type":"array","items":` + object + `,"default":{"name":"a"}}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default"},
		"map unknown key":       {`{"type":"object","additionalProperties":{"type":"object","properties":{"v":{"type":"integer"}}},"default":{"k":{"zz":1}}}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/k/zz"},
		"map wrong kind":        {`{"type":"object","additionalProperties":` + object + `,"default":[]}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default"},
		"narrow null member":    {`{"type":"array","nullable":true,"items":{"type":"object","properties":{"a":{"type":"string"}}},"default":[{"a":null}]}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/0/a"},
	} {
		t.Run(name, func(t *testing.T) {
			input := schemas(`{"Holder":{"properties":{"value":` + tc.field + `}}}`)
			r := request(input, "Dto")
			r.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatalf("want %s at %s, got %v", tc.code, tc.pointer, err)
			}
		})
	}
	// Object-free collection trees keep their existing profiles.
	for _, input := range []string{
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"type":"string"}},"default":[]}}}}`,
		`{"Holder":{"properties":{"value":{"type":"object","additionalProperties":{"type":"integer"},"default":{"a":1}}}}}`,
	} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("scalar collection regressed", input, err)
		}
	}
}

func TestNestedObjectCollectionDefaultsBudgets(t *testing.T) {
	big := strings.Repeat("a", 33000)
	large := schemas(`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"object","properties":{"s":{"type":"string"}}},"default":[{"s":"` + big + `"}]}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(large, "Dto")); err == nil {
		t.Fatal("oversized object collection default accepted")
	}
	leaves := make([]string, 0, 17000)
	for i := 0; i < 17000; i++ {
		leaves = append(leaves, `{"n":1}`)
	}
	deep := schemas(`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"object","properties":{"n":{"type":"integer"}}},"default":[` + strings.Join(leaves, ",") + `]}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(deep, "Dto")); err == nil {
		t.Fatal("over-work object collection default accepted")
	}
}
