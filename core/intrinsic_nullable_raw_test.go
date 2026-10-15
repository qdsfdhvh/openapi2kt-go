package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func intrinsicRawInput(t testing.TB, nullable bool) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-raw.json")
	if e != nil {
		t.Fatal(e)
	}
	if nullable {
		return b
	}
	var doc, components, catalog map[string]json.RawMessage
	if e := json.Unmarshal(b, &doc); e != nil {
		t.Fatal(e)
	}
	_ = json.Unmarshal(doc["components"], &components)
	_ = json.Unmarshal(components["schemas"], &catalog)
	catalog["ZRaw"] = json.RawMessage(`{"type":"object"}`)
	catalog["ZEmpty"] = json.RawMessage(`{"type":"object","properties":{}}`)
	components["schemas"], _ = json.Marshal(catalog)
	doc["components"], _ = json.Marshal(components)
	b, _ = json.Marshal(doc)
	return b
}
func TestIntrinsicNullableRawWholeEnvelopeControl(t *testing.T) {
	for _, validation := range []bool{false, true} {
		r := request(intrinsicRawInput(t, false), "Dto")
		r.Options.Validation = validation
		if _, e := core.GenerateWithDiagnostics(r); e != nil {
			t.Fatal(e)
		}
	}
}
func TestIntrinsicNullableRawDesiredSuccess(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			r := request(intrinsicRawInput(t, true), suffix)
			r.Options.Validation = validation
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil {
				t.Fatal(e)
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", e)
			}
			for _, n := range []string{"ZRaw", "ZEmpty"} {
				s := source(t, got, n+suffix)
				if !strings.Contains(s, "= kotlinx.serialization.json.JsonElement?") || strings.Contains(s, "data class") {
					t.Fatal(s)
				}
			}
			if s := source(t, got, "AFalse"+suffix); !strings.Contains(s, "= kotlinx.serialization.json.JsonElement\n") {
				t.Fatal(s)
			}
			if s := source(t, got, "SubmitInput"+suffix); !strings.Contains(s, "Patch<kotlinx.serialization.json.JsonElement>") || !strings.Contains(s, "kotlin.collections.Map<kotlin.String, org.example.models.models.ZRaw"+suffix+"?>") {
				t.Fatal("request/model forward linking must retain identical raw map item intent", s)
			}
		}
	}
}
