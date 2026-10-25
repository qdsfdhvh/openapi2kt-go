package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleFieldDefaults(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/tuple-field-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			first, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(source(t, first, "HolderDto"), "openapi-kt default array") {
				t.Fatal("tuple field lost typed scalar helper")
			}
			second, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(first, second) {
				t.Fatal("tuple default nondeterminism", err)
			}
		}
	}
}

func TestTupleFieldDefaultsTailAndOriginalValuePointers(t *testing.T) {
	valid := schemas(`{"Holder":{"properties":{"values":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"items":{"type":"integer"},"default":[1,2]}}}}`)
	if _, err := core.Generate(request(valid, "Dto")); err != nil {
		t.Fatal("validated explicit tail should own default element type", err)
	}
	for _, extra := range []string{"", `,"required":["values"]`} {
		input := schemas(`{"Holder":{"properties":{"values":{"type":"array","prefixItems":[{"type":"integer"}],"default":[1,"bad"]}}` + extra + `}}`)
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/values/default/1")
	}
	for _, value := range []string{`[]`, `null`} {
		input := schemas(`{"Holder":{"properties":{"values":{"type":["array","null"],"prefixItems":[{"type":"string","format":"unsupported"}],"default":` + value + `}}}}`)
		failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/values/prefixItems/0")
	}
	input := schemas(`{"Holder":{"properties":{"values":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"default":[]}}}}`)
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/values/prefixItems")
	input = schemas(`{"Holder":{"properties":{"values":{"type":"array","prefixItems":[{"type":"string"}],"default":["` + strings.Repeat("x", 32769) + `"]}}}}`)
	failure(t, input, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/values/default/0")
}

func TestTupleFieldDefaultsNodeCountAndMappedValueChecks(t *testing.T) {
	for _, count := range []int{16383, 16384} {
		values := strings.TrimSuffix(strings.Repeat("0,", count), ",")
		rq := request(schemas(`{"Holder":{"properties":{"values":{"type":"array","prefixItems":[{"type":"integer"}],"default":[`+values+`]}}}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if count == 16383 && err != nil {
			t.Fatal("exact value node/byte budget rejected", err)
		}
		if count == 16384 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("tuple default escaped node/byte budget", err)
		}
	}
	for _, field := range []string{
		`{"type":"array","prefixItems":[{"type":"integer"}],"default":[],"minItems":1}`,
		`{"type":"array","prefixItems":[{"type":"integer"}],"items":{"type":"integer"},"default":[0,-0],"uniqueItems":true}`,
		`{"type":"array","prefixItems":[{"type":"string","format":"uuid"}],"items":{"type":"string","format":"uuid"},"default":["00112233445566778899aabbccddeeff","00112233-4455-6677-8899-AABBCCDDEEFF"],"uniqueItems":true}`,
		`{"type":"array","prefixItems":[{"type":"string","format":"date-time"}],"items":{"type":"string","format":"date-time"},"default":["2026-01-02T03:04:05Z","2026-01-02T05:04:05+02:00"],"uniqueItems":true}`,
	} {
		rq := request(schemas(`{"Holder":{"properties":{"values":`+field+`}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || !strings.Contains(*err.(core.Diagnostics)[0].Pointer, "/default") || len(got.Artifacts.Files) != 0 {
			t.Fatal("tuple default lost mapped-value checks", field, err)
		}
	}
}

func TestTupleFieldDefaultPrefixOnlyUniqueRemainsSeparate(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"values":{"type":"array","prefixItems":[{"type":"integer"}],"default":[1],"uniqueItems":true}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/values/uniqueItems" || len(got.Artifacts.Files) != 0 {
		t.Fatal("tuple default granted new prefix-only uniqueness source role", err)
	}
}
