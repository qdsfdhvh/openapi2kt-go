package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableRecursiveCoverEdgesMinimal(t *testing.T) {
	base := coverChainRoot("Value", "Value")
	for _, intent := range []string{"true", "false"} {
		changed := strings.ReplaceAll(base, `"$ref":"#/components/schemas/Value"`, `"$ref":"#/components/schemas/Value","nullable":`+intent)
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Value":`+changed+`}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(intent, checked, err)
			}
			control := rq
			control.Document = schemas(`{"Value":` + base + `}`)
			before, e := core.GenerateWithDiagnostics(control)
			if e != nil || !reflect.DeepEqual(got, before) {
				t.Fatal("JSON cover edge changed its existing carrier/wire/warning", intent, checked, e)
			}
		}
	}
}

func TestNullableRecursiveCoverEdgesGrammarBoundaries(t *testing.T) {
	for _, tc := range []struct{ schema, pointer string }{
		{strings.Replace(coverChainRoot("Value", "Value"), `"$ref":"#/components/schemas/Value"`, `"$ref":"#/components/schemas/Value","nullable":"bad"`, 1), "/components/schemas/Value/oneOf/4/items/nullable"},
		{strings.Replace(coverChainRoot("Value", "Value"), `"$ref":"#/components/schemas/Value"`, `"$ref":"#/components/schemas/Value","nullable":true,"default":null`, 1), "/components/schemas/Value/oneOf/4/items/default"},
		{strings.Replace(coverChainRoot("Value", "Value"), `"$ref":"#/components/schemas/Value"`, `"$ref":"#/components/schemas/Value","nullable":true,"minimum":0`, 1), "/components/schemas/Value/oneOf/4/items/minimum"},
		{strings.TrimSuffix(coverChainRoot("Value", "Value"), "}") + `,"nullable":true}`, "/components/schemas/Value/nullable"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Value":`+tc.schema+`}`), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || e.(core.Diagnostics)[0].Pointer == nil || *e.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("edge intent hid unsupported original grammar", tc, checked, e)
			}
		}
	}
}
