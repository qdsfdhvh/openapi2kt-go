package core_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInferredNullLiteralsMatchExplicitCanonicalTypes(t *testing.T) {
	for _, literal := range []string{`{"const":null}`, `{"enum":[null]}`} {
		input := schemas(`{"Value":` + literal + `,"Alias":{"$ref":"#/components/schemas/Value"},"Holder":{"properties":{"required":` + literal + `,"optional":` + literal + `,"rows":{"type":"array","items":` + literal + `},"map":{"additionalProperties":` + literal + `}},"required":["required"]}}`)
		explicit := []byte(strings.ReplaceAll(string(input), literal, `{"type":"null",`+literal[1:]))
		for _, validation := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = validation, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(literal, validation, style, err)
				}
				r.Document = explicit
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("Inferred null changed canonical API, warnings or wire plan", literal, validation, style, err)
				}
			}
		}
	}
}

func TestInferredNullFixtureMatchesExplicitOperationsAndOwnedOptions(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("..", "tests", "fixtures", "http", "explicit-null-literals.json"))
	if err != nil {
		t.Fatal(err)
	}
	inferred := []byte(strings.ReplaceAll(string(input), `"type": "null",`, ""))
	if reflect.DeepEqual(input, inferred) {
		t.Fatal("Missing original explicit null source")
	}
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, optimized := range []bool{false, true} {
				r := request(inferred, "Dto")
				r.Options.Validation, r.Options.ModelStyle = validation, style
				r.Options.DeduplicateModels, r.Options.HoistSharedShapes, r.Options.MergeIdenticalModels = optimized, optimized, optimized
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(validation, style, optimized, err)
				}
				r.Document = input
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("Original null inference changed HTTP/SSE/Patch/owned declarations", validation, style, optimized, err)
				}
			}
		}
	}
}
