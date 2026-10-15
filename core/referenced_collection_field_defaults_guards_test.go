package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedFieldDoc(terminal, annotation, override string) []byte {
	return schemas(`{"AHolder":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/narrow","default":` + annotation + override + `}}},"Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Alias","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}},"Alias":{"$ref":"#/components/schemas/ZValue"},"ZValue":` + terminal + `}`)
}

func TestReferencedCollectionFieldDefaultsSourceKindsAndNullLayers(t *testing.T) {
	for _, tc := range []struct{ terminal, value, tail string }{
		{`{"type":"array","items":{"additionalProperties":{"type":"integer"}}}`, `[{"a/~":1.0}]`, "/0/a~1~0"},
		{`{"type":"array","items":{"additionalProperties":{"type":"integer"}}}`, `[{"z":-0,"a":true}]`, "/0/z"},
		{`{"type":"array","items":{"additionalProperties":{"type":"integer","format":"int64"}}}`, `[{"a":9223372036854775808}]`, "/0/a"},
		{`{"additionalProperties":{"type":"array","items":{"type":"number","format":"float"}}}`, `{"a":[1e300]}`, "/a/0"},
		{`{"additionalProperties":{"type":"array","items":{"type":"boolean"}}}`, `{"z":["true"],"a":false}`, "/z/0"},
		{`{"type":"array","items":{"type":"string"}}`, `null`, ""},
		{`{"type":"array","items":{"type":"array","nullable":true,"items":{"type":"string"}}}`, `[[null]]`, "/0/0"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string","nullable":true}}}`, `[null]`, "/0"},
	} {
		for _, checked := range []bool{false, true} {
			r := request(referencedFieldDoc(tc.terminal, tc.value, ""), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			want := "/components/schemas/AHolder/properties/value/default" + tc.tail
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != want || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, want, err)
			}
		}
	}
	for _, terminal := range []string{`{"type":"array","nullable":true,"items":{"type":"array","nullable":true,"items":{"type":"integer","nullable":true}}}`, `{"nullable":true,"additionalProperties":{"type":"array","items":{"type":"string"}}}`} {
		if _, err := core.GenerateWithDiagnostics(request(referencedFieldDoc(terminal, "null", `,"nullable":true`), "Dto")); err != nil {
			t.Fatal("nearest true", err)
		}
	}
}

func TestReferencedCollectionFieldDefaultsSourceRolesAndNoDrop(t *testing.T) {
	for _, terminal := range []string{
		`{"type":"array","items":{"type":"string","default":"a"}}`,

		`{"type":"array","items":{"properties":{"id":{"type":"integer"}}}}`,
		`{"additionalProperties":true}`,
		`{"type":"array","items":{"type":"string","x-native-type":"kotlin.String"}}`,
		`{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"null"}]}}`,
		`{"type":"array","items":{"type":"string"},"default":[]}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(referencedFieldDoc(terminal, "[]", ""), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("ungranted role", terminal, checked, err)
			}
		}
	}
	for _, body := range []string{
		`{"Rows":{"type":"array","items":{"type":"string"}},"Root":{"$ref":"#/components/schemas/Rows","default":[]}}`,
		`{"Rows":{"type":"array","items":{"type":"string"}},"Holder":{"properties":{"value":{"allOf":[{"$ref":"#/components/schemas/Rows"}],"default":[]}}}}`,
		`{"Rows":{"type":"array","items":{"type":"string"}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Rows","default":[]}}}}}`,
		`{"Holder":{"properties":{"target":{"type":"array","items":{"type":"string"},"default":[]},"value":{"$ref":"#/components/schemas/Holder/properties/target","default":[]}}}}`,
		`{"Rows":{"type":"array","items":{"type":"string"}},"Holder":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[1]}}},{"properties":{"value":{"type":"string"}}}]}}`,
		`{"Unused":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[false]}}},"Rows":{"type":"array","items":{"type":"integer"}}}`,
	} {
		if got, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto")); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("source lost", body, err)
		}
	}
	// A discarded valid annotation is checked, while its runtime identity is absent.
	body := `{"Rows":{"type":"array","items":{"type":"integer"}},"Holder":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[1]}}},{"properties":{"value":{"type":"string"}}}]}}`
	if _, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto")); err != nil {
		t.Fatal(err)
	}
}

func TestReferencedCollectionFieldDefaultsCheckedSourceBeforeSuppression(t *testing.T) {
	for _, tc := range []struct{ terminal, value, tail, code string }{
		{`{"type":"array","minItems":1,"items":{"type":"integer"}}`, `[]`, "", "E_SCHEMA_INVALID"},
		{`{"additionalProperties":{"type":"array","uniqueItems":true,"items":{"type":"integer","nullable":true}}}`, `{"a":[null,null]}`, "/a/1", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"additionalProperties":{"type":"integer","minimum":1}}}`, `[{"a":0}]`, "/0/a", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"string","minLength":2}}`, `["x"]`, "/0", "E_SCHEMA_INVALID"},
		{`{"type":"array","items":{"type":"number","format":"float","exclusiveMinimum":1}}`, `[1.000000001]`, "/0", "E_SCHEMA_INVALID"},
	} {
		for _, role := range []string{"optional", "required", "patch", "discarded"} {
			field := `{"$ref":"#/components/schemas/Rows","default":` + tc.value + `}`
			holder := `{"properties":{"value":` + field + `}}`
			pointer := "/components/schemas/Holder/properties/value/default" + tc.tail
			if role == "required" {
				holder = `{"properties":{"value":` + field + `},"required":["value"]}`
			}
			if role == "discarded" {
				holder = `{"allOf":[` + holder + `,{"properties":{"value":{"type":"string"}}}]}`
				pointer = "/components/schemas/Holder/allOf/0/properties/value"
			}
			doc := schemas(`{"Rows":` + tc.terminal + `,"Holder":` + holder + `}`)
			if role == "patch" {
				field = strings.TrimSuffix(field, "}") + `,"nullable":true}`
				doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Rows":` + tc.terminal + `}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}},"responses":{"204":{}}}}}}`)
				pointer = "/paths/~1p/patch/requestBody/content/application~1json/schema/properties/value/default" + tc.tail
			}
			r := request(doc, "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(role, tc, err)
			}
			if role != "discarded" && (*err.(core.Diagnostics)[0].Pointer != pointer || err.(core.Diagnostics)[0].Code != tc.code) {
				t.Fatal(role, pointer, err)
			}
		}
	}
}

func TestReferencedCollectionFieldDefaultsBudgets(t *testing.T) {
	for _, count := range []int{45, 46} {
		props := []string{}
		for i := 0; i < count; i++ {
			target := "Rows"
			if i > 0 {
				target = fmt.Sprintf("Hops/properties/h%d", i-1)
			}
			props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
		}
		doc := schemas(fmt.Sprintf(`{"Rows":{"type":"array","items":{"type":"string"}},"Hops":{"properties":{%s}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/h%d","default":[]}}}}`, strings.Join(props, ","), count-1))
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if count == 45 && err != nil || count == 46 && (err == nil || len(got.Artifacts.Files) != 0) {
			t.Fatal("source budget", count, err)
		}
	}
	for _, n := range []int{16, 17} {
		terminal := `{"type":"integer","minimum":0}`
		for i := 0; i < n; i++ {
			terminal = `{"type":"array","items":` + terminal + `}`
		}
		r := request(referencedFieldDoc(terminal, "[]", ""), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if n == 16 && err != nil || n == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal("checked", n, err)
		}
	}
	for _, size := range []int{32764, 32765} {
		r := request(referencedFieldDoc(`{"type":"array","items":{"type":"string"}}`, `["`+strings.Repeat("a", size)+`"]`, ""), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if size == 32764 && err != nil || size == 32765 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("bytes", size, err)
		}
	}
	for _, n := range []int{16382, 16383} {
		annotation := `[[` + strings.Repeat(`0,`, n-1) + `0]]`
		// Zero nodes fit 32 KiB exactly; two containers also count as nodes.
		got, err := core.GenerateWithDiagnostics(request(referencedFieldDoc(`{"type":"array","items":{"type":"array","items":{"type":"integer"}}}`, annotation, ""), "Dto"))
		if n == 16382 && err != nil || n == 16383 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !strings.Contains(err.(core.Diagnostics)[0].Message, "value nodes") || len(got.Artifacts.Files) != 0) {
			t.Fatal("nodes", n, err)
		}
	}
	// Original compact JSON fits, but mapped float canonical JSON independently
	// exceeds 32 KiB because each normalized exponent contains a plus sign.
	annotation := `[` + strings.Repeat(`1e30,`, 6549) + `1e30]`
	got, err := core.GenerateWithDiagnostics(request(referencedFieldDoc(`{"type":"array","items":{"type":"number","format":"float"}}`, annotation, ""), "Dto"))
	if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !strings.Contains(err.(core.Diagnostics)[0].Message, "Canonical") || len(got.Artifacts.Files) != 0 {
		t.Fatal("canonical bytes not independently bounded", err)
	}
	for _, n := range []int{240, 241} {
		props := []string{}
		for i := 0; i < n; i++ {
			props = append(props, fmt.Sprintf(`"v%d":{"$ref":"#/components/schemas/Rows","default":[1]}`, i))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Rows":{"type":"array","items":{"type":"integer"}},"Holder":{"properties":{`+strings.Join(props, ",")+`}}}`), "Dto"))
		if n == 240 && err != nil || n == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("slots", n, err)
		}
	}
}
