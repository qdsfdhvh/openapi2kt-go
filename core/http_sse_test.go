package core_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func sseDoc(schema string) []byte {
	return httpDoc(`{"responses":{"200":{"content":{"text/event-stream":{"schema":` + schema + `}}}}}`)
}

func TestHTTPServerSentEvents(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "sse.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	for name, tokens := range map[string][]string{
		"StreamApi":      {"public fun events(", "kotlinx.coroutines.flow.Flow<org.example.models.api.models.EventsEventDto>", "client.sse(", "lastEventId: kotlin.String? = null", "public suspend fun ping(", "json: kotlinx.serialization.json.Json"},
		"EventsEventDto": {"public sealed class EventsEventDto", "public data class Update(", "public data class Done(", "public abstract val id: kotlin.String?", "org.example.models.models.PayloadDto", "data: org.example.models.api.models.EventsDataInlineDto", "kotlin.collections.List<org.example.models.api.models.EventsDataRowsDtoInner>", "kotlin.collections.Map<kotlin.String, org.example.models.api.models.EventsDataLookupDtoValue>"},
		"ApiClient":      {"ignoreUnknownKeys = true", "StreamApi(httpClient, basePath, json)"},
	} {
		s := source(t, r, name)
		for _, token := range tokens {
			if !strings.Contains(s, token) {
				t.Fatal(name, token, s)
			}
		}
	}
	for _, file := range r.Artifacts.Files {
		for _, line := range strings.Split(file.Content, "\n") {
			if strings.HasPrefix(line, "import ") && strings.HasSuffix(line, ".*") {
				t.Fatal("wildcard import", file.Path)
			}
		}
	}
	if !reflect.DeepEqual(r, report(t, input, "Dto")) {
		t.Fatal("nondeterministic SSE")
	}
	req := request(input, "Dto")
	req.Options.ExcludeAPIs = []string{"StreamApi"}
	excluded, err := core.Generate(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range excluded.Files {
		if strings.Contains(file.Path, "/api/") || strings.Contains(file.Path, "/Patch.kt") || strings.Contains(file.Content, "io.ktor.client.plugins.sse") {
			t.Fatal("excluded SSE leaked", file.Path)
		}
	}
}

func TestSSEGuardsAndSelection(t *testing.T) {
	prefix := "/paths/~1ping/get/responses/200/content/text~1event-stream/schema"
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"type":"string"}`, "E_OPERATIONS_UNSUPPORTED", ""},
		{`false`, "E_SCHEMA_INVALID", ""},
		{`{"type":"object","properties":{},"additionalProperties":false}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties"},
		{`{"type":"object","properties":{"extra":{"type":"string"}}}`, "E_OPERATIONS_UNSUPPORTED", "/properties/extra"},
		{`{"type":"object","properties":{"event":{"type":"string","const":""}}}`, "E_SCHEMA_INVALID", "/properties/event/const"},
		{`{"type":"object","properties":{"event":{"type":"string","const":"a\nb"}}}`, "E_SCHEMA_INVALID", "/properties/event/const"},
		{`{"type":"object","properties":{"id":{"type":"integer"}}}`, "E_OPERATIONS_UNSUPPORTED", "/properties/id"},
		{`{"type":"object","properties":{"data":{"type":"string","pattern":"x"}}}`, "E_SCHEMA_UNSUPPORTED", "/properties/data/pattern"},
		{`{"type":"object","required":["data","data"],"properties":{"data":{"type":"string"}}}`, "E_SCHEMA_INVALID", "/required/1"},
		{`{"type":"object","required":["missing"],"properties":{}}`, "E_SCHEMA_INVALID", "/required/0"},
		{`{"type":["object","null"],"properties":{}}`, "E_OPERATIONS_UNSUPPORTED", ""},
		{`{"anyOf":[{"type":"object","properties":{}},{"type":"null"}]}`, "E_OPERATIONS_UNSUPPORTED", ""},
		{`{"oneOf":[{"type":"object","properties":{}},{"type":"object","properties":{}}]}`, "E_SCHEMA_INVALID", "/oneOf/1"},
	} {
		failure(t, sseDoc(tc.schema), tc.code, prefix+tc.tail)
	}
	failure(t, httpDoc(`{"responses":{"201":{"content":{"text/event-stream":{"schema":{"type":"object","properties":{}}}}}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/responses/201/content/text~1event-stream")
	// JSON wins when both are valid; unused SSE declarations/helpers are not emitted.
	response := `{"application/json":{"schema":{"type":"integer"}},"text/event-stream":{"schema":{"type":"object","properties":{}}}}`
	r := report(t, httpDoc(`{"responses":{"200":{"content":`+response+`}}}`), "")
	if len(r.Artifacts.Files) != 2 || strings.Contains(source(t, r, "DefaultApi"), "client.sse") || strings.Contains(source(t, r, "ApiClient"), "ignoreUnknownKeys") {
		t.Fatal(r)
	}
	// Validation cannot be bypassed by choosing JSON or excluding the API.
	bad := httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"integer"}},"text/event-stream":{"schema":{"type":"object","properties":{"data":{"type":"string","pattern":"x"}}}}}}}}`)
	req := request(bad, "")
	req.Options.ExcludeAPIs = []string{"DefaultApi"}
	rpt, err := core.GenerateWithDiagnostics(req)
	if err == nil || len(rpt.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" {
		t.Fatal(rpt, err)
	}
}

func TestSSEFallbackAndEventLimits(t *testing.T) {
	envelope := `{"type":"object","properties":{"event":{"type":"string","const":"item"},"data":{"type":"object","properties":{"value":{"type":"string"}}}}}`
	operation := func(param string) string {
		return `{"parameters":[{"name":"` + param + `","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"content":{"text/event-stream":{"schema":` + envelope + `}}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","paths":{"/feed/{a}":{"get":` + operation("a") + `},"/feed/{b}":{"get":` + operation("b") + `}}}`)
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	for _, token := range []string{"public fun getFeed(", "public fun getFeed2(", "GetFeed2EventDto", "GetFeed2DataItemDto"} {
		if !strings.Contains(api, token) {
			t.Fatal(token, api)
		}
	}
	for _, n := range []int{240, 241} {
		branches := make([]string, n)
		for i := range n {
			branches[i] = fmt.Sprintf(`{"type":"object","properties":{"event":{"type":"string","const":"v%d"}}}`, i)
		}
		input := sseDoc(`{"oneOf":[` + strings.Join(branches, ",") + `]}`)
		if n == 240 {
			report(t, input, "")
		} else {
			failure(t, input, "E_UNION_LIMIT", "/paths/~1ping/get/responses/200/content/text~1event-stream/schema/oneOf")
		}
	}
}

func TestSSECursorNamingAndSlotLimits(t *testing.T) {
	envelope := `{"type":"object","properties":{}}`
	for _, params := range []string{
		`[{"name":"lastEventId","in":"query","schema":{"type":"string"}}]`,
		`[{"name":"lastEventId","in":"query","schema":{"type":"string"}},{"name":"lastEventIdQuery","in":"query","schema":{"type":"string"}},{"name":"last-event-id","in":"header","required":true,"schema":{"type":"string"}}]`,
	} {
		r := report(t, httpDoc(`{"parameters":`+params+`,"responses":{"200":{"content":{"text/event-stream":{"schema":`+envelope+`}}}}}`), "")
		content := source(t, r, "DefaultApi")
		if !strings.Contains(content, "lastEventId: kotlin.String") || strings.Count(content, "header(\"Last-Event-ID\"")+strings.Count(content, "header(\"last-event-id\"") != 1 {
			t.Fatal(content)
		}
	}
	for _, n := range []int{119, 120} {
		var params []string
		for i := range n {
			params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"type":"integer","format":"int64"}}`, i))
		}
		input := httpDoc(`{"parameters":[` + strings.Join(params, ",") + `],"responses":{"200":{"content":{"text/event-stream":{"schema":` + envelope + `}}}}}`)
		if n == 119 {
			report(t, input, "")
		} else {
			failure(t, input, "E_OPERATION_LIMIT", "/paths/~1ping/get/parameters")
		}
	}
	failure(t, httpDoc(`{"parameters":[{"name":"Last-Event-ID","in":"header","schema":{"type":"integer"}}],"responses":{"200":{"content":{"text/event-stream":{"schema":`+envelope+`}}}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters")
	failure(t, httpDoc(`{"parameters":[{"name":"Last-Event-ID","in":"header","schema":{"type":"string"}},{"name":"last-event-id","in":"header","schema":{"type":"string"}}],"responses":{"200":{"content":{"text/event-stream":{"schema":`+envelope+`}}}}}`), "E_SCHEMA_INVALID", "/paths/~1ping/get/parameters")
}
