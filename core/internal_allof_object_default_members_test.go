package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func internalAllOfMemberDocument(target, member, value string) []byte {
	return schemas(`{"Owner":{"properties":{"node":` + target + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
}

func TestInternalAllOfObjectDefaultMemberScopes(t *testing.T) {
	for _, tc := range []struct{ owner, path, actual, value string }{
		{`{"properties":{"nodes":{"type":"array","items":` + allOfObjectMember + `}}}`, "Owner/properties/nodes/items", "OwnerDto.NodesInner", `{"label":"x"}`},
		{`{"properties":{"nodes":{"additionalProperties":` + allOfObjectMember + `}}}`, "Owner/properties/nodes/additionalProperties", "OwnerDto.NodesValue", `{"label":"x"}`},
		{`{"type":"array","items":` + allOfObjectMember + `}`, "Owner/items", "OwnerDtoInner", `{"label":"x"}`},
		{`{"additionalProperties":` + allOfObjectMember + `}`, "Owner/additionalProperties", "OwnerDtoValue", `{"label":"x"}`},
		{`{"properties":{"wrapper":{"properties":{"node":` + allOfObjectMember + `}}}}`, "Owner/properties/wrapper/properties/node", "OwnerDto.Wrapper.Node", `{"label":"x"}`},
		{`{"oneOf":[{"allOf":[{"properties":{"kind":{"const":"all"},"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer","default":7}}}]},{"properties":{"kind":{"const":"other"},"label":{"type":"integer"}},"required":["label"]}]}`, "Owner/oneOf/0", "OwnerDto.All", `{"kind":"all","label":"x"}`},
		{`{"properties":{"n/~":` + allOfObjectMember + `}}`, "Owner/properties/n~1~0", "", `{"label":"escaped"}`},
	} {
		for _, checked := range []bool{false, true} {
			input := schemas(`{"Owner":` + tc.owner + `,"Alias":{"$ref":"#/components/schemas/` + tc.path + `"},"Holder":{"properties":{"v":` + allOfMemberField(`{"$ref":"#/components/schemas/Alias"}`, `{"member":`+tc.value+`}`) + `}}}`)
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(tc.path, checked, err)
			}
			if tc.actual != "" && !strings.Contains(source(t, got, "HolderDto"), "public val member: org.example.models.models."+tc.actual+"?") {
				t.Fatal(source(t, got, "HolderDto"))
			}
		}
	}
}

func TestInternalAllOfObjectDefaultMemberNullable(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, targetIntent := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			target := strings.TrimSuffix(strings.Replace(allOfObjectMember, `"allOf":[{`, `"allOf":[{"nullable":true,`, 1), "}") + targetIntent + "}"
			for _, aliasIntent := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					member := `{"$ref":"#/components/schemas/Alias"` + consumer + `}`
					input := internalAllOfMemberDocument(target, member, `{"member":null}`)
					input = []byte(strings.Replace(string(input), `"Alias":{"$ref":"#/components/schemas/Owner/properties/node"}`, `"Alias":{"$ref":"#/components/schemas/Owner/properties/node"`+aliasIntent+`}`, 1))
					nullable := targetIntent != `,"nullable":false`
					for _, intent := range []string{aliasIntent, consumer} {
						if intent != "" {
							nullable = intent == `,"nullable":true`
						}
					}
					if !nullable {
						carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
						continue
					}
					r := request(input, "Dto")
					r.Options.Validation = checked
					if _, err := core.GenerateWithDiagnostics(r); err != nil {
						t.Fatal(targetIntent, aliasIntent, consumer, err)
					}
				}
			}
		}
	}
}

func TestInternalAllOfObjectDefaultMemberGuards(t *testing.T) {
	member := `{"$ref":"#/components/schemas/Alias"}`
	definitions := `"Owner":{"properties":{"node":` + allOfObjectMember + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"}`
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail, code string }{
			{`{"member":{"label":false}}`, "/member/label", "E_SCHEMA_INVALID"},
			{`{"member":{}}`, "/member", "E_SCHEMA_INVALID"},
			{`{"member":null}`, "/member", "E_SCHEMA_INVALID"},
			{`{"member":{"z/~":1,"label":false}}`, "/member/z~1~0", "E_SCHEMA_INVALID"},
			{`{"member":{"label":"x","child":{"flag":"bad"}}}`, "/member/child/flag", "E_SCHEMA_INVALID"},
			{`{"member":{"label":"` + strings.Repeat("x", 32767) + `"}}`, "/member/label", "E_MODEL_LIMIT"},
		} {
			input := internalAllOfMemberDocument(allOfObjectMember, member, tc.value)
			carrierDefaultFailure(t, input, checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
			required := []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1))
			carrierDefaultFailure(t, required, checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
			field := allOfMemberField(member, tc.value)
			patch := strings.TrimSuffix(string(carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)), "}") + `,"components":{"schemas":{` + definitions + `}}}`
			carrierDefaultFailure(t, []byte(patch), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default"+tc.tail)
			discarded := schemas(`{` + definitions + `,"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default"+tc.tail)
		}
	}
	for _, target := range []string{
		`{"allOf":[{},{}]}`,
		`{"allOf":[{"properties":{"tag":{"const":"x"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"additionalProperties":true}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"minProperties":1}]}`,
		strings.TrimSuffix(allOfObjectMember, "}") + `,"default":{"label":"target"}}`,
	} {
		for _, value := range []string{`{}`, `{"member":{"label":"x"}}`} {
			r := request(internalAllOfMemberDocument(target, member, value), "Dto")
			r.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("excluded source admitted", target, err)
			}
		}
	}
	hierarchy := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind","mapping":{"first":"First","second":"Second"}}},"First":{"properties":{"kind":{"const":"first"},"node":` + allOfObjectMember + `}},"Second":{"properties":{"kind":{"const":"second"},"id":{"type":"integer"}}},"Holder":{"properties":{"v":` + allOfMemberField(`{"$ref":"#/components/schemas/First/properties/node"}`, `{}`) + `}}}`)
	carrierDefaultFailure(t, hierarchy, false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/First/properties/node/allOf")
	bounded := strings.Replace(allOfObjectMember, `"type":"integer","default":2`, `"type":"integer","minimum":1,"default":2`, 1)
	carrierDefaultFailure(t, internalAllOfMemberDocument(bounded, member, `{"member":{"label":"x","count":0}}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/count")
}

func TestInternalAllOfObjectDefaultMemberContexts(t *testing.T) {
	object := `{"properties":{"member":{"$ref":"#/components/schemas/Alias"}}}`
	value := `{"member":{"label":"x"}}`
	definitions := `"Owner":{"properties":{"node":` + allOfObjectMember + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Envelope":` + object + `,`
	for _, field := range []string{
		`{"type":"array","items":` + object + `,"default":[` + value + `]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":` + value + `}}`,
		`{"$ref":"#/components/schemas/Envelope","default":` + value + `}`,
		`{"allOf":[` + object + `,{"properties":{"extra":{"type":"boolean"}}}],"default":` + value + `}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(`{`+definitions+`"Holder":{"properties":{"v":`+field+`}}}`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(field, err)
			}
		}
	}
}

func TestInternalAllOfObjectDefaultMembers(t *testing.T) {
	for _, ref := range []string{"Owner/properties/node", "Alias"} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := internalAllOfMemberDocument(allOfObjectMember, `{"$ref":"#/components/schemas/`+ref+`"}`, `{"member":{"label":"initial"}}`)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(ref, checked, style, err)
				}
				if text := source(t, got, "HolderDto"); !strings.Contains(text, `public val member: org.example.models.models.OwnerDto.Node?`) || !strings.Contains(text, `\"member\":{\"label\":\"initial\"}`) {
					t.Fatal(text)
				}
				r.Document = []byte(strings.Replace(string(input), `,"default":{"member":{"label":"initial"}}`, "", 1))
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
					t.Fatal("warnings changed", err)
				}
				for _, name := range []string{"OwnerDto", "AliasDto"} {
					if source(t, got, name) != source(t, baseline, name) {
						t.Fatal("target changed", name)
					}
				}
				r.Document = input
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic", err)
				}
			}
		}
	}
}
