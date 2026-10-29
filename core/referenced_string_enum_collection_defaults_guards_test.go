package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func stringEnumDefaultFailure(t *testing.T, input []byte, checked bool, code, pointer string) {
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

func TestReferencedStringEnumCollectionFieldDefaultsOriginalGraduation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		// Verbatim former referenced builtin-collection source-role enum negative:
		// the named collection leaf enum was rejected before this role existed.
		r := request(referencedFieldDoc(`{"type":"array","items":{"type":"string","enum":["on","off"]}}`, `["on"]`, ""), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
		// Verbatim former named-enum-leaf negative through the same alias chain.
		r.Document = schemas(`{"Status":{"type":"string","enum":["on","off"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Status"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/narrow","default":["off"]}}},"Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Rows","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}}}`)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
		// Typeless plain enum leaves graduate with the same role.
		r.Document = referencedFieldDoc(`{"type":"array","items":{"enum":["raw","wire"]}}`, `["wire"]`, "")
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReferencedStringEnumCollectionFieldDefaultsValuesAndNullLayers(t *testing.T) {
	valid := []string{`"on"`, `"off"`, `"a~z"`}
	invalid := []string{`"ON"`, `"nope"`, `"on "`, `" on"`, `true`, `1`, `{}`, `[]`}
	terminal := `{"type":"array","items":{"additionalProperties":{"type":["string","null"],"enum":["on","off","a~z"]}}}`
	for _, checked := range []bool{false, true} {
		for _, value := range valid {
			r := request(referencedFieldDoc(terminal, `[{"a/~":`+value+`}]`, ""), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(value, err)
			}
		}
		for _, value := range invalid {
			stringEnumDefaultFailure(t, referencedFieldDoc(terminal, `[{"a/~":`+value+`}]`, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0/a~1~0")
		}
		// Source order wins over canonical map key order.
		stringEnumDefaultFailure(t, referencedFieldDoc(terminal, `[{"z":false,"a":"on"}]`, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0/z")
		for _, value := range []string{`null`, `[null]`} {
			stringEnumDefaultFailure(t, referencedFieldDoc(terminal, value, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default"+map[string]string{`[null]`: "/0"}[value])
		}
		r := request(referencedFieldDoc(terminal, `[{"a":null}]`, ""), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("leaf null", err)
		}
		r.Document = referencedFieldDoc(terminal, `null`, `,"nullable":true`)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("nearest true", err)
		}
		// Null list members require the nullable row layer, not the field flag.
		stringEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","items":{"type":"array","items":{"type":"string","enum":["on","off"]}}}`, `[null]`, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0")
		r.Document = referencedFieldDoc(`{"type":"array","items":{"type":["array","null"],"items":{"type":"string","enum":["on","off"]}}}`, `[null,[]]`, "")
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("row null", err)
		}
	}
}

func TestReferencedStringEnumCollectionFieldDefaultsExcludedSourceRolesAndNoDrop(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"string","format":"uuid","enum":["on"]}`, `{"type":"string","enum":["on"],"const":"on"}`,
		`{"type":"string","enum":["on"],"minLength":0}`, `{"type":"string","enum":["on"],"pattern":".*"}`,
		`{"enum":[1,2]}`, `{"type":["string","integer"],"enum":["on",1]}`,
		`{"oneOf":[{"type":"string","enum":["on"]},{"type":"null"}]}`, `{"type":"string","x-native-type":"kotlin.String","enum":["on"]}`,
		`{"type":"string","enum":[]}`, `{"type":"string","enum":["on",null,false]}`,
	} {
		for _, checked := range []bool{false, true} {
			for _, value := range []string{`[]`, `null`} {
				r := request(referencedFieldDoc(`{"type":"array","items":`+leaf+`}`, value, `,"nullable":true`), "Dto")
				r.Options.Validation = checked
				if metadataStringEnumFixture(leaf) {
					metadataStringEnumGraduation(t, r)
					continue
				}
				if checked && (leaf == `{"type":"string","enum":["on"],"minLength":0}` || leaf == `{"type":"string","enum":["on"],"pattern":".*"}`) {
					boundedEnumRootsGenerate(t, r.Document)
					continue
				}
				got, err := core.GenerateWithDiagnostics(r)
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("excluded source", leaf, value, checked, err)
				}
			}
		}
	}
	for _, body := range []string{
		`{"Statuses":{"type":"array","items":{"type":"string","enum":["on"]}},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Statuses"}],"default":[]}}}}`,
		`{"Statuses":{"type":"array","items":{"type":"string","enum":["on"]}},"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Statuses","default":[]}}}}}`,
		`{"Statuses":{"type":"array","items":{"$ref":"#/components/schemas/Statuses"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Statuses","default":[]}}}}`,
		`{"Status":{"type":"string","enum":["on"]},"Statuses":{"type":"array","items":{"$ref":"#/components/schemas/Status"}},"Holder":{"properties":{"v":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Statuses","default":[]}}}}}}`,
		`{"Statuses":{"type":"array","items":{"type":"string","enum":["on"]}},"Choice":{"oneOf":[{"properties":{"tag":{"const":"a"},"v":{"$ref":"#/components/schemas/Statuses","default":[]}}},{"properties":{"tag":{"const":"b"}}}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("role", body, err)
			}
		}
	}
	stringEnumUniqueGraduation(t, referencedFieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"string","enum":["on"]}}`, `[]`, ""))
	for _, checked := range []bool{false, true} {
		// Validation-only counts stay admitted and checked against the default.
		r := request(referencedFieldDoc(`{"type":"array","maxItems":2,"items":{"type":"string","enum":["on","off"]}}`, `["on","off"]`, ""), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); checked && err != nil {
			t.Fatal("validated count", err)
		} else if !checked && err == nil {
			t.Fatal("unvalidated count source admitted")
		}
		stringEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","maxItems":1,"items":{"type":"string","enum":["on","off"]}}`, `["on","off"]`, ""), true, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default")
	}
	// Validated false remains exactly effect-free, including default identity.
	r := request(referencedFieldDoc(`{"type":"array","items":{"type":"string","enum":["on"]}}`, `["on"]`, ""), "Dto")
	r.Options.Validation = true
	before, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Document = referencedFieldDoc(`{"type":"array","uniqueItems":false,"items":{"type":"string","enum":["on"]}}`, `["on"]`, "")
	after, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("explicit false drift", err)
	}
	// Required and Patch annotations that emit no initializer validate but add
	// no runtime default identity.
	for _, role := range []string{"required", "patch"} {
		holder := `{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Statuses","default":["on"]}}}`
		doc := schemas(`{"Statuses":{"type":"array","items":{"type":"string","enum":["on","off"]}},"Holder":` + holder + `}`)
		if role == "patch" {
			doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Statuses":{"type":"array","nullable":true,"items":{"type":"string","enum":["on","off"]}}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"v":{"$ref":"#/components/schemas/Statuses","default":["on"]}}}}}},"responses":{"204":{}}}}}}`)
		}
		first, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if err != nil {
			t.Fatal(err)
		}
		second, err := core.GenerateWithDiagnostics(request([]byte(strings.ReplaceAll(string(doc), `"default":["on"]`, `"default":["off"]`)), "Dto"))
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("suppressed annotation invented identity", role, err)
		}
	}
}

func TestReferencedStringEnumCollectionFieldDefaultsVocabularyMembershipBoundaries(t *testing.T) {
	// Default members must match an original lexical member exactly; supplied
	// values keep the open runtime vocabulary and are never generation-checked.
	leaf := `{"type":["string","null"],"enum":["on","off"]}`
	terminal := `{"type":"array","items":` + leaf + `}`
	for _, value := range []string{`["on",null]`, `[]`, `["off","off"]`} {
		for _, checked := range []bool{false, true} {
			r := request(referencedFieldDoc(terminal, value, ""), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(value, checked, err)
			}
		}
	}
	// The consumed field annotation is strict even when a colliding target
	// default exists on an unrelated component with the same members.
	stringEnumDefaultFailure(t, referencedFieldDoc(`{"type":"array","items":{"type":"string","enum":["on","off"]}}`, `["gone"]`, ""), false, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0")
}
