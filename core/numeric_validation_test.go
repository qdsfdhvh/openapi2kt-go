package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNumericValidationPropagation(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/numeric-constraints.json")
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
	for _, name := range []string{"WindowDto", "RecordDto", "NumbersApi", "PatchInputDto", "WatchEventDto"} {
		text := source(t, r, name)
		if !strings.Contains(text, "exclusiveMinimum") && !strings.Contains(text, "exclusiveMaximum") {
			t.Fatal("dropped exclusive bounds", name, text)
		}
		if !strings.Contains(text, "%") || !strings.Contains(text, "multipleOf") {
			t.Fatal("dropped multipleOf", name, text)
		}
	}
	if strings.Contains(source(t, r, "RecordDto"), "kotlin.Long.MIN_VALUE") {
		t.Fatal("constraint expression shadowed by field kotlin")
	}
}
