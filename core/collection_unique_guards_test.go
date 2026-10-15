package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCollectionUniqueGrammarAndEquality(t *testing.T) {
	for _, literal := range []string{`null`, `0`, `1`, `"false"`, `[]`, `{}`} {
		aliasFailure(t, fieldDoc(`{"type":"array","items":{"type":"string"},"uniqueItems":`+literal+`}`), "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/uniqueItems")
	}
	for _, kind := range []string{`{"type":"string"}`, `{"type":"object","additionalProperties":true}`, `{"type":"object","additionalProperties":{"type":"string"}}`} {
		for _, flag := range []string{"true", "false"} {
			aliasFailure(t, fieldDoc(strings.TrimSuffix(kind, "}")+`,"uniqueItems":`+flag+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/uniqueItems")
		}
	}
	for _, leaf := range []string{
		`{"type":"number"}`, `{"type":"number","format":"float"}`, `{"type":"array","items":{"type":"string"}}`,
		`{"type":"object","additionalProperties":{"type":"string"}}`,
		`{"type":"object","properties":{"v":{"type":"string"}}}`, `{"type":"integer","enum":[1,2]}`,
		`{"type":"boolean","enum":[true]}`, `{"type":"string","enum":["x"]}`, `{"type":"integer","const":1}`,
		`{"type":"string","format":"date"}`, `{"type":"string","format":"date-time"}`, `{"type":"string","format":"byte"}`,
		`{"type":"string","x-native-type":"bigint"}`, `{"type":"integer","x-kotlin-value-class":true}`,
	} {
		// Nominal value classes are component-only; validate that separately below.
		if strings.Contains(leaf, "x-kotlin-value-class") {
			continue
		}
		aliasFailure(t, fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/uniqueItems")
	}
	// Keep the original JsonObject input as an explicit new-profile regression.
	jsonValues := aggregationReport(t, fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":true}}`))
	if !strings.Contains(source(t, jsonValues, "Record"), "`openapi-kt unique JSON items`(value)") {
		t.Fatal("original JsonObject equality projection lost", jsonValues)
	}
	for _, leaf := range []string{`{"type":"integer","enum":[1]}`, `{"type":"integer","const":1}`, `{"type":"string","x-native-type":"bigint"}`, `{"type":"integer","x-kotlin-value-class":true}`} {
		aliasFailure(t, schemas(`{"Leaf":`+leaf+`,"Record":{"type":"object","properties":{"value":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Leaf"}}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/uniqueItems")
	}
	for _, leaf := range []string{`{"type":"string"}`, `{"type":"integer"}`, `{"type":"integer","format":"int64"}`, `{"type":"boolean"}`, `{"type":["boolean","null"]}`, `{"anyOf":[{"type":"integer"},{"type":"null"}]}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`), "")
		rq.Options.Validation = true
		if _, err := core.Generate(rq); err != nil {
			t.Fatal(leaf, err)
		}
	}
	for _, flag := range []string{"true", "false"} {
		if _, err := core.Generate(request(fieldDoc(`{"type":"array","uniqueItems":`+flag+`,"items":{"type":"string"}}`), "")); err == nil {
			t.Fatal("missing opt-in accepted", flag)
		}
	}
	// Root component arrays remain unsupported.
	// Root array schemas now retain the same eligible scalar uniqueness profile.
	aggregationReport(t, schemas(`{"Root":{"type":"array","uniqueItems":true,"items":{"type":"string"}}}`))
}

func TestCollectionUniqueFalseNoop(t *testing.T) {
	for _, leaf := range []string{`{"type":"string"}`, `{"type":"number"}`, `{"type":"array","items":{"type":"string"}}`, `{"type":"object","additionalProperties":true}`, `{"type":"object","properties":{"v":{"type":"string"}}}`} {
		plain := fieldDoc(`{"type":"array","items":` + leaf + `}`)
		noop := fieldDoc(`{"type":"array","uniqueItems":false,"items":` + leaf + `}`)
		a, b := request(plain, ""), request(noop, "")
		a.Options.Validation = true
		b.Options.Validation = true
		before, e1 := core.GenerateWithDiagnostics(a)
		after, e2 := core.GenerateWithDiagnostics(b)
		if e1 != nil || e2 != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("false changed artifacts/diagnostics", leaf, e1, e2)
		}
	}
	aliasFailure(t, fieldDoc(`{"type":"array","uniqueItems":false,"items":{"not":{}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/items/not")
}

func TestCollectionUniqueNoDrop(t *testing.T) {
	for _, flag := range []string{"true", "false"} {
		shape := `{"type":"array","uniqueItems":` + flag + `,"items":{"type":"string"}}`
		extra := schemas(`{"Record":{"type":"object","properties":{"v":{"type":"string"}},"additionalProperties":` + shape + `}}`)
		discarded := schemas(`{"Record":{"allOf":[{"type":"object","properties":{"v":` + shape + `}}],"type":"object","properties":{"v":{"type":"array","items":{"type":"string"}}}}}`)
		raw := httpDoc(`{"parameters":[{"name":"q","in":"query","schema":{"type":"object","properties":{"rows":` + shape + `}}}],"responses":{"204":{}}}`)
		for i, doc := range [][]byte{extra, discarded, raw} {
			if flag == "true" {
				pointer := []string{"/components/schemas/Record/additionalProperties/uniqueItems", "/components/schemas/Record/allOf/0/properties/v/uniqueItems", "/paths/~1x/get/parameters/0/schema/properties/rows/uniqueItems"}[i]
				// httpDoc uses its own path name; extract it instead of assuming one.
				if i == 2 {
					var parsed map[string]json.RawMessage
					json.Unmarshal(doc, &parsed)
					var paths map[string]json.RawMessage
					json.Unmarshal(parsed["paths"], &paths)
					for path := range paths {
						pointer = "/paths/" + strings.ReplaceAll(path, "/", "~1") + "/get/parameters/0/schema/properties/rows/uniqueItems"
					}
				}
				aliasFailure(t, doc, "E_SCHEMA_UNSUPPORTED", pointer)
			} else {
				rq := request(doc, "")
				rq.Options.Validation = true
				if _, err := core.Generate(rq); err != nil {
					t.Fatal("false erased-source no-op rejected", i, err)
				}
			}
		}
	}
	aliasFailure(t, schemas(`{"Unused":{"type":"object","properties":{"v":{"type":"array","uniqueItems":true,"items":{"type":"number"}}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Unused/properties/v/uniqueItems")
}

func TestCollectionUniqueDepthSlotsAndIdentity(t *testing.T) {
	for _, edges := range []int{16, 17} {
		shape := `{"type":"array","uniqueItems":true,"items":{"type":"string"}}`
		for i := 1; i < edges; i++ {
			shape = `{"type":"array","items":` + shape + `}`
		}
		rq := request(fieldDoc(shape), "")
		rq.Options.Validation = true
		if edges == 16 {
			if _, err := core.Generate(rq); err != nil {
				t.Fatal(err)
			}
		} else {
			aliasFailure(t, rq.Document, "E_SCHEMA_DEPTH", "/components/schemas/Record/properties/value"+strings.Repeat("/items", 16))
		}
	}
	for _, count := range []int{240, 241} {
		var fields []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"array","uniqueItems":true,"items":{"type":"boolean"}}`, i))
		}
		rq := request(schemas(`{"Record":{"type":"object","properties":{`+strings.Join(fields, ",")+`}}}`), "")
		rq.Options.Validation = true
		if count == 240 {
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err)
			}
			text := source(t, r, "Record")
			if strings.Count(text, "private fun <T : kotlin.Comparable<T>>") != 1 || strings.Count(text, ".sorted()") != 1 || strings.Count(text, "`openapi-kt uniqueItems`(value)") != 240 {
				t.Fatal("wide uniqueness expansion must stay behind a shared non-inline helper", text)
			}
		} else {
			aliasFailure(t, rq.Document, "E_MODEL_LIMIT", "/components/schemas/Record")
		}
	}
	shape := `{"type":"object","properties":{"v":{"type":"array","uniqueItems":true,"items":{"type":"string"}}}}`
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":{"get":{"tags":["Left"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}},"/b":{"get":{"tags":["Right"],"operationId":"shared","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}}}`)
	rq := request(doc, "Dto")
	rq.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("source provenance changed identity", err)
	}
	if !strings.Contains(source(t, r, "SharedResponseDto"), "uniqueItems") {
		t.Fatal("shared shape lost uniqueness")
	}
}

func TestCollectionUniqueOperationProjectionSources(t *testing.T) {
	for _, leaf := range []string{`{"type":"integer","enum":[1]}`, `{"type":"boolean","enum":[true]}`, `{"type":"string","enum":["x"]}`, `{"type":"integer","const":1}`, `{"type":"string","x-native-type":"bigint"}`} {
		shape := `{"type":"array","uniqueItems":true,"items":` + leaf + `}`
		aliasFailure(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":`+shape+`}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/responses/200/content/application~1json/schema/uniqueItems")
	}
	// A false no-op cannot excuse malformed erased-source arrays or wrong-kind keys.
	for _, tc := range []struct{ shape, suffix, code string }{
		{`{"type":"array","uniqueItems":false,"items":{"not":{}}}`, "/items/not", "E_SCHEMA_UNSUPPORTED"},
		{`{"type":"array","uniqueItems":"false","items":{"type":"string"}}`, "/uniqueItems", "E_SCHEMA_INVALID"},
		{`{"type":"string","uniqueItems":false}`, "/uniqueItems", "E_SCHEMA_UNSUPPORTED"},
	} {
		aliasFailure(t, httpDoc(`{"parameters":[{"name":"q","in":"query","schema":{"type":"object","properties":{"v":`+tc.shape+`}}}],"responses":{"204":{}}}`), tc.code, "/paths/~1ping/get/parameters/0/schema/properties/v"+tc.suffix)
	}
}

func TestCollectionUniqueAliasDepthAndNominalBarrier(t *testing.T) {
	deep := `{"type":"array","uniqueItems":true,"items":{"type":"string"}}`
	for i := 0; i < 15; i++ {
		deep = `{"type":"object","additionalProperties":` + deep + `}`
	}
	// Use-site list adds the 17th collection edge; transparent Bag cannot reset it.
	aliasFailure(t, schemas(`{"Bag":`+deep+`,"Record":{"type":"object","properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Bag"}}}}}`), "E_SCHEMA_DEPTH", "/components/schemas/Bag"+strings.Repeat("/additionalProperties", 15))
	// A nominal Child constructor is a separate barrier, even through recursive refs.
	child := `{"type":"object","properties":{"v":{"type":"array","uniqueItems":true,"items":{"type":"string"}},"next":{"$ref":"#/components/schemas/Child"}}}`
	ty := `{"$ref":"#/components/schemas/Child"}`
	for i := 0; i < 16; i++ {
		ty = `{"type":"array","items":` + ty + `}`
	}
	rq := request(schemas(`{"Child":`+child+`,"Record":{"type":"object","properties":{"v":`+ty+`}}}`), "")
	rq.Options.Validation = true
	if _, err := core.Generate(rq); err != nil {
		t.Fatal("nominal constructor should reset unique validation depth", err)
	}
}
