package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nativeEnumUniqueGraduation(t *testing.T, input []byte) {
	t.Helper()
	r := request(input, "Dto")
	if _, err := core.GenerateWithDiagnostics(r); err == nil {
		t.Fatal("missing native uniqueness opt-in")
	}
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err != nil {
		t.Fatal("original native enum unique input did not graduate", err)
	}
}

func TestNativeEnumUniqueSources(t *testing.T) {
	for _, leaf := range []string{`{"type":"integer","enum":[1,2]}`, `{"type":"integer","format":"int64","enum":[-9223372036854775808,9223372036854775807]}`, `{"type":"boolean","enum":[true,false],"nullable":true}`, `{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2}`, `{"anyOf":[{"type":"integer","enum":[1,2]},{"type":"null"}]}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("direct native enum uniqueness rejected", leaf, err)
		}
		text := source(t, got, "RecordDto")
		if !strings.Contains(text, "`openapi-kt uniqueItems`(value)") || !strings.Contains(text, "does not belong to enum.") {
			t.Fatal("uniqueness or membership lost", text)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("native enum unique output drift", err)
		}
	}
	for _, ref := range []string{"Code", "Alias", "Owner/properties/code", "Owner/properties/hop", "Bridge"} {
		rq := request(schemas(`{"ARecord":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/`+ref+`","nullable":true}}}},"Alias":{"$ref":"#/components/schemas/Code"},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"},"Code":{"type":"integer","enum":[1,2]},"Owner":{"properties":{"code":{"type":"integer","enum":[1,2]},"hop":{"$ref":"#/components/schemas/Alias"}}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("existing native enum reference rejected uniqueness", ref, err)
		}
	}
}

func TestNativeEnumUniqueDefaults(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, values := range []string{`[0,1,null]`, `[0,-0]`, `[1,1]`, `[null,null]`, `[9]`} {
			shape := `{"type":"array","uniqueItems":true,"items":{"type":"integer","enum":[0,1,null],"nullable":true}`
			annotation, path := values, "/default/1"
			if values == `[9]` {
				path = "/default/0"
			}
			if nested {
				shape = `{"type":"array","items":` + shape + `}}`
				annotation = `[` + values + `]`
				path = strings.Replace(path, "/default/", "/default/0/", 1)
			} else {
				shape += "}"
			}
			shape = strings.TrimSuffix(shape, "}") + `,"default":` + annotation + `}`
			rq := request(fieldDoc(shape), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if values == `[0,1,null]` {
				if err != nil {
					t.Fatal("valid native default rejected", nested, err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+path {
				t.Fatal("native duplicate/membership default lost", nested, values, err)
			}
		}
	}
}

func TestNativeEnumUniqueOriginalGraduations(t *testing.T) {
	for _, leaf := range []string{`{"type":"integer","enum":[1,2]}`, `{"type":"boolean","enum":[true]}`} {
		nativeEnumUniqueGraduation(t, fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`))
	}
	nativeEnumUniqueGraduation(t, schemas(`{"Leaf":{"type":"integer","enum":[1]},"Record":{"type":"object","properties":{"value":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Leaf"}}}}}`))
	nativeEnumUniqueGraduation(t, schemas(`{"Code":{"type":"integer","enum":[1]},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}}}}`))
	for _, leaf := range []string{`{"type":"integer","enum":[1]}`, `{"type":"boolean","enum":[true]}`} {
		nativeEnumUniqueGraduation(t, httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"type":"array","uniqueItems":true,"items":`+leaf+`}}}}}}`))
	}
	nativeEnumUniqueGraduation(t, fieldDoc(`{"type":"array","items":{"type":"integer","enum":[1]},"uniqueItems":true,"default":[1]}`))
	nativeEnumUniqueGraduation(t, fieldDoc(`{"type":"array","items":{"type":"array","uniqueItems":true,"items":{"type":"integer","enum":[1]}},"default":[]}`))
}

func TestNativeEnumUniqueResourceBudgets(t *testing.T) {
	enumUniqueCheckedBudgets(t, `{"type":"integer","enum":[1]}`)
	enumUniqueSourceCacheBudgets(t, `{"type":["integer","null"],"enum":[1]}`)
}

func TestNativeEnumUniqueExclusionsNoDropAndNoop(t *testing.T) {
	for _, leaf := range []string{`{"type":"integer","const":1}`, `{"type":"string","x-native-type":"kotlin.Int"}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("unsupported source gained native equality", leaf, err)
		}
	}
	shape := `{"type":"array","uniqueItems":true,"items":{"type":"integer","enum":[1,2]}}`
	for _, doc := range [][]byte{
		schemas(`{"Record":{"properties":{"v":{"type":"string"}},"additionalProperties":` + shape + `}}`),
		schemas(`{"Record":{"allOf":[{"properties":{"v":` + shape + `}}],"properties":{"v":{"type":"integer"}}}}`),
		httpDoc(`{"parameters":[{"name":"q","in":"query","schema":{"properties":{"v":` + shape + `}}}],"responses":{"204":{}}}`),
	} {
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("native checks dropped", err)
		}
	}
	a, b := request(fieldDoc(strings.Replace(shape, `true`, `false`, 1)), "Dto"), request(fieldDoc(strings.Replace(shape, `,"uniqueItems":true`, "", 1)), "Dto")
	a.Options.Validation, b.Options.Validation = true, true
	first, e1 := core.GenerateWithDiagnostics(a)
	second, e2 := core.GenerateWithDiagnostics(b)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("false changed native artifacts", e1, e2)
	}
}

func TestNativeEnumUniqueOptimizationIdentity(t *testing.T) {
	shape := `{"properties":{"codes":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}}}`
	op := func(id, shape string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}`
	}
	doc := `{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2]},"Other":{"type":"integer","enum":[1,3]}}},"paths":{"/a":` + op("alpha", shape) + `,"/b":` + op("beta", shape) + `,"/check":` + op("check", strings.Replace(shape, `true`, `false`, 1)) + `,"/vocabulary":` + op("vocabulary", strings.Replace(shape, `schemas/Code`, `schemas/Other`, 1)) + `}}`
	for _, mode := range []string{"dedup", "merge", "hoist"} {
		rq := request([]byte(doc), "Dto")
		rq.Options.Validation = true
		rq.Options.DeduplicateModels = mode == "dedup"
		rq.Options.MergeIdenticalModels = mode == "merge"
		rq.Options.HoistSharedShapes = mode == "hoist"
		rq.Options.LockHoistedNames = mode == "hoist"
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(mode, err)
		}
		if !strings.Contains(source(t, got, "BetaResponseDto"), "public typealias") || strings.Contains(source(t, got, "CheckResponseDto"), "public typealias") || strings.Contains(source(t, got, "VocabularyResponseDto"), "public typealias") {
			t.Fatal("native membership/uniqueness identity lost", mode)
		}
		if mode == "hoist" {
			rq.PreviousNameLock = got.Artifacts.NameLock
			rq.Document = []byte(strings.ReplaceAll(doc, `"uniqueItems":true`, `"uniqueItems":false`))
			if changed, err := core.GenerateWithDiagnostics(rq); err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
				t.Fatal("frozen native unique check erased", err)
			}
		}
	}
}
