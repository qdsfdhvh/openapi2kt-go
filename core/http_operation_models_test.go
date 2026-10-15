package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPOperationLocalModels(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "operation-models.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	for _, want := range []string{
		"kotlin.collections.List<org.example.models.api.models.FetchOrderDto>? = null",
		"org.example.models.api.models.FetchInputParamDto? = null",
		"body<org.example.models.api.models.FetchResponseDto>()",
		"body<kotlin.collections.List<org.example.models.api.models.ListItemDto>>()",
		"body<org.example.models.api.models.LookupResponseDto>()",
		"body<kotlin.collections.List<kotlin.Int>>()",
	} {
		if !strings.Contains(api, want) {
			t.Errorf("missing %q in %s", want, api)
		}
	}
	response := source(t, r, "FetchResponseDto")
	if !strings.Contains(response, "package org.example.models.api.models") || !strings.Contains(response, "org.example.models.models.StateDto") {
		t.Fatal(response)
	}
	alias := source(t, r, "LookupResponseDto")
	if !strings.Contains(alias, "typealias LookupResponseDto = kotlin.collections.Map<kotlin.String, kotlin.collections.List<org.example.models.models.RecordDto>>") {
		t.Fatal(alias)
	}
}
