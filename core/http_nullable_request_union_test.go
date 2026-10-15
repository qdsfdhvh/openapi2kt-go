package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPNullableRequestUnions(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "nullable-request-unions.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 4 {
		t.Fatal(r)
	}
	projectionWarnings := 0
	for _, warning := range r.Warnings {
		if warning.Code == "W_HTTP_REQUEST_UNION_PROJECTION" {
			projectionWarnings++
		}
	}
	if projectionWarnings != 2 {
		t.Fatal(r.Warnings)
	}
	api := source(t, r, "DefaultApi")
	if !strings.Contains(api, "input: org.example.models.api.models.RequiredChoiceInputDto,") || !strings.Contains(api, "input: org.example.models.api.models.OptionalChoiceInputDto? = null,") {
		t.Fatal(api)
	}
	model := source(t, r, "RequiredChoiceInputDto")
	if !strings.Contains(model, "memo: kotlin.String? = null") || strings.Contains(model, "data object Null") || strings.Contains(model, "Patch<") {
		t.Fatal(model)
	}
}
