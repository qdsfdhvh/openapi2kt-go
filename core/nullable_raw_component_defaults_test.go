package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableRawComponentNullFactories(t *testing.T) {
	for _, shape := range []string{
		`"type":"object","nullable":true`,
		`"type":["object","null"],"properties":{}`,
		`"type":"object","nullable":true,"additionalProperties":true`,
		`"type":["object","null"],"additionalProperties":true`,
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				r := request(schemas(`{"R":{`+shape+`,"default":null},"Holder":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/R"},"required":{"$ref":"#/components/schemas/R"}}}}`), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(shape, checked, style, err)
				}
				carrier := "kotlinx.serialization.json.JsonElement?"
				if strings.Contains(shape, "additionalProperties") {
					carrier = "kotlinx.serialization.json.JsonObject?"
				}
				text := source(t, got, "RDto")
				if !strings.Contains(text, "public typealias RDto = "+carrier) || !strings.Contains(text, "public fun `RDto default`(): RDto =") || !strings.Contains(text, `("null")`) || !strings.Contains(text, "private fun `openapi-kt component default ") {
					t.Fatal("null factory must retain its nullable alias and helper", text)
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic raw null factory", err)
				}
				r.Document = []byte(strings.Replace(string(r.Document), `,"default":null`, "", 1))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal("no-default control", err)
				}
				if source(t, got, "HolderDto") != source(t, control, "HolderDto") || !reflect.DeepEqual(got.Warnings, control.Warnings) {
					t.Fatal("component default inherited or warnings changed")
				}
				if strings.Contains(source(t, control, "RDto"), "fun `RDto default`") {
					t.Fatal("unannotated carrier gained a factory")
				}
			}
		}
	}
}

func TestNullableRawComponentNullFactoryGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, body := range []string{
			`{"R":{"type":"object","nullable":true,"minProperties":0,"default":null}}`,
			`{"R":{"type":"object","nullable":true,"additionalProperties":true,"maxProperties":1,"default":null}}`,
			`{"R":{"type":"object","nullable":true,"properties":[],"default":null}}`,
			`{"R":{"type":"object","nullable":true,"required":true,"default":null}}`,
			`{"R":{"type":"object","nullable":true,"properties":{"pin":{"type":"string","const":"x"}},"default":null}}`,
			`{"R":{"type":"object","nullable":true,"additionalProperties":true,"properties":{"a":{"type":"string"}},"default":null}}`,
			`{"R":{"type":"object","nullable":true,"oneOf":[{"type":"string"},{"type":"integer"}],"default":null}}`,
			`{"R":{"type":"object","nullable":true,"format":"raw","default":null}}`,
			`{"T":{"type":"object","nullable":true},"R":{"$ref":"#/components/schemas/T","nullable":true,"default":null}}`,
		} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Pointer == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("raw null default bypassed the original source role", checked, body, got, err)
			}
		}
		for _, shape := range []string{
			`"type":"object"`,
			`"type":"object","nullable":false`,
			`"type":"object","additionalProperties":true`,
		} {
			r := request(schemas(`{"R":{`+shape+`,"default":null}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_UNSUPPORTED" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/R/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("non-nullable raw default changed its original diagnostic", shape, checked, err)
			}
		}
		r := request(schemas(`{"R":{"type":"object","nullable":true,"additionalProperties":true,"default":[1]}}`), "Dto")
		r.Options.Validation = checked
		_, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/R/default" {
			t.Fatal("nullable open dictionary accepted a non-object value", checked, err)
		}
		// Nullable carriers retain the original non-null value byte budget.
		r.Document = schemas(`{"R":{"type":"object","nullable":true,"default":"` + strings.Repeat("x", 32*1024) + `"}}`)
		_, err = core.GenerateWithDiagnostics(r)
		ds, ok = err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/R/default" {
			t.Fatal("nullable raw value bypassed the default byte budget", checked, err)
		}
	}
}
