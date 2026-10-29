package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUUIDUniqueSources(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"string","format":"uuid","nullable":true}`,
		`{"anyOf":[{"type":"string","format":"uuid"},{"type":"null"}]}`,
		`{"$ref":"#/components/schemas/Id","nullable":true}`,
		`{"$ref":"#/components/schemas/Maybe"}`,
		`{"$ref":"#/components/schemas/Bridge"}`,
		`{"$ref":"#/components/schemas/Source/properties/hop","nullable":false}`,
	} {
		doc := schemas(`{"Id":{"type":"string","format":"uuid","minLength":36},"Maybe":{"$ref":"#/components/schemas/Id","nullable":true},"Source":{"properties":{"id":{"type":"string","format":"uuid","nullable":true},"hop":{"$ref":"#/components/schemas/Source/properties/id"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"Holder":{"properties":{"values":{"type":"array","items":` + leaf + `,"uniqueItems":true}}}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(leaf, err)
		}
		if !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt uniqueItems`(value)") {
			t.Fatal("UUID uniqueness lost", leaf)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("UUID source/output drift", err)
		}
		rq.Options.Validation = false
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("UUID uniqueness missing opt-in", err)
		}
	}
}

func TestUUIDUniqueDefaults(t *testing.T) {
	for _, tc := range []struct {
		values string
		valid  bool
	}{
		{`["550e8400-e29b-41d4-a716-446655440000","550E8400E29B41D4A716446655440000"]`, false},
		{`["550E8400-E29B-41D4-A716-446655440000","550e8400-e29b-41d4-a716-446655440000"]`, false},
		{`["550E8400E29B41D4A716446655440000","550e8400-e29b-41d4-a716-446655440001",null]`, true},
		{`[null,null]`, false},
		{`["550e8400-e29b-41d4-a716-446655440000","invalid"]`, false},
	} {
		for _, nested := range []bool{false, true} {
			field := `{"type":"array","uniqueItems":true,"items":{"type":"string","format":"uuid","nullable":true},"default":` + tc.values + `}`
			pointer := "/components/schemas/Record/properties/value/default/1"
			if nested {
				field = `{"type":"array","items":{"type":"array","uniqueItems":true,"items":{"type":"string","format":"uuid","nullable":true}},"default":[` + tc.values + `]}`
				pointer = "/components/schemas/Record/properties/value/default/0/1"
			}
			rq := request(fieldDoc(field), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if tc.valid {
				if err != nil {
					t.Fatal(nested, tc, err)
				}
			} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != pointer {
				t.Fatal("UUID default equality/grammar/Pointer lost", nested, tc, err)
			}
		}
	}
}

func TestUUIDUniqueDefaultRoles(t *testing.T) {
	terminal := `{"type":"array","uniqueItems":true,"items":{"type":"string","format":"uuid"}}`
	duplicate := `["550e8400-e29b-41d4-a716-446655440000","550E8400E29B41D4A716446655440000"]`
	for _, tc := range []struct{ body, pointer string }{
		{`{"Value":` + strings.TrimSuffix(terminal, "}") + `,"default":` + duplicate + `}}`, "/components/schemas/Value/default/1"},
		{`{"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","default":` + duplicate + `}}`, "/components/schemas/Alias/default/1"},
		{`{"Value":` + terminal + `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":` + duplicate + `}}}}`, "/components/schemas/Holder/properties/value/default/1"},
	} {
		rq := request(schemas(tc.body), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
			t.Fatal("UUID default role equality/Pointer lost", tc, err)
		}
	}
}

func TestUUIDUniqueBudgetsAndExclusions(t *testing.T) {
	terminal := `{"type":["string","null"],"format":"uuid"}`
	enumUniqueCheckedBudgets(t, terminal)
	enumUniqueSourceCacheBudgets(t, terminal)
	for _, leaf := range []string{
		`{"type":"string","format":"uuid","enum":["550e8400-e29b-41d4-a716-446655440000"]}`,
		`{"type":"string","format":"uuid","const":"550e8400-e29b-41d4-a716-446655440000"}`,
		`{"type":"string","x-native-type":"kotlin.uuid.Uuid"}`,
		`{"type":"string","format":"uuid","x-kotlin-value-class":true}`,
	} {
		rq := request(schemas(`{"Id":`+leaf+`,"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Id"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if strings.Contains(leaf, `"enum"`) {
			metadataStringEnumGraduation(t, rq)
			continue
		}
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("projected source acquired UUID equality", leaf, err)
		}
	}
	jsonUniqueReferencedNoDropAndNoop(t, `"Value":`+terminal+`, `)
}

func TestUUIDUniqueCanonicalDefaultChecks(t *testing.T) {
	leaf := `{"type":"string","format":"uuid","minLength":36,"maxLength":36,"pattern":"^[0-9a-f-]+$"}`
	rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`,"default":["550E8400E29B41D4A716446655440000"]}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("UUID constraints did not observe canonical default text", err)
	}
}
