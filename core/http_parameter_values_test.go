package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPParameterValues(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "parameter-values.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	for _, want := range []string{
		"pathList: kotlin.collections.List<kotlin.String?>,",
		"nested: kotlin.collections.List<kotlin.collections.List<kotlin.String>>? = null",
		"states: kotlin.collections.List<org.example.models.models.StateDto?>? = null",
		"map: kotlin.collections.Map<kotlin.String, kotlin.Int?>? = null",
		"record: org.example.models.models.RecordDto? = null",
		"it?.serialName", "it?.serialName ?: \"null\"",
	} {
		if !strings.Contains(api, want) {
			t.Errorf("missing %q: %s", want, api)
		}
	}
	filter := source(t, r, "FilterValuesFilterDto")
	if !strings.Contains(filter, "opaque: kotlinx.serialization.json.JsonElement? = null") || !strings.Contains(filter, "nullableOpaque: kotlinx.serialization.json.JsonElement? = null") || strings.Contains(filter, "Patch<") || strings.Contains(filter, "StringOpaque") {
		t.Fatal("filter must retain ordinary defaults/raw union without payload wrappers", filter)
	}
}
