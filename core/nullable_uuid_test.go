package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nullableUUIDFixture(t testing.TB, checked bool) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/nullable-uuid.json")
	if err != nil {
		t.Fatal(err)
	}
	if checked {
		return input
	}
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	allSchemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	branches := allSchemas["Aggregate"].(map[string]any)["allOf"].([]any)
	delete(branches[0].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any), "minLength")
	delete(branches[1].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any), "pattern")
	bounded := allSchemas["Bounded"].(map[string]any)
	for _, key := range []string{"minLength", "maxLength", "pattern"} {
		delete(bounded, key)
	}
	input, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestNullableUUIDWholeFixture(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(nullableUUIDFixture(t, checked), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			report, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct{ name, spelling string }{
				{"MaybeDto", "kotlin.uuid.Uuid?"}, {"OwnDto", "kotlin.uuid.Uuid?"}, {"BridgeDto", "kotlin.uuid.Uuid"},
				{"RowsDto", "kotlin.collections.List<kotlin.uuid.Uuid?>"}, {"ValuesDto", "kotlin.collections.Map<kotlin.String, kotlin.uuid.Uuid>"},
			} {
				if !strings.Contains(source(t, report, tc.name), "= "+tc.spelling+"\n") {
					t.Fatal("nullable UUID borrowed/own mapping lost", tc)
				}
			}
			text := source(t, report, "PatchUuidInputDto")
			for _, want := range []string{"Patch<kotlin.uuid.Uuid>", "val requiredNull: kotlin.uuid.Uuid?,", "val requiredFalse: kotlin.uuid.Uuid,"} {
				if !strings.Contains(text, want) {
					t.Fatal("nullable UUID Patch/required mapping lost", want, text)
				}
			}
		}
	}
}

func TestNullableUUIDProfiles(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, shape := range []string{
				`"type":"string","format":"uuid","nullable":true`,
				`"type":["string","null"],"format":"uuid"`,
				`"type":"string","format":"uuid"`,
			} {
				r := request(schemas(`{"Z":{`+shape+`},"A":{"$ref":"#/components/schemas/Z","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false},"C":{"$ref":"#/components/schemas/B"},"D":{"$ref":"#/components/schemas/C","nullable":true},"Holder":{"properties":{"nullable":{"$ref":"#/components/schemas/A"},"narrow":{"$ref":"#/components/schemas/C"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/D"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/B"}}},"required":["nullable","narrow"]}}`), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				report, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"ADto", "DDto"} {
					if !strings.Contains(source(t, report, name), "= kotlin.uuid.Uuid?\n") {
						t.Fatal("UUID nullable intent lost", name)
					}
				}
				for _, name := range []string{"BDto", "CDto"} {
					if !strings.Contains(source(t, report, name), "= kotlin.uuid.Uuid\n") {
						t.Fatal("UUID false backing lost", name)
					}
				}
				text := source(t, report, "HolderDto")
				for _, want := range []string{"val nullable: kotlin.uuid.Uuid? = null", "val narrow: kotlin.uuid.Uuid,", "kotlin.collections.List<kotlin.uuid.Uuid?>", "kotlin.collections.Map<kotlin.String, kotlin.uuid.Uuid>"} {
					if !strings.Contains(text, want) {
						t.Fatal("UUID consumer mapping drift", want, text)
					}
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(report, again) {
					t.Fatal("nullable UUID generation drift", err)
				}
			}
		}
	}
}

func TestNullableUUIDDefaultOwnership(t *testing.T) {
	for _, shape := range []string{`"type":"string","format":"uuid","nullable":true`, `"type":["string","null"],"format":"uuid"`} {
		for _, value := range []string{"null", `"550e8400-e29b-41d4-a716-446655440000"`} {
			for _, checked := range []bool{false, true} {
				body := `{"T":{` + shape + `,"default":` + value + `}}`
				if checked {
					body = strings.TrimSuffix(body, "}") + `,"M":{"$ref":"#/components/schemas/T","nullable":false,"default":"550e8400-e29b-41d4-a716-446655440001"},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":null},"Plain":{"$ref":"#/components/schemas/N"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/M"}}}}`
				}
				r := request(schemas(body), "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(source(t, report, "TDto"), "typealias TDto = kotlin.uuid.Uuid?") || !strings.Contains(source(t, report, "TDto"), "public fun `TDto default`(): TDto =") {
					t.Fatal("nullable UUID root factory missing")
				}
				r.Document = schemas(strings.Replace(body, `,"default":`+value, "", 1))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(report.Warnings, control.Warnings) {
					t.Fatal(err)
				}
				if checked {
					for _, name := range []string{"MDto", "NDto", "PlainDto", "HolderDto"} {
						if source(t, report, name) != source(t, control, name) {
							t.Fatal("UUID default inherited/changed another owner", name)
						}
					}
				}
			}
		}
	}
}

func TestNullableUUIDSourceAndRoleGuards(t *testing.T) {
	for _, tc := range []struct{ body, code, pointer string }{
		{`{"T":{"type":"string","format":"uuid","nullable":true,"minLength":-1,"default":null}}`, "E_SCHEMA_INVALID", "/components/schemas/T/minLength"},
		{`{"T":{"type":"string","format":"uuid","nullable":true,"minLength":2,"default":"x"}}`, "E_SCHEMA_INVALID", "/components/schemas/T/default"},
		{`{"T":{"type":"string","format":"uuid","nullable":true},"M":{"$ref":"#/components/schemas/T","nullable":false,"default":null}}`, "E_SCHEMA_INVALID", "/components/schemas/M/default"},
		{`{"T":{"type":"boolean","format":"uuid","nullable":true}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/T"},
		{`{"T":{"type":"string","format":"uri","nullable":true}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/T"},
	} {
		r := request(schemas(tc.body), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("UUID nullable source/role guard drift", tc, err)
		}
	}
	for _, target := range []string{"T", "A"} {
		body := `{"T":{"type":"string","format":"uuid","nullable":true},"A":{"$ref":"#/components/schemas/T","nullable":false},"C":{"allOf":[{"$ref":"#/components/schemas/` + target + `"},{"properties":{"id":{"type":"string"}}}]}}`
		aliasFailure(t, schemas(body), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	}
}
