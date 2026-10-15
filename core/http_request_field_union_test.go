package core_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestHTTPRequestFieldUnions(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "request-field-unions.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 5 {
		t.Fatal(r)
	}
	model := source(t, r, "ValuesInputDto")
	for _, want := range []string{
		"choice: kotlinx.serialization.json.JsonElement,",
		"nullableChoice: kotlinx.serialization.json.JsonElement?,",
		"optionalPlain: kotlinx.serialization.json.JsonElement? = null",
		"optionalPatch: org.example.models.models.Patch<kotlinx.serialization.json.JsonElement>",
		"nullablePayload: kotlinx.serialization.json.JsonElement?,",
		"payload: kotlinx.serialization.json.JsonElement? = null",
		"kinds: kotlin.collections.List<kotlinx.serialization.json.JsonElement>? = null",
		"values: kotlinx.serialization.json.JsonObject? = null",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("missing %q: %s", want, model)
		}
	}
	if strings.Contains(model, "StringChoice") || strings.Contains(model, "StringPayload") || strings.Contains(model, "nullableChoice: kotlinx.serialization.json.JsonElement? = null") {
		t.Fatal("request-only fields must not gain native wrappers or required-nullable defaults", model)
	}
	if !strings.Contains(source(t, r, "SharedDto"), "data class StringChoice") {
		t.Fatal("component wrapper contract was changed")
	}
	_ = source(t, r, "Patch")
	count := 0
	for _, warning := range r.Warnings {
		if warning.Code == "W_HTTP_FIELD_UNION_PROJECTION" {
			count++
		}
	}
	if count != 18 || !reflect.DeepEqual(r, report(t, input, "Dto")) {
		t.Fatal("field projection warning or deterministic contract", count, r.Warnings)
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 2 {
		t.Fatal("excluded operation left request model/runtime", excluded, err)
	}
}
