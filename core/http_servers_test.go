package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func serverDocument(servers, item string) []byte {
	return []byte(`{"openapi":"3.1.0","servers":` + servers + `,"paths":{"/ping":` + item + `}}`)
}

func TestHTTPServerVariableDefaults(t *testing.T) {
	input := serverDocument(`[{"url":"https://{host}/{version}/{version}","variables":{"host":{"default":"api.example.test","enum":["api.example.test","other.example.test"]},"version":{"default":"v2","description":"Selected version"}}}]`, `{"get":{"responses":{"204":{}}}}`)
	got := report(t, input, "")
	if !strings.Contains(source(t, got, "ApiClient"), `basePath: kotlin.String = "https://api.example.test/v2/v2/"`) {
		t.Fatal(source(t, got, "ApiClient"))
	}
	if strings.Contains(source(t, got, "DefaultApi"), "basePath: kotlin.String?") {
		t.Fatal("root-only defaults changed the established constructor")
	}
	var reordered any
	if err := json.Unmarshal(input, &reordered); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(reordered)
	if !reflect.DeepEqual(got, report(t, encoded, "")) {
		t.Fatal("server variable map order changed output")
	}
}

func serverDefaultReference(t testing.TB, got core.GenerationReport, value string) string {
	t.Helper()
	for _, file := range got.Artifacts.Files {
		text := string(file.Content)
		if !strings.Contains(text, "val basePath: kotlin.String = "+fmt.Sprintf("%q", value)) {
			continue
		}
		start := strings.Index(text, "internal object ")
		if start >= 0 {
			name := strings.SplitN(text[start+len("internal object "):], " {", 2)[0]
			return name + ".basePath"
		}
	}
	t.Fatalf("no shared server default for %q", value)
	return ""
}

func TestHTTPServerScopePrecedence(t *testing.T) {
	root := `[{"url":"https://root.example.test/v1"}]`
	for _, tc := range []struct {
		name, item, base string
	}{
		{"path", `{"servers":[{"url":"https://path.example.test/p"}],"get":{"responses":{"204":{}}}}`, "https://path.example.test/p/"},
		{"operation", `{"servers":[{"url":"https://path.example.test/p"}],"get":{"servers":[{"url":"https://op.example.test/o"}],"responses":{"204":{}}}}`, "https://op.example.test/o/"},
		{"emptyOperation", `{"servers":[{"url":"https://path.example.test/p"}],"get":{"servers":[],"responses":{"204":{}}}}`, ""},
		{"emptyPath", `{"servers":[],"get":{"responses":{"204":{}}}}`, ""},
		{"relative", `{"get":{"servers":[{"url":"/relative/"}],"responses":{"204":{}}}}`, "relative//"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := report(t, serverDocument(root, tc.item), "")
			api := source(t, got, "DefaultApi")
			want := "(this@DefaultApi.basePath ?: " + serverDefaultReference(t, got, tc.base) + `) + "ping"`
			if !strings.Contains(api, "basePath: kotlin.String?") || !strings.Contains(api, want) {
				t.Fatal(api)
			}
			if !strings.Contains(source(t, got, "ApiClient"), "basePath: kotlin.String? = null") {
				t.Fatal(source(t, got, "ApiClient"))
			}
		})
	}
}

func TestHTTPServerInheritedDefaultAndExclusion(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","servers":[{"url":"https://root.example.test/root"}],"paths":{"/a":{"get":{"tags":["Root"],"responses":{"204":{}}}},"/b":{"get":{"tags":["Local"],"servers":[{"url":"https://local.example.test/local"}],"responses":{"204":{}}}}}}`)
	got := report(t, input, "")
	if !strings.Contains(source(t, got, "RootApi"), "basePath ?: "+serverDefaultReference(t, got, "https://root.example.test/root/")) {
		t.Fatal(source(t, got, "RootApi"))
	}
	rq := request(input, "")
	rq.Options.ExcludeAPIs = []string{"LocalApi"}
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(source(t, got, "RootApi"), "basePath: kotlin.String?") || !strings.Contains(source(t, got, "ApiClient"), `basePath: kotlin.String = "https://root.example.test/root/"`) {
		t.Fatal("excluded scope changed retained constructors")
	}
}

func TestHTTPServerClientNameKeepsSharedAPIFiles(t *testing.T) {
	input := serverDocument(`[{"url":"https://root.example.test"}]`, `{"get":{"servers":[{"url":"https://local.example.test"}],"responses":{"204":{}}}}`)
	before := report(t, input, "")
	rq := request(input, "")
	rq.Options.APIClientName = "AlternateClient"
	after, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if source(t, before, "DefaultApi") != source(t, after, "DefaultApi") {
		t.Fatal("client naming changed shared API dependency bytes")
	}
	for _, file := range before.Artifacts.Files {
		if strings.Contains(file.Path, "/ServerDefault_") {
			found := false
			for _, alternate := range after.Artifacts.Files {
				found = found || reflect.DeepEqual(file, alternate)
			}
			if !found {
				t.Fatal("client naming changed shared URL helper")
			}
		}
	}
}

func TestHTTPServerExpansionBudget(t *testing.T) {
	value := strings.Repeat("a", 3*1024*1024)
	servers := `[{"url":"https://{part}/{part}/{part}","variables":{"part":{"default":"` + value + `"}}}]`
	failure(t, serverDocument(servers, `{"get":{"responses":{"204":{}}}}`), "E_INPUT_LIMIT", "/servers/0/url")
}

func TestHTTPServerAggregateExpansionBudget(t *testing.T) {
	value := strings.Repeat("a", 3*1024*1024)
	servers := `[{"url":"https://{part}/{part}","variables":{"part":{"default":"` + value + `"}}},{"url":"https://{part}","variables":{"part":{"default":"` + value[:2*1024*1024] + `"}}}]`
	failure(t, serverDocument(servers, `{"get":{"responses":{"204":{}}}}`), "E_INPUT_LIMIT", "/servers/1/url")
}

func TestHTTPServerSharedDefaultsAreEmittedOnce(t *testing.T) {
	base := "https://example.test/" + strings.Repeat("a", 40000)
	paths := map[string]any{}
	for i := range 80 {
		paths[fmt.Sprintf("/p%d", i)] = map[string]any{"get": map[string]any{"tags": []string{fmt.Sprintf("Group%d", i)}, "responses": map[string]any{"204": map[string]any{}}}}
	}
	paths["/local"] = map[string]any{"get": map[string]any{"servers": []any{}, "responses": map[string]any{"204": map[string]any{}}}}
	encoded, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "servers": []any{map[string]any{"url": base}}, "paths": paths})
	got := report(t, encoded, "")
	count := 0
	for _, file := range got.Artifacts.Files {
		count += strings.Count(string(file.Content), base)
	}
	if count != 1 {
		t.Fatalf("shared root literal emitted %d times", count)
	}
}

func TestHTTPServerEscapedVariableSource(t *testing.T) {
	servers := `[{"url":"https://{a/~}","variables":{"a/~":{"default":"api.example.test"}}}]`
	got := report(t, serverDocument(servers, `{"get":{"responses":{"204":{}}}}`), "")
	if !strings.Contains(source(t, got, "ApiClient"), `"https://api.example.test/"`) {
		t.Fatal(source(t, got, "ApiClient"))
	}
	failure(t, serverDocument(strings.Replace(servers, `"api.example.test"`, `42`, 1), `{"get":{"responses":{"204":{}}}}`), "E_SCHEMA_INVALID", "/servers/0/variables/a~1~0/default")
}

func TestHTTPServerGrammarAndDiscardedSources(t *testing.T) {
	for _, tc := range []struct {
		servers, code, pointer string
	}{
		{`{}`, "E_SCHEMA_INVALID", "/servers"},
		{`[{"url":"https://example.test","variables":[]}]`, "E_SCHEMA_INVALID", "/servers/0/variables"},
		{`[{"url":"https://{host}","variables":{"host":{}}}]`, "E_SCHEMA_INVALID", "/servers/0/variables/host/default"},
		{`[{"url":"https://{host}","variables":{"host":{"default":42}}}]`, "E_SCHEMA_INVALID", "/servers/0/variables/host/default"},
		{`[{"url":"https://{host}","variables":{"host":{"default":"a","enum":[]}}}]`, "E_SCHEMA_INVALID", "/servers/0/variables/host/enum"},
		{`[{"url":"https://{host}","variables":{"host":{"default":"a","enum":["b"]}}}]`, "E_SCHEMA_INVALID", "/servers/0/variables/host/default"},
		{`[{"url":"https://{host}","variables":{"host":{"default":"a","enum":[1]}}}]`, "E_SCHEMA_INVALID", "/servers/0/variables/host/enum/0"},
		{`[{"url":"https://{host}","variables":{"host":{"default":"{nested}"}}}]`, "E_OPERATIONS_UNSUPPORTED", "/servers/0/variables/host/default"},
		{`[{"url":"https://{host}","variables":{"host":{"default":"a\nb"}}}]`, "E_OPERATIONS_UNSUPPORTED", "/servers/0/variables/host/default"},
		{`[{"url":"https://{missing}","variables":{}}]`, "E_OPERATIONS_UNSUPPORTED", "/servers/0/url"},
		{`[{"url":"https://example.test/}"}]`, "E_OPERATIONS_UNSUPPORTED", "/servers/0/url"},
		{`[{"url":"https://example.test/{"}]`, "E_OPERATIONS_UNSUPPORTED", "/servers/0/url"},
		{`[{"url":"https://example.test"},{"url":"https://{host}","variables":{"host":{"default":false}}}]`, "E_SCHEMA_INVALID", "/servers/1/variables/host/default"},
		{`[{"url":"https://example.test","variables":{"unused":{"default":"safe","unknown":true}}}]`, "E_OPERATIONS_UNSUPPORTED", "/servers/0/variables/unused/unknown"},
	} {
		t.Run(tc.pointer+tc.servers, func(t *testing.T) {
			failure(t, serverDocument(tc.servers, `{"get":{"responses":{"204":{}}}}`), tc.code, tc.pointer)
		})
	}
	for _, item := range []string{
		`{"servers":[{"url":"https://{missing}"}],"get":{"servers":[{"url":"https://valid.example.test"}],"responses":{"204":{}}}}`,
		`{"get":{"servers":[{"url":"https://{missing}"}],"responses":{"204":{}}}}`,
	} {
		rq := request(serverDocument(`[]`, item), "")
		rq.Options.ExcludeAPIs = []string{"DefaultApi"}
		got, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("overridden or excluded invalid server was ignored", err)
		}
	}
}
