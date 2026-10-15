package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestAliasValidationPropagation(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/alias-constraints.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"IdDto", "LinkDto"} {
		if !strings.Contains(source(t, r, name), "typealias "+name+" = kotlin.String") {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"RecordDto", "PatchInputDto", "WatchEventDto", "AliasesApi"} {
		text := source(t, r, name)
		if !strings.Contains(text, "require(") || !strings.Contains(text, "Field value ") {
			t.Fatal("lostrefchecks", name, text)
		}
	}
	text := source(t, r, "RecordDto")
	if !strings.Contains(text, "forEach") || !strings.Contains(text, "value >= -2L") {
		t.Fatal("elementvalidationdropped", text)
	}
	text = source(t, r, "AliasesApi")
	if !strings.Contains(text, ".also") {
		t.Fatal("uncheckedrootresponse", text)
	}
}
