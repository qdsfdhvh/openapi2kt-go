package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPLocalReferences(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "references.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := source(t, report(t, input, "Dto"), "DefaultApi")
	for _, want := range []string{"id: kotlin.String,", "limit: kotlin.Long? = null,", "xTrace: kotlin.String? = null,", "input: org.example.models.models.WidgetDto,", "body<org.example.models.models.WidgetDto>()"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
}

func TestHTTPReferenceFailures(t *testing.T) {
	for _, tc := range []struct{ op, extra, code, pointer string }{
		{`{"parameters":[{"$ref":"https://example.test/param"}],"responses":{"204":{}}}`, ``, "E_REF_UNSUPPORTED", "/paths/~1ping/get/parameters/0/$ref"},
		{`{"parameters":[{"$ref":"#/missing"}],"responses":{"204":{}}}`, ``, "E_REF_MISSING", "/paths/~1ping/get/parameters/0/$ref"},
		{`{"parameters":[{"$ref":"#/pool/01"}],"responses":{"204":{}}}`, `,"pool":[{},{}]`, "E_SCHEMA_INVALID", "/paths/~1ping/get/parameters/0/$ref"},
		{`{"parameters":[{"$ref":"#/bad~2name"}],"responses":{"204":{}}}`, ``, "E_SCHEMA_INVALID", "/paths/~1ping/get/parameters/0/$ref"},
		{`{"parameters":[{"$ref":"#/pool/0"}],"responses":{"204":{}}}`, `,"pool":[false]`, "E_SCHEMA_INVALID", "/pool/0"},
		{`{"parameters":[{"$ref":"#/pool/0","required":true}],"responses":{"204":{}}}`, `,"pool":[{}]`, "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters/0/required"},
		{`{"parameters":[{"$ref":"#/pool/0"}],"responses":{"204":{}}}`, `,"pool":[{"$ref":"#/pool/1"},{"$ref":"#/pool/0"}]`, "E_REF_CYCLE", "/pool/0/$ref"},
		{`{"parameters":[{"$ref":"#/pool/0"},{"$ref":"#/pool/0"}],"responses":{"204":{}}}`, `,"pool":[{"name":"v","in":"query","schema":{"type":"string"}}]`, "E_SCHEMA_INVALID", "/paths/~1ping/get/parameters/1"},
		{`{"requestBody":{"$ref":"#/missing"},"responses":{"204":{}}}`, ``, "E_REF_MISSING", "/paths/~1ping/get/requestBody/$ref"},
		{`{"responses":{"200":{"$ref":"#/missing"}}}`, ``, "E_REF_MISSING", "/paths/~1ping/get/responses/200/$ref"},
		{`{"parameters":[{"$ref":"#/pool/0"}],"responses":{"204":{}}}`, `,"pool":[{"name":"v","in":"query","schema":{"type":"string","default":"secret-marker"}}]`, "E_SCHEMA_UNSUPPORTED", "/pool/0/schema/default"},
	} {
		input := []byte(`{"openapi":"3.1.0","paths":{"/ping":{"get":` + tc.op + `}}` + tc.extra + `}`)
		failure(t, input, tc.code, tc.pointer)
	}
}

func TestHTTPWarningOrderIsIndependentOfObjectOrder(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{"/z":{"get":{"parameters":[{"name":"p","in":"query","schema":{"type":"string","const":"z"}}],"requestBody":{"content":{"application/z+json":{"schema":{"type":"string","const":"z"}},"application/json":{"schema":{"type":"string","const":"a"}}}},"responses":{"202":{"content":{"application/json":{"schema":{"type":"string","const":"later"}}}},"200":{"content":{"application/json":{"schema":{"type":"string","const":"first"}}}}}}},"/a":{"get":{"parameters":[{"name":"p","in":"query","schema":{"type":"string","const":"a"}}],"responses":{"204":{}}}}}}`)
	var decoded any
	if err := json.Unmarshal(input, &decoded); err != nil {
		t.Fatal(err)
	}
	reordered, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report(t, input, ""), report(t, reordered, "")) {
		t.Fatal("unordered HTTP object keys changed warnings or artifacts")
	}
}

func TestHTTPReferenceDepth(t *testing.T) {
	for _, n := range []int{48, 49} {
		var entries []string
		for i := 0; i < n-1; i++ {
			entries = append(entries, fmt.Sprintf(`{"$ref":"#/pool/%d"}`, i+1))
		}
		entries = append(entries, `{"name":"v","in":"query","schema":{"type":"string"}}`)
		input := []byte(`{"openapi":"3.1.0","paths":{"/ping":{"get":{"parameters":[{"$ref":"#/pool/0"}],"responses":{"204":{}}}}},"pool":[` + strings.Join(entries, ",") + `]}`)
		if n == 48 {
			report(t, input, "")
		} else {
			failure(t, input, "E_SCHEMA_DEPTH", "/pool/47/$ref")
		}
	}
}
