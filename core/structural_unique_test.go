package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuralUniqueOriginalGraduations(t *testing.T) {
	for _, input := range [][]byte{
		fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"array","items":{"type":"number"}}}`),
		fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"array","items":{"type":"integer","enum":[1]}}}`),
		fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"enum":["x"]}}}`),
		schemas(`{"Id":{"type":"string","format":"uuid"},"Holder":{"properties":{"v":{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Id"}}}}}}`),
		fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"anyOf":[{"type":"string","format":"date"},{"type":"null"}]}}}`),
		schemas(`{"Value":{"type":"object","additionalProperties":{"type":"string"}},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"uniqueItems":true}}}}`),
		fieldDoc(`{"type":"array","items":{"type":"array","items":{"type":"integer"}},"uniqueItems":true,"default":[]}`),
		fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"array","items":{"type":"boolean","enum":[true]}},"default":[]}`),
		pointerDoc(`{"type":"array","items":{"type":"array","items":{"type":"integer"}},"uniqueItems":true}`, pointerRef("Source/properties/leaf")),
	} {
		mapUniqueGraduation(t, input)
	}
}

func TestStructuralUniqueTrees(t *testing.T) {
	for _, item := range []string{
		`{"type":"array","nullable":true,"items":{"type":"string","nullable":true}}`,
		`{"type":"object","additionalProperties":{"type":"array","items":{"type":"string","format":"date-time"}}}`,
		`{"type":"array","items":{"type":"string","enum":["a","b"]}}`,
		`{"type":"array","items":{"type":"number","format":"float"}}`,
		`{"type":"object","additionalProperties":{"type":"number"}}`,
		`{"$ref":"#/components/schemas/Rows"}`,
		`{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Rows"}}`,
	} {
		rq := request(schemas(`{"Rows":{"type":"array","items":{"type":"integer"}},"Holder":{"properties":{"rows":{"type":"array","uniqueItems":true,"items":`+item+`}}}}`), "Dto")
		rq.Options.Validation = true
		first, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(item, err)
		}
		if !strings.Contains(source(t, first, "HolderDto"), "`openapi-kt unique structural items`") {
			t.Fatal("structural check missing", item)
		}
		second, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("non-deterministic structural check", item, err)
		}
	}
}

func TestStructuralUniqueDefaultEquality(t *testing.T) {
	for _, tc := range []struct{ item, value, suffix string }{
		{`{"type":"array","items":{"type":"integer"}}`, `[[1,2],[1,2]]`, "/1"},
		{`{"type":"array","items":{"type":"integer"}}`, `[[1,2],[2,1]]`, ""},
		{`{"type":"object","additionalProperties":{"type":"array","items":{"type":"string","format":"date-time"}}}`, `[{"a":["2026-01-02T03:04:05Z"]},{"a":["2026-01-02T05:04:05+02:00"]}]`, "/1"},
		{`{"type":"array","items":{"type":"number","format":"float"}}`, `[[16777216],[16777217]]`, "/1"},
		{`{"type":"array","items":{"type":"number"}}`, `[[0],[-0.0]]`, "/1"},
		{`{"type":"array","items":{"type":"string","enum":["a","b"]}}`, `[["a"],["a"]]`, "/1"},
		{`{"type":"array","nullable":true,"items":{"type":"string","nullable":true}}`, `[null,[],[null],["null"]]`, ""},
	} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+tc.item+`,"default":`+tc.value+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if tc.suffix == "" {
			if err != nil {
				t.Fatal(tc, err)
			}
		} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default"+tc.suffix {
			t.Fatal("mapped structural default equality/Pointer lost", tc, err)
		}
	}
}

func TestStructuralUniqueBudgets(t *testing.T) {
	for _, layers := range []int{16, 17} {
		item := `{"type":"string"}`
		for i := 1; i < layers; i++ {
			item = `{"type":"array","items":` + item + `}`
		}
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+item+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (layers == 16) || err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("checked traversal depth lost", layers, err)
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"type":"array","uniqueItems":true,"items":{"type":"array","items":{"type":"string"}}}`, i)
		}
		rq := request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (width == 240) || err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("slot budget lost", width, err)
		}
	}
}

func TestStructuralUniqueNoDropAndNoop(t *testing.T) {
	shape := `{"type":"array","uniqueItems":true,"items":{"type":"array","items":{"type":"string"}}}`
	aliasFailure(t, schemas(`{"Holder":{"properties":{"v":{"type":"string"}},"additionalProperties":`+shape+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/additionalProperties/uniqueItems")
	aliasFailure(t, schemas(`{"Holder":{"allOf":[{"properties":{"v":`+shape+`}}],"properties":{"v":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/allOf/0/properties/v/uniqueItems")
	rq := request(fieldDoc(strings.Replace(shape, `"uniqueItems":true,`, "", 1)), "Dto")
	rq.Options.Validation = true
	first, e1 := core.GenerateWithDiagnostics(rq)
	rq.Document = fieldDoc(strings.Replace(shape, `"uniqueItems":true`, `"uniqueItems":false`, 1))
	second, e2 := core.GenerateWithDiagnostics(rq)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("false structural check changed output", e1, e2)
	}
}
