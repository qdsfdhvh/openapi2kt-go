package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestDTOPropertyCountGrammar(t *testing.T) {
	for _, tc := range []struct{ bound, code, key string }{
		{`"minProperties":-1`, "E_SCHEMA_INVALID", "minProperties"},
		{`"minProperties":1.0`, "E_SCHEMA_INVALID", "minProperties"},
		{`"minProperties":1e0`, "E_SCHEMA_INVALID", "minProperties"},
		{`"minProperties":-0`, "E_SCHEMA_INVALID", "minProperties"},
		{`"minProperties":"1"`, "E_SCHEMA_INVALID", "minProperties"},
		{`"maxProperties":null`, "E_SCHEMA_INVALID", "maxProperties"},
		{`"maxProperties":true`, "E_SCHEMA_INVALID", "maxProperties"},
		{`"maxProperties":2147483648`, "E_SCHEMA_INVALID", "maxProperties"},
		{`"minProperties":2,"maxProperties":1`, "E_SCHEMA_INVALID", "maxProperties"},
		{`"minItems":0`, "E_SCHEMA_UNSUPPORTED", "minItems"},
	} {
		aliasFailure(t, fieldDoc(`{"type":"object","properties":{"v":{"type":"string"}},`+tc.bound+`}`), tc.code, "/components/schemas/Record/properties/value/"+tc.key)
	}
	for _, bounds := range []string{`"minProperties":0`, `"minProperties":2147483647`, `"maxProperties":0`, `"minProperties":0,"maxProperties":0`} {
		input := fieldDoc(`{"type":["object","null"],"properties":{"v":{"type":"string"}},` + bounds + `}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		text := source(t, r, "RecordDto")
		if !strings.Contains(text, "KeepGeneratedSerializer") || strings.Contains(text, "init {") {
			t.Fatal("count must be explicit wire metadata, not constructor check", text)
		}
		rq.Options.Validation = false
		_, err = core.Generate(rq)
		d, ok := err.(core.Diagnostics)
		if !ok || len(d) != 1 || d[0].Code != "E_SCHEMA_UNSUPPORTED" || d[0].Pointer == nil || *d[0].Pointer != "/components/schemas/Record/properties/value/"+strings.Split(strings.TrimPrefix(bounds, `"`), `"`)[0] {
			t.Fatal("defaultfalse pointer", err)
		}
	}
}

func TestDTOPropertyCountFailClosed(t *testing.T) {
	count := `{"type":"object","minProperties":1,"properties":{"v":{"type":"string"}}}`
	for _, tc := range []struct {
		doc            []byte
		pointer, bound string
	}{
		{schemas(`{"Unused":{"allOf":[` + count + `]}}`), "/components/schemas/Unused/allOf/0/minProperties", "element.size < 1"},
		{schemas(`{"Unused":{"type":"object","minProperties":0,"allOf":[{"type":"object","properties":{"v":{"type":"string"}}}]}}`), "/components/schemas/Unused/minProperties", "element.size < 0"},
	} {
		rq := request(tc.doc, "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		text := source(t, r, "Unused")
		if !strings.Contains(text, "generatedSerializer()") || !strings.Contains(text, tc.bound) {
			t.Fatal("lost count propagation", text)
		}
		rq.Options.Validation = false
		_, err = core.Generate(rq)
		d, ok := err.(core.Diagnostics)
		if !ok || len(d) != 1 || d[0].Code != "E_SCHEMA_UNSUPPORTED" || d[0].Pointer == nil || *d[0].Pointer != tc.pointer {
			t.Fatal("default false source pointer", err)
		}
	}
	// Preserve the exact historical direct nullable input as a bounded
	// validation-only graduation; all other source-role guards below remain.
	originalNullable := schemas(`{"Unused":{"type":["object","null"],"properties":{"v":{"type":"string"}},"minProperties":0}}`)
	if text := source(t, aggregationReport(t, originalNullable), "Unused"); !strings.Contains(text, "generatedSerializer()") || !strings.Contains(text, "element.size < 0") {
		t.Fatal("original direct nullable count erased", text)
	}
	if _, err := core.Generate(request(originalNullable, "")); err == nil {
		t.Fatal("validation-off admitted original direct nullable count")
	}
	for _, tc := range []struct {
		doc     []byte
		pointer string
	}{
		{schemas(`{"Unused":{"type":"object","properties":{},"minProperties":0}}`), "/components/schemas/Unused/minProperties"},
		{schemas(`{"Unused":{"type":"object","minProperties":0,"properties":{"tag":{"type":"string","const":"x"}}}}`), "/components/schemas/Unused/minProperties"},
		{schemas(`{"Unused":{"type":"string","minProperties":0}}`), "/components/schemas/Unused/minProperties"},
		{schemas(`{"Unused":{"type":"array","items":{"type":"string"},"minProperties":0}}`), "/components/schemas/Unused/minProperties"},
		{schemas(`{"Unused":{"type":"string","enum":["x"],"minProperties":0}}`), "/components/schemas/Unused/minProperties"},
		{schemas(`{"Unused":{"type":"integer","x-kotlin-value-class":true,"minProperties":0}}`), "/components/schemas/Unused/minProperties"},
		{schemas(`{"Unused":{"type":"string","const":"x","minProperties":0}}`), "/components/schemas/Unused/minProperties"},

		{fieldDoc(`{"oneOf":[` + count + `,{"type":"object","properties":{"w":{"type":"integer"}}}]}`), "/components/schemas/Record/properties/value/oneOf/0/minProperties"},
		{fieldDoc(`{"oneOf":[{"type":"string"},` + count + `]}`), "/components/schemas/Record/properties/value/oneOf/1/minProperties"},
		{schemas(`{"Base":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"type":"object","minProperties":0,"properties":{"kind":{"type":"string","const":"leaf"},"v":{"type":"string"}}}}`), "/components/schemas/Leaf/minProperties"},
		{schemas(`{"Record":{"allOf":[{"type":"object","properties":{"v":` + count + `}}],"type":"object","properties":{"v":{"type":"string"}}}}`), "/components/schemas/Record/allOf/0/properties/v/minProperties"},
		{httpDoc(`{"parameters":[{"name":"q","in":"query","schema":` + count + `}],"responses":{"204":{}}}`), "/paths/~1ping/get/parameters/0/schema/minProperties"},
		{httpDoc(`{"parameters":[{"name":"q","in":"query","style":"deepObject","schema":` + count + `}],"responses":{"204":{}}}`), "/paths/~1ping/get/parameters/0/schema/minProperties"},
		{httpDoc(`{"parameters":[{"name":"q","in":"query","style":"deepObject","schema":{"type":"object","properties":{"nested":` + count + `}}}],"responses":{"204":{}}}`), "/paths/~1ping/get/parameters/0/schema/properties/nested/minProperties"},
		{fieldDoc(`{"type":"object","properties":{"v":{"type":"string"}},"additionalProperties":` + count + `}`), "/components/schemas/Record/properties/value/additionalProperties/minProperties"},
		{fieldDoc(`{"$ref":"#/components/schemas/Record","minProperties":0}`), "/components/schemas/Record/properties/value/minProperties"},
		{httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"oneOf":[{},` + count + `]}}}}}}`), "/paths/~1ping/get/responses/200/content/application~1json/schema/oneOf/1/minProperties"},
	} {
		aliasFailure(t, tc.doc, "E_SCHEMA_UNSUPPORTED", tc.pointer)
	}
	// Original valid referenced count input now owns the same whole-object
	// serializer; neither the source fields nor their count become a subobject.
	original := schemas(`{"Base":` + count + `,"Unused":{"allOf":[{"$ref":"#/components/schemas/Base"},{"type":"object","properties":{"x":{"type":"string"}}}]}}`)
	got := aggregationReport(t, original)
	if !strings.Contains(source(t, got, "Unused"), "element.size < 1") || !strings.Contains(source(t, got, "Unused"), "generatedSerializer()") {
		t.Fatal("Referenced count erased")
	}
	if _, err := core.Generate(request(original, "")); err == nil {
		t.Fatal("Default-off admitted original referenced count")
	}
}

func TestDTOPropertyCountIdentityAndLimits(t *testing.T) {
	shape := `{"type":"object","minProperties":1,"maxProperties":2,"properties":{"v":{"type":"string"}}}`
	input := []byte(`{"openapi":"3.1.0","paths":{"/a":{"get":{"tags":["Left"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}},"/b":{"get":{"tags":["Right"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("source provenance changed nominal equality", err)
	}
	if !strings.Contains(source(t, r, "SharedResponseDto"), "element.size < 1") {
		t.Fatal("lost metadata")
	}
	rq.Document = []byte(strings.Replace(string(input), `"maxProperties":2`, `"maxProperties":3`, 1))
	_, err = core.Generate(rq)
	if d, ok := err.(core.Diagnostics); !ok || len(d) != 1 || d[0].Code != "E_NAME_COLLISION" {
		t.Fatal("differing counts merged", err)
	}
	for _, tc := range []struct {
		n        int
		required bool
	}{{120, true}, {121, true}, {240, false}, {241, false}} {
		fields, required := []string{}, []string{}
		for i := 0; i < tc.n; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"integer","format":"int64"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		req := ""
		if tc.required {
			req = `,"required":[` + strings.Join(required, ",") + `]`
		}
		input := schemas(`{"Wide":{"type":"object","minProperties":0,"properties":{` + strings.Join(fields, ",") + `}` + req + `}}`)
		if tc.n == 121 || tc.n == 241 {
			aliasFailure(t, input, "E_MODEL_LIMIT", "/components/schemas/Wide")
		} else {
			rq := request(input, "")
			rq.Options.Validation = true
			if _, err := core.Generate(rq); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Object recursion remains nominal; object count metadata does not turn it
	// into an emitted use-site collection validation chain.
	shape = `{"type":"object","minProperties":0,"properties":{"v":{"type":"string"}}}`
	for i := 0; i < 49; i++ {
		shape = `{"type":"object","properties":{"v":` + shape + `}}`
	}
	aliasFailure(t, schemas(`{"Deep":`+shape+`}`), "E_SCHEMA_DEPTH", "/components/schemas/Deep"+strings.Repeat("/properties/v", 49))
	plain := request(fieldDoc(`{"type":"object","properties":{"v":{"type":"string"}}}`), "")
	before, err := core.GenerateWithDiagnostics(plain)
	if err != nil {
		t.Fatal(err)
	}
	plain.Options.Validation = true
	after, err := core.GenerateWithDiagnostics(plain)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("unbounded changed", err)
	}
}
