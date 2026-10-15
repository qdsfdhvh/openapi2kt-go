package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPResponseProjectionValidation(t *testing.T) {
	failure(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"anyOf":[{"type":"string"},{"$ref":"#/components/schemas/Missing"}]}}}}}}`), "E_REF_MISSING", "/paths/~1ping/get/responses/200/content/application~1json/schema/anyOf/1")
	failure(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"anyOf":[{"type":"string"},{"type":"object","properties":{"v":{"type":"string","minLength":1}}}]}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/responses/200/content/application~1json/schema/anyOf/1/properties/v/minLength")
	failure(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"null","properties":{}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/responses/200/content/application~1json/schema/properties")
}

func TestHTTPResponseKinds(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "response-kinds.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 2 {
		t.Fatal("response unions must not create sealed DTOs", r)
	}
	projections := 0
	for _, warning := range r.Warnings {
		if warning.Code == "W_HTTP_RESPONSE_PROJECTION" {
			projections++
		}
	}
	if projections != 3 {
		t.Fatal(r.Warnings)
	}
	api := source(t, r, "DefaultApi")
	if strings.Count(api, "): kotlinx.serialization.json.JsonElement {") != 3 || !strings.Contains(api, "): kotlin.Unit {") || !strings.Contains(api, "): kotlin.String {") {
		t.Fatal(api)
	}
}
