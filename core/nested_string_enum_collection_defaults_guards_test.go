package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestNestedStringEnumCollectionDefaultsOriginalGrammarMembershipAndNoDrop(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"additionalProperties":{"enum":["one"]}},"default":[{"a/~":"future"}]}`, "E_SCHEMA_INVALID", "/default/0/a~1~0"},
		{`{"additionalProperties":{"additionalProperties":{"enum":["one"]}},"default":{"z":{"z":false,"a":"future"},"a":false}}`, "E_SCHEMA_INVALID", "/default/z/z"},
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"]}},"default":[[1]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"]}},"default":null}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"]}},"default":[null]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"array","nullable":true,"items":{"enum":["one"]}},"default":[[null]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"type":"array","items":{"type":"array","items":{"enum":[]}},"default":[]}`, "E_SCHEMA_INVALID", "/items/items/enum"},
		{`{"type":"array","nullable":true,"items":{"type":"array","items":{"enum":["one","one"]}},"default":null}`, "E_SCHEMA_INVALID", "/items/items/enum/1"},
		{`{"additionalProperties":{"additionalProperties":{"enum":[null,"one"]}},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/additionalProperties/enum/0"},
		{`{"additionalProperties":{"additionalProperties":{"enum":[null,"one",null],"nullable":true}},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/additionalProperties/enum/2"},
		{`{"additionalProperties":{"additionalProperties":{"type":"string","enum":[null],"nullable":true}},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/additionalProperties/enum"},
		{`{"type":"array","items":{"type":"array","items":{"enum":[null,"one",false],"nullable":true}},"default":[]}`, "E_SCHEMA_INVALID", "/items/items/enum/2"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string","format":"uuid","enum":["one"]}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/items/enum"},
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"],"minLength":1}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/items/minLength"},
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"],"pattern":"^o"}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/items/pattern"},
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"],"default":"one"}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/items/default"},
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"]},"default":[]},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/default"},
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = checked
			if tc.field == `{"type":"array","items":{"type":"array","items":{"enum":["one"],"default":"one"}},"default":[]}` {
				inlineEnumChildGraduation(t, r, `,"default":"one"`, `,"default":"other"`, "/components/schemas/Record/properties/value/items/items/default")
				continue
			}
			if checked && (tc.field == `{"type":"array","items":{"type":"array","items":{"enum":["one"],"minLength":1}},"default":[]}` || tc.field == `{"type":"array","items":{"type":"array","items":{"enum":["one"],"pattern":"^o"}},"default":[]}`) {
				nestedStringEnumBoundsGraduation(t, r.Document)
				continue
			}
			if tc.field == `{"additionalProperties":{"additionalProperties":{"enum":[null,"one"]}},"default":{}}` {
				implicitStringEnumGraduation(t, fieldDoc(tc.field), checked)
				continue
			}
			if tc.code == "E_SCHEMA_UNSUPPORTED" && strings.HasSuffix(tc.tail, "/enum") && metadataStringEnumCollectionFixture(tc.field) {
				metadataStringEnumGraduation(t, r)
				continue
			}
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, err)
			}
		}
	}
}

func TestNestedStringEnumCollectionDefaultsNullableEmptyLongerAndBeforeSuppression(t *testing.T) {
	for _, field := range []string{
		`{"type":"array","items":{"type":"array","nullable":true,"items":{"type":["string","null"],"enum":[null,"one","null"]}},"default":[null,[null,"one","null"],[]]}`,
		`{"nullable":true,"additionalProperties":{"additionalProperties":{"enum":["one"]}},"default":null}`,
		`{"additionalProperties":{"type":"array","items":{"additionalProperties":{"type":"array","items":{"enum":["one"]}}}},"default":{"$ref":[{"type":["one"],"/~":[]}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(`{"Holder":{"properties":{"value":`+field+`}}}`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(field, err)
			}
		}
	}
	invalid := `{"type":"array","items":{"type":"array","items":{"enum":["one"]}},"default":[["future"]]}`
	for _, input := range [][]byte{
		schemas(`{"Holder":{"properties":{"value":` + invalid + `},"required":["value"]}}`),
		schemas(`{"Holder":{"allOf":[{"properties":{"value":` + invalid + `}},{"properties":{"value":{"type":"string"}}}]}}`),
		[]byte(`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + strings.Replace(invalid, `"type":"array"`, `"type":"array","nullable":true`, 1) + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`),
	} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || !strings.HasSuffix(*err.(core.Diagnostics)[0].Pointer, "/default/0/0") || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("lost annotation", err)
			}
		}
	}
	for _, tc := range []struct{ field, tail string }{
		{`{"type":"array","items":{"type":"array","items":{"enum":["one"]},"minItems":1},"default":[[]]}`, "/default/0"},
		{`{"additionalProperties":{"additionalProperties":{"enum":["one"]},"maxProperties":0},"default":{"z":{"a/~":"one"}}}`, "/default/z"},
	} {
		r := request(fieldDoc(tc.field), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail {
			t.Fatal(tc, err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("lost opt-in", tc)
		}
	}
}

func TestNestedStringEnumCollectionDefaultsIndependentScopeAndCaps(t *testing.T) {
	field := `{"type":"array","items":{"additionalProperties":{"enum":["one"]}},"default":[]}`
	for _, input := range []string{
		`{"Holder":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":` + field + `}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + field + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("scope leak", input, err)
			}
		}
	}
	// The direct nested role never owns $ref fields; the referenced collection
	// field role separately admits this same shape with its own source grammar.
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"Tree":{"type":"array","items":{"type":"array","items":{"enum":["one"]}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Tree","default":[]}}}}`), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("referenced role graduation", checked, err)
		}
	}
	for _, checked := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			members := make([]string, count-1)
			for i := range members {
				members[i] = fmt.Sprintf(`"v%d"`, i)
			}
			r := request(fieldDoc(nestedDefaultChain(2, `{"enum":[null,`+strings.Join(members, ",")+`],"nullable":true}`, `[]`)), "Dto")
			r.Options.Validation = checked
			_, err := core.GenerateWithDiagnostics(r)
			if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT") {
				t.Fatal("raw240", count, err)
			}
		}
	}
	for _, edges := range []int{47, 48} {
		_, err := core.GenerateWithDiagnostics(request(fieldDoc(nestedDefaultChain(edges, `{"enum":["one"]}`, `[]`)), "Dto"))
		if edges == 47 && err != nil || edges == 48 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("source48", edges, err)
		}
	}
	for _, edges := range []int{16, 17} {
		leaf := `{"enum":["one"]}`
		for i := 0; i < edges; i++ {
			leaf = `{"type":"array","minItems":0,"items":` + leaf + `}`
		}
		leaf = strings.TrimSuffix(leaf, "}") + `,"default":[]}`
		r := request(fieldDoc(leaf), "Dto")
		r.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(r)
		if edges == 16 && err != nil || edges == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("checked16", edges, err)
		}
	}
	for _, size := range []int{32762, 32763} {
		token := strings.Repeat("a", size)
		field := nestedDefaultChain(2, `{"enum":["`+token+`"]}`, `[["`+token+`"]]`)
		_, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
		if size == 32762 && err != nil || size == 32763 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("bytes", size, err)
		}
	}
	for _, nodes := range []int{16382, 16383} {
		annotation := `[[` + strings.TrimSuffix(strings.Repeat(`null,`, nodes), ",") + `]]`
		_, err := core.GenerateWithDiagnostics(request(fieldDoc(nestedDefaultChain(2, `{"enum":["one"],"nullable":true}`, annotation)), "Dto"))
		// The original byte cap is independent and can close before node work.
		if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
			t.Fatal("independent null-byte cap", nodes, err)
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%d":%s`, i, nestedDefaultChain(2, `{"enum":["one"]}`, `[["one"]]`))
		}
		_, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("slots240", count, err)
		}
	}
}
