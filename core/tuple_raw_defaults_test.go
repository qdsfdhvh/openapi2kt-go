package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRawTupleOwnDefaults(t *testing.T) {
	for _, tail := range []string{"", `,"items":true`, `,"items":false`} {
		for _, checked := range []bool{false, true} {
			input := schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}]` + tail + `,"default":[null,true,{"default":{"$ref":"data"}},[1.0,"1"]]},"Alias":{"$ref":"#/components/schemas/Rows","default":[false]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}]` + tail + `,"default":[null,{"description":"data"}]}}}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(tail, checked, err)
			}
			if !strings.Contains(source(t, report, "RowsDto"), "List<kotlinx.serialization.json.JsonElement>") || !strings.Contains(source(t, report, "HolderDto"), "openapi-kt default array JsonElement") {
				t.Fatal("raw tuple default lost its actual List carrier/helper")
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(report, again) {
				t.Fatal("raw tuple default nondeterminism", err)
			}
		}
	}
}

func TestRawTupleReferencePrefixDefaultSources(t *testing.T) {
	for _, tail := range []string{"", `,"items":true`, `,"items":false`} {
		input := schemas(`{"Item":{"properties":{"name":{"type":"string"}},"required":["name"]},"Choice":{"type":"string","enum":["ready"]},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"},{"$ref":"#/components/schemas/Choice"}]` + tail + `,"default":[true,null]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"},{"$ref":"#/components/schemas/Choice"}]` + tail + `,"default":[{}]}}}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatal("raw annotation lost admitted original prefix reference authority", tail, err)
		}
	}
}

func TestRawTupleAliasDefaultsRetainNearestNullableIntent(t *testing.T) {
	input := schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"string"}],"items":true,"nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Rows","nullable":false,"default":[null]},"Outer":{"$ref":"#/components/schemas/Narrow","nullable":true,"default":null},"Independent":{"$ref":"#/components/schemas/Rows"}}`)
	if _, err := core.Generate(request(input, "Dto")); err != nil {
		t.Fatal("raw tuple alias defaults lost nearest nullable intent", err)
	}
	input = []byte(strings.Replace(string(input), `"nullable":false,"default":[null]`, `"nullable":false,"default":null`, 1))
	failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Narrow/default")
}

func TestRawTupleDefaultStrictValuesAndSuppression(t *testing.T) {
	for _, required := range []string{"", `,"required":["rows"]`} {
		for _, value := range []string{`null`, `{}`, `true`, `1`, `"value"`} {
			input := schemas(`{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"type":"string"}],"items":true,"default":` + value + `}}` + required + `}}`)
			failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/rows/default")
		}
	}
	for _, root := range []bool{false, true} {
		for _, value := range []string{`[]`, `null`} {
			schema := `{"type":["array","null"],"prefixItems":[{"type":"integer","unknown":true}],"items":true,"default":` + value + `}`
			input := schemas(`{"Rows":` + schema + `}`)
			pointer := "/components/schemas/Rows"
			if !root {
				input = schemas(`{"Holder":{"properties":{"rows":` + schema + `}}}`)
				pointer = "/components/schemas/Holder/properties/rows"
			}
			report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Pointer == nil || !strings.HasPrefix(*err.(core.Diagnostics)[0].Pointer, pointer) {
				t.Fatal("raw empty/null annotation hid original invalid prefix", root, value, err)
			}
		}
	}
}

func TestRawTupleDefaultExactJSONUniqueness(t *testing.T) {
	for _, value := range []string{`[null,null]`, `[1,1.0]`, `[0,-0e+5]`, `[{"a":1,"b":2},{"b":2.0,"a":1e0}]`, `[[1,null],[1.0,null]]`} {
		for _, root := range []bool{false, true} {
			schema := `{"type":"array","prefixItems":[{"type":"string"}],"items":true,"uniqueItems":true,"default":` + value + `}`
			input, pointer := schemas(`{"Rows":`+schema+`}`), "/components/schemas/Rows/default/1"
			if !root {
				input, pointer = schemas(`{"Holder":{"properties":{"rows":`+schema+`}}}`), "/components/schemas/Holder/properties/rows/default/1"
			}
			rq := request(input, "Dto")
			rq.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != pointer {
				t.Fatal("raw tuple annotation lost exact JSON duplicate check", value, root, err)
			}
		}
	}
}

func TestRawTupleDefaultOriginalBudgets(t *testing.T) {
	for _, count := range []int{16383, 16384} {
		value := "[" + strings.TrimSuffix(strings.Repeat("0,", count), ",") + "]"
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"string"}],"items":true,"default":`+value+`}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(rq)
		if count == 16383 && err != nil || count == 16384 && (err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("raw tuple reset default value budget", count, err)
		}
	}
	for _, size := range []int{32764, 32765} {
		input := schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"integer"}],"items":true,"default":["` + strings.Repeat("x", size) + `"]}}`)
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if size == 32764 && err != nil || size == 32765 && (err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("raw tuple reset default byte budget", size, err)
		}
	}
}

func TestRawTupleFormerRootRefusals(t *testing.T) {
	for _, schema := range []string{
		`{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"default":[]}`,
		`{"type":"array","prefixItems":[{"type":"integer"}],"items":true,"default":[]}`,
	} {
		if _, err := core.Generate(request(schemas(`{"Rows":`+schema+`}`), "Dto")); err != nil {
			t.Fatal("former raw factory refusal remains", err)
		}
	}
}

func TestRawTupleDefaultJSONUniqueNumbers(t *testing.T) {
	for _, literal := range []string{"1e-513", strings.Repeat("1", 257)} {
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"string"}],"items":true,"uniqueItems":true,"default":[`+literal+`]}}`), "Dto")
		rq.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/default/0" {
			t.Fatal("singleton default escaped JSON unique numeric limits", literal, err)
		}
	}
}

func TestRawTupleDefaultLockedInitializerIdentity(t *testing.T) {
	operation := `{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"rows":{"type":"array","prefixItems":[{"type":"string"}],"items":true,"default":[null]}}}}}},"responses":{"204":{}}}}`
	rq := request([]byte(`{"openapi":"3.1.0","paths":{"/one":`+operation+`,"/two":`+operation+`}}`), "Dto")
	rq.Options.HoistSharedShapes, rq.Options.LockHoistedNames = true, true
	first, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(first.Artifacts.NameLock) == 0 || strings.Contains(string(first.Artifacts.NameLock), `"entries":[]`) {
		t.Fatal("raw initializer probe did not create actual locked allocations", err)
	}
	rq.PreviousNameLock = first.Artifacts.NameLock
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("same raw initializer failed locked replay", err)
	}
	rq.Document = []byte(strings.ReplaceAll(string(rq.Document), `"default":[null]`, `"default":[true]`))
	if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" {
		t.Fatal("changed effective raw initializer reused a frozen shape", err)
	}
}
