package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBoundedEnumCollectionRootsStrictIndependentValues(t *testing.T) {
	body := `{"Tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}},"default":[{"root":"one"}]},"Alias":{"$ref":"#/components/schemas/Tree","default":[{"alias":"two"}]},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":[{"field":"one"}]}},"default":{"value":[{"outer":"two"}]}}}`
	for _, tc := range []struct{ from, to, pointer string }{
		{`"root":"one"`, `"root":"x"`, "/components/schemas/Tree/default/0/root"},
		{`"alias":"two"`, `"alias":"future"`, "/components/schemas/Alias/default/0/alias"},
		{`"field":"one"`, `"field":"x"`, "/components/schemas/Use/properties/value/default/0/field"},
		{`"outer":"two"`, `"outer":"x"`, "/components/schemas/Use/default/value/0/outer"},
	} {
		enumBoundsDiagnostic(t, schemas(strings.Replace(body, tc.from, tc.to, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
	}
	for _, nullable := range []string{"true", "false"} {
		doc := schemas(`{"Tree":{"type":"array","nullable":true,"items":{"enum":["one","x"],"minLength":2},"default":null},"Alias":{"$ref":"#/components/schemas/Tree","nullable":` + nullable + `,"default":null}}`)
		if nullable == "false" {
			enumBoundsDiagnostic(t, doc, true, "E_SCHEMA_INVALID", "/components/schemas/Alias/default")
		} else {
			boundedEnumRootsGenerate(t, doc)
		}
	}
}

func TestBoundedEnumCollectionRootsOriginalDepthAndVocabularyCaps(t *testing.T) {
	for _, depth := range []int{16, 17} {
		tree := `{"enum":["one"],"minLength":1}`
		for i := 0; i < depth; i++ {
			tree = `{"type":"array","items":` + tree + `}`
		}
		rq := request(schemas(`{"Tree":`+tree+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 16 && err != nil {
			t.Fatal("bounded root rejected original checked16", err)
		}
		if depth == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("bounded root escaped original checked16", err)
		}
	}
	for _, count := range []int{240, 241} {
		words := make([]string, count)
		for i := range words {
			words[i] = fmt.Sprintf(`"w%d"`, i)
		}
		rq := request(schemas(`{"Tree":{"type":"array","items":{"enum":[`+strings.Join(words, ",")+`],"minLength":1}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if count == 240 && err != nil {
			t.Fatal("bounded root rejected enum240", err)
		}
		if count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("bounded root escaped enum240", err)
		}
	}
}

func TestBoundedEnumCollectionRootsInternalBridge(t *testing.T) {
	boundedEnumRootsGenerate(t, schemas(`{"Tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2}},"default":[{"root":"one"}]},"Alias":{"$ref":"#/components/schemas/Tree","default":[{"alias":"two"}]},"Hops":{"properties":{"tree":{"$ref":"#/components/schemas/Alias"}}},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/tree","default":[{"field":"one"}]}}}}`))
}

func TestBoundedEnumCollectionRootsRequiredAndPatchDefaults(t *testing.T) {
	for _, required := range []bool{false, true} {
		body := `{"type":"object","properties":{"value":{"$ref":"#/components/schemas/Tree","nullable":true,"default":[{"a/~":"x"}]}}`
		if required {
			body += `,"required":["value"]`
		}
		body += `}`
		doc := `{"openapi":"3.1.0","components":{"schemas":{"Tree":{"type":"array","items":{"additionalProperties":{"enum":["one","x"],"minLength":2}}}}},"paths":{"/patch":{"patch":{"operationId":"patch","requestBody":{"content":{"application/json":{"schema":` + body + `}}},"responses":{"204":{"description":"Updated"}}}}}}`
		enumBoundsDiagnostic(t, []byte(doc), true, "E_SCHEMA_INVALID", "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/value/default/0/a~1~0")
	}
}

func TestBoundedEnumCollectionRootsSourceBeforeNullAlias(t *testing.T) {
	for _, tc := range []struct{ leaf, code, tail string }{
		{`{"enum":["one","one"],"minLength":1}`, "E_SCHEMA_INVALID", "/enum/1"},
		{`{"enum":["one"],"minLength":-1}`, "E_SCHEMA_INVALID", "/minLength"},
		{`{"enum":["one"],"minLength":1.0}`, "E_SCHEMA_INVALID", "/minLength"},
		{`{"enum":["one"],"pattern":"(?=x)"}`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`{"enum":["one"],"minLength":1,"format":"unsupported"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"enum":["one"],"minLength":1,"default":"one"}`, "E_SCHEMA_UNSUPPORTED", "/default"},
	} {
		doc := schemas(`{"Tree":{"type":"array","nullable":true,"items":{"additionalProperties":` + tc.leaf + `},"default":null},"Alias":{"$ref":"#/components/schemas/Tree","default":null}}`)
		enumBoundsDiagnostic(t, doc, true, tc.code, "/components/schemas/Tree/items/additionalProperties"+tc.tail)
	}
}
