package core_test

import (
	"openapi-kt/core"
	"openapi-kt/internal/testutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func booleanRootNullableInput(t *testing.T) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/boolean-root-nullable.json")
	if err != nil {
		t.Fatal(err)
	}
	return input
}
func TestBooleanRootNullableWholeLoweredControl(t *testing.T) {
	input, err := testutil.BooleanRootControl(booleanRootNullableInput(t))
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
			checkBooleanRootNullableTypes(t, r, got)
		}
	}
}
func TestBooleanRootNullableDesiredWholeEnvelope(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(booleanRootNullableInput(t), "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, "desired nullable true-root alias success", err)
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic", e)
			}
			checkBooleanRootNullableTypes(t, r, got)
			if len(got.Warnings) != 1 || got.Warnings[0].Code != "W_BOOLEAN_SCHEMA_PROJECTION" || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/ZAny" {
				t.Fatal(got.Warnings)
			}
		}
	}
}
func checkBooleanRootNullableTypes(t *testing.T, r core.GenerationRequest, got core.GenerationReport) {
	t.Helper()
	root := r.Options.PackageName + ".models.ZAnyDto"
	for _, name := range []string{"ANullableDto", "BInheritedDto", "DNullableAgainDto"} {
		if !strings.Contains(source(t, got, "models/"+name), "typealias "+name+" = "+root+"?") {
			t.Fatal(name, source(t, got, "models/"+name))
		}
	}
	if !strings.Contains(source(t, got, "models/CFalseDto"), "typealias CFalseDto = "+root+"\n") {
		t.Fatal(source(t, got, "models/CFalseDto"))
	}
	holder := source(t, got, "models/HolderDto")
	if !strings.Contains(holder, "inherited: "+root+"? = null") || !strings.Contains(holder, "narrowed: "+root+",") {
		t.Fatal(holder)
	}
	patch := source(t, got, "api/models/PatchInputDto")
	if !strings.Contains(patch, "Patch<"+root+">") || !strings.Contains(patch, "requiredNullable: "+root+"?,") {
		t.Fatal(patch)
	}
}
