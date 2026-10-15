package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestDTOPropertyCountFixture(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/dto-property-counts.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"RecordDto", "NodeDto", "ZeroDto", "PatchInputDto", "CountsApi", "WatchEventDto"} {
		text := source(t, r, name)
		if text == "" {
			t.Fatal(name)
		}
	}
	text := source(t, r, "RecordDto")
	for _, fragment := range []string{"KeepGeneratedSerializer", "generatedSerializer()", "private object", "by kotlin.lazy", "get() = delegate.descriptor", "element.size", "JsonDecoder", "JsonEncoder"} {
		if !strings.Contains(text, fragment) {
			t.Fatal("missing", fragment, text)
		}
	}
	if strings.Contains(text, "RecordDto.serializer()") {
		t.Fatal("self delegate", text)
	}
}
