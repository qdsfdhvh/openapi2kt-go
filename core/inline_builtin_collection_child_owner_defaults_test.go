package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInlineBuiltinCollectionChildOwnerDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, field, outer, valid, invalid, suffix string
	}{
		{"list", `{"type":"array","items":{"type":"integer"@}}`, `{"value":[2]}`, `7`, `"bad"`, "/items/default"},
		{"map", `{"type":"object","additionalProperties":{"type":"string"@}}`, `{"value":{"key":"field"}}`, `"child"`, `true`, "/additionalProperties/default"},
		{"missing list", `{"type":"array","items":{"type":"integer"@}}`, `{}`, `7`, `"bad"`, "/items/default"},
		{"null owner", `{"type":"array","items":{"type":"integer"@}}`, `null`, `7`, `"bad"`, "/items/default"},
		{"nullable child", `{"type":"array","items":{"type":["integer","null"]@}}`, `{"value":[null,1]}`, `null`, `false`, "/items/default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Holder":{"type":["object","null"],"properties":{"value":` + tc.field + `},"default":` + tc.outer + `}}`
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, checked := range []bool{false, true} {
					rq := request(schemas(strings.Replace(body, "@", "", 1)), "Dto")
					rq.Options.ModelStyle = style
					rq.Options.Validation = checked
					control, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal("owner default without child annotation rejected", style, checked, err)
					}
					rq.Document = schemas(strings.Replace(body, "@", `,"default":`+tc.valid, 1))
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(got, control) {
						t.Fatal("child annotation changed owner default", style, checked, err)
					}
					rq.Document = schemas(strings.Replace(body, "@", `,"default":`+tc.invalid, 1))
					report, err := core.GenerateWithDiagnostics(rq)
					ds, ok := err.(core.Diagnostics)
					want := "/components/schemas/Holder/properties/value" + tc.suffix
					if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
						*ds[0].Pointer != want || !reflect.DeepEqual(report, core.GenerationReport{}) {
						t.Fatal("invalid child escaped independent validation", style, checked, err)
					}
				}
			}
		})
	}
}

func TestInlineBuiltinCollectionChildAndOwnerDefaultsFailIndependently(t *testing.T) {
	for _, checked := range []bool{false, true} {
		body := `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"integer","default":7}}},"default":{"value":[true]}}}`
		carrierDefaultFailure(t, schemas(body), checked, "E_SCHEMA_INVALID",
			"/components/schemas/Holder/default/value/0")
	}
}

func TestInlineBuiltinCollectionChildOwnerDefaultChecks(t *testing.T) {
	body := `{"Holder":{"type":["object","null"],"properties":{"value":{"type":"array","items":{"type":"integer","minimum":2,"default":@}}},"default":null}}`
	rq := request(schemas(strings.Replace(body, "@", "2", 1)), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("valid checked child rejected", err)
	}
	carrierDefaultFailure(t, schemas(strings.Replace(body, "@", "1", 1)), true, "E_SCHEMA_INVALID",
		"/components/schemas/Holder/properties/value/items/default")
}

func TestInlineBuiltinCollectionChildOwnerDefaultBudget(t *testing.T) {
	for _, size := range []int{32750, 32768} {
		body := `{"Holder":{"type":["object","null"],"properties":{"value":{"type":"array","items":{"type":"string","default":"` + strings.Repeat("a", size) + `"}}},"default":null}}`
		rq := request(schemas(body), "Dto")
		report, err := core.GenerateWithDiagnostics(rq)
		if size == 32750 && err != nil {
			t.Fatal("bounded child default rejected", err)
		}
		if size == 32768 {
			ds, ok := err.(core.Diagnostics)
			want := "/components/schemas/Holder/properties/value/items/default"
			if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil ||
				*ds[0].Pointer != want || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("null owner bypassed child default budget", err)
			}
		}
	}
}
