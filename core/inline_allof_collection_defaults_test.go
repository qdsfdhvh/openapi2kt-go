package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func inlineAllOfCollectionDocument(field string) []byte {
	return schemas(`{"Holder":{"properties":{"v":` + field + `}}}`)
}

func TestInlineAllOfCollectionPresenceAndNullability(t *testing.T) {
	for _, checked := range []bool{false, true} {
		field := `{"type":"array","nullable":true,"items":` + referencedAllOfCollectionItem + `,"default":[{"name":"a"}]}`
		r := request(carrierDefaultDocument(field, true), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "model default") {
			t.Fatal("required inline collection initializer", err)
		}
		r.Document = carrierRequestDocument(field, false)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("Patch allOf collection", err)
		}
		seen := false
		for _, file := range got.Artifacts.Files {
			seen = seen || strings.Contains(file.Content, ".Patch<kotlin.collections.List<VInner>>") && !strings.Contains(file.Content, "model default")
		}
		if !seen {
			t.Fatal("lost actual Patch collection element")
		}
		discarded := schemas(`{"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		// No-drop runtime checks keep their existing rejection before discarded
		// annotation processing; remove them only to isolate suppression.
		if checked {
			carrierDefaultFailure(t, discarded, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/allOf/0/properties/v/items/allOf/1/properties/code/enum")
		}
		discarded = []byte(strings.Replace(string(discarded), `,"enum":[1,2]`, "", 1))
		r.Document = discarded
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "model default") {
			t.Fatal("discarded initializer or scratch declaration leaked", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"name":"a"`, `"name":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default/0/name")
		nilField := strings.Replace(field, `[{"name":"a"}]`, "null", 1)
		r.Document = inlineAllOfCollectionDocument(nilField)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("explicit nullable container null", err)
		}
		carrierDefaultFailure(t, inlineAllOfCollectionDocument(strings.Replace(nilField, `,"nullable":true`, "", 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
		// Effective element null intent may come from an allOf member, without
		// making the outer container nullable. Root false narrows the element.
		element := strings.Replace(referencedAllOfCollectionItem, `"allOf":[{`, `"allOf":[{"nullable":true,`, 1)
		for _, value := range []string{"[]", "[null]", `[{"name":"a"}]`} {
			doc := inlineAllOfCollectionDocument(`{"type":"array","items":` + element + `,"default":` + value + `}`)
			r.Document = doc
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("effective member-derived element nullable", err)
			}
		}
		narrow := `{"type":"array","items":{"nullable":false,` + element[1:] + `,"default":[null]}`
		carrierDefaultFailure(t, inlineAllOfCollectionDocument(narrow), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/0")
	}
}

func TestInlineAllOfCollectionDiagnosticsAndBoundaries(t *testing.T) {
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
			field := `{"type":"array","items":` + referencedAllOfCollectionItem + `,"default":` + tc.value + `}`
			carrierDefaultFailure(t, inlineAllOfCollectionDocument(field), checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
		}
		for _, element := range []string{
			`{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"name":{"type":"string"}}}]}`,
			`{"allOf":[{"properties":{"name":{"type":"string"}},"additionalProperties":true},{}]}`,
			`{"allOf":[{"properties":{"name":{"const":"x"}}}]}`,
			`{"allOf":[{},{}]}`,
			`{"default":{"name":"element"},` + referencedAllOfCollectionItem[1:],
		} {
			r := request(inlineAllOfCollectionDocument(`{"type":"array","items":`+element+`,"default":[]}`), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("empty annotation admitted excluded source/constructor", element, err)
			}
		}
		// Keep permission on the original leaf of the consuming collection,
		// rather than allowing allOf members inside another object default.
		for _, field := range []string{
			`{"properties":{"rows":{"type":"array","items":` + referencedAllOfCollectionItem + `}},"default":{}}`,
			`{"type":"array","items":{"properties":{"node":` + referencedAllOfCollectionItem + `}},"default":[]}`,
		} {
			r := request(inlineAllOfCollectionDocument(field), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err == nil {
				t.Fatal("inline collection permission escaped to object member")
			}
		}
	}
	bounded := `{"type":"array","minItems":1,"items":` + referencedAllOfCollectionItem + `,"default":[]}`
	carrierDefaultFailure(t, inlineAllOfCollectionDocument(bounded), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
}

func TestInlineAllOfCollectionSourceBudgetAndMemberReferences(t *testing.T) {
	for _, checked := range []bool{false, true} {
		field := `{"type":"array","items":{"allOf":[{"properties":{"child":{"$ref":"#/components/schemas/Node00"}}},{"properties":{"n":{"type":"integer"}}}]},"default":[]}`
		r := request(objectDefaultFanoutDocument(14, field), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Node") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("empty inline collection bypassed shared source work", err)
		}
		// The ordinary merged shape retains its existing named contribution
		// expansion and original required fields; no fake element owner is added.
		input := schemas(`{"Base":{"properties":{"name":{"type":"string"}},"required":["name"]},"Holder":{"properties":{"v":{"type":"array","items":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"n":{"type":"integer"}}}]},"default":[{"name":"member","n":2}]}}}}`)
		r.Document = input
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("existing named allOf contribution rejected", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"name":"member"`, `"name":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/0/name")
		// Direct inline collection elements that are refs to allOf Models retain
		// their separate role; this increment grants only a bare original leaf.
		refField := schemas(`{"Item":` + referencedAllOfCollectionItem + `,"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[]}}}}`)
		r.Document = refField
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("inline collection role leaked to referenced element")
		}
	}
}

func TestInlineAllOfCollectionDefaults(t *testing.T) {
	for _, tc := range []struct{ shape, value, marker string }{
		{`{"type":"array","items":` + referencedAllOfCollectionItem + `}`, `[{"name":"a","child":{}}]`, "kotlin.collections.List<VInner>"},
		{`{"type":"object","additionalProperties":` + referencedAllOfCollectionItem + `}`, `{"k":{"name":"b"}}`, "kotlin.collections.Map<kotlin.String, VValue>"},
		{`{"type":"array","items":{"type":"object","additionalProperties":` + referencedAllOfCollectionItem + `}}`, `[{"z":{"name":"z"},"a":{"name":"a"}}]`, "kotlin.collections.List<kotlin.collections.Map<kotlin.String, VInnerValue>>"},
	} {
		field := strings.TrimSuffix(tc.shape, "}") + `,"default":` + tc.value + `}`
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				r := request(inlineAllOfCollectionDocument(field), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.marker, checked, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, "public val v: "+tc.marker+"? = `openapi-kt model default ") || !strings.Contains(text, "public val age: kotlin.Int? = 7") {
					t.Fatal("lost actual collection/constructor/default binding", text)
				}
				r.Document = inlineAllOfCollectionDocument(tc.shape)
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(baseline.Warnings, got.Warnings) || !strings.Contains(source(t, baseline, "HolderDto"), "public val v: "+tc.marker) {
					t.Fatal("annotation changed type or duplicated source warnings", err)
				}
				r.Document = inlineAllOfCollectionDocument(field)
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic inline allOf collection", err)
				}
			}
		}
	}
}
