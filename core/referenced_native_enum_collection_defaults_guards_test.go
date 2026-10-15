package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nativeEnumDefaultFailure(t *testing.T, input []byte, checked bool, code, pointer string) {
	t.Helper()
	r := request(input, "Dto")
	r.Options.Validation = checked
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("expected no artifacts/report", err, got)
	}
	d := err.(core.Diagnostics)[0]
	if d.Code != code || pointer != "" && (d.Pointer == nil || *d.Pointer != pointer) {
		t.Fatal(code, pointer, d)
	}
}

func TestReferencedNativeEnumCollectionFieldDefaultsOriginalGraduation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		// Verbatim former referenced builtin-collection source-role native negative:
		// the named collection leaf native enum was rejected before this role.
		r := request(referencedFieldDoc(`{"type":"array","items":{"type":"integer","format":"int32","enum":[1,2]}}`, `[2]`, ""), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
		// Named-enum-leaf negative through the same alias chain.
		r.Document = schemas(`{"Code":{"type":"integer","format":"int64","enum":[1,2]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Code"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/narrow","default":[1]}}},"Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Rows","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}}}`)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
		// Boolean leaves graduate with the same role.
		r.Document = referencedFieldDoc(`{"type":"array","items":{"type":"boolean","enum":[true,false]}}`, `[false]`, "")
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReferencedNativeEnumCollectionFieldDefaultsValuesAndNullLayers(t *testing.T) {
	terminal := `{"type":"array","items":{"additionalProperties":{"type":"integer","format":"int32","enum":[1,2,3]}}}`
	for _, checked := range []bool{false, true} {
		for _, value := range []string{`[{"a/~":1}]`, `[{"a/~":3}]`} {
			r := request(referencedFieldDoc(terminal, value, ""), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(value, err)
			}
		}
		for _, value := range []string{`[{"a/~":9}]`, `[{"a/~":"1"}]`, `[{"a/~":true}]`, `[{"a/~":null}]`, `[{"a/~":{}}]`, `[{"a/~":[]}]`, `[{"a/~":1.0}]`} {
			nativeEnumDefaultFailure(t, referencedFieldDoc(terminal, value, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0/a~1~0")
		}
		// Source order wins over canonical map key order.
		nativeEnumDefaultFailure(t, referencedFieldDoc(terminal, `[{"z":false,"a":1}]`, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0/z")
		for _, value := range []string{`null`, `[null]`} {
			nativeEnumDefaultFailure(t, referencedFieldDoc(terminal, value, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default"+map[string]string{`[null]`: "/0"}[value])
		}
		// Row null requires the nullable row layer from the actual backing.
		nativeEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","items":{"type":"array","items":{"type":"integer","format":"int32","enum":[1,2]}}}`, `[null]`, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0")
		r := request(referencedFieldDoc(`{"type":"array","items":{"type":["array","null"],"items":{"type":"integer","format":"int32","enum":[1,2]}}}`, `[null,[1]]`, ""), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("row null", err)
		}
		r.Document = referencedFieldDoc(terminal, `null`, `,"nullable":true`)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("nearest true", err)
		}
	}
}

func TestReferencedNativeEnumCollectionFieldDefaultsExcludedSourceRolesAndNoDrop(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"integer","format":"int32","enum":[1],"const":1}`,
		`{"type":"integer","format":"int32","enum":[1],"x-native-type":"org.example.Code"}`,
		`{"type":"integer","format":"int32","enum":[1],"x-kotlin-value-class":true}`,
		`{"enum":[1,2]}`,
		`{"oneOf":[{"type":"integer","enum":[1]}]}`,
		`{"type":"integer","enum":[]}`,
	} {
		for _, checked := range []bool{false, true} {
			for _, value := range []string{`[]`, `null`} {
				r := request(referencedFieldDoc(`{"type":"array","items":`+leaf+`}`, value, `,"nullable":true`), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("excluded source", leaf, value, checked, err)
				}
			}
		}
	}
	for _, body := range []string{
		`{"Codes":{"type":"array","items":{"type":"integer","format":"int32","enum":[1]},"default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[]}}}}`,
		`{"Codes":{"type":"array","items":{"type":"integer","format":"int32","enum":[1]}},"Alias":{"$ref":"#/components/schemas/Codes","default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":[]}}}}`,
		`{"Codes":{"type":"array","items":{"type":"integer","format":"int32","enum":[1]}},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Codes"}],"default":[]}}}}`,
		`{"Codes":{"type":"array","items":{"type":"integer","format":"int32","enum":[1]}},"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Codes","default":[]}}}}}`,
		`{"Codes":{"type":"array","items":{"$ref":"#/components/schemas/Codes"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[]}}}}`,
		`{"Code":{"type":"integer","format":"int32","enum":[1],"default":1},"Codes":{"type":"array","items":{"$ref":"#/components/schemas/Code"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[]}}}}`,
		`{"Codes":{"type":"array","items":{"type":"integer","format":"int32","enum":[1]}},"Choice":{"oneOf":[{"properties":{"tag":{"const":"a"},"v":{"$ref":"#/components/schemas/Codes","default":[]}}},{"properties":{"tag":{"const":"b"}}}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("role", body, err)
			}
		}
	}
	// Float/Double membership is exact against the source token before rounding.
	for _, value := range []string{`[1.6]`, `[1.5000000000000001]`} {
		nativeEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","items":{"type":"number","format":"float","enum":[0.5,1,1.5]}}`, value, ""), false, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0")
	}
	r := request(referencedFieldDoc(`{"type":"array","items":{"type":"number","format":"float","enum":[0.5,1,1.5]}}`, `[1.5]`, ""), "Dto")
	if _, err := core.GenerateWithDiagnostics(r); err != nil {
		t.Fatal("exact float member", err)
	}
	// Counts/bounds without validation stay rejected; with validation the
	// default must satisfy every retained check.
	{
		nativeEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","maxItems":1,"items":{"type":"integer","format":"int32","enum":[1,2]}}`, `[2]`, ""), false, "E_SCHEMA_UNSUPPORTED", "")
		r := request(referencedFieldDoc(`{"type":"array","maxItems":2,"items":{"type":"integer","format":"int32","enum":[1,2],"minimum":2}}`, `[2]`, ""), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("validated bounds", err)
		}
		nativeEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","maxItems":1,"items":{"type":"integer","format":"int32","enum":[1,2],"minimum":2}}`, `[2,1]`, ""), true, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default")
		// The enum/bounds mapped intersection is empty before any member walk.
		nativeEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","items":{"type":"integer","format":"int32","enum":[1,2],"minimum":3}}`, `[2]`, ""), true, "E_SCHEMA_INVALID", "")
		// Floating multipleOf never joins the default walk.
		nativeEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","items":{"type":"number","format":"double","enum":[1.5],"multipleOf":0.5}}`, `[1.5]`, ""), true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/ZValue/items/multipleOf")
	}
	// Validated false remains exactly effect-free, including default identity.
	r = request(referencedFieldDoc(`{"type":"array","items":{"type":"integer","format":"int32","enum":[1]}}`, `[1]`, ""), "Dto")
	r.Options.Validation = true
	before, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Document = referencedFieldDoc(`{"type":"array","uniqueItems":false,"items":{"type":"integer","format":"int32","enum":[1]}}`, `[1]`, "")
	after, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("explicit false drift", err)
	}
	// Required and Patch annotations that emit no initializer validate but add
	// no runtime default identity.
	for _, role := range []string{"required", "patch"} {
		holder := `{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[1]}}}`
		doc := schemas(`{"Codes":{"type":"array","items":{"type":"integer","format":"int32","enum":[1,2]}},"Holder":` + holder + `}`)
		if role == "patch" {
			doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Codes":{"type":"array","nullable":true,"items":{"type":"integer","format":"int32","enum":[1,2]}}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[1]}}}}}},"responses":{"204":{}}}}}}`)
		}
		first, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if err != nil {
			t.Fatal(err)
		}
		second, err := core.GenerateWithDiagnostics(request([]byte(strings.ReplaceAll(string(doc), `"default":[1]`, `"default":[2]`)), "Dto"))
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("suppressed annotation invented identity", role, err)
		}
	}
}
