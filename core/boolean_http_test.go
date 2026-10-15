package core_test

import (
	"openapi-kt/core"
	"openapi-kt/internal/testutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func booleanHTTPInput(t *testing.T) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/boolean-http.json")
	if err != nil {
		t.Fatal(err)
	}
	return input
}
func TestBooleanHTTPWholeLoweredControl(t *testing.T) {
	input, err := testutil.BooleanHTTPControl(booleanHTTPInput(t))
	if err != nil {
		t.Fatal(err)
	}
	booleanHTTPWhole(t, input, false)
}
func TestBooleanHTTPDesiredWholeEnvelope(t *testing.T) {
	booleanHTTPWhole(t, booleanHTTPInput(t), true)
}
func booleanHTTPWhole(t *testing.T, input []byte, desired bool) {
	t.Helper()
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, "desired whole direct true JSON HTTP success", err)
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", e)
			}
			api := source(t, got, "api/DefaultApi")
			for _, part := range []string{"input: kotlinx.serialization.json.JsonElement,", "input: kotlinx.serialization.json.JsonElement? = null,", "): kotlinx.serialization.json.JsonElement {", ".body<kotlinx.serialization.json.JsonElement>()"} {
				if !strings.Contains(api, part) {
					t.Fatal(part, api)
				}
			}
			if len(got.Artifacts.Files) != 4 {
				t.Fatal("unexpected DTO/codec", got.Artifacts.Files)
			}
			if desired {
				if len(got.Warnings) != 11 {
					t.Fatal(got.Warnings)
				}
				seen := map[string]bool{}
				for _, w := range got.Warnings {
					if w.Code != "W_BOOLEAN_SCHEMA_PROJECTION" || w.Pointer == nil || seen[*w.Pointer] {
						t.Fatal(w)
					}
					seen[*w.Pointer] = true
				}
				for _, p := range []string{"/components/requestBodies/Any/content/application~1json/schema", "/components/responses/Any/content/application~1json/schema", "/paths/~1watch/post/responses/200/content/text~1event-stream/schema/oneOf/0/properties/data"} {
					if !seen[p] {
						t.Fatal(p, seen)
					}
				}
			}
		}
	}
}
