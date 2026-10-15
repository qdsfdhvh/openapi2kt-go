package core_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestHTTPOperationScalarUnions(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "operation-scalar-unions.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 3 {
		t.Fatal("simple scalar projections must not own enum DTOs", r)
	}
	api := source(t, r, "DefaultApi")
	for _, want := range []string{"input: kotlin.String,", "input: kotlin.String? = null,", "input: kotlin.collections.List<kotlin.String?>,", "input: kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.String?>>", "kotlin.collections.List<org.example.models.models.StateDto?>", "): kotlin.String?"} {
		if !strings.Contains(api, want) {
			t.Errorf("missing %q: %s", want, api)
		}
	}
	overlap, enums := 0, 0
	for _, w := range r.Warnings {
		if w.Code == "W_ONEOF_NULL_OVERLAP" {
			overlap++
		}
		if w.Code == "W_ENUM_NATIVE_APPROXIMATION" {
			enums++
		}
	}
	if overlap != 1 || enums != 7 {
		t.Fatal(r.Warnings)
	}
	if again := report(t, input, "Dto"); !reflect.DeepEqual(r, again) {
		t.Fatal("scalar projection changed on repeat")
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 2 {
		t.Fatal(excluded, err)
	}
}

func TestHTTPOperationScalarUnionGuards(t *testing.T) {
	base := "/paths/~1apply/post/requestBody/content/application~1json/schema"
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"oneOf":[{"enum":[true]},{"type":"null"}]}`, "E_SCHEMA_INVALID", "/oneOf/0/enum/0"},
		{`{"anyOf":[{"enum":["a"]},{"type":"string","pattern":false}]}`, "E_SCHEMA_INVALID", "/anyOf/1/pattern"},
		{`{"anyOf":[{"type":"string","enum":["a"]},{"type":"string","unknownKeyword":true}]}`, "E_SCHEMA_UNSUPPORTED", "/anyOf/1/unknownKeyword"},
		{`{"anyOf":[{"$ref":"#/components/schemas/Missing"},{"type":"null"}]}`, "E_REF_MISSING", "/anyOf/0"},
	} {
		input := presenceDocument(t, `{"content":{"application/json":{"schema":`+tc.schema+`}}}`, nil)
		failure(t, input, tc.code, base+tc.tail)
	}
	// A nullable fold in a DTO property remains a model enum, not a raw String.
	body := `{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"state":{"anyOf":[{"enum":["known"]},{"type":"null"}]}}}}}}`
	r := report(t, presenceDocument(t, body, nil), "Dto")
	if !strings.Contains(source(t, r, "PostApplyInputDto"), "sealed class State") {
		t.Fatal(r)
	}
	// Closed enum unions do not take the known+free-string fold.
	closed := `{"anyOf":[{"type":"string","enum":["left"]},{"type":"string","enum":["right"]}]}`
	r = report(t, presenceDocument(t, `{"required":true,"content":{"application/json":{"schema":`+closed+`}}}`, nil), "Dto")
	if !strings.Contains(source(t, r, "PostApplyInputDto"), "data object Variant2") {
		t.Fatal(r)
	}
}
