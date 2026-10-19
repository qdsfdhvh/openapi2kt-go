package core_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBytesDefaultTreesKeepOriginalGraduatedInputsAndTypedWire(t *testing.T) {
	for _, field := range []string{
		// These exact original inputs graduated from the Map and temporal tests.
		`{"type":"object","additionalProperties":{"type":"string","format":"byte"},"default":{}}`,
		`{"type":"array","items":{"type":"string","format":"byte"},"default":[]}`,
		`{"type":"array","items":{"type":"string","format":"byte"},"default":[[-128,127],[]]}`,
		`{"type":"object","additionalProperties":{"type":["string","null"],"format":"binary"},"default":{"z":[1],"a":null}}`,
		`{"type":"array","items":{"type":"object","additionalProperties":{"type":"string","format":"byte"}},"default":[{"a":[-128,127]}]}`,
		`{"type":"object","properties":{"blob":{"type":"string","format":"byte"}},"default":{"blob":[-128,127]}}`,
	} {
		for _, validation := range []bool{false, true} {
			input := fieldDoc(field)
			original := bytes.Clone(input)
			r := request(input, "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "RecordDto"), "kotlin.ByteArray") || !bytes.Equal(original, input) {
				t.Fatal("Bytes default tree lost type or mutated source", field, err)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Nondeterministic bytes default tree", err)
			}
		}
	}
	// Exact original component-array rejection now owns the existing root factory.
	for _, validation := range []bool{false, true} {
		r := request(schemas(`{"S":{"type":"array","items":{"type":"string","format":"byte"},"default":[[1]]}}`), "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "SDto"), "public fun `SDto default`()") || !strings.Contains(source(t, got, "SDto"), `"[[1]]"`) {
			t.Fatal("Original bytes component-array input lost factory", validation, err)
		}
	}
}

func TestBytesDefaultTreesKeepOriginalElementDiagnosticsAndSchemaGuards(t *testing.T) {
	for _, tc := range []struct{ schema, tail, code string }{
		{`{"type":"array","items":{"type":"string","format":"byte"},"default":[[128]]}`, "/default/0/0", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string","format":"byte"},"default":[[-129]]}`, "/default/0/0", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string","format":"byte"},"default":[[1.0]]}`, "/default/0/0", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string","format":"byte"},"default":[[null]]}`, "/default/0/0", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string","format":"byte"},"default":["AAEC"]}`, "/default/0", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string","format":"byte"},"default":[null]}`, "/default/0", "E_SCHEMA_INVALID"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"binary"},"default":{"a/~":[128]}}`, "/default/a~1~0/0", "E_SCHEMA_INVALID"},
		{`{"type":"object","properties":{"blob":{"type":"string","format":"byte"}},"default":{"blob":[128]}}`, "/default/blob/0", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string","format":"byte","default":[1]},"default":[]}`, "/items/default", "E_SCHEMA_UNSUPPORTED"},
		{`{"type":"array","items":{"type":"string","format":"byte","minLength":1},"default":[]}`, "/items/minLength", "E_SCHEMA_UNSUPPORTED"},
		{`{"type":"array","items":{"type":"string","format":"byte"},"uniqueItems":true,"default":[]}`, "/uniqueItems", "E_SCHEMA_UNSUPPORTED"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.schema), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
}

func TestBytesDefaultTreesKeepCountsAndWholeAnnotationLimits(t *testing.T) {
	for _, tc := range []struct {
		values   string
		accepted bool
	}{
		{strings.Repeat("0,", 16381) + "0", true}, // 16,384 nodes, 32,767 bytes.
		{strings.Repeat("0,", 16382) + "0", false},
		{strings.Repeat("-128,", 6552) + "-128", true}, // Exactly 32 KiB.
		{strings.Repeat("-128,", 6553) + "-128", false},
	} {
		r := request(fieldDoc(`{"type":"array","items":{"type":"string","format":"byte"},"default":[[`+tc.values+`]]}`), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if tc.accepted {
			if err != nil {
				t.Fatal("Existing whole-tree boundary narrowed", len(tc.values), err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !strings.Contains(*err.(core.Diagnostics)[0].Pointer, "/default/") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Byte values escaped existing whole-tree budgets", len(tc.values), err)
		}
	}
	for _, field := range []string{
		`{"type":"array","items":{"type":"string","format":"byte"},"minItems":1,"default":[]}`,
		`{"type":"object","additionalProperties":{"type":"string","format":"byte"},"maxProperties":0,"default":{"a":[1]}}`,
	} {
		r := request(fieldDoc(field), "Dto")
		r.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(r)
		if err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default" {
			t.Fatal("Byte trees dropped outer container counts", err)
		}
	}
}
