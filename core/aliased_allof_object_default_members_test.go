package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func aliasedAllOfMemberDocument(target, aliases, member, value string) []byte {
	return schemas(`{"Target":` + target + `,` + aliases + `"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
}

const allOfMemberAliases = `"Alias":{"$ref":"#/components/schemas/Target"},"Outer":{"$ref":"#/components/schemas/Alias"},`

func TestAliasedAllOfObjectDefaultMemberNullability(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, targetNull := range []bool{false, true} {
			target := allOfObjectMember
			if targetNull {
				target = strings.TrimSuffix(target, "}") + `,"nullable":true}`
			}
			for _, inner := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, outer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
						aliases := `"Alias":{"$ref":"#/components/schemas/Target"` + inner + `},"Outer":{"$ref":"#/components/schemas/Alias"` + outer + `},`
						member := `{"$ref":"#/components/schemas/Outer"` + consumer + `}`
						input := aliasedAllOfMemberDocument(target, aliases, member, `{"member":null}`)
						nullable := targetNull
						for _, intent := range []string{inner, outer, consumer} {
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
							t.Fatal(targetNull, inner, outer, consumer, checked, err)
						}
					}
				}
			}
		}
	}
}

func TestAliasedAllOfObjectDefaultMemberGuardsAndContexts(t *testing.T) {
	member := `{"$ref":"#/components/schemas/Outer"}`
	definitions := `"Target":` + allOfObjectMember + `,` + allOfMemberAliases
	object := `{"properties":{"member":` + member + `}}`
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail string }{
			{`{"member":{"label":false}}`, "/member/label"},
			{`{"member":{"label":"x","extra/~":1}}`, "/member/extra~1~0"},
			{`{"member":{}}`, "/member"},
			{`{"member":{"label":"x","child":{"flag":4}}}`, "/member/child/flag"},
		} {
			field := allOfMemberField(member, tc.value)
			for _, required := range []bool{false, true} {
				input := aliasedAllOfMemberDocument(allOfObjectMember, allOfMemberAliases, member, tc.value)
				if required {
					input = []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1))
				}
				carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default"+tc.tail)
			}
			discarded := schemas(`{` + definitions + `"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default"+tc.tail)
			patch := carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)
			patch = []byte(strings.Replace(string(patch), `"openapi":"3.1.0",`, `"openapi":"3.1.0","components":{"schemas":{`+strings.TrimSuffix(definitions, ",")+`}},`, 1))
			carrierDefaultFailure(t, patch, checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default"+tc.tail)
		}
		value := `{"member":{"label":"x"}}`
		for _, field := range []string{
			`{"type":"array","items":` + object + `,"default":[` + value + `]}`,
			`{"type":"object","additionalProperties":` + object + `,"default":{"k":` + value + `}}`,
			`{"$ref":"#/components/schemas/Envelope","default":` + value + `}`,
			`{"allOf":[` + object + `,{"properties":{"extra":{"type":"boolean"}}}],"default":` + value + `}`,
		} {
			r := request(schemas(`{`+definitions+`"Envelope":`+object+`,"Holder":{"properties":{"v":`+field+`}}}`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(field, err)
			}
		}
		// Neither omitted members nor explicit consumer intent hide alias defaults.
		for _, name := range []string{"Alias", "Outer"} {
			aliases := strings.Replace(allOfMemberAliases, `"`+name+`":{`, `"`+name+`":{"default":{"label":"hidden"},`, 1)
			for _, value := range []string{`{}`, `{"member":null}`, `{"member":{"label":"x"}}`} {
				input := aliasedAllOfMemberDocument(allOfObjectMember, aliases, `{"$ref":"#/components/schemas/Outer","nullable":true}`, value)
				carrierDefaultFailure(t, input, checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/"+name+"/default")
			}
		}
	}
	// This extension must not admit aliases ending in other source roles.
	for _, target := range []string{
		`{"properties":{"label":{"type":"string"}}}`,
		`{"type":"string"}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"tag":{"const":"x"}}}]}`,
		`{"allOf":[{},{}]}`,
		strings.TrimSuffix(allOfObjectMember, "}") + `,"default":{"label":"root"}}`,
	} {
		r := request(aliasedAllOfMemberDocument(target, allOfMemberAliases, member, `{}`), "Dto")
		if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("unsupported alias target admitted", target, err)
		}
	}
	internal := schemas(`{"Scalar":{"type":"string"},"Owner":{"properties":{"n":` + allOfObjectMember + `,"hop":{"$ref":"#/components/schemas/Scalar"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Holder":{"properties":{"v":` + allOfMemberField(`{"$ref":"#/components/schemas/Alias"}`, `{}`) + `}}}`)
	if _, err := core.GenerateWithDiagnostics(request(internal, "Dto")); err == nil {
		t.Fatal("internal chain to scalar admitted by allOf role")
	}
	cycle := strings.Replace(allOfMemberAliases, `"Alias":{"$ref":"#/components/schemas/Target"}`, `"Alias":{"$ref":"#/components/schemas/Outer"}`, 1)
	if _, err := core.GenerateWithDiagnostics(request(aliasedAllOfMemberDocument(allOfObjectMember, cycle, member, `{}`), "Dto")); err == nil {
		t.Fatal("alias cycle admitted")
	}
	bounded := strings.Replace(allOfObjectMember, `"type":"integer","default":2`, `"type":"integer","minimum":1,"default":2`, 1)
	carrierDefaultFailure(t, aliasedAllOfMemberDocument(bounded, allOfMemberAliases, member, `{"member":{"label":"x","count":0}}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/count")
	large := `{"member":{"label":"` + strings.Repeat("a", 32767) + `"}}`
	carrierDefaultFailure(t, aliasedAllOfMemberDocument(allOfObjectMember, allOfMemberAliases, member, large), false, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/member/label")
}

func TestAliasedAllOfObjectDefaultMembers(t *testing.T) {
	for _, name := range []string{"Alias", "Outer"} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := aliasedAllOfMemberDocument(allOfObjectMember, allOfMemberAliases, `{"$ref":"#/components/schemas/`+name+`"}`, `{"member":{"label":"initial"}}`)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(name, checked, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, `public val member: org.example.models.models.TargetDto?`) || !strings.Contains(text, `\"member\":{\"label\":\"initial\"}`) {
					t.Fatal(text)
				}
				r.Document = []byte(strings.Replace(string(input), `,"default":{"member":{"label":"initial"}}`, "", 1))
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
					t.Fatal("warnings changed", err)
				}
				for _, name := range []string{"TargetDto", "AliasDto", "OuterDto"} {
					if source(t, got, name) != source(t, baseline, name) {
						t.Fatal("declaration changed", name)
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
