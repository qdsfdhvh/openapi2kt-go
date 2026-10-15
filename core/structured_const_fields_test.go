package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuredConstFieldsKeepBodyApiTypedJsonAndOriginalData(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"kotlinx":{"type":"string"},"kotlin":{"type":"string"},"payload":{"type":"object","const":{"$ref":"literal data","type":"data","values":[null,true,1e2,"null"]}},"rows":{"const":[{"enum":[null]},null,"quoted"]}},"required":["payload","rows"]},"Only":{"properties":{"payload":{"const":{"value":1}}}},"Nested":{"properties":{"child":{"properties":{"rows":{"type":"array","const":[1,2]}}}}}}`)
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(input, "Dto")
			r.Options.Validation, r.Options.ModelStyle = validation, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(validation, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, marker := range []string{"public val payload: kotlinx.serialization.json.JsonObject = `openapi-kt object const`(", "public val rows: kotlinx.serialization.json.JsonArray = `openapi-kt array const`(", "private fun `openapi-kt object const`(", "private fun `openapi-kt array const`(", "literal data", `1e2`, `EncodeDefault`} {
				if !strings.Contains(text, marker) {
					t.Fatal("Structured const lost typed body/data or private initialization", marker, text)
				}
			}
			if strings.Contains(strings.Split(text, ") {")[0], "public val payload:") || !strings.Contains(source(t, got, "OnlyDto"), "public class OnlyDto") {
				t.Fatal("Structured const became constructor/copy/Patch or raw model")
			}
			if len(got.Warnings) != 4 {
				t.Fatal("Structured const approximation warnings", got.Warnings)
			}
			for _, warning := range got.Warnings {
				if warning.Code != "W_CONST_STRUCTURED_APPROXIMATION" || warning.Pointer == nil || !strings.HasSuffix(*warning.Pointer, "/const") {
					t.Fatal("Structured source warning not located", warning)
				}
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Structured const rerun changed", err)
			}
		}
	}
}
