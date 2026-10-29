package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableCollectionLeafComponentFactories(t *testing.T) {
	for _, leaf := range []struct{ schema, value, kotlin string }{
		{`"type":"string"`, `"x"`, "kotlin.String?"},
		{`"type":"integer"`, `7`, "kotlin.Int?"},
		{`"type":"number"`, `1.5`, "kotlin.Double?"},
		{`"type":"boolean"`, `true`, "kotlin.Boolean?"},
		{`"type":"string","format":"date"`, `"2024-02-29"`, "kotlinx.datetime.LocalDate?"},
		{`"type":"string","format":"date-time"`, `"2024-02-29T00:00:00Z"`, "kotlin.time.Instant?"},
		{`"type":"string","format":"byte"`, `[-128,127]`, "kotlin.ByteArray?"},
		{`"type":"string","format":"binary"`, `[1]`, "kotlin.ByteArray?"},
		{`"type":"string","enum":["x","y"]`, `"x"`, ""},
		{`"type":"string","enum":["x",null]`, `"x"`, ""},
		{`"type":"integer","format":"int32","enum":[1,2]`, `1`, ""},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, container := range []string{"list", "map"} {
					for _, outer := range []string{"", `,"nullable":true`} {
						shape, value, ty := `"type":"array","items":{`, `[null,`+leaf.value+`]`, "kotlin.collections.List<"+leaf.kotlin+">"
						if container == "map" {
							shape, value, ty = `"type":"object","additionalProperties":{`, `{"a":null,"z":`+leaf.value+`}`, "kotlin.collections.Map<kotlin.String, "+leaf.kotlin+">"
						}
						shape += leaf.schema + `,"nullable":true}` + outer
						if outer != "" {
							ty += "?"
						}
						r := request(schemas(`{"N":{`+shape+`,"default":`+value+`},"Holder":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/N"},"required":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
						r.Options.Validation, r.Options.ModelStyle = checked, style
						got, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal(leaf.schema, container, outer, checked, style, err)
						}
						text := source(t, got, "NDto")
						if !strings.Contains(text, "public fun `NDto default`(): NDto =") || (leaf.kotlin != "" && !strings.Contains(text, "public typealias NDto = "+ty+"\n")) {
							t.Fatal("nullable leaf type/factory lost", text)
						}
						again, err := core.GenerateWithDiagnostics(r)
						if err != nil || !reflect.DeepEqual(got, again) {
							t.Fatal("nondeterministic nullable leaf factory", err)
						}
						r.Document = []byte(strings.Replace(string(r.Document), `,"default":`+value, "", 1))
						control, err := core.GenerateWithDiagnostics(r)
						if err != nil || source(t, got, "HolderDto") != source(t, control, "HolderDto") || !reflect.DeepEqual(got.Warnings, control.Warnings) {
							t.Fatal("factory changed consumer presence or warnings", err)
						}
						if strings.Contains(source(t, control, "NDto"), "fun `NDto default`") {
							t.Fatal("unannotated collection gained a factory")
						}
						if outer != "" {
							r.Document = schemas(`{"N":{` + shape + `,"default":null}}`)
							if _, err := core.GenerateWithDiagnostics(r); err != nil {
								t.Fatal("nullable outer null lost", err)
							}
						}
					}
				}
			}
		}
	}
}

func TestNullableCollectionLeafFactoryGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ name, body, code, pointer string }{
			{"outer null", `{"N":{"type":"array","items":{"type":"string","nullable":true},"default":null}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default"},
			{"nonnull list leaf", `{"N":{"type":"array","nullable":true,"items":{"type":"string"},"default":[null]}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/0"},
			{"nonnull map leaf", `{"N":{"type":"object","additionalProperties":{"type":"string"},"default":{"a/b":null}}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/a~1b"},
			{"wrong leaf", `{"N":{"type":"array","items":{"type":"string","nullable":true},"default":[null,1]}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/1"},
			{"enum vocabulary", `{"N":{"type":"array","items":{"type":"string","nullable":true,"enum":["a"]},"default":[null,"z"]}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/1"},
			{"temporal lexeme", `{"N":{"type":"array","items":{"type":"string","format":"date","nullable":true},"default":[null,"2023-02-30"]}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/1"},
			{"byte range", `{"N":{"type":"array","items":{"type":"string","format":"byte","nullable":true},"default":[null,[128]]}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/1/0"},
			{"child default", `{"N":{"type":"array","nullable":true,"items":{"type":"string","nullable":true,"default":null},"default":null}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/N/default"},
			{"composition leaf", `{"N":{"type":"array","nullable":true,"items":{"nullable":true,"oneOf":[{"type":"string"},{"type":"integer"}]},"default":null}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/N/default"},
			{"wrapper wrong leaf", `{"T":{"type":"array","items":{"type":"string","nullable":true}},"N":{"$ref":"#/components/schemas/T","default":[null,1]}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/1"},
		} {
			t.Run(tc.name+"/checked="+map[bool]string{false: "off", true: "on"}[checked], func(t *testing.T) {
				r := request(schemas(tc.body), "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatalf("want %s @ %s, got %v", tc.code, tc.pointer, err)
				}
			})
		}
		r := request(schemas(`{"T":{"type":"string"},"N":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/T","nullable":true},"default":null}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "NDto"), "fun `NDto default`") {
			t.Fatal("nullable referenced scalar factory rejected", checked, err)
		}
		// Null outer values must still validate malformed nullable leaf sources.
		for _, leaf := range []string{
			`"type":"string","nullable":"yes"`,
			`"type":"string","nullable":true,"enum":["a",1]`,
			`"type":"string","nullable":true,"enum":["a","a"]`,
			`"type":"string","nullable":true,"format":"unknown"`,
		} {
			r := request(schemas(`{"N":{"type":"array","nullable":true,"items":{`+leaf+`},"default":null}}`), "Dto")
			r.Options.Validation = checked
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("null skipped malformed leaf source", leaf, err)
			}
		}
	}
}
