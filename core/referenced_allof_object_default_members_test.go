package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedAllOfMemberDocument(target, member, value string) []byte {
	return schemas(`{"Target":` + target + `,"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
}

func TestReferencedAllOfObjectDefaultMemberGuards(t *testing.T) {
	member := `{"$ref":"#/components/schemas/Target"}`
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail string }{
			{`{"member":{"label":false}}`, "/member/label"},
			{`{"member":{"label":"x","extra/~":true}}`, "/member/extra~1~0"},
			{`{"member":{}}`, "/member"},
			{`{"member":null}`, "/member"},
			{`{"member":{"label":"x","child":{"flag":"bad"}}}`, "/member/child/flag"},
		} {
			input := referencedAllOfMemberDocument(allOfObjectMember, member, tc.value)
			for _, required := range []bool{false, true} {
				doc := input
				if required {
					doc = []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1))
				}
				carrierDefaultFailure(t, doc, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default"+tc.tail)
			}
			field := allOfMemberField(member, tc.value)
			patch := carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)
			patch = []byte(strings.Replace(string(patch), `"openapi":"3.1.0",`, `"openapi":"3.1.0","components":{"schemas":{"Target":`+allOfObjectMember+`}},`, 1))
			carrierDefaultFailure(t, patch, checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default"+tc.tail)
			discarded := schemas(`{"Target":` + allOfObjectMember + `,"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default"+tc.tail)
		}
		for _, target := range []string{allOfObjectMember, strings.Replace(allOfObjectMember, `{"allOf":`, `{"nullable":true,"allOf":`, 1)} {
			for _, override := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				ref := strings.TrimSuffix(member, "}") + override + "}"
				input := referencedAllOfMemberDocument(target, ref, `{"member":null}`)
				nullable := override == `,"nullable":true` || override == "" && strings.Contains(target, `"nullable":true`)
				r := request(input, "Dto")
				r.Options.Validation = checked
				if _, err := core.GenerateWithDiagnostics(r); (err == nil) != nullable {
					t.Fatal("nullable precedence", target, override, err)
				}
			}
		}
	}
	for _, target := range []string{
		`{"allOf":[{},{}]}`,
		`{"allOf":[{"properties":{"tag":{"const":"x"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"additionalProperties":true}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"minProperties":1}]}`,
		strings.TrimSuffix(allOfObjectMember, "}") + `,"default":{"label":"root"}}`,
	} {
		for _, value := range []string{`{}`, `{"member":{}}`} {
			r := request(referencedAllOfMemberDocument(target, member, value), "Dto")
			r.Options.Validation = true
			if target == allOfOverrideDuplicateA {
				allOfDefaultOverridePositive(t, r)
				continue
			}
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("excluded target admitted", target, value, err)
			}
		}
	}
	for _, ref := range []string{"#/components/schemas/Owner/properties/hop"} {
		input := schemas(`{"Target":` + allOfObjectMember + `,"Scalar":{"type":"string"},"Alias":{"$ref":"#/components/schemas/Target"},"Owner":{"properties":{"n":` + allOfObjectMember + `,"hop":{"$ref":"#/components/schemas/Scalar"}}},"Holder":{"properties":{"v":` + allOfMemberField(`{"$ref":"`+ref+`"}`, `{}`) + `}}}`)
		// Preserve the original scalar-hop input as a graduated builtin member case.
		if got, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil || len(got.Artifacts.Files) == 0 {
			t.Fatal("internal scalar-chain member failed", ref, err)
		}
	}
	bounded := strings.Replace(allOfObjectMember, `"type":"integer","default":2`, `"type":"integer","minimum":1,"default":2`, 1)
	carrierDefaultFailure(t, referencedAllOfMemberDocument(bounded, member, `{"member":{"label":"x","count":0}}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/count")
	large := `{"member":{"label":"` + strings.Repeat("a", 32767) + `"}}`
	carrierDefaultFailure(t, referencedAllOfMemberDocument(allOfObjectMember, member, large), false, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/member/label")
}

func TestReferencedAllOfObjectDefaultMemberContexts(t *testing.T) {
	object := `{"properties":{"member":{"$ref":"#/components/schemas/Target"}}}`
	value := `{"member":{"label":"x"}}`
	for _, field := range []string{
		`{"type":"array","items":` + object + `,"default":[` + value + `]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":` + value + `}}`,
		`{"$ref":"#/components/schemas/Envelope","default":` + value + `}`,
		`{"allOf":[` + object + `,{"properties":{"extra":{"type":"boolean"}}}],"default":` + value + `}`,
	} {
		input := schemas(`{"Target":` + allOfObjectMember + `,"Envelope":` + object + `,"Holder":{"properties":{"v":` + field + `}}}`)
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(field, err)
			}
		}
	}
}

func TestReferencedAllOfObjectDefaultMembers(t *testing.T) {
	member := `{"$ref":"#/components/schemas/Target"}`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			input := referencedAllOfMemberDocument(allOfObjectMember, member, `{"member":{"label":"initial"}}`)
			r := request(input, "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			text := source(t, got, "HolderDto")
			if !strings.Contains(text, `public val member: org.example.models.models.TargetDto?`) || !strings.Contains(text, `\"member\":{\"label\":\"initial\"}`) {
				t.Fatal(text)
			}
			r.Document = []byte(strings.Replace(string(input), `,"default":{"member":{"label":"initial"}}`, "", 1))
			baseline, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) || source(t, got, "TargetDto") != source(t, baseline, "TargetDto") {
				t.Fatal("target or warnings changed", err)
			}
			r.Document = input
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic", err)
			}
		}
	}
}
