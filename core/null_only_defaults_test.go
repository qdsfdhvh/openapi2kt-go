package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullOnlyDefaultsKeepCanonicalCarrierAndRequiredPresence(t *testing.T) {
	for _, field := range []string{`{"type":"null","default":null}`, `{"type":"null","enum":[null],"default":null}`, `{"enum":[null],"default":null}`} {
		input := schemas(`{"Holder":{"properties":{"required":` + field + `,"optional":` + field + `},"required":["required"]}}`)
		for _, validation := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = validation, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(field, validation, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, "public val required: kotlinx.serialization.json.JsonNull,") || !strings.Contains(text, "public val optional: kotlinx.serialization.json.JsonNull? = null") || len(got.Warnings) != 0 {
					t.Fatal("Null default replaced canonical carrier or required argument", text, got.Warnings)
				}
				r.Document = []byte(strings.ReplaceAll(string(input), `,"default":null`, ""))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("Null initializer annotation changed the existing API/wire", field, err)
				}
			}
		}
	}
}
