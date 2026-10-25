package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const referencedAllOfCollectionItem = `{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"age":{"type":"integer","default":7},"code":{"type":"integer","enum":[1,2]},"child":{"properties":{"note":{"type":"string","default":"inner"}}}}}]}`

func referencedAllOfCollectionDefinitions() string {
	return `"Item":` + referencedAllOfCollectionItem + `,"Alias":{"$ref":"#/components/schemas/Item"},"Owner":{"properties":{"node":` + referencedAllOfCollectionItem + `}},"CollectionOwner":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"RowsAlias":{"$ref":"#/components/schemas/Rows"},"Nested":{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Owner/properties/node"}}}`
}

func referencedAllOfCollectionDocument(field string) []byte {
	return schemas(`{` + referencedAllOfCollectionDefinitions() + `,"Holder":{"properties":{"v":` + field + `}}}`)
}

func TestReferencedAllOfCollectionNullableAndSuppression(t *testing.T) {
	field := `{"$ref":"#/components/schemas/Rows","default":[null,{"name":"a"}]}`
	for _, checked := range []bool{false, true} {
		for intentCase, nullable := range []string{
			strings.Replace(string(referencedAllOfCollectionDocument(field)), `"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}`, `"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":true}`, 1),
			strings.ReplaceAll(string(referencedAllOfCollectionDocument(field)), referencedAllOfCollectionItem, `{"nullable":true,`+referencedAllOfCollectionItem[1:]),
		} {
			r := request([]byte(nullable), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("lost nullable element intent", err)
			}
			narrow := strings.Replace(nullable, `"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"`, `"Rows":{"type":"array","items":{"nullable":false,"$ref":"#/components/schemas/Alias"`, 1)
			if intentCase == 0 {
				narrow = strings.Replace(nullable, `"nullable":true`, `"nullable":false`, 1)
			}
			carrierDefaultFailure(t, []byte(narrow), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/0")
		}
		valid := `{"$ref":"#/components/schemas/Rows","nullable":true,"default":[{"name":"a"}]}`
		doc := referencedAllOfCollectionDocument(valid)
		r := request([]byte(strings.Replace(string(doc), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1)), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "default tree") {
			t.Fatal("required collection acquired initializer", err)
		}
		patch := strings.TrimSuffix(string(carrierRequestDocument(valid, false)), "}") + `,"components":{"schemas":{` + referencedAllOfCollectionDefinitions() + `}}}`
		r.Document = []byte(patch)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("Patch collection rejected", err)
		}
		seen := false
		for _, file := range got.Artifacts.Files {
			seen = seen || strings.Contains(file.Content, ".Patch<org.example.models.models.RowsDto>") && !strings.Contains(file.Content, "default tree")
		}
		if !seen {
			t.Fatal("lost Patch collection alias")
		}
		badPatch := strings.Replace(patch, `"name":"a"`, `"name":false`, 1)
		if got, err := core.GenerateWithDiagnostics(request([]byte(badPatch), "Dto")); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Patch skipped annotation checks", err)
		}
		discarded := schemas(`{` + referencedAllOfCollectionDefinitions() + `,"Holder":{"allOf":[{"properties":{"v":` + valid + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		if checked {
			carrierDefaultFailure(t, discarded, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Item/allOf/1/properties/code/enum")
		}
		// Existing no-drop checks still reject a discarded native enum. Use an
		// unchecked source to isolate valid/invalid annotation suppression.
		discarded = []byte(strings.ReplaceAll(string(discarded), `,"enum":[1,2]`, ""))
		r.Document = discarded
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "default tree") {
			t.Fatal("discarded collection leaked initializer", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"name":"a"`, `"name":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default/0/name")
		carrierDefaultFailure(t, referencedAllOfCollectionDocument(`{"$ref":"#/components/schemas/Rows","default":null}`), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
	}
}

func TestReferencedAllOfCollectionDiagnosticsAndExclusions(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail, code string }{
			{`[{}]`, "/0", "E_SCHEMA_INVALID"},
			{`[{"name":false}]`, "/0/name", "E_SCHEMA_INVALID"},
			{`[{"name":"a","code":3}]`, "/0/code", "E_SCHEMA_INVALID"},
			{`[{"z/~":1,"a":false}]`, "/0/z~1~0", "E_SCHEMA_INVALID"},
			{`[null]`, "/0", "E_SCHEMA_INVALID"},
			{`{}`, "", "E_SCHEMA_INVALID"},
			{`[{"name":"` + strings.Repeat("a", 32767) + `"}]`, "/0/name", "E_MODEL_LIMIT"},
		} {
			carrierDefaultFailure(t, referencedAllOfCollectionDocument(`{"$ref":"#/components/schemas/Rows","default":`+tc.value+`}`), checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
		}
		for _, target := range []string{
			`{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"name":{"type":"string"}}}]}`,
			`{"allOf":[{"properties":{"name":{"type":"string"}},"additionalProperties":true},{}]}`,
			`{"allOf":[{},{}]}`,
			`{"default":{"name":"target"},` + referencedAllOfCollectionItem[1:],
		} {
			r := request([]byte(strings.ReplaceAll(string(referencedAllOfCollectionDocument(`{"$ref":"#/components/schemas/Rows","default":[]}`)), referencedAllOfCollectionItem, target)), "Dto")
			r.Options.Validation = checked
			if checked && target == allOfOverrideDuplicateName {
				allOfDefaultOverridePositive(t, r)
				continue
			}
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("empty collection default authorized excluded allOf source", err)
			}
		}
		// A bare leaf uses its own retained constructor; a separate typed use
		// neither replaces its lexical owner nor changes the collection type.
		inline := schemas(`{"Rows":{"type":"array","items":` + referencedAllOfCollectionItem + `},"Peer":{"properties":{"node":{"$ref":"#/components/schemas/Rows/items"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":[]}}}}`)
		r := request(inline, "Dto")
		r.Options.Validation = checked
		if got, err := core.GenerateWithDiagnostics(r); err != nil || !strings.Contains(source(t, got, "HolderDto"), "org.example.models.models.RowsDto? = `openapi-kt default tree ") {
			t.Fatal("retained inline allOf collection leaf rejected or renamed", err)
		}
	}
}

func TestReferencedAllOfCollectionBudgetsAndCounts(t *testing.T) {
	input := referencedAllOfCollectionDocument(`{"$ref":"#/components/schemas/Rows","default":[]}`)
	counts := []byte(strings.Replace(string(input), `"Rows":{"type":"array"`, `"Rows":{"minItems":1,"type":"array"`, 1))
	carrierDefaultFailure(t, counts, true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
	tooMany := referencedAllOfCollectionDocument(`{"$ref":"#/components/schemas/Rows","default":[` + strings.Repeat("0,", 16383) + `0]}`)
	for _, checked := range []bool{false, true} {
		carrierDefaultFailure(t, tooMany, checked, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/16383")
		// Empty annotations must still bound the original source expansion.
		fanout := objectDefaultFanoutDocument(14, `{"$ref":"#/components/schemas/Rows","default":[]}`)
		fanout = []byte(strings.Replace(string(fanout), `"schemas":{`, `"schemas":{"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Item":{"allOf":[{"properties":{"child":{"$ref":"#/components/schemas/Node00"}}},{"properties":{"n":{"type":"integer"}}}]},`, 1))
		r := request(fanout, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Node") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("empty collection bypassed object source budget", err)
		}
	}
}

func TestReferencedAllOfCollectionDefaults(t *testing.T) {
	for _, tc := range []struct{ field, name string }{
		{`{"$ref":"#/components/schemas/Rows","default":[{"name":"a","child":{}}]}`, "org.example.models.models.RowsDto"},
		{`{"$ref":"#/components/schemas/RowsAlias","default":[{"name":"a"}]}`, "org.example.models.models.RowsDto"},
		{`{"$ref":"#/components/schemas/Nested","default":[{"z":{"name":"a"},"a":{"name":"b"}}]}`, "org.example.models.models.NestedDto"},
		{`{"$ref":"#/components/schemas/CollectionOwner/properties/rows","default":[{"name":"borrowed"}]}`, "kotlin.collections.List<org.example.models.models.ItemDto>"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := referencedAllOfCollectionDocument(tc.field)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.name, checked, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, "public val v: "+tc.name+"? = `openapi-kt default tree ") {
					t.Fatal("lost actual collection alias and helper", text)
				}
				r.Document = []byte(strings.Replace(string(input), tc.field, strings.Split(tc.field, `,"default":`)[0]+`}`, 1))
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || source(t, baseline, "ItemDto") != source(t, got, "ItemDto") || source(t, baseline, "OwnerDto") != source(t, got, "OwnerDto") || !reflect.DeepEqual(baseline.Warnings, got.Warnings) {
					t.Fatal("annotation changed original owner bytes/warnings", err)
				}
				r.Document = input
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic collection defaults", err)
				}
			}
		}
	}
}
