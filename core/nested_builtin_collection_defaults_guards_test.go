package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedBuiltinCollectionDefaultsOriginalKindsNullabilityAndNoDrop(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"type":"array","items":{"type":"integer"}},"default":[[1.0]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"type":"array","items":{"type":"array","items":{"type":"integer"}},"default":[[2147483648]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"type":"array","items":{"type":"array","items":{"type":"integer"}},"default":[[-0]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"type":"array","items":{"type":"array","items":{"type":"number","format":"float"}},"default":[[1e300]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"type":"array","items":{"additionalProperties":{"type":"boolean"}},"default":[{"a/~":"true"}]}`, "E_SCHEMA_INVALID", "/default/0/a~1~0"},
		{`{"additionalProperties":{"additionalProperties":{"type":"string"}},"default":{"z":{"z":false,"a":1},"a":false}}`, "E_SCHEMA_INVALID", "/default/z/z"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string"}},"default":null}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string"}},"default":[null]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"array","nullable":true,"items":{"type":"string"}},"default":[[null]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"additionalProperties":{"type":"array","items":{"type":"string"}},"default":{"a":{}}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string"},"default":[]},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/default"},
		{`{"type":"array","items":{"additionalProperties":{"type":"string","x-native-type":"bigint"}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/additionalProperties/x-native-type"},
		{`{"additionalProperties":{"type":"array","items":true},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/items"},
		{`{"type":"array","items":{"type":"array","items":{"oneOf":[{"type":"integer"},{"type":"null"}]}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/items/oneOf"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string"},"prefixItems":[{"type":"string"}]},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/prefixItems"},
		{`{"type":"array","items":{"additionalProperties":true},"default":[]}`, "E_SCHEMA_INVALID", "/items/additionalProperties"},
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, err)
			}
		}
	}
}

func TestNestedBuiltinCollectionDefaultsChecksBeforeRequiredPatchAndEmpty(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"type":"array","items":{"type":"integer"},"minItems":1},"default":[[]]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"additionalProperties":{"type":"integer"},"maxProperties":0},"default":[{"a":1}]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"additionalProperties":{"type":"array","items":{"type":"integer","minimum":1}},"default":{"a/~":[0]}}`, "E_SCHEMA_INVALID", "/default/a~1~0/0"},
		{`{"additionalProperties":{"additionalProperties":{"type":"string","minLength":2}},"default":{"z":{"a/~":"x"}}}`, "E_SCHEMA_INVALID", "/default/z/a~1~0"},
		{`{"type":"array","items":{"type":"array","items":{"type":"integer"},"uniqueItems":true},"default":[[1,1]]}`, "E_SCHEMA_INVALID", "/default/0/1"},
		{`{"type":"array","items":{"type":"array","items":{"type":"integer","nullable":true},"uniqueItems":true},"default":[[null,null]]}`, "E_SCHEMA_INVALID", "/default/0/1"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string","minLength":-1}},"default":[]}`, "E_SCHEMA_INVALID", "/items/items/minLength"},
		{`{"additionalProperties":{"additionalProperties":{"type":"number","format":"float","multipleOf":1}},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/additionalProperties/multipleOf"},
		{`{"type":"array","items":{"type":"array","items":{"type":"integer"}},"uniqueItems":true,"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/uniqueItems"},
	} {
		r := request(fieldDoc(tc.field), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("lost noDrop", tc)
		}
	}
	field := `{"type":"array","items":{"type":"array","items":{"type":"integer"}},"default":[[false]]}`
	for _, input := range [][]byte{
		schemas(`{"Holder":{"properties":{"value":` + field + `},"required":["value"]}}`),
		[]byte(`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + strings.Replace(field, `"items":{"type":"array"`, `"nullable":true,"items":{"type":"array"`, 1) + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`),
		schemas(`{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}`),
	} {
		if got, err := core.GenerateWithDiagnostics(request(input, "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("invalid required/Patch/discarded annotation lost", err)
		}
	}
}

func TestNestedBuiltinCollectionDefaultsExcludeIndependentRoles(t *testing.T) {
	field := `{"type":"array","items":{"additionalProperties":{"type":"string"}},"default":[{"a":"one"}]}`
	for _, input := range []string{
		`{"Root":` + field + `}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":` + field + `}}}}`,
		`{"Holder":{"properties":{"value":{"additionalProperties":` + field + `}}}}`,
		`{"Holder":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		`{"Leaf":{"type":"string"},"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}},"default":[]}}}}`,
		// Object leaves inside collections now use the separate object default role.
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + field + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("independent default role leaked", input, err)
			}
		}
	}
}

func nestedDefaultChain(edges int, leaf, annotation string) string {
	for i := 0; i < edges; i++ {
		leaf = `{"type":"array","items":` + leaf + `}`
	}
	return strings.TrimSuffix(leaf, "}") + `,"default":` + annotation + `}`
}

func TestNestedBuiltinCollectionDefaultsResourceBoundaries(t *testing.T) {
	for _, edges := range []int{47, 48} {
		r := request(fieldDoc(nestedDefaultChain(edges, `{"type":"integer"}`, `[]`)), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 47 && err != nil || edges == 48 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal("source48", edges, err)
		}
	}
	for _, edges := range []int{16, 17} {
		r := request(fieldDoc(nestedDefaultChain(edges, `{"type":"integer","minimum":0}`, `[]`)), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 16 && err != nil || edges == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal("checked16", edges, err)
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%d":%s`, i, nestedDefaultChain(2, `{"type":"integer"}`, `[[1]]`))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("slots240", count, err)
		}
	}
	// Original and canonical encoded JSON are both bounded. These examples do
	// not infer a combined maximum width/depth JVM compilation guarantee.
	for _, size := range []int{32762, 32763} {
		field := nestedDefaultChain(2, `{"type":"string"}`, `[["`+strings.Repeat("a", size)+`"]]`)
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
		if size == 32762 && err != nil || size == 32763 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("byte budget", size, err)
		}
	}
	for _, annotation := range []string{`[[` + strings.Repeat("0,", 16384) + `0]]`, `[["` + strings.Repeat("😀", 9000) + `"]]`} {
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(nestedDefaultChain(2, `{"type":"string"}`, annotation)), "Dto"))
		if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("unbounded annotation admitted", err)
		}
	}
}
