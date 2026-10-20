package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestChainedWrapperDefaultFactories(t *testing.T) {
	for _, tc := range []struct{ shape, value string }{
		{`"type":"object","required":["z"],"properties":{"z":{"type":"string"}}`, `{"z":"value"}`},
		{`"type":"object","nullable":true,"properties":{"z":{"type":"string"}}`, `null`},
		{`"type":"array","items":{"additionalProperties":{"type":"integer","nullable":true}}`, `[{"a":null,"z":1}]`},
		{`"type":["array","null"],"items":{"type":"string"}`, `null`},
		{`"type":"string","enum":["a","b"]`, `"a"`},
		{`"type":"string","format":"date"`, `"2024-02-29"`},
		{`"type":"string","nullable":true,"format":"byte"`, `null`},
		{`"type":"string","format":"binary"`, `[-128,127]`},
		{`"type":"object"`, `[1,true,null]`},
		{`"type":["object","null"],"additionalProperties":true`, `null`},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				r := request(schemas(`{"T":{`+tc.shape+`},"B":{"$ref":"#/components/schemas/T"},"M":{"$ref":"#/components/schemas/B"},"N":{"$ref":"#/components/schemas/M","default":`+tc.value+`},"Holder":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/N"},"required":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc, checked, style, err)
				}
				text := source(t, got, "NDto")
				if !strings.Contains(text, "public fun `NDto default`(): NDto =") || !strings.Contains(text, "serializer<NDto>()") {
					t.Fatal("chain factory lost actual alias/helper", text)
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("chain default generation drifted", err)
				}
				r.Document = []byte(strings.Replace(string(r.Document), `,"default":`+tc.value, "", 1))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got.Warnings, control.Warnings) {
					t.Fatal("chain factory changed source warnings", err)
				}
				for _, file := range []string{"TDto", "BDto", "MDto", "HolderDto"} {
					if source(t, got, file) != source(t, control, file) {
						t.Fatal("factory changed target/chain/consumer output", file)
					}
				}
			}
		}
	}
}
