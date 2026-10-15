package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestDTOPropertyCountOrdinaryContexts(t *testing.T) {
	counted := `{"type":["object","null"],"minProperties":0,"maxProperties":2,"properties":{"v":{"type":"string","minLength":1}}}`
	for _, shape := range []string{counted, `{"oneOf":[` + counted + `,{"type":"null"}]}`, `{"type":"array","items":` + counted + `}`, `{"type":"object","additionalProperties":` + counted + `}`} {
		req := request(fieldDoc(shape), "Dto")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err != nil {
			t.Fatal(err)
		}
		text := source(t, r, "RecordDto")
		for _, token := range []string{"KeepGeneratedSerializer", "generatedSerializer()", "element.size < 0", "element.size > 2", "minLength", "init {"} {
			if !strings.Contains(text, token) {
				t.Fatal("lost object/leaf coexistence", token, text)
			}
		}
	}
	counted = `{"type":"object","minProperties":0,"properties":{"v":{"type":"string"}}}`
	for _, input := range [][]byte{
		httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":` + counted + `}}}}}`),
		httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"array","items":` + counted + `}}}}}}`),
		httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"object","additionalProperties":` + counted + `}}}}}}`),
		sseDoc(`{"type":"object","properties":{"data":` + counted + `}}`),
	} {
		req := request(input, "Dto")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, file := range r.Artifacts.Files {
			if strings.Contains(file.Content, "KeepGeneratedSerializer") {
				found = true
			}
		}
		if !found {
			t.Fatal("inline operation context lost nominal counts", r)
		}
	}
	// Pure annotation-only references retain the independent nominal DTO codec.
	input := schemas(`{"Counted":` + counted + `,"Record":{"type":"object","properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Counted"},{"nullable":true}]}}}}`)
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, r, "CountedDto"), "generatedSerializer()") || strings.Contains(source(t, r, "RecordDto"), "KeepGeneratedSerializer") {
		t.Fatal("reference consumer fork")
	}
}

func TestDTOPropertyCountPrivateNameAllocation(t *testing.T) {
	for _, input := range [][]byte{
		schemas(`{"OpenapiKtPropertyCountSerializer":{"type":"object","minProperties":0,"properties":{"v":{"type":"string"}}}}`),
		schemas(`{"Record":{"type":"object","minProperties":0,"properties":{"openapiKtPropertyCountSerializer":{"type":"string"}}}}`),
		schemas(`{"Record":{"type":"object","minProperties":0,"properties":{"openapiKtPropertyCountSerializer":{"type":"object","properties":{"v":{"type":"string"}}}}}}`),
	} {
		req := request(input, "")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err != nil {
			t.Fatal(err)
		}
		text := r.Artifacts.Files[0].Content
		if !strings.Contains(text, "private object OpenapiKtPropertyCountSerializer2") || !strings.Contains(text, ".OpenapiKtPropertyCountSerializer2::class") {
			t.Fatal("private symbol shadowing", text)
		}
	}
}

func TestDTOPropertyCountFilterReferencesFailClosed(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Counted":{"type":"object","minProperties":0,"properties":{"v":{"type":"string"}}},"Holder":{"type":"object","properties":{"counted":{"$ref":"#/components/schemas/Counted"},"self":{"$ref":"#/components/schemas/Holder"}}}}},"paths":{"/x":{"get":{"parameters":[{"name":"q","in":"query","style":"deepObject","schema":{"type":"object","properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Holder"}}}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Counted/minProperties")
	// This guard must not replace the old container/primitive constructor checks.
	input = httpDoc(`{"parameters":[{"name":"q","in":"query","style":"deepObject","schema":{"type":"object","properties":{"rows":{"type":"array","minItems":1,"items":{"type":"integer"}},"bag":{"type":"object","minProperties":1,"additionalProperties":{"type":"integer"}},"v":{"type":"string","minLength":1}}}}],"responses":{"204":{}}}`)
	req := request(input, "")
	req.Options.Validation = true
	if _, err := core.Generate(req); err != nil {
		t.Fatal("old filter constructor checks changed", err)
	}
}
