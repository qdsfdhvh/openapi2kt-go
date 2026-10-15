package core_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuredConstLiteralDataRetainsDocumentDepthAndInputBoundaries(t *testing.T) {
	for _, depth := range []int{120, 121} {
		literal := "null"
		for range depth {
			literal = "[" + literal + "]"
		}
		input := fieldDoc(`{"const":{"deep":` + literal + `}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if (err == nil) != (depth == 120) || err != nil && (err.(core.Diagnostics)[0].Code != "E_JSON_INVALID" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Literal data reset the document JSON depth budget", depth, err)
		}
	}
	input := fieldDoc(`{"const":{"key":"literal"}}`)
	input = append(input, bytes.Repeat([]byte{' '}, core.MaxDocumentBytes-len(input))...)
	if _, err := core.Generate(request(input, "Dto")); err != nil {
		t.Fatal("Structured literal lost exact 8MiB input", err)
	}
	got, err := core.GenerateWithDiagnostics(request(append(input, ' '), "Dto"))
	if err == nil || err.(core.Diagnostics)[0].Code != "E_INPUT_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("Structured literal bypassed document input budget", err)
	}
	for _, literal := range []string{`{"duplicate":1,"duplicate":2}`, `{"number":1e400}`} {
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(`{"const":`+literal+`}`), "Dto"))
		if err == nil || err.(core.Diagnostics)[0].Code != "E_JSON_INVALID" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Literal duplicate/range checks disappeared", literal, err)
		}
	}
}

func TestStructuredConstLiteralNumbersAndEscapedKeysAreData(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"v$/~":{"const":{"n":1e2,"large":9007199254740993,"$ref":"#/not-a-schema","default":false,"type":["array"],"properties":{"key":{"const":[null]}}}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || len(got.Warnings) != 1 || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/Holder/properties/v$~1~0/const" {
		t.Fatal("Structured literal keys were reinterpreted as schemas", err, got.Warnings)
	}
	if !strings.Contains(source(t, got, "HolderDto"), "1e2") || !strings.Contains(source(t, got, "HolderDto"), "9007199254740993") {
		t.Fatal("Numeric literal text changed before backing JSON codec")
	}
}
