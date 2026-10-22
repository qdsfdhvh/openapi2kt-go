package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHTTPAdditionalParameterStyles(t *testing.T) {
	for _, style := range []string{"label", "matrix"} {
		for _, schema := range []string{`{"type":"string"}`, `{"type":"array","items":{"type":"string"}}`, `{"type":"object","additionalProperties":{"type":"string"}}`} {
			for _, explode := range []string{"true", "false"} {
				input := []byte(`{"openapi":"3.1.0","paths":{"/values/{value}":{"get":{"parameters":[{"name":"value","in":"path","required":true,"style":"` + style + `","explode":` + explode + `,"schema":` + schema + `}],"responses":{"204":{}}}}}}`)
				r, err := core.GenerateWithDiagnostics(request(input, ""))
				if err != nil {
					t.Fatal(style, schema, explode, err)
				}
				text := source(t, r, "DefaultApi")
				if !strings.Contains(text, "encodeURLPathPart()") {
					t.Fatal(text)
				}
				again, err := core.GenerateWithDiagnostics(request(input, ""))
				if err != nil || !reflect.DeepEqual(r, again) {
					t.Fatal("parameter style output drift", err)
				}
			}
		}
	}
	input := httpDoc(`{"parameters":[{"name":"filter","in":"query","style":"form","explode":false,"schema":{"type":"object","additionalProperties":{"type":"string"}}}],"responses":{"204":{}}}`)
	if _, err := core.Generate(request(input, "")); err != nil {
		t.Fatal("form dictionary comma encoding should generate", err)
	}
}

func TestHTTPAdditionalParameterStylesKeepUnsupportedRoles(t *testing.T) {
	for _, style := range []string{"label", "matrix"} {
		failure(t, httpDoc(`{"parameters":[{"name":"q","in":"query","style":"`+style+`","schema":{"type":"string"}}],"responses":{"204":{}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters/0/style")
		failure(t, httpDoc(`{"parameters":[{"name":"h","in":"header","style":"`+style+`","schema":{"type":"string"}}],"responses":{"204":{}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters/0/style")
		input := []byte(`{"openapi":"3.1.0","paths":{"/values/{value}":{"get":{"parameters":[{"name":"value","in":"path","required":true,"style":"` + style + `","schema":{"type":"object","additionalProperties":{"type":"array","items":{"type":"string"}}}}],"responses":{"204":{}}}}}}`)
		failure(t, input, "E_OPERATIONS_UNSUPPORTED", "/paths/~1values~1{value}/get/parameters/0/schema")
	}
}

func TestHTTPDeepObjectTypedDictionaries(t *testing.T) {
	for _, leaf := range []string{`{"type":"string"}`, `{"type":"integer"}`, `{"type":"boolean","nullable":true}`, `{"$ref":"#/components/schemas/Choice","nullable":true}`} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Choice":{"type":"string","enum":["ready/value","done"]}}},"paths":{"/ping":{"get":{"parameters":[{"name":"filter","in":"query","style":"deepObject","schema":{"additionalProperties":` + leaf + `}}],"responses":{"204":{}}}}}}`)
		rq := request(input, "Dto")
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("typed dictionary deepObject should generate", leaf, err)
		}
		text := source(t, r, "DefaultApi")
		if !strings.Contains(text, `"filter[" + key + "]"`) || strings.Contains(text, "FilterDto") {
			t.Fatal("dictionary gained fixed-field filter or lost bracket keys", text)
		}
		if strings.Contains(leaf, `"$ref"`) && !strings.Contains(text, ".serialName") {
			t.Fatal("dictionary enum wire spelling lost", text)
		}
		for _, w := range r.Warnings {
			if w.Code == "W_HTTP_PARAMETER_PROJECTION" {
				t.Fatal("scalar dictionary must not use structured projection", w)
			}
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(r, again) {
			t.Fatal("deepObject dictionary output drift", err)
		}
	}
}

func TestHTTPDeepObjectDictionariesExcludeStructuredValues(t *testing.T) {
	for _, schema := range []string{`{"additionalProperties":true}`, `{"additionalProperties":{"type":"array","items":{"type":"string"}}}`, `{"additionalProperties":{"properties":{"id":{"type":"string"}}}}`} {
		if r, err := core.GenerateWithDiagnostics(request(httpDoc(`{"parameters":[{"name":"filter","in":"query","style":"deepObject","schema":`+schema+`}],"responses":{"204":{}}}`), "Dto")); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("structured/open dictionary acquired recursive encoding", schema, r, err)
		}
	}
	failure(t, httpDoc(`{"parameters":[{"name":"filter","in":"query","style":"deepObject","explode":false,"schema":{"additionalProperties":{"type":"string"}}}],"responses":{"204":{}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters/0/explode")
	failure(t, httpDoc(`{"parameters":[{"name":"filter","in":"header","style":"deepObject","schema":{"additionalProperties":{"type":"string"}}}],"responses":{"204":{}}}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1ping/get/parameters/0/style")
}

func TestHTTPDeepObjectDictionaryAliasesAndChecks(t *testing.T) {
	for _, intent := range []string{"", `,"nullable":false`, `,"nullable":true`} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Codes":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Code","nullable":true}},"Alias":{"$ref":"#/components/schemas/Codes","nullable":true},"Code":{"type":"string","x-kotlin-value-class":true}}},"paths":{"/ping":{"get":{"parameters":[{"name":"filter","in":"query","required":true,"style":"deepObject","explode":true,"schema":{"$ref":"#/components/schemas/Alias"` + intent + `}}],"responses":{"204":{}}}}}}`)
		r := report(t, input, "Dto")
		text := source(t, r, "DefaultApi")
		if !strings.Contains(text, "value?.rawValue") || strings.Contains(text, "FilterDto") || strings.Contains(text, "CodesDto? = null") || strings.Contains(text, "filter?.forEach") != (intent != `,"nullable":false`) {
			t.Fatal("alias source/nullability/value-class wire lost", intent, text)
		}
	}
	input := httpDoc(`{"parameters":[{"name":"filter","in":"query","style":"deepObject","schema":{"additionalProperties":{"type":"string","minLength":2},"maxProperties":4}}],"responses":{"204":{}}}`)
	rq := request(input, "Dto")
	if _, err := core.GenerateWithDiagnostics(rq); err == nil {
		t.Fatal("validation-off silently discarded dictionary checks")
	}
	rq.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !strings.Contains(source(t, r, "DefaultApi"), "minLength") || !strings.Contains(source(t, r, "DefaultApi"), "maxProperties") {
		t.Fatal("typed dictionary checks lost", r, err)
	}
}

func TestHTTPDeepObjectDictionarySourceDepth(t *testing.T) {
	for _, count := range []int{46, 47} {
		components := []string{`"Values":{"additionalProperties":{"type":"string"}}`}
		for i := range count {
			ref := "Values"
			if i > 0 {
				ref = fmt.Sprintf("A%02d", i-1)
			}
			components = append(components, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, ref))
		}
		input := []byte(fmt.Sprintf(`{"openapi":"3.1.0","components":{"schemas":{%s}},"paths":{"/ping":{"get":{"parameters":[{"name":"filter","in":"query","style":"deepObject","schema":{"$ref":"#/components/schemas/A%02d","nullable":false}}],"responses":{"204":{}}}}}}`, strings.Join(components, ","), count-1))
		r, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 46 && err != nil || count == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0) {
			t.Fatal("wire backing inspection erased source depth", count, r, err)
		}
	}
}
