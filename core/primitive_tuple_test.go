package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func primitiveTupleInput(t *testing.T) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/primitive-tuples.json")
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestPrimitiveTupleWholeLoweredControl(t *testing.T) {
	input, err := testutil.PrimitiveTupleControl(primitiveTupleInput(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []string{"data_class", "plain_class"} {
		for _, validation := range []bool{false, true} {
			t.Run(style+map[bool]string{false: "/off", true: "/on"}[validation], func(t *testing.T) {
				r := request(input, "Dto")
				r.Options.ModelStyle = core.ModelStyle(style)
				r.Options.Validation = validation
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || len(got.Artifacts.Files) == 0 {
					t.Fatal("whole envelope control", err)
				}
				for name, element := range map[string]string{"StringTupleDto": "kotlin.String", "LongTupleDto": "kotlin.Long", "MixedTupleDto": "kotlinx.serialization.json.JsonElement", "PriorityTupleDto": "kotlin.String"} {
					if !strings.Contains(source(t, got, "models/"+name), "List<"+element+">") {
						t.Fatal(name, element)
					}
				}
			})
		}
	}
}

func TestPrimitiveTupleDesiredWholeEnvelope(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, validation := range []bool{false, true} {
			t.Run(style+map[bool]string{false: "/off", true: "/on"}[validation], func(t *testing.T) {
				r := request(primitiveTupleInput(t), "Dto")
				r.Options.ModelStyle = core.ModelStyle(style)
				r.Options.Validation = validation
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal("desired successful prefix tuple projection", err)
				}
				again, e := core.GenerateWithDiagnostics(r)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic", e)
				}
				for name, element := range map[string]string{"StringTupleDto": "kotlin.String", "LongTupleDto": "kotlin.Long", "MixedTupleDto": "kotlinx.serialization.json.JsonElement", "PriorityTupleDto": "kotlin.String"} {
					if !strings.Contains(source(t, got, "models/"+name), "List<"+element+">") {
						t.Fatal(name, element)
					}
				}
				tuples := 0
				for _, warning := range got.Warnings {
					if warning.Code == "W_TUPLE_APPROXIMATION" {
						tuples++
						if warning.Pointer == nil || !strings.HasSuffix(*warning.Pointer, "/prefixItems") {
							t.Fatal(warning)
						}
					}
				}
				if tuples == 0 {
					t.Fatal("missing positional/tail approximation diagnostic")
				}
			})
		}
	}
}
