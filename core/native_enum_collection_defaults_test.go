package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumCollectionDefaultsTypedPresence(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"codes":{"type":"array","items":{"type":["integer","null"],"enum":[1,2]},"default":[1,null,2]},"flags":{"type":"object","additionalProperties":{"type":"boolean","enum":[true]},"default":{"a/~":true}},"required":{"type":"array","items":{"type":"number","format":"float","enum":[0.5]},"default":[0.5]},"empty":{"type":"object","additionalProperties":{"type":"integer","format":"int64","enum":[-9223372036854775808]},"default":{}},"nil":{"type":"array","nullable":true,"items":{"type":"boolean","enum":[false]},"default":null}},"required":["required"]}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{
			"public val codes: kotlin.collections.List<kotlin.Int?>? =",
			"public val flags: kotlin.collections.Map<kotlin.String, kotlin.Boolean>? =",
			"public val required: kotlin.collections.List<kotlin.Float>,",
			"`openapi-kt default array Int nullable`",
			"`openapi-kt default map Boolean`",
		} {
			if !strings.Contains(text, marker) {
				t.Fatal("Native enum collection lost scalar/presence/default contract", marker, text)
			}
		}
		warnings := 0
		for _, diagnostic := range got.Warnings {
			if diagnostic.Code == "W_ENUM_NATIVE_APPROXIMATION" {
				warnings++
			}
		}
		if validation && warnings != 0 || !validation && warnings != 5 {
			t.Fatal("Native enum warning policy drifted", validation, got.Warnings)
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Native enum collection default generation drifted", err)
		}
	}
}

func TestNativeEnumCollectionDefaultsOriginalDiagnostics(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"type":"integer","enum":[1,2]},"default":[1,3]}`, "E_SCHEMA_INVALID", "/default/1"},
		{`{"type":"object","additionalProperties":{"type":"boolean","enum":[true]},"default":{"a/~":false}}`, "E_SCHEMA_INVALID", "/default/a~1~0"},
		{`{"type":"array","items":{"type":"number","format":"float","enum":[1]},"default":[1.000000001]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"object","additionalProperties":{"type":"number","enum":[1]},"default":{"a":1.00000000000000001}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"array","items":{"type":"integer","enum":[1]},"default":[null]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"integer","enum":[]},"default":[]}`, "E_SCHEMA_INVALID", "/items/enum"},
		{`{"type":"object","additionalProperties":{"type":"boolean","enum":[true,"bad"]},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/enum/1"},
		{`{"type":"array","items":{"type":"integer","enum":[1,1]},"default":[]}`, "E_SCHEMA_INVALID", "/items/enum/1"},
		{`{"type":"object","additionalProperties":{"type":"boolean","enum":[true]},"default":{"z":false,"a":false}}`, "E_SCHEMA_INVALID", "/default/z"},
		{`{"type":"array","nullable":true,"items":{"type":"integer","enum":[]},"default":null}`, "E_SCHEMA_INVALID", "/items/enum"},
		{`{"type":"array","items":{"type":"integer","enum":[1],"default":1},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/default"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
}

func TestNativeEnumCollectionDefaultsCheckedBoundsAndNullVocabulary(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"type":"integer","enum":[1,2],"minimum":2},"default":[1]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"object","additionalProperties":{"type":"integer","enum":[1,2],"minimum":2},"default":{"a/~":1}}`, "E_SCHEMA_INVALID", "/default/a~1~0"},
		{`{"type":"array","items":{"type":"integer","enum":[1],"minimum":2},"default":[]}`, "E_SCHEMA_INVALID", "/items/enum"},
		{`{"type":"array","items":{"type":"integer","enum":[1]},"minItems":1,"default":[]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","additionalProperties":{"type":"integer","enum":[1]},"maxProperties":0,"default":{"a":1}}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","additionalProperties":{"type":"boolean","nullable":true,"enum":[null,true,null]},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/enum/2"},
	} {
		r := request(fieldDoc(tc.field), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
	}
	for _, field := range []string{
		`{"type":"array","items":{"type":"integer","nullable":true,"enum":[null,1,2],"minimum":2},"default":[null,2]}`,
		`{"type":"object","additionalProperties":{"type":["boolean","null"],"enum":[true,null]},"default":{"a":null,"b":true}}`,
	} {
		r := request(fieldDoc(field), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("Checked null vocabulary admitted without validation", field)
		}
	}
}

func TestNativeEnumCollectionDefaultsExcludedRolesAndOriginalSources(t *testing.T) {
	field := `{"type":"array","items":{"type":"integer","enum":[1,2]},"default":[1]}`
	for _, input := range []string{
		`{"Code":{"type":"integer","enum":[1,2]},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"default":[1]}}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":` + field + `}}}}`,
		`{"Holder":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + field + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":` + strings.Replace(field, `"default":[1]`, `"default":[3]`, 1) + `}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Collection enum default expanded a role or lost an original annotation", input, checked, err)
			}
		}
	}
	for _, checked := range []bool{false, true} {
		members := make([]string, 241)
		for i := range members {
			members[i] = fmt.Sprint(i)
		}
		r := request(fieldDoc(`{"type":"array","items":{"type":"integer","enum":[`+strings.Join(members, ",")+`]},"default":[]}`), "Dto")
		r.Options.Validation = checked
		_, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/items/enum" {
			t.Fatal("Native enum default lost raw vocabulary budget", checked, err)
		}
	}
}

func TestNativeEnumCollectionDefaultsAllOfRetainsOriginalChecks(t *testing.T) {
	field := `{"type":"array","items":{"type":"integer","enum":[1,2]},"default":[1]}`
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"Holder":{"allOf":[{"properties":{"value":`+field+`}}],"properties":{"label":{"type":"string"}}}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt default array Int`(\"[1]\")") {
			t.Fatal("Ordinary allOf lost its enum collection default", checked, err)
		}
		r.Document = schemas(`{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}`)
		got, err = core.GenerateWithDiagnostics(r)
		if checked {
			if err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/allOf/0/properties/value/items/enum" {
				t.Fatal("Discarded enforcing vocabulary was lost", err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			for _, warning := range got.Warnings {
				if warning.Code == "W_ENUM_NATIVE_APPROXIMATION" {
					t.Fatal("Discarded probe leaked its warning", warning)
				}
			}
		}
	}
}

func TestNativeEnumCollectionDefaultsFrozenIdentity(t *testing.T) {
	field := `{"type":"object","additionalProperties":{"type":"integer","enum":[1,2]},"default":{"b":2,"a":1}}`
	op := `{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	for _, checked := range []bool{false, true} {
		r := request([]byte(`{"openapi":"3.1.0","paths":{"/a":`+op+`,"/b":`+op+`}}`), "Dto")
		r.Options.Validation = checked
		r.Options.HoistSharedShapes = true
		r.Options.LockHoistedNames = true
		r.Options.DeduplicateModels = true
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || len(got.Artifacts.NameLock) == 0 {
			t.Fatal("Native enum defaults failed owned optimization", err)
		}
		r.PreviousNameLock = got.Artifacts.NameLock
		r.Document = []byte(strings.ReplaceAll(string(r.Document), `{"b":2,"a":1}`, `{"a":1,"b":2}`))
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Key order changed frozen default identity", checked, err)
		}
		r.Document = []byte(strings.ReplaceAll(string(r.Document), `{"a":1,"b":2}`, `{"a":2,"b":2}`))
		changed, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
			t.Fatal("Changed enum default reused a frozen model", checked, err)
		}
	}
}
