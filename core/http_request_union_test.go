package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPRequestUnionBoundaries(t *testing.T) {
	for _, tc := range []struct{ schema, suffix string }{
		{`{"type":"object","properties":{"rows":{"type":"array","items":{"anyOf":[{"type":"string","pattern":"x"},{"type":"boolean"}]}}}}`, "/properties/rows/items/anyOf/0/pattern"},
		{`{"type":"object","properties":{"payload":{"anyOf":[{"type":"string","pattern":"x"},{"type":"boolean"}]}}}`, "/properties/payload/anyOf/0/pattern"},
	} {
		input := presenceDocument(t, `{"content":{"application/json":{"schema":`+tc.schema+`}}}`, nil)
		failure(t, input, "E_SCHEMA_UNSUPPORTED", "/paths/~1apply/post/requestBody/content/application~1json/schema"+tc.suffix)
	}
}

func TestHTTPRequestUnionUsageContexts(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "request-unions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(input, &document); err != nil {
		t.Fatal(err)
	}
	paths := document["paths"].(map[string]any)
	for path, operation := range paths {
		document["paths"] = map[string]any{path: operation}
		isolated, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		result := report(t, isolated, "Dto")
		hasRuntime := false
		for _, file := range result.Artifacts.Files {
			if strings.HasSuffix(file.Path, "/models/Patch.kt") {
				hasRuntime = true
			}
		}
		if hasRuntime != (path != "/choice") {
			t.Fatalf("runtime ownership for %s: %v", path, hasRuntime)
		}
	}
	r := report(t, input, "Dto")
	root := strings.ReplaceAll(source(t, r, "ChooseInputDto"), "kotlin.String", "String")
	if strings.Contains(root, "Patch<") || !strings.Contains(root, "token: String? = null") {
		t.Fatalf("root union must retain ordinary defaults: %s", root)
	}
	nested := strings.ReplaceAll(source(t, r, "EnvelopeInputDto"), "kotlin.String", "String")
	for _, want := range []string{"token: String?,", "note: String? = null", "memo: org.example.models.models.Patch<String>"} {
		if !strings.Contains(nested, want) {
			t.Errorf("nested union missing %q: %s", want, nested)
		}
	}
	if strings.Contains(nested, "token: String? = null,") || strings.Contains(nested, "note: org.example.models.models.Patch") {
		t.Fatalf("variant context leaked plain-object Patch/default rules: %s", nested)
	}
	folded := strings.ReplaceAll(source(t, r, "MaybeInputDto"), "kotlin.String", "String")
	if !strings.Contains(folded, "memo: org.example.models.models.Patch<String>") || !strings.Contains(folded, "token: String?,") {
		t.Fatalf("nullable object fold lost request context: %s", folded)
	}
	if !strings.Contains(source(t, r, "DefaultApi"), "input: org.example.models.api.models.MaybeInputDto? = null") {
		t.Fatal("optional folded body lost nullable default")
	}
}
