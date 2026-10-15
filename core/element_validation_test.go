package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestElementValidationProvenanceDoesNotChangeDeclarationIdentity(t *testing.T) {
	shape := `{"type":"object","properties":{"ids":{"type":"array","items":{"type":"string","minLength":1}}}}`
	input := []byte(`{"openapi":"3.1.0","paths":{"/a":{"get":{"tags":["Left"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}},"/b":{"get":{"tags":["Right"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}}}`)
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal("diagnostic source must not change shape identity", err)
	}
	text := source(t, r, "SharedResponseDto")
	if !strings.Contains(text, "require(") {
		t.Fatal("shared response lost checks", text)
	}
}

func TestDirectElementValidationPropagation(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/element-constraints.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"RecordDto", "PatchInputDto", "WatchEventDto", "ElementsApi"} {
		text := source(t, r, name)
		if !strings.Contains(text, "require(") || !strings.Contains(text, "forEach") || !strings.Contains(text, "Field value ") {
			t.Fatal("lost element checks", name, text)
		}
	}
	if !strings.Contains(source(t, r, "BagDto"), "typealias") {
		t.Fatal("dictionary alias API changed")
	}
	if !strings.Contains(source(t, r, "ElementsApi"), ".also") {
		t.Fatal("response element checks dropped")
	}
}
