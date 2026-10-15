package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPStructuralValueBoundaries(t *testing.T) {
	for _, tc := range []struct{ item, want string }{
		{`{"type":"object"}`, `kotlin.collections.List<kotlinx.serialization.json.JsonElement>`},
		{`{"type":"object","additionalProperties":true}`, `kotlin.collections.List<kotlinx.serialization.json.JsonObject>`},
		{`{"type":"object","additionalProperties":false}`, `kotlin.collections.List<kotlinx.serialization.json.JsonObject>`},
		{`{"type":"object","additionalProperties":{"type":"object","properties":{"id":{"type":"string"}}}}`, `kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, kotlinx.serialization.json.JsonElement?>>>`},
	} {
		input := presenceDocument(t, `{"content":{"application/json":{"schema":{"type":"array","items":`+tc.item+`}}}}`, nil)
		r := report(t, input, "")
		if len(r.Artifacts.Files) != 2 || !strings.Contains(source(t, r, "DefaultApi"), tc.want) {
			t.Fatal(tc, r)
		}
	}
	input := presenceDocument(t, `{"content":{"application/json":{"schema":{"type":"array","items":{"type":"object","properties":{"next":{"$ref":"#/components/schemas/Missing"}}}}}}}`, nil)
	failure(t, input, "E_REF_MISSING", "/paths/~1apply/post/requestBody/content/application~1json/schema/items/properties/next")
}

func TestHTTPStructuralValues(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "structural-values.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 3 {
		t.Fatal("structural operation values acquired declarations", r)
	}
	api := source(t, r, "DefaultApi")
	for _, want := range []string{
		"kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlinx.serialization.json.JsonElement?>>",
		"kotlin.collections.List<kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlinx.serialization.json.JsonElement?>?>>",
		"kotlin.collections.List<kotlin.collections.List<org.example.models.models.RecordDto>>",
		"kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.String?>>",
	} {
		if !strings.Contains(api, want) {
			t.Errorf("missing %q: %s", want, api)
		}
	}
}
