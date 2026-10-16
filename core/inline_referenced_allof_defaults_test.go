package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func inlineReferencedAllOfDocument(field string) []byte {
	return schemas(`{` + referencedAllOfCollectionDefinitions() + `,"Bridge":{"properties":{"node":{"$ref":"#/components/schemas/Owner/properties/node"}}},"NodeAlias":{"$ref":"#/components/schemas/Bridge/properties/node"},"Inline":{"properties":{"v":` + field + `}}}`)
}

func TestInlineReferencedAllOfPresenceNullableAndSuppression(t *testing.T) {
	for _, checked := range []bool{false, true} {
		field := `{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Item"},"default":[{"name":"a"}]}`
		input := inlineReferencedAllOfDocument(field)
		r := request([]byte(strings.Replace(string(input), `"Inline":{"properties"`, `"Inline":{"required":["v"],"properties"`, 1)), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "InlineDto"), "default tree") {
			t.Fatal("required inline ref collection acquired initializer", err)
		}
		patch := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{` + referencedAllOfCollectionDefinitions() + `}}}`
		r.Document = []byte(patch)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("Patch collection", err)
		}
		seen := false
		for _, file := range got.Artifacts.Files {
			seen = seen || strings.Contains(file.Content, ".Patch<kotlin.collections.List<org.example.models.models.ItemDto>>") && !strings.Contains(file.Content, "default tree")
		}
		if !seen {
			t.Fatal("lost actual Patch element type")
		}
		if got, err := core.GenerateWithDiagnostics(request([]byte(strings.Replace(patch, `"name":"a"`, `"name":false`, 1)), "Dto")); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Patch annotation skipped checks", err)
		}
		discarded := schemas(`{` + referencedAllOfCollectionDefinitions() + `,"Inline":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		if checked {
			carrierDefaultFailure(t, discarded, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Item/allOf/1/properties/code/enum")
		}
		discarded = []byte(strings.ReplaceAll(string(discarded), `,"enum":[1,2]`, ""))
		r.Document = discarded
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "InlineDto"), "default tree") {
			t.Fatal("discarded default leaked or failed", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"name":"a"`, `"name":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Inline/allOf/0/properties/v/default/0/name")
		for _, sourceNullable := range []bool{false, true} {
			field := `{"type":"array","items":{"$ref":"#/components/schemas/Item","nullable":true},"default":[null,{"name":"a"}]}`
			doc := inlineReferencedAllOfDocument(field)
			if sourceNullable {
				doc = []byte(strings.Replace(string(doc), `"Item":`+referencedAllOfCollectionItem, `"Item":{"nullable":true,`+referencedAllOfCollectionItem[1:], 1))
			}
			r.Document = doc
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("explicit nullable element rejected", err)
			}
			carrierDefaultFailure(t, []byte(strings.Replace(string(doc), `"#/components/schemas/Item","nullable":true`, `"#/components/schemas/Item","nullable":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Inline/properties/v/default/0")
		}
		nilField := strings.Replace(field, `[{"name":"a"}]`, "null", 1)
		r.Document = inlineReferencedAllOfDocument(nilField)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("nullable outer collection null", err)
		}
		carrierDefaultFailure(t, inlineReferencedAllOfDocument(strings.Replace(nilField, `,"nullable":true`, "", 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Inline/properties/v/default")
	}
}

func TestInlineReferencedAllOfDiagnosticsBudgetsAndExclusions(t *testing.T) {
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
			field := `{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":` + tc.value + `}`
			carrierDefaultFailure(t, inlineReferencedAllOfDocument(field), checked, tc.code, "/components/schemas/Inline/properties/v/default"+tc.tail)
		}
		field := `{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":[]}`
		for _, target := range []string{
			`{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"name":{"type":"string"}}}]}`,
			`{"allOf":[{"properties":{"name":{"type":"string"}},"additionalProperties":true},{}]}`,
			`{"allOf":[{},{}]}`,
			`{"default":{"name":"target"},` + referencedAllOfCollectionItem[1:],
		} {
			r := request([]byte(strings.ReplaceAll(string(inlineReferencedAllOfDocument(field)), referencedAllOfCollectionItem, target)), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("empty value authorized excluded target", err)
			}
		}
		chainDefault := strings.Replace(string(inlineReferencedAllOfDocument(field)), `"Alias":{"$ref":"#/components/schemas/Item"}`, `"Alias":{"$ref":"#/components/schemas/Item","default":{"name":"alias"}}`, 1)
		carrierDefaultFailure(t, []byte(chainDefault), checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Alias/default")
		childDefault := strings.Replace(field, `"items":{"$ref"`, `"items":{"default":{"name":"child"},"$ref"`, 1)
		carrierDefaultFailure(t, inlineReferencedAllOfDocument(childDefault), checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Inline/properties/v/items/default")
		// Role detection must stop at object-member edges. A default on that
		// outer object cannot authorize the nested collection's source role.
		r := request(inlineReferencedAllOfDocument(`{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}}},"default":{}}`), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("inline collection role escaped into object default")
		}
		fanout := objectDefaultFanoutDocument(14, `{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[]}`)
		fanout = []byte(strings.Replace(string(fanout), `"schemas":{`, `"schemas":{"Item":{"allOf":[{"properties":{"child":{"$ref":"#/components/schemas/Node00"}}},{"properties":{"n":{"type":"integer"}}}]},`, 1))
		r.Document = fanout
		got, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Node") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("empty inline collection bypassed original source work budget", err)
		}
	}
	bounded := `{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Item"},"default":[]}`
	carrierDefaultFailure(t, inlineReferencedAllOfDocument(bounded), true, "E_SCHEMA_INVALID", "/components/schemas/Inline/properties/v/default")
}

func TestInlineReferencedAllOfDefaults(t *testing.T) {
	for _, tc := range []struct{ shape, value, ty string }{
		{`{"type":"array","items":{"$ref":"#/components/schemas/Item"}}`, `[{"name":"a","child":{}}]`, "kotlin.collections.List<org.example.models.models.ItemDto>"},
		{`{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Alias"}}`, `{"z":{"name":"a"},"a":{"name":"b"}}`, "kotlin.collections.Map<kotlin.String, org.example.models.models.ItemDto>"},
		{`{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/NodeAlias"}}}`, `[{"k":{"name":"internal"}}]`, "kotlin.collections.List<kotlin.collections.Map<kotlin.String, org.example.models.models.OwnerDto.Node>>"},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				field := strings.TrimSuffix(tc.shape, "}") + `,"default":` + tc.value + `}`
				r := request(inlineReferencedAllOfDocument(field), "Dto")
				r.Options.ModelStyle, r.Options.Validation = style, checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(style, checked, tc.ty, err)
				}
				if text := source(t, got, "InlineDto"); !strings.Contains(text, "public val v: "+tc.ty+"? = `openapi-kt default tree ") {
					t.Fatal("lost original public collection type/helper", text)
				}
				r.Document = inlineReferencedAllOfDocument(tc.shape)
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || source(t, baseline, "ItemDto") != source(t, got, "ItemDto") || source(t, baseline, "OwnerDto") != source(t, got, "OwnerDto") || !reflect.DeepEqual(baseline.Warnings, got.Warnings) {
					t.Fatal("annotation changed owner sources/warnings", err)
				}
				r.Document = inlineReferencedAllOfDocument(field)
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic default plan", err)
				}
			}
		}
	}
}
