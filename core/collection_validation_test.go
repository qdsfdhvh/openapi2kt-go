package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestCollectionNullableFoldCounts(t *testing.T) {
	for _, schema := range []string{
		`{"anyOf":[{"type":"array","minItems":1,"items":{"type":"string"}},{"type":"null"}]}`,
		`{"oneOf":[{"type":"object","additionalProperties":{"type":"string"},"minProperties":1},{"type":"null"}]}`,
	} {
		req := request(fieldDoc(schema), "Dto")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err != nil {
			t.Fatal(err)
		}
		text := source(t, r, "RecordDto")
		if !strings.Contains(text, ".size") || !strings.Contains(text, "?.let") {
			t.Fatal("nullable fold erased count", text)
		}
	}
}

func TestCollectionCountPropagation(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/collection-counts.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		if ds, ok := err.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
			t.Fatal(err, *ds[0].Pointer)
		}
		t.Fatal(err)
	}
	for _, name := range []string{"RecordDto", "CountsApi", "PatchInputDto", "WatchEventDto"} {
		text := source(t, r, name)
		if !strings.Contains(text, "minItems") || !strings.Contains(text, "maxItems") || !strings.Contains(text, ".size") {
			t.Fatal("lost container checks", name, text)
		}
	}
	if !strings.Contains(source(t, r, "RecordDto"), "minProperties") || !strings.Contains(source(t, r, "RecordDto"), "Field value is shorter than minLength") {
		t.Fatal("container/element checks cannot coexist")
	}
	if !strings.Contains(source(t, r, "BagDto"), "typealias") {
		t.Fatal("dictionary shape changed")
	}
}
