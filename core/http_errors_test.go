package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestHTTPErrorSchemaBoundaries(t *testing.T) {
	input := httpDoc(`{"responses":{"204":{},"400":{},"401":{"content":{"text/html":{"schema":{"type":"string"}}}},"402":{"content":{"application/json":{"example":{"$ref":"opaque"}}}}}}`)
	r := report(t, input, "")
	if len(r.Artifacts.Files) != 2 || strings.Contains(source(t, r, "DefaultApi"), "[errorBody]") {
		t.Fatal(r)
	}
	input = httpDoc(`{"responses":{"204":{},"400":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Missing"}}}}}}`)
	failure(t, input, "E_REF_MISSING", "/paths/~1ping/get/responses/400/content/application~1json/schema")
	defaultOnly := report(t, httpDoc(`{"responses":{"default":{"content":{"application/json":{"schema":{"type":"string"}}}}}}`), "")
	if !strings.Contains(source(t, defaultOnly, "DefaultApi"), "): kotlin.Unit") || len(defaultOnly.Artifacts.Files) != 3 {
		t.Fatal(defaultOnly)
	}
	rq := request(input, "")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	if _, err := core.GenerateWithDiagnostics(rq); err == nil {
		t.Fatal("excluded error schema bypassed validation")
	}
}

func TestHTTPDeclaredErrors(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "errors.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	last := -1
	for _, status := range []string{"400", "404", "409", "5XX", "default"} {
		index := strings.Index(api, "`"+status+"`")
		if index <= last {
			t.Fatalf("error status ordering %s: %s", status, api)
		}
		last = index
	}
	if !strings.Contains(api, "org.example.models.models.ProblemDto") || !strings.Contains(api, "org.example.models.api.models.FetchError409Dto") {
		t.Fatal(api)
	}
	helper := source(t, r, "ErrorBody")
	if !strings.Contains(helper, "CancellationException") || !strings.Contains(helper, "errorBody") {
		t.Fatal(helper)
	}
	if len(r.Artifacts.Files) != 6 || !strings.Contains(source(t, r, "FetchError409Dto"), "val reason: kotlin.String? = null,") {
		t.Fatal(r)
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range excluded.Artifacts.Files {
		if strings.HasSuffix(file.Path, "/ErrorBody.kt") {
			t.Fatal("excluded errors left helper")
		}
	}
	if len(excluded.Artifacts.Files) != 3 {
		t.Fatal(excluded)
	}
}
