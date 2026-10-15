package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

func namespaceCollisionDoc() []byte {
	return httpDoc(`{"operationId":"watch","parameters":[
		{"name":"org","in":"query","schema":{"type":"string"}},
		{"name":"io","in":"query","schema":{"type":"string"}},
		{"name":"kotlinx","in":"query","schema":{"type":"string"}}
	],"responses":{"200":{"content":{"text/event-stream":{"schema":{"type":"object","properties":{
		"event":{"type":"string","const":"tick"},"data":{"type":"string"}},"required":["data"]}}}}}}`)
}

func TestHTTPParameterNamespaceCollisions(t *testing.T) {
	got := report(t, namespaceCollisionDoc(), "")
	api := source(t, got, "DefaultApi")
	for _, want := range []string{
		"import io.ktor.http.HttpMethod as OpenapiKtHttpMethodImport",
		"import kotlinx.coroutines.flow.flow as OpenapiKtFlowImport",
		"import org.example.models.api.models.WatchEvent as OpenapiKtSSEEventImport1",
		"= OpenapiKtFlowImport {",
		"this.method = OpenapiKtHttpMethodImport(\"GET\")",
		"OpenapiKtSSEEventImport1.Tick(",
	} {
		if !strings.Contains(api, want) {
			t.Fatalf("missing %q:\n%s", want, api)
		}
	}
	for _, name := range []string{"org", "io", "kotlinx"} {
		if !strings.Contains(api, name+": kotlin.String? = null") || !strings.Contains(api, `parameter("`+name+`", it)`) {
			t.Fatal("parameter API or wire name changed", api)
		}
	}
}

func TestSSEGeneratedLocalsAndBodyDoNotShadowEventPackages(t *testing.T) {
	for _, prefix := range []string{"frame", "value", "requestBuilder", "input", "client", "basePath", "json"} {
		input := namespaceCollisionDoc()
		if prefix == "input" {
			input = []byte(strings.Replace(string(input), `"operationId":"watch"`, `"operationId":"watch","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"string"}}}}`, 1))
		}
		rq := request(input, "")
		rq.Options.PackageName = prefix + ".example"
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "DefaultApi"), "OpenapiKtSSEEventImport1.Tick(") {
			t.Fatal(prefix, source(t, got, "DefaultApi"))
		}
	}
}

func TestHTTPNamespaceAliasesReserveMethodNames(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(namespaceCollisionDoc(), &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]any)
	for _, name := range []string{"OpenapiKtHttpMethodImport", "OpenapiKtFlowImport", "OpenapiKtSSEEventImport1"} {
		paths["/reserved/"+name] = map[string]any{"get": map[string]any{
			"operationId": name, "responses": map[string]any{"204": map[string]any{}},
		}}
	}
	encoded, _ := json.Marshal(doc)
	api := source(t, report(t, encoded, ""), "DefaultApi")
	for _, want := range []string{
		"import io.ktor.http.HttpMethod as OpenapiKtHttpMethodImport2",
		"import kotlinx.coroutines.flow.flow as OpenapiKtFlowImport2",
		"import org.example.models.api.models.WatchEvent as OpenapiKtSSEEventImport12",
	} {
		if !strings.Contains(api, want) {
			t.Fatalf("alias stole a method name: missing %q", want)
		}
	}
	for _, name := range []string{"OpenapiKtHttpMethodImport", "OpenapiKtFlowImport", "OpenapiKtSSEEventImport1"} {
		if !strings.Contains(api, "fun "+name+"(") {
			t.Fatal("method was renamed for an internal import", name)
		}
	}
}

func TestHTTPNamespaceBodyContentType(t *testing.T) {
	input := []byte(strings.Replace(string(namespaceCollisionDoc()), `"operationId":"watch"`, `"operationId":"watch","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"string"}}}}`, 1))
	got := report(t, input, "")
	api := source(t, got, "DefaultApi")
	if !strings.Contains(api, "import io.ktor.http.ContentType as OpenapiKtContentTypeImport") ||
		!strings.Contains(api, "contentType(OpenapiKtContentTypeImport.Application.Json)") {
		t.Fatal("body content type remains captured by io", api)
	}
}

func TestHTTPNamespaceAggregateMembers(t *testing.T) {
	for _, prefix := range []string{"httpClient", "basePath", "json", "default"} {
		rq := request(namespaceCollisionDoc(), "")
		rq.Options.PackageName = prefix + ".example"
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		api := source(t, got, "ApiClient")
		if !strings.Contains(api, "import "+prefix+".example.api.DefaultApi as OpenapiKtAPIImport1") ||
			!strings.Contains(api, "by lazy { OpenapiKtAPIImport1(") {
			t.Fatal("aggregate member captures its API constructor", prefix, api)
		}
	}
}

func TestHTTPNamespaceAliasesOnlyWhenNeeded(t *testing.T) {
	input := httpDoc(`{"responses":{"204":{}}}`)
	got := report(t, input, "")
	api := source(t, got, "DefaultApi")
	if strings.Contains(api, "OpenapiKtHttpMethodImport") || !strings.Contains(api, `io.ktor.http.HttpMethod("GET")`) {
		t.Fatal("ordinary method representation changed", api)
	}
}
