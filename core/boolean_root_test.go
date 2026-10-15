package core_test

import (
	"openapi-kt/core"
	"openapi-kt/internal/testutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func booleanRootInput(t *testing.T) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/boolean-roots.json")
	if err != nil {
		t.Fatal(err)
	}
	return input
}
func TestBooleanRootWholeLoweredControl(t *testing.T) {
	input, err := testutil.BooleanRootControl(booleanRootInput(t))
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
				t.Fatal(style, checked, err)
			}
			if !strings.Contains(source(t, got, "models/ZAnyDto"), "typealias ZAnyDto = kotlinx.serialization.json.JsonElement") {
				t.Fatal(got)
			}
		}
	}
}
func TestBooleanRootDesiredWholeEnvelope(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(booleanRootInput(t), "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, "desired named true success", err)
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic", e)
			}
			root := source(t, got, "models/ZAnyDto")
			if !strings.Contains(root, "typealias ZAnyDto = kotlinx.serialization.json.JsonElement") || strings.Contains(root, "class ZAnyDto") {
				t.Fatal(root)
			}
			if len(got.Warnings) != 1 || got.Warnings[0].Code != "W_BOOLEAN_SCHEMA_PROJECTION" || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/ZAny" {
				t.Fatal(got.Warnings)
			}
		}
	}
}
