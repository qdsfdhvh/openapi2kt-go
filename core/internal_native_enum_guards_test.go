package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInternalNativeEnumPointersKeepNullViewsAndOriginalDiagnostics(t *testing.T) {
	for _, use := range []string{pointerRef("Source/properties/leaf"), `{"$ref":"#/components/schemas/Source/properties/leaf","nullable":false}`, `{"allOf":[{"$ref":"#/components/schemas/Source/properties/leaf"}],"nullable":false}`} {
		r := request(schemas(`{"Consumer":{"properties":{"v":`+use+`},"required":["v"]},"Source":{"properties":{"leaf":{"type":["integer","null"],"enum":[1,2],"minimum":2}}}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(use, err)
		}
		text := source(t, got, "ConsumerDto")
		if !strings.Contains(text, "value >= 2") || !strings.Contains(text, "value == 1 || value == 2") {
			t.Fatal(text)
		}
		narrowed := strings.Contains(use, `"nullable":false`)
		if strings.Contains(text, "v: kotlin.Int?") == narrowed {
			t.Fatal("required nullable view lost", use, text)
		}
	}
	for _, tc := range []struct{ leaf, code, tail string }{
		{`{"type":"integer","enum":[1,1]}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"type":"integer","enum":[1,2],"minimum":3}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":"integer","enum":[1],"default":1}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"type":"number","enum":[1],"multipleOf":0.5}`, "E_SCHEMA_UNSUPPORTED", "multipleOf"},
		{`{"type":"boolean","enum":[true],"minimum":0}`, "E_SCHEMA_UNSUPPORTED", "minimum"},
		{`{"type":"integer","enum":[null]}`, "E_SCHEMA_INVALID", "enum/0"},
	} {
		r := request(schemas(`{"ABridge":{"$ref":"#/components/schemas/Source/properties/leaf"},"Source":{"properties":{"leaf":`+tc.leaf+`}}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Source/properties/leaf/"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
	}
}

func TestInternalNativeEnumPointersRetainSourceAndCheckedContainerDepth(t *testing.T) {
	for _, count := range []int{45, 46} {
		fields := []string{`"leaf":{"type":["integer","null"],"enum":[1,2],"minimum":2}`}
		for i := range count {
			target := "Source/properties/leaf"
			if i > 0 {
				target = fmt.Sprintf("Source/properties/h%d", i-1)
			}
			fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
		}
		r := request(schemas(`{"Source":{"properties":{`+strings.Join(fields, ",")+`}},"Bridge":{"$ref":"#/components/schemas/Source/properties/h`+fmt.Sprint(count-1)+`","nullable":false},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if (err == nil) != (count == 45) {
			t.Fatal(count, err)
		}
		if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(err)
		}
	}
	for _, depth := range []int{16, 17} {
		leaf := pointerRef("Source/properties/leaf")
		for range depth {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		r := request(pointerDoc(`{"type":"integer","enum":[1,2]}`, leaf), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if (err == nil) != (depth == 16) {
			t.Fatal(depth, err)
		}
		if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(err)
		}
	}
}
