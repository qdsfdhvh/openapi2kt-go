package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedStringEnumUniqueSources(t *testing.T) {
	// Preserve the original direct-reference negative as an opt-in graduation.
	stringEnumUniqueGraduation(t, schemas(`{"Code":{"enum":["ready"]},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}}}}`))
	for _, ref := range []string{"Code", "Alias", "Owner/properties/code", "Owner/properties/alias", "Bridge"} {
		input := schemas(`{"ARecord":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/` + ref + `","nullable":true},"uniqueItems":true}}},"Alias":{"$ref":"#/components/schemas/Code"},"Bridge":{"$ref":"#/components/schemas/Owner/properties/alias"},"Code":{"enum":["ready","done"]},"Owner":{"properties":{"code":{"enum":["ready","done"]},"alias":{"$ref":"#/components/schemas/Code"}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("existing enum reference rejected uniqueness", ref, err)
		}
		text := source(t, got, "ARecordDto")
		if !strings.Contains(text, "`openapi-kt uniqueItems`(value.map { it?.serialName })") {
			t.Fatal("actual enum payload check lost", text)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("reference uniqueness output drift", ref, err)
		}
		rq.Options.Validation = false
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("missing validation opt-in", ref, err)
		}
	}
}

func TestReferencedStringEnumUniqueDefaults(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, values := range []string{`["ready",null,"null"]`, `["ready","\u0072eady"]`, `[null,null]`, `["future"]`} {
			rows := `{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"uniqueItems":true}`
			annotation, path := values, "/default/1"
			if values == `["future"]` {
				path = "/default/0"
			}
			if nested {
				rows = `{"type":"array","items":` + rows + `}`
				annotation = `[` + values + `]`
				path = strings.Replace(path, "/default/", "/default/0/", 1)
			}
			rq := request(schemas(`{"ARecord":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":`+annotation+`}}},"Rows":`+rows+`,"Alias":{"$ref":"#/components/schemas/Code"},"Code":{"enum":["ready","null",null],"nullable":true}}`), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if values == `["ready",null,"null"]` {
				if err != nil {
					t.Fatal("valid alias-backed default rejected", nested, err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/ARecord/properties/value"+path {
				t.Fatal("default equality/membership/source lost", nested, values, err)
			}
		}
	}
}

func TestReferencedStringEnumUniqueCheckedBudgets(t *testing.T) {
	enumUniqueCheckedBudgets(t, `{"enum":["ready"]}`)
}

func enumUniqueCheckedBudgets(t *testing.T, terminal string) {
	t.Helper()
	for _, edges := range []int{16, 17} {
		field := `{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}`
		for i := 1; i < edges; i++ {
			field = `{"type":"array","items":` + field + `}`
		}
		rq := request(schemas(`{"Code":`+terminal+`,"Record":{"properties":{"value":`+field+`}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (edges == 16) || err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("enum reference checked edge budget lost", edges, err)
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}`, i)
		}
		rq := request(schemas(`{"Code":`+terminal+`,"Record":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (width == 240) || err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("enum reference slot budget lost", width, err)
		}
	}
}

func TestReferencedStringEnumUniqueSourceCacheBudgets(t *testing.T) {
	enumUniqueSourceCacheBudgets(t, `{"type":["string","null"],"enum":["ready"]}`)
}

func enumUniqueSourceCacheBudgets(t *testing.T, terminal string) {
	t.Helper()
	for _, cached := range []bool{false, true} {
		for _, hops := range []int{42, 43, 44} {
			props := []string{`"terminal":{"$ref":"#/components/schemas/Code"}`}
			next := "terminal"
			for i := 0; i < hops; i++ {
				key := fmt.Sprintf("h%d", i)
				props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
				next = key
			}
			props = append(props, `"rows":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Source/properties/`+next+`","nullable":false}}`)
			field, root := `{"$ref":"#/components/schemas/Source/properties/rows"}`, ""
			if cached {
				root = `"Bridge":{"$ref":"#/components/schemas/Source/properties/rows","nullable":false},`
				field = `{"$ref":"#/components/schemas/Bridge","nullable":false}`
			}
			rq := request(schemas(`{"Code":`+terminal+`,"Source":{"properties":{`+strings.Join(props, ",")+`}},`+root+`"Holder":{"properties":{"value":`+field+`},"required":["value"]}}`), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if hops <= 42 || hops == 43 && !cached {
				if err != nil {
					t.Fatal("original source48 rejected", cached, hops, err)
				}
			} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
				t.Fatal("enum uniqueness erased alias source work", cached, hops, err)
			}
		}
	}
}

func TestReferencedStringEnumUniqueNoDropAndNoop(t *testing.T) {
	shape := `{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}`
	prefix := `"Code":{"enum":["ready"]},`
	aliasFailure(t, schemas(`{`+prefix+`"Record":{"properties":{"v":{"type":"string"}},"additionalProperties":`+shape+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/additionalProperties/uniqueItems")
	aliasFailure(t, schemas(`{`+prefix+`"Record":{"allOf":[{"properties":{"v":`+shape+`}}],"properties":{"v":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/v/uniqueItems")
	base := schemas(`{` + prefix + `"Record":{"properties":{"value":` + strings.Replace(shape, `true`, `false`, 1) + `}}}`)
	plain := []byte(strings.Replace(string(base), `,"uniqueItems":false`, "", 1))
	a, b := request(base, "Dto"), request(plain, "Dto")
	a.Options.Validation, b.Options.Validation = true, true
	first, e1 := core.GenerateWithDiagnostics(a)
	second, e2 := core.GenerateWithDiagnostics(b)
	if e1 != nil || e2 != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("false changed reference artifacts", e1, e2)
	}
	for _, leaf := range []string{`{"type":"string","enum":["ready"],"format":"date"}`, `{"type":"string","enum":["ready"],"minLength":1}`, `{"type":"string","const":"ready"}`, `{"type":"string","x-kotlin-value-class":true}`, `{"type":"string","x-native-type":"org.example.State"}`, `{"anyOf":[{"enum":["ready"]},{"type":"string"}]}`} {
		rq := request(schemas(`{"Code":`+leaf+`,"Record":{"properties":{"value":`+shape+`}}}`), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("excluded reference source acquired enum equality", leaf, err)
		}
	}
}

func TestReferencedStringEnumUniqueOptimizationIdentity(t *testing.T) {
	shape := `{"properties":{"codes":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}}}`
	op := func(id, shape string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}`
	}
	doc := `{"openapi":"3.1.0","components":{"schemas":{"Code":{"enum":["ready","done"]},"Other":{"enum":["ready","done"]}}},"paths":{"/a":` + op("alpha", shape) + `,"/b":` + op("beta", shape) + `,"/different":` + op("different", strings.Replace(shape, `true`, `false`, 1)) + `,"/nominal":` + op("nominal", strings.Replace(shape, `schemas/Code`, `schemas/Other`, 1)) + `}}`
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
		if !strings.Contains(source(t, got, "BetaResponseDto"), "public typealias") || strings.Contains(source(t, got, "DifferentResponseDto"), "public typealias") || strings.Contains(source(t, got, "NominalResponseDto"), "public typealias") != (mode == "merge") {
			t.Fatal("constraint/nominal identity lost", mode)
		}
		if mode == "hoist" {
			rq.PreviousNameLock = got.Artifacts.NameLock
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("frozen reference identity drift", err)
			}
			rq.Document = []byte(strings.ReplaceAll(doc, `"uniqueItems":true`, `"uniqueItems":false`))
			if changed, err := core.GenerateWithDiagnostics(rq); err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
				t.Fatal("frozen uniqueness check erased", err)
			}
		}
	}
}
