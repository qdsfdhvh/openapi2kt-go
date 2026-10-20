package core_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// A `$ref` wrapper is a canonical alias, so the alias itself owns the factory
// while the target gains nothing. Both halves are asserted, because "the factory
// exists" and "the target was silently modified" are different claims.
func TestReferenceWrapperComponentDefaultsOwnTheFactoryOnTheAlias(t *testing.T) {
	doc := schemas(`{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}`)
	for _, validation := range []bool{false, true} {
		r := request(doc, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		wrapper := source(t, got, "SDto")
		if !strings.Contains(wrapper, "public typealias SDto = org.example.models.models.TDto") {
			t.Fatalf("Wrapper lost its alias shape:\n%s", wrapper)
		}
		if !strings.Contains(wrapper, "public fun `SDto default`(): SDto =") {
			t.Fatalf("Wrapper lost its factory:\n%s", wrapper)
		}
		if !strings.Contains(wrapper, `"{\"z\":\"1\"}"`) {
			t.Fatalf("Wrapper default did not keep its canonical JSON:\n%s", wrapper)
		}
		if target := source(t, got, "TDto"); strings.Contains(target, "`TDto default`") {
			t.Fatalf("Wrapper default leaked onto the target:\n%s", target)
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Wrapper default generation drifted", err)
		}
	}
}

// The wrapper validates its value against the target's own default role, and a
// bad value keeps the wrapper's `.../default` location.
func TestReferenceWrapperComponentDefaultsValidateAgainstTheTarget(t *testing.T) {
	admitted := []struct{ name, doc, marker string }{
		{"object", `{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{"z":"1"}}}`, `"{\"z\":\"1\"}"`},
		{"collection", `{"T":{"type":"array","items":{"type":"string"}},"S":{"$ref":"#/components/schemas/T","default":["x"]}}`, `"[\"x\"]"`},
		{"string enum", `{"T":{"type":"string","enum":["a","b"]},"S":{"$ref":"#/components/schemas/T","default":"a"}}`, `"\"a\""`},
		{"bytes", `{"T":{"type":"string","format":"byte"},"S":{"$ref":"#/components/schemas/T","default":[1,2]}}`, `"[1,2]"`},
		{"temporal", `{"T":{"type":"string","format":"date"},"S":{"$ref":"#/components/schemas/T","default":"2024-02-29"}}`, `"\"2024-02-29\""`},
	}
	for _, tc := range admitted {
		for _, validation := range []bool{false, true} {
			r := request(schemas(tc.doc), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatalf("%s (validation=%v): %v", tc.name, validation, err)
			}
			if text := source(t, got, "SDto"); !strings.Contains(text, tc.marker) {
				t.Fatalf("%s lost its canonical JSON:\n%s", tc.name, text)
			}
		}
	}
	for _, tc := range []struct{ name, doc, pointer string }{
		{"object missing required", `{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T","default":{}}}`, "/components/schemas/S/default"},
		{"enum non-member", `{"T":{"type":"string","enum":["a","b"]},"S":{"$ref":"#/components/schemas/T","default":"zz"}}`, "/components/schemas/S/default"},
		{"bytes out of range", `{"T":{"type":"string","format":"byte"},"S":{"$ref":"#/components/schemas/T","default":[999]}}`, "/components/schemas/S/default/0"},
		{"temporal bad lexeme", `{"T":{"type":"string","format":"date"},"S":{"$ref":"#/components/schemas/T","default":"2023-02-30"}}`, "/components/schemas/S/default"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(tc.doc), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			var ds core.Diagnostics
			if !errors.As(err, &ds) || len(ds) == 0 {
				t.Fatalf("%s: expected a diagnostic, got %v", tc.name, err)
			}
			if ds[0].Code != "E_SCHEMA_INVALID" {
				t.Fatalf("%s: expected E_SCHEMA_INVALID, got %s", tc.name, ds[0].Code)
			}
			if ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer {
				t.Fatalf("%s: expected %s, got %v", tc.name, tc.pointer, ds[0].Pointer)
			}
			if len(got.Artifacts.Files) != 0 {
				t.Fatalf("%s: rejected input emitted files", tc.name)
			}
		}
	}
}

// Roles outside this contract keep the baseline rejection at the wrapper's own
// `.../default`, so no shape loses precision relative to before.
func TestReferenceWrapperComponentDefaultsKeepOtherRolesRejected(t *testing.T) {
	for _, tc := range []struct{ name, doc, pointer string }{
		{"chained plain scalar, validation off", `{"T":{"type":"string"},"M":{"$ref":"#/components/schemas/T"},"S":{"$ref":"#/components/schemas/M","default":"x"}}`, "/components/schemas/S/default"},
		{"nullable wrapper", `{"T":{"type":"string"},"S":{"$ref":"#/components/schemas/T","nullable":true,"default":"x"}}`, "/components/schemas/S/default"},
		{"missing reference", `{"S":{"$ref":"#/components/schemas/Nope","default":{}}}`, "/components/schemas/S/default"},
		{"union target", `{"T":{"oneOf":[{"type":"string"},{"type":"integer"}]},"S":{"$ref":"#/components/schemas/T","default":"x"}}`, "/components/schemas/S/default"},
		{"plain scalar target, validation off", `{"T":{"type":"string"},"S":{"$ref":"#/components/schemas/T","default":"x"}}`, "/components/schemas/S/default"},
	} {
		r := request(schemas(tc.doc), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		var ds core.Diagnostics
		if !errors.As(err, &ds) || len(ds) == 0 {
			t.Fatalf("%s: expected a diagnostic, got %v", tc.name, err)
		}
		if ds[0].Code != "E_SCHEMA_UNSUPPORTED" {
			t.Fatalf("%s: expected E_SCHEMA_UNSUPPORTED, got %s", tc.name, ds[0].Code)
		}
		if ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer {
			t.Fatalf("%s: expected %s, got %v", tc.name, tc.pointer, ds[0].Pointer)
		}
		if len(got.Artifacts.Files) != 0 {
			t.Fatalf("%s: rejected input emitted files", tc.name)
		}
	}
	// A `$ref` to a plain scalar is a validation-opt-in alias, so with validation
	// enabled the wrapper is admitted like the plain-scalar component root.
	r := request(schemas(`{"T":{"type":"string"},"S":{"$ref":"#/components/schemas/T","default":"x"}}`), "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	if text := source(t, got, "SDto"); !strings.Contains(text, "public fun `SDto default`(): SDto =") {
		t.Fatalf("validation-enabled scalar wrapper lost its factory:\n%s", text)
	}
}

// A wrapper without the annotation is unchanged: a plain alias, no factory.
func TestReferenceWrapperWithoutAnnotationStaysAPlainAlias(t *testing.T) {
	got, err := core.GenerateWithDiagnostics(request(schemas(`{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"S":{"$ref":"#/components/schemas/T"}}`), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "SDto")
	if !strings.Contains(text, "public typealias SDto =") || strings.Contains(text, "default`") {
		t.Fatalf("Annotation-less wrapper gained a factory:\n%s", text)
	}
}
