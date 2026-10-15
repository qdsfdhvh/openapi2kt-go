package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func recursiveJSONInput(t *testing.T) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/recursive-json.json")
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestRecursiveJSONWholeControl(t *testing.T) {
	input, err := testutil.RecursiveJSONControl(recursiveJSONInput(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			if !strings.Contains(source(t, got, "models/ZAnyDto"), "typealias ZAnyDto = kotlinx.serialization.json.JsonElement") {
				t.Fatal(got)
			}
		}
	}
}

func TestRecursiveJSONDesiredWholeEnvelope(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(recursiveJSONInput(t), "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, "desired recursive any-JSON success", err)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("non-deterministic", err)
			}
			if !strings.Contains(source(t, got, "models/ZAnyDto"), "typealias ZAnyDto = kotlinx.serialization.json.JsonElement") {
				t.Fatal(got)
			}
			control, err := testutil.RecursiveJSONControl(r.Document)
			if err != nil {
				t.Fatal(err)
			}
			baseline := r
			baseline.Document = control
			before, err := core.GenerateWithDiagnostics(baseline)
			if err != nil || !reflect.DeepEqual(got.Artifacts.Files, before.Artifacts.Files) {
				t.Fatal("whole JSON carrier control files differ", err)
			}
			if len(got.Warnings) != 1 || got.Warnings[0].Code != "W_RECURSIVE_JSON_PROJECTION" || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/ZAny" {
				t.Fatal(got.Warnings)
			}
		}
	}
}
