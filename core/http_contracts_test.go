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

func httpDoc(operation string) []byte {
	return []byte(`{"openapi":"3.1.0","paths":{"/ping":{"get":` + operation + `}}}`)
}

func TestHTTPBasicClientContract(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "basic.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 6 {
		t.Fatalf("files: %v", r.Artifacts.Files)
	}
	api := source(t, r, "WidgetsApi")
	for _, want := range []string{"suspend fun getItem(", "id: kotlin.String", "limit: kotlin.Long? = null", "state: org.example.models.models.StateDto? = null", "xTraceId: kotlin.String? = null", "org.example.models.models.WidgetDto", "requestBuilder"} {
		if !strings.Contains(api, want) {
			t.Errorf("missing %q in %s", want, api)
		}
	}
	group := source(t, r, "UserAdminApi")
	if !strings.Contains(group, "fun batchGet(") || !strings.Contains(group, "fun ready(") {
		t.Fatal(group)
	}
	client := source(t, r, "WidgetApiClient")
	if !strings.Contains(client, `"https://api.example.test/v1/"`) || !strings.Contains(client, "by lazy") {
		t.Fatal(client)
	}
	if !reflect.DeepEqual(r, report(t, input, "Dto")) {
		t.Fatal("nondeterministic generation")
	}
}

func TestHTTPOperationOnlyAndUnit(t *testing.T) {
	r := report(t, httpDoc(`{"responses":{"204":{"description":"No body"}}}`), "")
	if len(r.Artifacts.Files) != 2 {
		t.Fatal(r.Artifacts.Files)
	}
	if !strings.Contains(source(t, r, "DefaultApi"), "fun getPing(") {
		t.Fatal(r)
	}
}

func TestHTTPMethodAndParameterCollisions(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{
		"/a/{id}":{"get":{"operationId":"fetch","parameters":[
			{"in":"header","name":"id","schema":{"type":"string"}},
			{"in":"query","name":"idQuery","schema":{"type":"string"}},
			{"in":"query","name":"id","schema":{"type":"string"}},
			{"in":"path","name":"id","required":true,"schema":{"type":"string"}},
			{"in":"query","name":"requestBuilder","schema":{"type":"string"}},
			{"in":"query","name":"when","schema":{"type":"boolean"}}],"responses":{"204":{}}}},
		"/b":{"get":{"operationId":"same","responses":{"204":{}}}},
		"/c":{"get":{"operationId":"same","responses":{"204":{}}}},
		"/d":{"get":{"operationId":"same2","responses":{"204":{}}}},
		"/e":{"get":{"operationId":"same","parameters":[{"in":"query","name":"v","required":true,"schema":{"type":"integer"}}],"responses":{"204":{}}}}
	}}`)
	r := report(t, input, "")
	s := source(t, r, "DefaultApi")
	for _, want := range []string{"id: kotlin.String", "idQuery2: kotlin.String?", "idQuery: kotlin.String?", "idHeader: kotlin.String?", "requestBuilderQuery: kotlin.String?", "`when`: kotlin.Boolean?", "fun same(", "fun same2(", "fun same3("} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
	var reordered any
	if err := json.Unmarshal(input, &reordered); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, report(t, encoded, "")) {
		t.Fatal("unordered object keys changed artifacts")
	}
}

func TestHTTPMediaPriorityAndBasePath(t *testing.T) {
	for _, tc := range []struct{ responses, want string }{
		{`{"200":{},"202":{"content":{"application/example+json":{"schema":{"type":"integer"}}}},"default":{}}`, "kotlin.Int"},
		{`{"2XX":{"content":{"APPLICATION/JSON; charset=utf-8":{"schema":{"type":"boolean"}}}}}`, "kotlin.Boolean"},
		{`{"201":{"content":{"application/json":{"schema":{"type":"string"}},"application/a+json":{"schema":{"type":"boolean"}}}}}`, "kotlin.String"},
		{`{"204":{},"default":{}}`, "kotlin.Unit"},
	} {
		r := report(t, httpDoc(`{"responses":`+tc.responses+`}`), "")
		if !strings.Contains(source(t, r, "DefaultApi"), ").body<"+tc.want+">()") && !strings.Contains(source(t, r, "DefaultApi"), "}.body<"+tc.want+">()") {
			t.Fatal(r)
		}
	}
	for _, tc := range []struct{ server, want string }{{"", ""}, {"/v1", "v1/"}, {"https://example.test/api/", "https://example.test/api//"}} {
		input := []byte(fmt.Sprintf(`{"openapi":"3.1.0","servers":[{"url":%q}],"paths":{"/ping":{"get":{"responses":{"204":{}}}}}}`, tc.server))
		if !strings.Contains(source(t, report(t, input, ""), "ApiClient"), fmt.Sprintf("= %q", tc.want)) {
			t.Fatal(tc)
		}
	}
}

func TestHTTPMalformedDocumentsAndLimits(t *testing.T) {
	for _, tc := range []struct{ input, code, pointer string }{
		{`{"openapi":"3.1.0","paths":{"/bad/{id}":{"get":{"responses":{"204":{}}}}}}`, "E_SCHEMA_INVALID", "/paths/~1bad~1{id}/get/parameters"},
		{`{"openapi":"3.1.0","paths":{"/bad}":{"get":{"responses":{"204":{}}}}}}`, "E_SCHEMA_INVALID", "/paths/~1bad}/get"},
		{`{"openapi":"3.1.0","paths":{"/ping":{"get":{"tags":["foo"],"responses":{"204":{}}},"post":{"tags":["FOO"],"responses":{"204":{}}}}}}`, "E_NAME_COLLISION", "/paths"},
		{`{"openapi":"3.1.0","paths":{"/ping":{"get":{"tags":["base path"],"responses":{"204":{}}},"post":{"tags":["base path api"],"responses":{"204":{}}}}}}`, "E_NAME_COLLISION", "/paths"},
		{`{"openapi":"3.1.0","servers":[{"url":"https://example.test/{version}"}],"paths":{"/ping":{"get":{"responses":{"204":{}}}}}}`, "E_OPERATIONS_UNSUPPORTED", "/servers/0/url"},
	} {
		failure(t, []byte(tc.input), tc.code, tc.pointer)
	}
	for _, count := range []int{120, 121} {
		var params []string
		for i := 0; i < count; i++ {
			params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"type":"integer","format":"int64"}}`, i))
		}
		input := httpDoc(`{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}`)
		if count == 120 {
			report(t, input, "")
		} else {
			failure(t, input, "E_OPERATION_LIMIT", "/paths/~1ping/get/parameters")
		}
	}
}

func TestHTTPUnsupportedSemanticsFailClosed(t *testing.T) {
	for _, tc := range []struct{ op, code, tail string }{
		{`{"responses":{"200":{"content":{"text/plain":{"schema":{"type":"string"}}}}}}`, "E_OPERATIONS_UNSUPPORTED", "/responses/200/content"},
		{`{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"payload":{"anyOf":[{"type":"string","pattern":"x"},{"type":"boolean"}]}}}}}},"responses":{"204":{}}}`, "E_SCHEMA_UNSUPPORTED", "/requestBody/content/application~1json/schema/properties/payload/anyOf/0/pattern"},
		{`{"parameters":[{"name":"token","in":"cookie","schema":{"type":"string"}}],"responses":{"204":{}}}`, "E_OPERATIONS_UNSUPPORTED", "/parameters/0/in"},
		{`{"parameters":[{"name":"v","in":"query","schema":{"type":"array","items":{"type":"array"}}}],"responses":{"204":{}}}`, "E_SCHEMA_INVALID", "/parameters/0/schema/items"},
		{`{"responses":{"200":{"content":{"application/json":{"schema":{"not":{}}}}}}}`, "E_SCHEMA_UNSUPPORTED", "/responses/200/content/application~1json/schema/not"},
		{`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"string"}},"text/event-stream":{"schema":{"type":"string"}}}}}}`, "E_OPERATIONS_UNSUPPORTED", "/responses/200/content/text~1event-stream/schema"},
		{`{"callbacks":{},"responses":{"204":{}}}`, "E_OPERATIONS_UNSUPPORTED", "/callbacks"},
		{`{"responses":{}}`, "E_SCHEMA_INVALID", "/responses"},
	} {
		t.Run(tc.tail, func(t *testing.T) { failure(t, httpDoc(tc.op), tc.code, "/paths/~1ping/get"+tc.tail) })
	}
}
