package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCollectionCountsGuards(t *testing.T) {
	for _, tc := range []struct{ shape, key, code string }{
		{`{"type":"array","items":{"type":"string"},"minItems":-1}`, "minItems", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string"},"maxItems":true}`, "maxItems", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string"},"minItems":2,"maxItems":1}`, "maxItems", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string"},"minProperties":1}`, "minProperties", "E_SCHEMA_UNSUPPORTED"},
		{`{"type":"object","additionalProperties":{"type":"string"},"minProperties":0.5}`, "minProperties", "E_SCHEMA_INVALID"},
		{`{"type":"object","additionalProperties":true,"maxProperties":2147483648}`, "maxProperties", "E_SCHEMA_INVALID"},
		{`{"type":"object","additionalProperties":true,"minProperties":2,"maxProperties":1}`, "maxProperties", "E_SCHEMA_INVALID"},
		{`{"type":"object","additionalProperties":true,"minItems":1}`, "minItems", "E_SCHEMA_UNSUPPORTED"},
		{`{"type":"string","minItems":1}`, "minItems", "E_SCHEMA_UNSUPPORTED"},
		{`{"type":"array","items":{"type":"number"},"uniqueItems":true}`, "uniqueItems", "E_SCHEMA_UNSUPPORTED"},
	} {
		aliasFailure(t, fieldDoc(tc.shape), tc.code, "/components/schemas/Record/properties/value/"+tc.key)
	}
	for _, shape := range []string{`{"type":"array","items":{"type":"string"},"minItems":1}`, `{"type":"object","additionalProperties":true,"minProperties":1}`} {
		if _, err := core.Generate(request(fieldDoc(shape), "")); err == nil {
			t.Fatal("defaultfalse accepted counts")
		}
	}
	aliasFailure(t, schemas(`{"Record":{"type":"object","properties":{"v":{"type":"string"}},"additionalProperties":{"type":"array","items":{"type":"string"},"minItems":1}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/additionalProperties/minItems")
	aliasFailure(t, schemas(`{"Record":{"allOf":[{"type":"object","properties":{"v":{"type":"array","items":{"type":"string"},"minItems":1}}}],"type":"object","properties":{"v":{"type":"array","items":{"type":"string"}}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/v/minItems")
	// Raw parameter objects have no constructor barrier and must not drop counts.
	raw := []byte(`{"openapi":"3.1.0","paths":{"/x":{"get":{"parameters":[{"name":"q","in":"query","schema":{"type":"object","properties":{"rows":{"type":"array","minItems":1,"items":{"type":"string"}}}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, raw, "E_SCHEMA_UNSUPPORTED", "/paths/~1x/get/parameters/0/schema/properties/rows/minItems")
	unused := schemas(`{"Unused":{"type":"object","properties":{"v":{"type":"string"}},"minProperties":1}}`)
	unusedRequest := request(unused, "Dto")
	unusedRequest.Options.Validation = true
	unusedReport, err := core.GenerateWithDiagnostics(unusedRequest)
	if err != nil {
		t.Fatal("unused ordinary DTO must retain counts", err)
	}
	for _, token := range []string{"KeepGeneratedSerializer", "generatedSerializer()", "element.size < 1"} {
		if !strings.Contains(source(t, unusedReport, "UnusedDto"), token) {
			t.Fatal("unused count propagation lost", token)
		}
	}
	unusedRequest.Options.Validation = false
	_, err = core.Generate(unusedRequest)
	if diagnostics, ok := err.(core.Diagnostics); !ok || len(diagnostics) != 1 || diagnostics[0].Code != "E_SCHEMA_UNSUPPORTED" || diagnostics[0].Pointer == nil || *diagnostics[0].Pointer != "/components/schemas/Unused/minProperties" {
		t.Fatal("disabled unused count must retain original keyword Pointer", err)
	}
	// Cap checked-container chains, retaining original container Pointer.
	deep := `{"type":"array","items":{"type":"string"},"minItems":1}`
	for i := 0; i < 16; i++ {
		deep = `{"type":"array","items":` + deep + `}`
	}
	aliasFailure(t, fieldDoc(deep), "E_SCHEMA_DEPTH", "/components/schemas/Record/properties/value"+strings.Repeat("/items", 16))
	// Diagnostic source locations must not change declaration shape identity.
	shape := `{"type":"object","properties":{"v":{"type":"array","minItems":1,"items":{"type":"string"}}}}`
	input := []byte(`{"openapi":"3.1.0","paths":{"/a":{"get":{"tags":["Left"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}},"/b":{"get":{"tags":["Right"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("source changed shape identity", err)
	}
	if !strings.Contains(source(t, r, "SharedResponseDto"), "minItems") {
		t.Fatal("shared declaration lost checks")
	}
	for _, count := range []int{240, 241} {
		var fields []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"array","minItems":1,"items":{"type":"integer"}}`, i))
		}
		rq := request(schemas(`{"Record":{"type":"object","properties":{`+strings.Join(fields, ",")+`}}}`), "")
		rq.Options.Validation = true
		if count == 240 {
			if _, err := core.Generate(rq); err != nil {
				t.Fatal(err)
			}
		} else {
			aliasFailure(t, rq.Document, "E_MODEL_LIMIT", "/components/schemas/Record")
		}
	}
	req := request(fieldDoc(`{"type":"array","items":{"type":"string"}}`), "")
	before := report(t, req.Document, "")
	req.Options.Validation = true
	after, err := core.GenerateWithDiagnostics(req)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("unbounded output changed", err)
	}
}
