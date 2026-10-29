package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedStringEnumBoundsValuesAndNullLayers(t *testing.T) {
	leaf := `{"type":"string","nullable":true,"enum":["one","two","x","UPPER","toolong"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}`
	shape := `{"type":"array","nullable":true,"items":{"type":"object","nullable":true,"additionalProperties":LEAF},"default":VALUE}`
	for _, value := range []string{`null`, `[]`, `[null]`, `[{"a":null}]`, `[{"$ref":"one","/~":"two"}]`} {
		field := strings.ReplaceAll(strings.Replace(shape, "LEAF", leaf, 1), "VALUE", value)
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("independent nested nullable/container/default view", value, err)
		}
	}
	for _, tc := range []struct{ value, tail string }{
		{`[{"z/~":"x","a":"one"}]`, "/0/z~1~0"},
		{`[{"z":"UPPER"}]`, "/0/z"},
		{`[{"z":"toolong"}]`, "/0/z"},
		{`[{"z":"future"}]`, "/0/z"},
		{`[{"z":7}]`, "/0/z"},
	} {
		field := strings.ReplaceAll(strings.Replace(shape, "LEAF", leaf, 1), "VALUE", tc.value)
		for _, required := range []bool{false, true} {
			doc := fieldDoc(field)
			if !required {
				doc = []byte(strings.Replace(string(doc), `,"required":["value"]`, "", 1))
			}
			enumBoundsDiagnostic(t, doc, true, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/default"+tc.tail)
		}
	}
}

func TestNestedStringEnumBoundsSourceBeforeEmptyDefaults(t *testing.T) {
	for _, tc := range []struct{ leaf, code, tail string }{
		{`{"enum":["one"],"minLength":4,"maxLength":2}`, "E_SCHEMA_INVALID", "/maxLength"},
		{`{"enum":["one","one"],"minLength":1}`, "E_SCHEMA_INVALID", "/enum/1"},
		{`{"enum":["one"],"minLength":1.0}`, "E_SCHEMA_INVALID", "/minLength"},
		{`{"enum":["one"],"pattern":"["}`, "E_SCHEMA_INVALID", "/pattern"},
		{`{"enum":["one"],"pattern":"(?=x)"}`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`{"enum":["one"],"minLength":1,"format":"unsupported"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"enum":["one"],"minLength":1,"default":"one"}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"type":["string","null"],"format":"date","enum":["one",null],"minLength":1}`, "E_SCHEMA_INVALID", "/enum/1"},
	} {
		for _, value := range []string{`[]`, `null`} {
			field := `{"type":"array","nullable":true,"items":{"additionalProperties":` + tc.leaf + `},"default":` + value + `}`
			enumBoundsDiagnostic(t, fieldDoc(field), true, tc.code, "/components/schemas/Record/properties/value/items/additionalProperties"+tc.tail)
		}
	}
	enumBoundsDiagnostic(t, fieldDoc(`{"type":"array","items":{"type":"array","items":{"enum":["one"],"minLength":1},"default":[]},"default":[]}`), true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/items/default")
}

func TestNestedStringEnumBoundsDefaultByteBudget(t *testing.T) {
	for _, size := range []int{32740, 32768} {
		word := strings.Repeat("a", size)
		field := `{"type":"array","items":{"additionalProperties":{"enum":["` + word + `"],"minLength":1}},"default":[{"a":"` + word + `"}]}`
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if size == 32740 && err != nil {
			t.Fatal("bounded nested default below original32KiB refused", err)
		}
		if size == 32768 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("bounded nested default escaped original32KiB", err)
		}
	}
}

func TestNestedStringEnumBoundsCheckedDepth(t *testing.T) {
	for _, depth := range []int{16, 17} {
		field := `{"type":"string","enum":["one"],"minLength":2}`
		for i := 0; i < depth; i++ {
			field = `{"type":"array","items":` + field + `}`
		}
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 16 {
			if err != nil {
				t.Fatal("bounded depth16 enum tree rejected", err)
			}
		} else {
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_DEPTH" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Record/properties/value"+strings.Repeat("/items", depth) || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("nested enum checks escaped existing depth16 budget", err)
			}
		}
	}
}

func TestNestedStringEnumBoundsKeepOtherSourceRoles(t *testing.T) {
	tree := `{"type":"array","items":{"additionalProperties":{"enum":["one"],"minLength":2}}}`
	for _, body := range []string{
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"a"},"value":` + tree + `}},{"properties":{"tag":{"const":"b"}}}]}}`,
		`{"Use":{"allOf":[{"properties":{"value":` + tree + `}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("inline tree granted an excluded source role", body, err)
		}
	}
	for _, count := range []int{240, 241} {
		words := make([]string, count)
		for i := range words {
			words[i] = fmt.Sprintf(`"w%d"`, i)
		}
		field := `{"type":"array","items":{"additionalProperties":{"enum":[` + strings.Join(words, ",") + `],"minLength":1}},"default":[]}`
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("original enum240 cap changed", count, err)
		}
	}
}
