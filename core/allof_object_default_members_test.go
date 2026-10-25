package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const allOfObjectMember = `{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"count":{"type":"integer","default":2},"child":{"properties":{"flag":{"type":"boolean"}},"default":{"flag":true}}}}]}`

func allOfMemberField(member, value string) string {
	return `{"properties":{"member":` + member + `},"default":` + value + `}`
}

func TestAllOfObjectDefaultMembers(t *testing.T) {
	field := allOfMemberField(allOfObjectMember, `{"member":{"label":"initial"}}`)
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(carrierDefaultDocument(field, false), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			text := source(t, got, "HolderDto")
			if !strings.Contains(text, `public val member: Member?`) || !strings.Contains(text, `\"member\":{\"label\":\"initial\"}`) {
				t.Fatal(text)
			}
			r.Document = carrierDefaultDocument(strings.TrimSuffix(field, `,"default":{"member":{"label":"initial"}}}`)+`}`, false)
			baseline, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
				t.Fatal("type warnings changed", err, got.Warnings, baseline.Warnings)
			}
			r.Document = carrierDefaultDocument(field, false)
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic", err)
			}
		}
	}
}

func TestAllOfObjectDefaultMemberDiagnosticsAndPresence(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail string }{
			{`{"member":{"label":false}}`, "/member/label"},
			{`{"member":{"label":"x","extra/~":true}}`, "/member/extra~1~0"},
			{`{"member":{}}`, "/member"},
			{`{"member":null}`, "/member"},
			{`{"member":{"label":"x","child":{"flag":"bad"}}}`, "/member/child/flag"},
		} {
			field := allOfMemberField(allOfObjectMember, tc.value)
			for _, required := range []bool{false, true} {
				carrierDefaultFailure(t, carrierDefaultDocument(field, required), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default"+tc.tail)
			}
			nullable := strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1)
			carrierDefaultFailure(t, carrierRequestDocument(nullable, false), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default"+tc.tail)
			discarded := schemas(`{"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default"+tc.tail)
		}
		for _, member := range []string{
			strings.Replace(allOfObjectMember, `{"allOf":`, `{"nullable":true,"allOf":`, 1),
			nullableAllOfMembers(t, allOfObjectMember),
		} {
			r := request(carrierDefaultDocument(allOfMemberField(member, `{"member":null}`), false), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("effective nullable member", err)
			}
			narrowed := strings.Replace(member, `{"allOf":`, `{"nullable":false,"allOf":`, 1)
			if narrowed != member {
				carrierDefaultFailure(t, carrierDefaultDocument(allOfMemberField(narrowed, `{"member":null}`), false), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
			}
		}
	}
}

func TestAllOfObjectDefaultMemberContexts(t *testing.T) {
	object := `{"properties":{"member":` + allOfObjectMember + `}}`
	value := `{"member":{"label":"x"}}`
	for _, checked := range []bool{false, true} {
		for _, input := range [][]byte{
			carrierDefaultDocument(`{"type":"array","items":`+object+`,"default":[`+value+`]}`, false),
			carrierDefaultDocument(`{"type":"object","additionalProperties":`+object+`,"default":{"k":`+value+`}}`, false),
			schemas(`{"Target":` + object + `,"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Target","default":` + value + `}}}}`),
			carrierDefaultDocument(`{"allOf":[`+object+`,{"properties":{"extra":{"type":"boolean"}}}],"default":`+value+`}`, false),
			carrierDefaultDocument(allOfMemberField(strings.TrimSuffix(allOfObjectMember, "}")+`,"default":{"label":"own"}}`, `{}`), false),
			carrierDefaultDocument(allOfMemberField(strings.Replace(allOfObjectMember, `"child":{"properties":{"flag":{"type":"boolean"}},"default":{"flag":true}}`, `"child":`+strings.Replace(allOfObjectMember, `"child":`, `"leaf":`, 1), 1), `{"member":{"label":"outer","child":{"label":"inner"}}}`), false),
		} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("shared object walk", string(input), err)
			}
		}
		// The member's own annotation remains validated even when omitted.
		bad := allOfMemberField(strings.TrimSuffix(allOfObjectMember, "}")+`,"default":{"label":false}}`, `{}`)
		carrierDefaultFailure(t, carrierDefaultDocument(bad, false), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/properties/member/default/label")
		// Annotation-only wrappers keep their separate role; direct named refs
		// are covered by TestReferencedAllOfObjectDefaultMembers.
		for _, member := range []string{`{"allOf":[{"$ref":"#/components/schemas/Target"}]}`} {
			input := schemas(`{"Target":` + allOfObjectMember + `,"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
			r := request(input, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err == nil {
				t.Fatal("reference member role expanded", member)
			}
		}
	}
}

func TestAllOfObjectDefaultMemberBoundaries(t *testing.T) {
	for _, member := range []string{
		`{"allOf":[{},{}]}`,
		`{"allOf":[{"properties":{"tag":{"const":"x"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"additionalProperties":true}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"minProperties":1}]}`,
	} {
		for _, value := range []string{`{}`, `{"member":{}}`} {
			r := request(carrierDefaultDocument(allOfMemberField(member, value), false), "Dto")
			r.Options.Validation = true
			if member == allOfOverrideDuplicateA {
				allOfDefaultOverridePositive(t, r)
				continue
			}
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("excluded member admitted", member, value, err)
			}
		}
	}
	field := allOfMemberField(allOfObjectMember, `{"member":{"label":"`+strings.Repeat("a", 32767)+`"}}`)
	carrierDefaultFailure(t, carrierDefaultDocument(field, false), false, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/member/label")
	bounded := strings.Replace(allOfObjectMember, `"type":"integer","default":2`, `"type":"integer","minimum":1,"default":2`, 1)
	carrierDefaultFailure(t, carrierDefaultDocument(allOfMemberField(bounded, `{"member":{"label":"x","count":0}}`), false), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/count")
}
