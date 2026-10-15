package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReportsAreComponentSortedButFieldsRetainSourceOrder(t *testing.T) {
	a := schemas(`{"Z":{"type":"object","properties":{"third":{"type":"string"},"first":{"type":"object"},"second":{"enum":["a"]}}},"A":{"type":"object"}}`)
	b := schemas(`{"A":{"type":"object"},"Z":{"type":"object","properties":{"third":{"type":"string"},"first":{"type":"object"},"second":{"enum":["a"]}}}}`)
	first, second := report(t, a, "Dto"), report(t, b, "Dto")
	if !reflect.DeepEqual(first, second) {
		t.Fatal("component ordering affects successful report")
	}
	z := source(t, first, "ZDto")
	if strings.Index(z, "val third:") > strings.Index(z, "val first:") || strings.Index(z, "val first:") > strings.Index(z, "val second:") {
		t.Fatal("property order changed")
	}
	rq := request(a, "Dto")
	rq.SourceName = "other/source.json"
	third, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(first, third) {
		t.Fatal("source name embedded in output", err)
	}
}

func TestProjectionReferenceAndBodySlotBoundaries(t *testing.T) {
	for _, constant := range []bool{false, true} {
		for _, count := range []int{120, 121} {
			fields := map[string]any{}
			required := []string{}
			for i := range count {
				name := fmt.Sprintf("field%d", i)
				fields[name] = map[string]any{"$ref": "#/components/schemas/Scalar"}
				required = append(required, name)
			}
			scalar := map[string]any{"type": "integer", "format": "int64"}
			if constant {
				scalar["const"] = int64(-9223372036854775808)
			} else {
				scalar["enum"] = []int{0, 1}
			}
			input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Scalar": scalar, "Record": map[string]any{"type": "object", "properties": fields, "required": required}}}})
			r, err := core.GenerateWithDiagnostics(request(input, ""))
			if (err == nil) != (count == 120) || err != nil && err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
				t.Fatal(constant, count, err)
			}
			if err == nil {
				s := source(t, r, "Record")
				if constant && (!strings.Contains(s, "public class Record {") || strings.Count(s, "kotlin.Long.MIN_VALUE") != 120) {
					t.Fatal("const refs incorrectly consume constructor parameters")
				}
				if !constant && strings.Contains(s, "= kotlin.Long.MIN_VALUE") {
					t.Fatal("enum ref became const")
				}
				if len(r.Warnings) != 1 || *r.Warnings[0].Pointer != "/components/schemas/Scalar/"+map[bool]string{true: "const", false: "enum"}[constant] {
					t.Fatal("reference warnings are not definition-located", r.Warnings)
				}
			}
		}
	}
}

func TestSourceValidationBeforeNarrowingAndReferenceWrappers(t *testing.T) {
	for _, tc := range []struct{ input, code, pointer string }{
		{`{"Scalar":{"type":"boolean","format":false}}`, "E_SCHEMA_INVALID", "/components/schemas/Scalar"},
		{`{"Status":{"type":"string","enum":["a"],"format":false}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/format"},
		{`{"Record":{"type":"object","properties":{"value":{"allOf":[{"$ref":"#/components/schemas/Base","nullable":"bad"}]}}},"Base":{"type":"object"}}`, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/allOf/0/nullable"},
		{`{"Record":{"allOf":[{"type":"object"},{"type":"string"}]}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/1/type"},
		{`{"Record":{"allOf":[{"type":"object"},{"oneOf":[{"type":"string"},{"type":"integer"}]}]}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/1/oneOf"},
		{`{"Record":{"type":"object","properties":{"v":{"type":"string","allOf":[{"type":"string"}]}}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/v/allOf"},
		{`{"Status":{"type":"string","enum":["a","a"]}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/enum/1"},
		{`{"Status":{"type":"string","enum":["a",null]}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/enum/1"},
		{`{"Status":{"type":"integer","enum":[9223372036854775808]}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/enum/0"},
		{`{"Status":{"type":"integer","format":"int64","enum":[9223372036854775808]}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/enum/0"},
		{`{"Status":{"type":"number","format":"float","enum":[1e100]}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/enum/0"},
		{`{"Status":{"enum":[1,2]}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/enum/0"},
		{`{"Status":{"type":"boolean","enum":[true,"false"]}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/enum/1"},
		{`{"Status":{"$ref":"#/components/schemas/Status"}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Status"},
		{`{"Status":{"const":2147483648}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Status/const"},
		{`{"Status":{"type":"string","enum":["a"],"deprecated":"bad"}}`, "E_SCHEMA_INVALID", "/components/schemas/Status/deprecated"},
	} {
		failure(t, schemas(tc.input), tc.code, tc.pointer)
	}
	r := report(t, schemas(`{"B/a~":{"type":"object","properties":{"id":{"type":"string"}}},"Holder":{"type":"object","required":["value"],"properties":{"value":{"title":"Metadata","allOf":[{"$ref":"#/components/schemas/B~1a~0","nullable":true},{"examples":[{"allOf":"not-schema"}]}],"nullable":false}}}}`), "")
	s := source(t, r, "Holder")
	if !strings.Contains(s, "public val value: org.example.models.models.BA,") || strings.Contains(s, "public data class Value") {
		t.Fatal("wrapper inherited member nullable or copied type", s)
	}
}

func TestCoreInputLimitAndOptionErrorPrecedence(t *testing.T) {
	input := append(schemas(`{"Empty":{"type":"object"}}`), []byte(" ")...)
	input = append(input, bytes.Repeat([]byte{' '}, core.MaxDocumentBytes-len(input))...)
	if _, err := core.Generate(request(input, "")); err != nil {
		t.Fatal(err)
	}
	input = append(input, ' ')
	if _, err := core.Generate(request(input, "")); err == nil || err.(core.Diagnostics)[0].Code != "E_INPUT_LIMIT" {
		t.Fatal(err)
	}
	rq := request([]byte("not JSON"), "")
	rq.Options.PackageName = "../escape"
	if _, err := core.Generate(rq); err == nil || err.(core.Diagnostics)[0].Code != "E_PACKAGE_INVALID" {
		t.Fatal("option validation precedence", err)
	}
}

func FuzzGenerate(f *testing.F) {
	f.Add(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false},"Z":{"type":"integer","format":"int64"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/A"},"n":{"$ref":"#/components/schemas/B"}},"required":["n"]}}`))
	f.Add(schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/Z"},"Z":{"type":"string","minLength":2}}`))
	f.Add(schemas(`{"Z":{"type":["integer","null"],"format":"int64"},"A":{"$ref":"#/components/schemas/Z","nullable":false},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/A"}}}}`))
	f.Add(schemas(`{"Z":{"type":["string","null"],"minLength":2},"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Z"}}}}}`))
	for _, seed := range [][]byte{schemas(`{"Empty":{"type":"object","nullable":true}}`), schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":false},"Z":{"type":["object","null"]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/A"}}}}`), schemas(`{"Z":{"type":"object","nullable":true},"C":{"allOf":[{"$ref":"#/components/schemas/Z","nullable":false},{"properties":{"v":{"type":"string"}}}]}}`), schemas(`{"Empty":{"type":"object"}}`), fieldDoc(`{"type":"string"}`), fieldDoc(`{"enum":["a", "雪"]}`), schemas(`{"A":{"type":"object","properties":{"next":{"$ref":"#/components/schemas/A"}}}}`), fieldDoc(`{"anyOf":[{"type":"integer"},{"type":"null"}]}`), fieldDoc(`{"type":"number","const":-0.0}`), schemas(`{"A":{"allOf":[{"$ref":"#/components/schemas/A"},{"type":"object"}]}}`)} {
		f.Add(seed)
	}
	// Exercise dictionary slots, transparent alias cycles and the composition
	// boundary in addition to the original model/union seeds.
	for _, seed := range [][]byte{
		fieldDoc(`{"type":"object","additionalProperties":{"type":"integer"}}`),
		fieldDoc(`{"type":"object","additionalProperties":null}`),
		schemas(`{"Bag":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Bag"}}}`),
		schemas(`{"Bag":{"allOf":[{"type":"object","additionalProperties":true},{"type":"object"}]}}`),
	} {
		f.Add(seed)
	}
	for _, seed := range [][]byte{
		schemas(`{"Id":{"type":"string","minLength":1},"Link":{"$ref":"#/components/schemas/Id"},"Bag":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Link"}},"Record":{"type":"object","properties":{"value":{"$ref":"#/components/schemas/Bag"}}}}`),
		schemas(`{"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"}}`),
		schemas(`{"A":{"$ref":"#/components/schemas/B"},"B":{"type":"integer","minimum":0}}`),
		schemas(`{"Id":{"type":"string","x-kotlin-value-class":true},"Record":{"type":"object","properties":{"value":{"$ref":"#/components/schemas/Id"}}}}`),
		schemas(`{"Id":{"type":"integer","format":"int64","x-kotlin-value-class":true}}`),
		schemas(`{"Id":{"type":"string","x-kotlin-value-class":true,"minLength":1,"maxLength":2}}`),
		schemas(`{"Id":{"type":"number","format":"float","x-kotlin-value-class":true,"minimum":-0.1,"maximum":0.1}}`),
		schemas(`{"Id":{"type":"number","x-kotlin-value-class":true,"minimum":1e-999999}}`),
		schemas(`{"Id":{"type":"string","x-kotlin-value-class":null}}`),
		fieldDoc(`{"type":["string","null"],"minLength":1,"maxLength":2}`),
		fieldDoc(`{"type":"number","format":"float","minimum":-0.1,"maximum":0.1}`),
		fieldDoc(`{"type":"number","minimum":1e-999999}`),
		fieldDoc(`{"type":"array","items":{"type":"object","additionalProperties":{"type":"array","items":{"type":["string","null"],"minLength":1}}}}`),
		fieldDoc(`{"type":"array","items":{"oneOf":[{"type":"string","minLength":1},{"type":"null"}]}}`),
		fieldDoc(`{"type":"object","additionalProperties":{"type":"array","items":{"type":"number","format":"float","minimum":-1}}}`),
		fieldDoc(`{"type":"array","items":{"type":"string","pattern":"(?<=a)b"}}`),
		fieldDoc(`{"type":"integer","exclusiveMinimum":2,"exclusiveMaximum":8,"multipleOf":2}`),
		fieldDoc(`{"type":"array","minItems":1,"maxItems":2,"items":{"type":"object","minProperties":1,"maxProperties":2,"additionalProperties":{"type":"integer"}}}`),
		fieldDoc(`{"type":"object","additionalProperties":true,"minProperties":2,"maxProperties":1}`),
		fieldDoc(`{"type":"array","uniqueItems":true,"minItems":1,"items":{"type":["boolean","null"]}}`),
		fieldDoc(`{"type":"array","uniqueItems":false,"items":{"type":"object","additionalProperties":true}}`),
		fieldDoc(`{"type":"array","uniqueItems":"true","items":{"type":"integer"}}`),
		fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"integer","enum":[1]}}`),
		fieldDoc(`{"type":"object","additionalProperties":true,"uniqueItems":false}`),
		schemas(`{"Token":{"type":"string"},"Link":{"$ref":"#/components/schemas/Token"},"Record":{"type":"object","properties":{"v":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Link"}}}}}`),
		fieldDoc(`{"type":"number","format":"float","multipleOf":1e-512}`),
		fieldDoc(`{"type":"integer","format":"int64","exclusiveMinimum":9223372036854775807}`),
		[]byte(`{"openapi":"3.0.3","components":{"schemas":{"Id":{"type":"integer","minimum":2,"maximum":4,"exclusiveMinimum":true,"multipleOf":2,"x-kotlin-value-class":true}}}}`),
		fieldDoc(`{"type":"string","pattern":"^[a-z]+$"}`),
		fieldDoc(`{"type":"string","pattern":"(?<=a)b"}`),
		schemas(`{"Id":{"type":"string","x-kotlin-value-class":true,"pattern":"(?:ab|cd)+"}}`),
		sseDoc(`{"type":"object","properties":{}}`),
		sseDoc(`{"type":"object","required":["data"],"properties":{"event":{"type":"string","const":"row"},"data":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"}}}}}}`),
		sseDoc(`{"oneOf":[{"type":"object","properties":{"event":{"type":"string","const":"same"}}},{"type":"object","properties":{"event":{"type":"string","const":"same"}}}]}`),
		httpDoc(`{"parameters":[{"name":"lastEventId","in":"query","schema":{"type":"string"}}],"responses":{"200":{"content":{"text/event-stream":{"schema":{"type":"object","properties":{}}}}}}}`),
		httpDoc(`{"responses":{"204":{}}}`),
		fieldDoc(`{"type":"string","format":"date-time"}`),
		fieldDoc(`{"type":"string","format":"date","enum":["2024-02-29"]}`),
		fieldDoc(`{"enum":["non-numeric"],"x-native-type":"bigint"}`),
		fieldDoc(`{"type":["string","null"],"enum":["non-numeric"],"x-native-type":"bigint"}`),
		fieldDoc(`{"enum":["x"],"x-native-type":"org.example.Bad<Type>"}`),
		fieldDoc(`{"type":"string","format":"date-time","const":"2024-02-29T12:34:56Z"}`),
		fieldDoc(`{"x-native-type":"bigint","const":"9223372036854775807"}`),
		fieldDoc(`{"type":"string","format":"binary"}`),
		fieldDoc(`{"type":"null"}`),
		fieldDoc(`{"type":["string","null"],"x-native-type":"bigint"}`),
		fieldDoc(`{"type":"string","x-native-type":"bad; injected"}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"big":{"type":["string","null"],"x-native-type":"bigint"}}}}}},"responses":{"204":{}}}`),
		httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"anyOf":[{}, {"not":{}}]}}}}}}`),
		httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"oneOf":[{"not":{}},{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}]}}}}}}`),
		httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"anyOf":[{"type":"string"},{"type":"boolean"}]}}}}}}`),
		httpDoc(`{"requestBody":` + presenceBody + `,"responses":{"204":{}}}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"payload":{"anyOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]}}}}}},"responses":{"204":{}}}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"rows":{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"boolean"}]}},"values":{"type":"object","additionalProperties":{"oneOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]}}}}}}},"responses":{"204":{}}}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"anyOf":[{"type":"string"},{"type":"boolean"}]}}}},"responses":{"204":{}}}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"anyOf":[{"enum":["known"]},{"type":"string"}]}}}},"responses":{"200":{"content":{"application/json":{"schema":{"oneOf":[{"enum":["reply"]},{"type":"null"}]}}}}}}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"oneOf":[{"type":"array","items":{"type":"string"}},{"type":"object","required":["code"],"properties":{"code":{"type":"integer"}}}]}}}},"responses":{"204":{}}}`),
		httpDoc(`{"parameters":[` + simpleDeepParam + `],"responses":{"204":{}}}`),
		httpDoc(`{"parameters":[{"name":"values","in":"query","schema":{"type":"array","items":{"type":["string","null"],"enum":["ready"]}}}],"responses":{"204":{}}}`),
		httpDoc(`{"parameters":[{"name":"filter","in":"query","style":"deepObject","schema":{"type":"object","properties":{"opaque":{"oneOf":[{"type":"string"},{"type":"boolean"}]},"rows":{"type":"array","items":{"type":"string"}}}}}],"responses":{"204":{}}}`),
		httpDoc(`{"responses":{"204":{},"400":{"content":{"application/json":{"schema":{"type":"object","properties":{"message":{"type":"string"}}}}}}}}`),
		httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"}}}}}}}}}`),
		httpDoc(`{"parameters":[{"name":"mode","in":"query","schema":{"type":"array","items":{"type":"string","enum":["a","b"]}}}],"responses":{"204":{}}}`),
		[]byte(`{"openapi":"3.1.0","paths":{"/ping":{"get":{"parameters":[{"$ref":"#/pool/0"}],"responses":{"204":{}}}}},"pool":[{"name":"p","in":"query","schema":{"type":"string"}}]}`),
		[]byte(`{"openapi":"3.1.0","paths":{"/ping":{"get":{"responses":{"200":{"$ref":"#/pool/0"}}}}},"pool":[{"$ref":"#/pool/0"}]}`),
		httpDoc(`{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"integer"}}}},"responses":{"204":{}}}`),
		httpDoc(`{"parameters":[{"name":"v","in":"query","style":"deepObject","schema":{"type":"array","items":{"type":"string"}}}],"responses":{"204":{}}}`),
		httpDoc(`{"parameters":[{"name":"id","in":"query","schema":{"type":"integer"}}],"responses":{"200":{"content":{"application/json":{"schema":{"type":"string"}}}}}}`),
		[]byte(`{"openapi":"3.1.0","paths":{"/items/{id}":{"get":{"parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"204":{}}}}}}`),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 65536 {
			t.Skip()
		}
		rq := request(input, "")
		rq.Options.Validation = len(input)%2 == 0 || bytes.Contains(input, []byte("minLength")) || bytes.Contains(input, []byte("minimum")) || bytes.Contains(input, []byte("exclusiveMinimum")) || bytes.Contains(input, []byte("multipleOf")) || bytes.Contains(input, []byte("minItems")) || bytes.Contains(input, []byte("minProperties")) || bytes.Contains(input, []byte("uniqueItems"))
		a, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			if len(a.Artifacts.Files) != 0 || len(a.Warnings) != 0 {
				t.Fatal("partial failure result")
			}
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || !strings.HasPrefix(ds[0].Code, "E_") {
				t.Fatal("unstructured error")
			}
			return
		}
		b, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(a, b) {
			t.Fatal("non-deterministic generation", err)
		}
		seen := map[string]bool{}
		for _, file := range a.Artifacts.Files {
			if seen[file.Path] || strings.Contains(file.Path, "..") || strings.HasPrefix(file.Path, "/") || strings.Contains(file.Path, "\\") {
				t.Fatal("unsafe generated path")
			}
			seen[file.Path] = true
		}
	})
}
