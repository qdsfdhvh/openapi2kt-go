package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestAllOfCollectionObjectDefaultMemberDiagnostics(t *testing.T) {
	collection := `{"type":"array","items":` + allOfObjectMember + `}`
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail, code string }{
			{`[{}]`, "/0", "E_SCHEMA_INVALID"},
			{`[{"label":false}]`, "/0/label", "E_SCHEMA_INVALID"},
			{`[{"z/~":1,"label":false}]`, "/0/z~1~0", "E_SCHEMA_INVALID"},
			{`[{"label":"x","child":{"flag":1}}]`, "/0/child/flag", "E_SCHEMA_INVALID"},
			{`[null]`, "/0", "E_SCHEMA_INVALID"},
			{`null`, "", "E_SCHEMA_INVALID"},
			{`{}`, "", "E_SCHEMA_INVALID"},
			{`[{"label":"` + strings.Repeat("x", 32767) + `"}]`, "/0/label", "E_MODEL_LIMIT"},
		} {
			field := allOfMemberField(collection, `{"member":`+tc.value+`}`)
			for _, required := range []bool{false, true} {
				carrierDefaultFailure(t, carrierDefaultDocument(field, required), checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
			}
			patch := strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1)
			carrierDefaultFailure(t, carrierRequestDocument(patch, false), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member"+tc.tail)
			discarded := schemas(`{"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default/member"+tc.tail)
		}
		// The member's own field default is validated even when the enclosing
		// object omits the collection. Element and inner-container defaults differ.
		own := strings.TrimSuffix(collection, "}") + `,"default":[{"label":false}]}`
		carrierDefaultFailure(t, carrierDefaultDocument(allOfMemberField(own, `{}`), false), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/properties/member/default/0/label")
	}
	for _, tc := range []struct{ schema, value, tail string }{
		{`{"type":"array","minItems":1,"items":` + allOfObjectMember + `}`, `[]`, ""},
		{`{"type":"object","minProperties":1,"additionalProperties":` + allOfObjectMember + `}`, `{}`, ""},
		{`{"type":"array","items":` + strings.Replace(allOfObjectMember, `"type":"integer","default":2`, `"type":"integer","minimum":1,"default":2`, 1) + `}`, `[{"label":"x","count":0}]`, "/0/count"},
	} {
		carrierDefaultFailure(t, carrierDefaultDocument(allOfMemberField(tc.schema, `{"member":`+tc.value+`}`), false), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member"+tc.tail)
	}
}

func TestAllOfCollectionObjectDefaultMemberNullability(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, containerNull := range []bool{false, true} {
			for _, elementNull := range []bool{false, true} {
				for _, narrow := range []bool{false, true} {
					leaf := allOfObjectMember
					if elementNull {
						leaf = strings.Replace(leaf, `"allOf":[{`, `"allOf":[{"nullable":true,`, 1)
					}
					if narrow {
						leaf = strings.Replace(leaf, `{"allOf":`, `{"nullable":false,"allOf":`, 1)
					}
					schema := `{"type":"array","items":` + leaf
					if containerNull {
						schema += `,"nullable":true`
					}
					schema += `}`
					for _, value := range []string{`null`, `[null]`, `[]`, `[{"label":"x"}]`} {
						input := carrierDefaultDocument(allOfMemberField(schema, `{"member":`+value+`}`), false)
						valid := value != "null" && value != "[null]" || value == "null" && containerNull || value == "[null]" && elementNull && !narrow
						if !valid {
							tail := ""
							if value == "[null]" {
								tail = "/0"
							}
							carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member"+tail)
							continue
						}
						r := request(input, "Dto")
						r.Options.Validation = checked
						if _, err := core.GenerateWithDiagnostics(r); err != nil {
							t.Fatal(containerNull, elementNull, narrow, value, err)
						}
					}
				}
			}
		}
	}
}

func TestAllOfCollectionObjectDefaultMemberGuards(t *testing.T) {
	for _, leaf := range []string{
		`{"allOf":[{},{}]}`,
		`{"allOf":[{"properties":{"tag":{"const":"x"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"additionalProperties":true}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"minProperties":1}]}`,
		strings.TrimSuffix(allOfObjectMember, "}") + `,"default":{"label":"child"}}`,
	} {
		for _, value := range []string{`{}`, `{"member":[]}`, `{"member":null}`} {
			input := carrierDefaultDocument(allOfMemberField(`{"type":"array","nullable":true,"items":`+leaf+`}`, value), false)
			r := request(input, "Dto")
			r.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("excluded source admitted", leaf, value, err)
			}
		}
	}
	for _, member := range []string{
		`{"type":"array","items":{"type":"array","items":` + allOfObjectMember + `,"default":[]}}`,
		`{"$ref":"#/components/schemas/Rows"}`,
		`{"type":"array","items":{"allOf":[{"$ref":"#/components/schemas/Target"}]}}`,
	} {
		input := schemas(`{"Target":` + allOfObjectMember + `,"Rows":{"type":"array","items":` + allOfObjectMember + `},"Holder":{"properties":{"v":` + allOfMemberField(member, `{}`) + `}}}`)
		if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err == nil {
			t.Fatal("collection permission escaped", member)
		}
	}
	// Planning all sources is mandatory, even with no collection entries.
	member := `{"type":"array","items":{"allOf":[{"properties":{"child":{"$ref":"#/components/schemas/Node00"}}},{"properties":{"n":{"type":"integer"}}}]}}`
	r := request(objectDefaultFanoutDocument(14, allOfMemberField(member, `{}`)), "Dto")
	got, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
		t.Fatal("source fanout unbounded", err)
	}
}

func TestAllOfCollectionObjectDefaultMemberContexts(t *testing.T) {
	collection := `{"type":"array","items":{"type":"object","additionalProperties":` + allOfObjectMember + `}}`
	object := `{"properties":{"member":` + collection + `}}`
	value := `{"member":[{"k":{"label":"x"}}]}`
	for _, input := range [][]byte{
		carrierDefaultDocument(`{"type":"array","items":`+object+`,"default":[`+value+`]}`, false),
		carrierDefaultDocument(`{"type":"object","additionalProperties":`+object+`,"default":{"k":`+value+`}}`, false),
		carrierDefaultDocument(`{"allOf":[`+object+`,{"properties":{"n":{"type":"integer"}}}],"default":`+value+`}`, false),
		schemas(`{"Target":` + object + `,"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Target","default":` + value + `}}}}`),
		carrierDefaultDocument(allOfMemberField(strings.TrimSuffix(collection, "}")+`,"default":[{"k":{"label":"own"}}]}`, `{}`), false),
		carrierDefaultDocument(allOfMemberField(`{"type":"array","items":{"allOf":[`+object+`,{"properties":{"n":{"type":"integer"}}}]}}`, `{"member":[`+value+`]}`), false),
	} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(string(input), err)
			}
		}
	}
}

func TestAllOfCollectionObjectDefaultMembers(t *testing.T) {
	for _, tc := range []struct{ schema, value, ty string }{
		{`{"type":"array","items":` + allOfObjectMember + `}`, `[{"label":"x"}]`, "kotlin.collections.List<MemberInner>"},
		{`{"type":"object","additionalProperties":` + allOfObjectMember + `}`, `{"k":{"label":"x"}}`, "kotlin.collections.Map<kotlin.String, MemberValue>"},
		{`{"type":"array","items":{"type":"object","additionalProperties":` + allOfObjectMember + `}}`, `[{"k":{"label":"x"}}]`, "kotlin.collections.List<kotlin.collections.Map<kotlin.String, MemberInnerValue>>"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				field := allOfMemberField(tc.schema, `{"member":`+tc.value+`}`)
				r := request(carrierDefaultDocument(field, false), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.schema, checked, style, err)
				}
				if text := source(t, got, "HolderDto"); !strings.Contains(text, "public val member: "+tc.ty+"?") || !strings.Contains(text, `\"label\":\"x\"`) {
					t.Fatal(text)
				}
				r.Document = carrierDefaultDocument(`{"properties":{"member":`+tc.schema+`}}`, false)
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
					t.Fatal("warning drift", err)
				}
				r.Document = carrierDefaultDocument(field, false)
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic", err)
				}
			}
		}
	}
}
