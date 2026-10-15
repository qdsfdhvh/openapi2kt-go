package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumConsumersRetainCheckedCollectionDepthAndAliasSourceWork(t *testing.T) {
	for _, depth := range []int{16, 17} {
		leaf := `{"type":["integer","null"],"enum":[1,2]}`
		for range depth {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		r := request(fieldDoc(leaf), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if (err == nil) != (depth == 16) {
			t.Fatal(depth, err)
		}
		if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !strings.HasSuffix(*err.(core.Diagnostics)[0].Pointer, "/items") || len(got.Artifacts.Files) != 0) {
			t.Fatal(depth, err)
		}
	}
	for _, length := range []int{46, 47} {
		parts := []string{`"Code":{"type":"integer","enum":[1,2]}`}
		for i := range length {
			target := "Code"
			if i > 0 {
				target = fmt.Sprintf("A%d", i-1)
			}
			parts = append(parts, fmt.Sprintf(`"A%d":{"$ref":"#/components/schemas/%s"}`, i, target))
		}
		parts = append(parts, fmt.Sprintf(`"Record":{"properties":{"code":{"$ref":"#/components/schemas/A%d"}}}`, length-1))
		r := request(schemas("{"+strings.Join(parts, ",")+"}"), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if (err == nil) != (length == 46) {
			t.Fatal(length, err)
		}
		if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(length, err)
		}
	}
}

func TestNativeEnumConsumersKeepUnsupportedAnnotationsAndAliasCycle(t *testing.T) {
	for _, body := range []string{`{"type":"integer","enum":[1],"default":1}`, `{"type":"integer","enum":[null]}`, `{"type":"integer","enum":[1],"x-native-type":"bigint"}`} {
		r := request(schemas(`{"Code":`+body+`}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(body, err)
		}
	}
	// Original enum+minimum component is now admitted with both checks retained
	// in a real typed consumer; its transparent alias remains unchanged.
	graduated := request(schemas(`{"Code":{"type":"integer","enum":[1],"minimum":0},"Record":{"properties":{"value":{"$ref":"#/components/schemas/Code"}}}}`), "Dto")
	graduated.Options.Validation = true
	accepted, e := core.GenerateWithDiagnostics(graduated)
	if e != nil || !strings.Contains(source(t, accepted, "RecordDto"), "value == 1") || !strings.Contains(source(t, accepted, "RecordDto"), "value >= 0") {
		t.Fatal(accepted, e)
	}
	r := request(schemas(`{"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"}}`), "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_CYCLE" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal(err)
	}
}
