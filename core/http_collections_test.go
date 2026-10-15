package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPCollections(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "collections.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := source(t, report(t, input, ""), "DefaultApi")
	for _, want := range []string{"ids: kotlin.collections.List<kotlin.String>", "repeat: kotlin.collections.List<org.example.models.models.State>? = null", "fun same(", "fun same2(", "forEachIndexed", `joinToString(",")`, `joinToString(" ")`, `joinToString("|")`, "serialName"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
}

func TestHTTPParameterInheritanceOrder(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{"/ping":{"parameters":[{"name":"shared","in":"query","schema":{"type":"string"}},{"name":"replace","in":"query","schema":{"type":"integer"}}],"get":{"parameters":[{"name":"own","in":"query","schema":{"type":"boolean"}},{"name":"replace","in":"query","schema":{"type":"string"}}],"responses":{"204":{}}}}}}`)
	s := source(t, report(t, input, ""), "DefaultApi")
	own, replaced, shared := strings.Index(s, "own: kotlin.Boolean"), strings.Index(s, "replace: kotlin.String"), strings.Index(s, "shared: kotlin.String")
	if own < 0 || replaced < own || shared < replaced {
		t.Fatal("operation parameters must precede non-overridden inherited parameters", s)
	}
}

func TestHTTPCollectionUnsupportedShapes(t *testing.T) {
	for _, tc := range []struct{ parameter, tail string }{
		{`{"in":"query","name":"v","style":"matrix","schema":{"type":"array","items":{"type":"string"}}}`, "/style"},
		{`{"in":"header","name":"v","style":"form","schema":{"type":"array","items":{"type":"string"}}}`, "/style"},
		{`{"in":"query","name":"v","style":"deepObject","schema":{"type":"object","additionalProperties":{"type":"string"}}}`, "/style"},
	} {
		failure(t, httpDoc(`{"parameters":[`+tc.parameter+`],"responses":{"204":{}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters/0"+tc.tail)
	}
}
