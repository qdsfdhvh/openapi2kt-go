package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedEnumCollectionIndependentDefaults(t *testing.T) {
	for _, bounds := range []string{"", `,"minLength":2,"maxLength":6`} {
		for _, tree := range []struct{ source, own, alias, field, suffix string }{
			{`{"type":"array","items":LEAF}`, `["one"]`, `["two"]`, `["one"]`, "/0"},
			{`{"additionalProperties":LEAF}`, `{"source":"one"}`, `{"alias":"two"}`, `{"field":"one"}`, "/alias"},
			{`{"type":"array","items":{"additionalProperties":LEAF}}`, `[{"source":"one"}]`, `[{"alias":"two"}]`, `[{"field":"one"}]`, "/0/alias"},
		} {
			for _, owner := range []string{"AOwner", "ZOwner"} {
				leaf := `{"type":"string","format":"date","enum":["one","two","x"]` + bounds + `}`
				treeSchema := strings.ReplaceAll(tree.source, "LEAF", leaf)
				treeSchema = strings.TrimSuffix(treeSchema, "}") + `,"default":` + tree.own + `}`
				body := `{"OWNER":{"properties":{"tree":SOURCE}},"Tree":{"$ref":"#/components/schemas/OWNER/properties/tree","default":ALIAS},"Alias":{"$ref":"#/components/schemas/Tree","default":FIELD},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":FIELD},"unannotated":{"$ref":"#/components/schemas/Tree"}},"default":{"value":ALIAS}}}`
				body = strings.NewReplacer("OWNER", owner, "SOURCE", treeSchema, "ALIAS", tree.alias, "FIELD", tree.field).Replace(body)
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = bounds != ""
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("independent original/borrowed collection defaults rejected", err)
				}
				if !strings.Contains(source(t, got, "TreeDto"), "TreeDto default") || !strings.Contains(source(t, got, "AliasDto"), "AliasDto default") {
					t.Fatal("borrowed collection lost alias-owned factories")
				}
				if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("borrowed collection factories nondeterministic", err)
				}
				if bounds != "" {
					bad := strings.Replace(body, `"default":`+tree.alias, `"default":`+strings.ReplaceAll(tree.alias, "two", "x"), 1)
					enumBoundsDiagnostic(t, schemas(bad), true, "E_SCHEMA_INVALID", "/components/schemas/Tree/default"+tree.suffix)
				}
			}
		}
	}
}

func TestBorrowedEnumCollectionDefaultSourceAndValueGuards(t *testing.T) {
	body := `{"Owner":{"properties":{"tree":{"type":"array","items":{"enum":["one","two","x"],"minLength":2},"default":["one"]}}},"Tree":{"$ref":"#/components/schemas/Owner/properties/tree","default":["two"]},"Maybe":{"$ref":"#/components/schemas/Tree","nullable":true,"default":null},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Tree","default":["one"]}},"required":["value"]}}`
	for _, tc := range []struct{ from, to, pointer string }{
		{`"default":["one"]`, `"default":["x"]`, "/components/schemas/Owner/properties/tree/default/0"},
		{`"default":["two"]`, `"default":["future"]`, "/components/schemas/Tree/default/0"},
		{`"default":["two"]`, `"default":["x"]`, "/components/schemas/Tree/default/0"},
		{`"nullable":true`, `"nullable":false`, "/components/schemas/Maybe/default"},
	} {
		enumBoundsDiagnostic(t, schemas(strings.Replace(body, tc.from, tc.to, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
	}
	for _, value := range []string{`[]`, `null`} {
		bad := strings.Replace(body, `"default":["two"]`, `"nullable":true,"default":`+value, 1)
		bad = strings.Replace(bad, `"minLength":2`, `"minLength":-1`, 1)
		enumBoundsDiagnostic(t, schemas(bad), true, "E_SCHEMA_INVALID", "/components/schemas/Owner/properties/tree/items/minLength")
	}
	for _, bad := range []string{
		`{"Owner":{"properties":{"tree":{"type":"array","items":{"type":"array","default":["one"],"items":{"enum":["one"]}}}}},"Tree":{"$ref":"#/components/schemas/Owner/properties/tree/items","default":["one"]}}`,
		`{"Owner":{"properties":{"tree":{"type":"array","items":{"enum":["one"],"default":"one"}}}}},"Tree":{"$ref":"#/components/schemas/Owner/properties/tree","default":["one"]}}`,
	} {
		rq := request(schemas(bad), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("factory granted a container/enum child annotation role", err)
		}
	}
	for _, size := range []int{32750, 32766} {
		word := strings.Repeat("a", size)
		rq := request(schemas(`{"Owner":{"properties":{"tree":{"type":"array","items":{"enum":["`+word+`"]}}}},"Tree":{"$ref":"#/components/schemas/Owner/properties/tree","default":["`+word+`"]}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if size == 32750 && err != nil {
			t.Fatal("valid borrowed factory below32KiB rejected", err)
		}
		if size == 32766 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("borrowed collection factory escaped32KiB", err)
		}
	}
}
