package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func booleanValueInput(t *testing.T) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/boolean-values.json")
	if err != nil {
		t.Fatal(err)
	}
	return input
}
func TestBooleanValueWholeLoweredControl(t *testing.T) {
	input, err := testutil.BooleanValueControl(booleanValueInput(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, "whole control", err)
			}
			holder := source(t, got, "models/HolderDto")
			if !strings.Contains(holder, "value: kotlinx.serialization.json.JsonElement") || !strings.Contains(holder, "optional: kotlinx.serialization.json.JsonElement? = null") || !strings.Contains(holder, "List<kotlinx.serialization.json.JsonElement>") {
				t.Fatal(holder)
			}
		}
	}
}
func TestBooleanValueDesiredWholeEnvelope(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(booleanValueInput(t), "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, "desired true value schema success", err)
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("not deterministic", e)
			}
			holder := source(t, got, "models/HolderDto")
			if !strings.Contains(holder, "value: kotlinx.serialization.json.JsonElement") || strings.Contains(holder, "data class Value") {
				t.Fatal(holder)
			}
			warnings := map[string]bool{}
			for _, warning := range got.Warnings {
				if warning.Code == "W_BOOLEAN_SCHEMA_PROJECTION" {
					if warning.Pointer == nil || warnings[*warning.Pointer] {
						t.Fatal("missing or duplicate original true pointer", warning)
					}
					warnings[*warning.Pointer] = true
				}
			}
			for _, pointer := range []string{"/components/schemas/Holder/properties/value", "/components/schemas/Holder/properties/optional", "/components/schemas/Holder/properties/rows/items", "/components/schemas/Holder/properties/mapping/additionalProperties/items"} {
				if !warnings[pointer] {
					t.Fatal("missing source diagnostic", pointer)
				}
			}
		}
	}
}
