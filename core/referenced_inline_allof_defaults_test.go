package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedInlineAllOfDefinitions() string {
	return `"Rows":{"type":"array","items":` + referencedAllOfCollectionItem + `},"RowsAlias":{"$ref":"#/components/schemas/Rows"},"Maps":{"type":"object","additionalProperties":` + referencedAllOfCollectionItem + `},"Mixed":{"type":"array","items":{"type":"object","additionalProperties":` + referencedAllOfCollectionItem + `}}`
}

func referencedInlineAllOfDocument(field string) []byte {
	return schemas(`{` + referencedInlineAllOfDefinitions() + `,"Holder":{"properties":{"v":` + field + `}}}`)
}

func TestReferencedInlineAllOfNullableAndSuppression(t *testing.T) {
	for _, checked := range []bool{false, true} {
		field := `{"$ref":"#/components/schemas/Rows","default":[null,{"name":"a"}]}`
		nullableLeaf := nullableAllOfMembers(t, referencedAllOfCollectionItem)
		input := []byte(strings.ReplaceAll(string(referencedInlineAllOfDocument(field)), referencedAllOfCollectionItem, nullableLeaf))
		r := request(input, "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("lost allOf member-derived element nullability", err)
		}
		narrow := strings.ReplaceAll(string(input), nullableLeaf, `{"nullable":false,`+nullableLeaf[1:])
		carrierDefaultFailure(t, []byte(narrow), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/0")
		container := referencedInlineAllOfDocument(`{"$ref":"#/components/schemas/RowsAlias","default":null}`)
		container = []byte(strings.Replace(string(container), `"Rows":{"type":"array"`, `"Rows":{"nullable":true,"type":"array"`, 1))
		r.Document = container
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("lost nullable container alias", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(container), `"RowsAlias":{"$ref"`, `"RowsAlias":{"nullable":false,"$ref"`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
		carrierDefaultFailure(t, referencedInlineAllOfDocument(`{"$ref":"#/components/schemas/Rows","default":null}`), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
		valid := `{"$ref":"#/components/schemas/Rows","nullable":true,"default":[{"name":"a"}]}`
		r.Document = []byte(strings.Replace(string(referencedInlineAllOfDocument(valid)), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1))
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "default tree") {
			t.Fatal("required annotation acquired initializer", err)
		}
		patch := strings.TrimSuffix(string(carrierRequestDocument(valid, false)), "}") + `,"components":{"schemas":{` + referencedInlineAllOfDefinitions() + `}}}`
		r.Document = []byte(patch)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("Patch annotation rejected", err)
		}
		seen := false
		for _, file := range got.Artifacts.Files {
			seen = seen || strings.Contains(file.Content, ".Patch<org.example.models.models.RowsDto>") && !strings.Contains(file.Content, "default tree")
		}
		if !seen {
			t.Fatal("lost Patch alias/absence")
		}
		carrierDefaultFailure(t, []byte(strings.Replace(patch, `"name":"a"`, `"name":false`, 1)), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/0/name")
		discarded := schemas(`{` + referencedInlineAllOfDefinitions() + `,"Holder":{"allOf":[{"properties":{"v":` + valid + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		if checked {
			carrierDefaultFailure(t, discarded, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/items/allOf/1/properties/code/enum")
		}
		discarded = []byte(strings.ReplaceAll(string(discarded), `,"enum":[1,2]`, ""))
		r.Document = discarded
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "default tree") {
			t.Fatal("discarded annotation acquired initializer", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"name":"a"`, `"name":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default/0/name")
	}
}

func TestReferencedInlineAllOfDiagnosticsBudgetsAndExclusions(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail, code string }{
			{`[{}]`, "/0", "E_SCHEMA_INVALID"},
			{`[{"name":false}]`, "/0/name", "E_SCHEMA_INVALID"},
			{`[{"name":"a","code":3}]`, "/0/code", "E_SCHEMA_INVALID"},
			{`[{"z/~":1,"a":false}]`, "/0/z~1~0", "E_SCHEMA_INVALID"},
			{`[null]`, "/0", "E_SCHEMA_INVALID"},
			{`{}`, "", "E_SCHEMA_INVALID"},
			{`[{"name":"` + strings.Repeat("a", 32767) + `"}]`, "/0/name", "E_MODEL_LIMIT"},
			{`[` + strings.Repeat("0,", 16383) + `0]`, "/16383", "E_MODEL_LIMIT"},
		} {
			carrierDefaultFailure(t, referencedInlineAllOfDocument(`{"$ref":"#/components/schemas/Rows","default":`+tc.value+`}`), checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
		}
		for _, target := range []string{
			`{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"name":{"type":"string"}}}]}`,
			`{"allOf":[{"properties":{"name":{"type":"string"}},"additionalProperties":true},{}]}`,
			`{"allOf":[{},{}]}`,
			`{"default":{"name":"target"},` + referencedAllOfCollectionItem[1:],
		} {
			r := request([]byte(strings.ReplaceAll(string(referencedInlineAllOfDocument(`{"$ref":"#/components/schemas/Rows","default":[]}`)), referencedAllOfCollectionItem, target)), "Dto")
			r.Options.Validation = checked
			if checked && target == allOfOverrideDuplicateName {
				allOfDefaultOverridePositive(t, r)
				continue
			}
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("empty default authorized excluded allOf source", err)
			}
		}
		fanout := objectDefaultFanoutDocument(14, `{"$ref":"#/components/schemas/Rows","default":[]}`)
		fanout = []byte(strings.Replace(string(fanout), `"schemas":{`, `"schemas":{"Rows":{"type":"array","items":{"allOf":[{"properties":{"child":{"$ref":"#/components/schemas/Node00"}}},{"properties":{"n":{"type":"integer"}}}]}},`, 1))
		r := request(fanout, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Node") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("empty annotation bypassed source expansion budget", err)
		}
	}
	counts := strings.Replace(string(referencedInlineAllOfDocument(`{"$ref":"#/components/schemas/Rows","default":[]}`)), `"Rows":{"type":"array"`, `"Rows":{"minItems":1,"type":"array"`, 1)
	carrierDefaultFailure(t, []byte(counts), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
}

func TestReferencedInlineAllOfCollectionDefaults(t *testing.T) {
	for _, tc := range []struct{ field, name string }{
		{`{"$ref":"#/components/schemas/Rows","default":[{"name":"a","child":{}}]}`, "org.example.models.models.RowsDto"},
		{`{"$ref":"#/components/schemas/RowsAlias","default":[{"name":"alias"}]}`, "org.example.models.models.RowsDto"},
		{`{"$ref":"#/components/schemas/Maps","default":{"k":{"name":"map"}}}`, "org.example.models.models.MapsDto"},
		{`{"$ref":"#/components/schemas/Mixed","default":[{"k":{"name":"mixed"}}]}`, "org.example.models.models.MixedDto"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := referencedInlineAllOfDocument(tc.field)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.name, checked, style, err)
				}
				if text := source(t, got, "HolderDto"); !strings.Contains(text, "public val v: "+tc.name+"? = `openapi-kt default tree ") {
					t.Fatal("lost existing collection type/default helper", text)
				}
				r.Document = []byte(strings.Replace(string(input), tc.field, strings.Split(tc.field, `,"default":`)[0]+`}`, 1))
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(baseline.Warnings, got.Warnings) {
					t.Fatal("annotation changed warnings", err)
				}
				for _, file := range baseline.Artifacts.Files {
					if !strings.HasSuffix(file.Path, "/HolderDto.kt") {
						found := false
						for _, other := range got.Artifacts.Files {
							found = found || file == other
						}
						if !found {
							t.Fatal("annotation changed source owner declaration", file.Path)
						}
					}
				}
				r.Document = input
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic output", err)
				}
			}
		}
	}
}
