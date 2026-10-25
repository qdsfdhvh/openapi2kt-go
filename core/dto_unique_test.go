package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNominalDTOListUniqueMappedFields(t *testing.T) {
	input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"version":{"type":"integer","default":0}},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(input, "Dto")
		rq.Options.Validation = true
		rq.Options.ModelStyle = style
		r, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !strings.Contains(source(t, r, "HolderDto"), "List<org.example.models.models.ItemDto>") {
			t.Fatal("typedactual DTOunique collection rejected", style, e)
		}
	}
}

func TestNominalDTOUniqueDefaultsUseConstructorValues(t *testing.T) {
	for _, value := range []string{`[{"id":"a"},{"id":"b"}]`, `[{"id":"a"},{"id":"a","version":0}]`} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"version":{"type":"integer","default":0}},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":` + value + `}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if strings.Contains(value, `"id":"b"`) {
			if e != nil {
				t.Fatal("validDTOdefault", e)
			}
		} else {
			ds, ok := e.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
				t.Fatal("mappedDTOdefault duplicate lost valuePointer", e)
			}
		}
	}
}

func TestDTOUniqueInlineOriginalConstructorAndNullableViews(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"properties":{"id":{"type":"string"}},"required":["id"]}}}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	if r, e := core.GenerateWithDiagnostics(rq); e != nil || !strings.Contains(source(t, r, "HolderDto"), "val id: kotlin.String,") {
		t.Fatal("originalinline DTO uniqueness", e)
	}
	for _, nullable := range []string{"true", "false"} {
		input := schemas(`{"Item":{"nullable":true,"properties":{"id":{"type":"string"}},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Item","nullable":` + nullable + `}}}}}`)
		rq.Document = input
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("DTO nearestnullable view", nullable, e)
		}
	}
}

func TestDTOUniqueCanonicalDefaultsAndExcludedRoles(t *testing.T) {
	for _, member := range []struct{ field, values string }{
		{`{"type":"string","format":"uuid"}`, `[{"id":"a","v":"550E8400-E29B-41D4-A716-446655440000"},{"id":"a","v":"550e8400-e29b-41d4-a716-446655440000"}]`},
		{`{"type":"string","format":"date-time"}`, `[{"id":"a","v":"2024-01-01T00:00:00Z"},{"id":"a","v":"2024-01-01T01:00:00+01:00"}]`},
		{`{"type":"number","format":"float"}`, `[{"id":"a","v":0},{"id":"a","v":-0}]`},
	} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"v":` + member.field + `},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":` + member.values + `}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("mappedDTO default equality mismatch", member.field, e)
		}
	}
	for _, item := range []string{
		`{"properties":{"id":{"type":"string"},"child":{"properties":{"v":{"type":"string"}}}}}`,
		`{"properties":{"id":{"type":"string"},"ro":{"type":"string","readOnly":true}}}`,
		`{"minProperties":1,"properties":{"id":{"type":"string"}}}`,
		`{"allOf":[{"properties":{"id":{"type":"string"}}}]}`,
	} {
		input := schemas(`{"Item":` + item + `,"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
		if item == `{"allOf":[{"properties":{"id":{"type":"string"}}}]}` {
			dtoAllOfUniqueGraduation(t, input, "HolderDto")
			continue
		}
		rq := request(input, "Dto")
		rq.Options.Validation = true
		if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("DTO uniqueness inferred unproved codec/projection/nested/const/count/composition", item, e)
		}
	}
}

func dtoUniqueGraduation(t *testing.T, input []byte) {
	t.Helper()
	rq := request(input, "Dto")
	rq.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(rq)
	if e != nil {
		t.Fatal("originalordinary DTO leaf nowuses mappedconstructor keys", e)
	}
	text := source(t, got, "RecordDto")
	if !strings.Contains(text, "kotlin.collections.List<") || !strings.Contains(text, "public data class ValueInner") || strings.Contains(text, "`openapi-kt unique JSON items`") {
		t.Fatal("ordinary DTO did not retain its actualconstructor/typed equality", text)
	}
	rq.Options.Validation = false
	if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
		t.Fatal("DTOuniqueness lost validation optin", e)
	}
}

func TestDTOUniqueRequiredSuppressionAndInvalidOwnDefault(t *testing.T) {
	input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"version":{"type":"integer","default":0}},"required":["id"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":[{"id":"a"},{"id":"a","version":0}]}},"required":["values"]}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, e := core.GenerateWithDiagnostics(rq)
	ds, ok := e.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
		t.Fatal("requiredsuppression skipped DTOduplicate annotation", e)
	}
	input = schemas(`{"Item":{"properties":{"id":{"type":"string"},"version":{"type":"integer","minimum":1,"default":0}},"required":["id","version"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	rq.Document = input
	r, e = core.GenerateWithDiagnostics(rq)
	ds, ok = e.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/version/default" || len(r.Artifacts.Files) != 0 {
		t.Fatal("required scalar initializer validation skipped", e)
	}
}
