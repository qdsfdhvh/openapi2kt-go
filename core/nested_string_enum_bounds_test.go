package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nestedStringEnumBoundsGraduation(t *testing.T, doc []byte) {
	t.Helper()
	rq := request(doc, "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !strings.Contains(source(t, got, "RecordDto"), "serialName.let") {
		t.Fatal("original bounded nested source did not migrate", err)
	}
}

func TestNestedStringEnumBoundsFieldsAndDefaults(t *testing.T) {
	for _, metadata := range []string{"", `,"format":"date"`, `,"x-native-type":"org.example.UnusedProvider"`} {
		leaf := `{"type":"string","enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"` + metadata + `}`
		for _, tc := range []struct{ field, value string }{
			{`{"type":"array","items":{"type":"array","items":LEAF}}`, `[["one"]]`},
			{`{"type":"array","items":{"additionalProperties":LEAF}}`, `[{"a":"one"}]`},
			{`{"additionalProperties":{"type":"array","items":LEAF}}`, `{"a":["one"]}`},
			{`{"additionalProperties":{"additionalProperties":LEAF}}`, `{"a":{"b":"one"}}`},
			{`{"type":"array","items":{"additionalProperties":{"type":"array","items":LEAF}}}`, `[{"a":["one"]}]`},
		} {
			for _, own := range []bool{false, true} {
				field := strings.Replace(tc.field, "LEAF", leaf, 1)
				if own {
					field = strings.TrimSuffix(field, "}") + `,"default":` + tc.value + `}`
				}
				rq := request(fieldDoc(field), "Dto")
				rq.Options.Validation = true
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("nested bounded lexical enum field", metadata, own, field, err)
				}
				text := source(t, got, "RecordDto")
				for _, marker := range []string{"serialName.let", "minLength", "maxLength", "containsMatchIn", "class Unrecognized"} {
					if !strings.Contains(text, marker) {
						t.Fatal("nested bounded enum lost actual lexical check", marker, text)
					}
				}
				if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nested bounded enum nondeterminism", err)
				}
				rq.Options.Validation = false
				if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("nested bounds bypassed validation opt-in", err)
				}
			}
		}
	}
}

func TestNestedStringEnumBoundsFormerGuards(t *testing.T) {
	for _, body := range []string{
		`{"Value":{"properties":{"state":{"enum":["one"],"minLength":2},"bad":{"type":"array","items":{"type":"array","items":{"enum":["one"],"minLength":2}}}},"default":{"state":"one"}}}`,
		`{"Use":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"enum":["one","x"],"minLength":2}}}}}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("original nested-bounds refusal did not migrate", body, err)
		}
	}
}
