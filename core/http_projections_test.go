package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPOperationProjectionValidation(t *testing.T) {
	failure(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"string","enum":["a","a"]}}}}}}`), "E_SCHEMA_INVALID", "/paths/~1ping/get/responses/200/content/application~1json/schema/enum/1")
	failure(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"array","items":{"type":"string","enum":[true]}}}}}}}`), "E_SCHEMA_INVALID", "/paths/~1ping/get/responses/200/content/application~1json/schema/items/enum/0")
}

func TestHTTPOperationProjections(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "projections.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	for _, want := range []string{"kotlin.collections.List<kotlin.String>", "input: kotlin.String,", "org.example.models.api.models.NullableObjectResponseDto?", "org.example.models.models.RecordDto?"} {
		if !strings.Contains(api, want) {
			t.Errorf("missing %q: %s", want, api)
		}
	}
	if len(r.Artifacts.Files) != 4 {
		t.Fatal("unexpected enum/model declarations", r)
	}
	_ = source(t, r, "NullableObjectResponseDto")
	if len(r.Warnings) != 3 {
		t.Fatal("expected three scalar enum projection warnings", r.Warnings)
	}
	for _, warning := range r.Warnings {
		if warning.Code != "W_ENUM_NATIVE_APPROXIMATION" {
			t.Fatal(warning)
		}
	}
}
