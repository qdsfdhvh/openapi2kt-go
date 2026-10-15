package core_test

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuredConstValueAliasesRetainCachedSourceHeightAndBodySlots(t *testing.T) {
	for _, width := range []int{240, 241} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Value"}`, i))
		}
		input := schemas(`{"Value":{"type":"array","const":[null]},"Holder":{"properties":{` + strings.Join(fields, ",") + `}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if (err == nil) != (width == 240) || err != nil && (err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Referenced structured const changed serialized body budget", width, err)
		}
	}
	for _, hops := range []int{46, 47} {
		components := []string{`"Base":{"type":"array","const":[]}`}
		previous := "Base"
		for i := range hops {
			name := fmt.Sprintf("Alias%d", i)
			components = append(components, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s"}`, name, previous))
			previous = name
		}
		components = append(components, fmt.Sprintf(`"Holder":{"properties":{"fixed":{"$ref":"#/components/schemas/%s"}}}`, previous))
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(components, ",")+`}`), "Dto"))
		if (err == nil) != (hops == 46) || err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Structured const reset erased alias source work", hops, err)
		}
	}
}

func TestStructuredConstValueJsonClassifierMethodErasureMatchesActualBacking(t *testing.T) {
	op := func(schema string) string {
		return `{"post":{"operationId":"send","requestBody":{"required":true,"content":{"application/json":{"schema":` + schema + `}}},"responses":{"204":{}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Object":{"const":{}},"Array":{"type":"array","const":[]},"NativeObject":{"type":"string","x-native-type":"kotlinx.serialization.json.JsonObject"},"NativeArray":{"type":"string","x-native-type":"kotlinx.serialization.json.JsonArray"}}},"paths":{"/a":` + op(`{"$ref":"#/components/schemas/Object"}`) + `,"/b":` + op(`{"$ref":"#/components/schemas/NativeObject"}`) + `,"/c":` + op(`{"$ref":"#/components/schemas/Array"}`) + `,"/d":` + op(`{"$ref":"#/components/schemas/NativeArray"}`) + `}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	api := source(t, got, "DefaultApi")
	methods := regexp.MustCompile(`public suspend fun ([A-Za-z0-9_]+)\(\s+input: kotlinx.serialization.json.(JsonObject|JsonArray),`).FindAllStringSubmatch(api, -1)
	seen := map[string]bool{}
	if len(methods) != 4 {
		t.Fatal("Missing structured/native JSON overloads", api)
	}
	for _, method := range methods {
		key := method[1] + ":" + method[2]
		if seen[key] {
			t.Fatal("Actual same JSON classifier caused overloaded signatures", api)
		}
		seen[key] = true
	}
	if strings.Count(api, "input: kotlinx.serialization.json.JsonArray,") != 2 || strings.Count(api, "input: kotlinx.serialization.json.JsonObject,") != 2 {
		t.Fatal("Structured JSON overload escaped into raw/list classifiers", api)
	}
}
