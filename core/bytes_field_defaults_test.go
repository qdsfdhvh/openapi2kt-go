package core_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// A bytes default must match the wire the existing ByteArrayFieldDefaultWire
// contract already produces: a JSON array of signed byte integers. Base64 text
// is rejected because the generated serializer could not round-trip it.
func TestBytesFieldDefaultsUseTheExistingIntegerArrayWire(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"required":{"type":"string","format":"byte","default":[-128,0,127]},"optional":{"type":"string","format":"binary","default":[4]},"nullable":{"type":"string","format":"byte","nullable":true,"default":null},"empty":{"type":"string","format":"byte","default":[]}},"required":["required"]}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{
			"public val required: kotlin.ByteArray,",
			"public val optional: kotlin.ByteArray? = byteArrayOf(4),",
			"public val nullable: kotlin.ByteArray? = null,",
			"public val empty: kotlin.ByteArray? = byteArrayOf(),",
		} {
			if !strings.Contains(text, marker) {
				t.Fatal("Bytes default changed typed required/nullable policy", marker, text)
			}
		}
		if strings.Contains(text, "required: kotlin.ByteArray? =") {
			t.Fatal("A required bytes field must not gain an initializer", text)
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Bytes default source generation drifted", err)
		}
	}
}

// Only the shape widened. A value the wire cannot carry still fails at the
// offending element, not at the whole annotation.
func TestBytesFieldDefaultsRejectNonIntegerArrayValues(t *testing.T) {
	cases := []struct{ name, value, pointer string }{
		{"aboveSignedRange", `[128]`, "/components/schemas/Holder/properties/v/default/0"},
		{"belowSignedRange", `[-129]`, "/components/schemas/Holder/properties/v/default/0"},
		{"base64Text", `"AAEC"`, "/components/schemas/Holder/properties/v/default"},
		{"fraction", `[1.5]`, "/components/schemas/Holder/properties/v/default/0"},
		{"nullElement", `[null]`, "/components/schemas/Holder/properties/v/default/0"},
		{"stringElement", `["1"]`, "/components/schemas/Holder/properties/v/default/0"},
		{"objectValue", `{}`, "/components/schemas/Holder/properties/v/default"},
	}
	for _, c := range cases {
		for _, validation := range []bool{false, true} {
			input := schemas(`{"Holder":{"properties":{"v":{"type":"string","format":"byte","default":` + c.value + `}}}}`)
			r := request(input, "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			var ds core.Diagnostics
			if !errors.As(err, &ds) || len(ds) == 0 {
				t.Fatalf("%s: expected a diagnostic, got %v", c.name, err)
			}
			if ds[0].Code != "E_SCHEMA_INVALID" {
				t.Fatalf("%s: expected E_SCHEMA_INVALID, got %s", c.name, ds[0].Code)
			}
			if ds[0].Pointer == nil || *ds[0].Pointer != c.pointer {
				t.Fatalf("%s: expected pointer %s, got %v", c.name, c.pointer, ds[0].Pointer)
			}
			if len(got.Artifacts.Files) != 0 {
				t.Fatalf("%s: rejected input emitted files", c.name)
			}
		}
	}
}

// A byte integer is reported exactly as written; the same wire value reached
// through different JSON spellings must not change emitted identity.
func TestBytesFieldDefaultsKeepSignedIntegerIdentity(t *testing.T) {
	generate := func(value string) core.GenerationReport {
		t.Helper()
		input := schemas(`{"Holder":{"properties":{"v":{"type":"string","format":"byte","default":` + value + `}}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if !reflect.DeepEqual(generate(`[1,2]`), generate(`[1,2]`)) {
		t.Fatal("Equivalent bytes defaults changed emitted identity")
	}
	if reflect.DeepEqual(generate(`[1,2]`), generate(`[2,1]`)) {
		t.Fatal("Bytes default element ordering disappeared")
	}
	if !strings.Contains(source(t, generate(`[-128,127]`), "HolderDto"), "byteArrayOf(-128, 127)") {
		t.Fatal("Signed byte bounds lost their literal form")
	}
}

// A component root is a separate role with its own factory: it must not inherit
// the field initializer, and it keeps the same signed-array grammar.
func TestBytesComponentDefaultsEmitTopLevelFactory(t *testing.T) {
	for _, validation := range []bool{false, true} {
		r := request(schemas(`{"S":{"type":"string","format":"byte","default":[-128,0,127]}}`), "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		text := source(t, got, "SDto")
		// A typealias cannot own members, so the factory is top level and decodes
		// through the component's own serializer.
		if !strings.Contains(text, "public typealias SDto = kotlin.ByteArray") {
			t.Fatalf("Bytes component lost its alias shape:\n%s", text)
		}
		if !strings.Contains(text, "public fun `SDto default`(): SDto =") {
			t.Fatalf("Bytes component lost its factory:\n%s", text)
		}
		if !strings.Contains(text, `"[-128,0,127]"`) {
			t.Fatalf("Bytes component default did not keep its canonical JSON:\n%s", text)
		}
		if !strings.Contains(text, "serializer<SDto>()") {
			t.Fatalf("Bytes component factory did not decode through its serializer:\n%s", text)
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Bytes component default generation drifted", err)
		}
	}
}

func TestBytesComponentNullableNullFactoryGraduation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"S":{"type":"string","format":"byte","nullable":true,"default":null}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if text := source(t, got, "SDto"); !strings.Contains(text, "typealias SDto = kotlin.ByteArray?") || !strings.Contains(text, `("null")`) {
			t.Fatal("original nullable byte case lost", text)
		}
	}
}

// Bad byte values retain their original dedicated diagnostics and Pointers.
func TestBytesComponentDefaultsKeepRoleBoundaries(t *testing.T) {
	for _, c := range []struct{ name, doc, pointer string }{
		{"badElement", `{"S":{"type":"string","format":"byte","default":[128]}}`, "/components/schemas/S/default/0"},
		{"base64", `{"S":{"type":"string","format":"byte","default":"AAEC"}}`, "/components/schemas/S/default"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(c.doc), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			var ds core.Diagnostics
			if !errors.As(err, &ds) || len(ds) == 0 {
				t.Fatalf("%s: expected a diagnostic, got %v", c.name, err)
			}
			if ds[0].Pointer == nil || *ds[0].Pointer != c.pointer {
				t.Fatalf("%s: expected pointer %s, got %v", c.name, c.pointer, ds[0].Pointer)
			}
			if len(got.Artifacts.Files) != 0 {
				t.Fatalf("%s: rejected input emitted files", c.name)
			}
		}
	}
}
