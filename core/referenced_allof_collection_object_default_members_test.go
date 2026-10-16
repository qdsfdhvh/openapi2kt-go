package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedAllOfCollectionObjectDefaultMemberNullable(t *testing.T) {
	target := strings.TrimSuffix(allOfObjectMember, "}") + `,"nullable":true}`
	for _, internal := range []bool{false, true} {
		for _, checked := range []bool{false, true} {
			for _, near := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, element := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					for _, container := range []string{"", `,"nullable":true`} {
						hops := strings.Replace(allOfMemberHops, `"first":{"$ref":"#/components/schemas/Hops/properties/second"}`, `"first":{"$ref":"#/components/schemas/Hops/properties/second"`+near+`}`, 1)
						member := `{"type":"array"` + container + `,"items":{"$ref":"#/components/schemas/Bridge"` + element + `}}`
						for _, value := range []string{`null`, `[null]`, `[{"label":"x"}]`} {
							nullable := true
							for _, intent := range []string{near, element} {
								if intent != "" {
									nullable = intent == `,"nullable":true`
								}
							}
							valid := value == `[{"label":"x"}]` || value == "null" && container != "" || value == "[null]" && nullable
							input := chainedAllOfMemberDocument(target, hops, member, `{"member":`+value+`}`, internal)
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
								t.Fatal(internal, near, element, container, value, err)
							}
						}
					}
				}
			}
		}
	}
}

func TestReferencedAllOfCollectionObjectDefaultMemberDiagnostics(t *testing.T) {
	member := `{"type":"array","items":{"$ref":"#/components/schemas/Bridge"}}`
	for _, internal := range []bool{false, true} {
		for _, checked := range []bool{false, true} {
			for _, tc := range []struct{ value, tail, code string }{
				{`[{}]`, "/0", "E_SCHEMA_INVALID"},
				{`[{"label":false}]`, "/0/label", "E_SCHEMA_INVALID"},
				{`[{"z/~":1,"label":false}]`, "/0/z~1~0", "E_SCHEMA_INVALID"},
				{`[{"label":"x","child":{"flag":false}}]`, "", ""},
				{`[{"label":"x","child":{"flag":1}}]`, "/0/child/flag", "E_SCHEMA_INVALID"},
				{`[null]`, "/0", "E_SCHEMA_INVALID"},
				{`null`, "", "E_SCHEMA_INVALID"},
				{`[{"label":"` + strings.Repeat("x", 32767) + `"}]`, "/0/label", "E_MODEL_LIMIT"},
			} {
				input := chainedAllOfMemberDocument(allOfObjectMember, allOfMemberHops, member, `{"member":`+tc.value+`}`, internal)
				if tc.code == "" {
					r := request(input, "Dto")
					r.Options.Validation = checked
					if _, err := core.GenerateWithDiagnostics(r); err != nil {
						t.Fatal(err)
					}
					continue
				}
				carrierDefaultFailure(t, input, checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
				required := []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1))
				carrierDefaultFailure(t, required, checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
				// Keep identical definitions while changing only the consuming context.
				defs := strings.SplitN(strings.TrimPrefix(string(input), `{"openapi":"3.1.0","components":{"schemas":{`), `"Holder":`, 2)[0]
				field := allOfMemberField(member, `{"member":`+tc.value+`}`)
				discarded := schemas(`{` + defs + `"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
				carrierDefaultFailure(t, discarded, checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default/member"+tc.tail)
				patch := strings.TrimSuffix(string(carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(defs, ",") + `}}}`
				carrierDefaultFailure(t, []byte(patch), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member"+tc.tail)
			}
			for _, value := range []string{`{}`, `{"member":[]}`, `{"member":null}`} {
				nullableMember := strings.Replace(member, `{"type":"array"`, `{"type":"array","nullable":true`, 1)
				hops := strings.Replace(allOfMemberHops, `"second":{"$ref":"#/components/schemas/Alias"}`, `"second":{"$ref":"#/components/schemas/Alias","default":{"label":"hidden"}}`, 1)
				input := chainedAllOfMemberDocument(allOfObjectMember, hops, nullableMember, value, internal)
				carrierDefaultFailure(t, input, checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Hops/properties/second/default")
				badElement := strings.Replace(nullableMember, `"items":{"$ref":`, `"items":{"default":{"label":"hidden"},"$ref":`, 1)
				carrierDefaultFailure(t, chainedAllOfMemberDocument(allOfObjectMember, allOfMemberHops, badElement, value, internal), checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/v/properties/member/items/default")
			}
		}
	}
	bounded := strings.Replace(allOfObjectMember, `"type":"integer","default":2`, `"type":"integer","minimum":1,"default":2`, 1)
	carrierDefaultFailure(t, chainedAllOfMemberDocument(bounded, allOfMemberHops, member, `{"member":[{"label":"x","count":0}]}`, true), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/0/count")
	counted := strings.Replace(member, `"type":"array"`, `"type":"array","minItems":1`, 1)
	carrierDefaultFailure(t, chainedAllOfMemberDocument(allOfObjectMember, allOfMemberHops, counted, `{"member":[]}`, true), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
}

func TestReferencedAllOfCollectionObjectDefaultMemberGuards(t *testing.T) {
	member := `{"type":"array","items":{"$ref":"#/components/schemas/Bridge"}}`
	for _, target := range []string{
		`{"allOf":[{},{}]}`,
		`{"allOf":[{"properties":{"tag":{"const":"x"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"additionalProperties":true}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"minProperties":1}]}`,
		strings.TrimSuffix(allOfObjectMember, "}") + `,"default":{"label":"target"}}`,
		`{"type":"array","items":` + allOfObjectMember + `}`,
	} {
		for _, value := range []string{`{}`, `{"member":[]}`} {
			r := request(chainedAllOfMemberDocument(target, allOfMemberHops, member, value, false), "Dto")
			r.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("excluded terminal admitted", target, value, err)
			}
		}
	}
	cycle := strings.Replace(allOfMemberHops, `"second":{"$ref":"#/components/schemas/Alias"}`, `"second":{"$ref":"#/components/schemas/Hops/properties/first"}`, 1)
	if _, err := core.GenerateWithDiagnostics(request(chainedAllOfMemberDocument(allOfObjectMember, cycle, member, `{}`, false), "Dto")); err == nil {
		t.Fatal("cycle admitted")
	}
}

func TestReferencedAllOfCollectionObjectDefaultMemberContexts(t *testing.T) {
	collection := `{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Bridge"}}}`
	object := `{"properties":{"member":` + collection + `}}`
	value := `{"member":[{"k":{"label":"x"}}]}`
	definitions := `"Owner":{"properties":{"node":` + allOfObjectMember + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},` + allOfMemberHops + `"Envelope":` + object + `,`
	for _, field := range []string{
		`{"type":"array","items":` + object + `,"default":[` + value + `]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":` + value + `}}`,
		`{"allOf":[` + object + `,{"properties":{"n":{"type":"integer"}}}],"default":` + value + `}`,
		`{"$ref":"#/components/schemas/Envelope","default":` + value + `}`,
		allOfMemberField(strings.TrimSuffix(collection, "}")+`,"default":[{"k":{"label":"own"}}]}`, `{}`),
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(`{`+definitions+`"Holder":{"properties":{"v":`+field+`}}}`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(field, err)
			}
		}
	}
	for _, tc := range []struct{ owner, path, value string }{
		{`{"type":"array","items":` + allOfObjectMember + `}`, "items", `{"label":"x"}`},
		{`{"type":"object","additionalProperties":` + allOfObjectMember + `}`, "additionalProperties", `{"label":"x"}`},
		{`{"oneOf":[{"allOf":[{"properties":{"kind":{"const":"all"},"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer"}}}]},{"properties":{"kind":{"const":"other"},"label":{"type":"integer"}},"required":["label"]}]}`, "oneOf/0", `{"kind":"all","label":"x"}`},
	} {
		input := schemas(`{"Owner":` + tc.owner + `,"Holder":{"properties":{"v":` + allOfMemberField(`{"type":"array","items":{"$ref":"#/components/schemas/Owner/`+tc.path+`"}}`, `{"member":[`+tc.value+`]}`) + `}}}`)
		if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
			t.Fatal(tc.path, err)
		}
	}
}

func TestReferencedAllOfCollectionObjectDefaultMembers(t *testing.T) {
	for _, internal := range []bool{false, true} {
		for _, ref := range []string{"Alias", "Bridge", "Hops/properties/first"} {
			for _, tc := range []struct{ member, value, ty string }{
				{`{"type":"array","items":{"$ref":"#/components/schemas/` + ref + `"}}`, `[{"label":"x"}]`, "kotlin.collections.List<"},
				{`{"type":"object","additionalProperties":{"$ref":"#/components/schemas/` + ref + `"}}`, `{"k":{"label":"x"}}`, "kotlin.collections.Map<kotlin.String, "},
				{`{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/` + ref + `"}}}`, `[{"k":{"label":"x"}}]`, "kotlin.collections.List<kotlin.collections.Map<kotlin.String, "},
			} {
				for _, checked := range []bool{false, true} {
					for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
						input := chainedAllOfMemberDocument(allOfObjectMember, allOfMemberHops, tc.member, `{"member":`+tc.value+`}`, internal)
						r := request(input, "Dto")
						r.Options.Validation, r.Options.ModelStyle = checked, style
						got, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal(internal, ref, tc.member, checked, style, err)
						}
						owner, actual := "TargetDto", "TargetDto"
						if internal {
							owner, actual = "OwnerDto", "OwnerDto.Node"
						}
						if text := source(t, got, "HolderDto"); !strings.Contains(text, "public val member: "+tc.ty+"org.example.models.models."+actual) || !strings.Contains(text, `\"label\":\"x\"`) {
							t.Fatal(text)
						}
						r.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+tc.value+`}`, "", 1))
						baseline, err := core.GenerateWithDiagnostics(r)
						if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
							t.Fatal("warning drift", err)
						}
						for _, name := range []string{owner, "AliasDto", "BridgeDto", "HopsDto"} {
							if source(t, got, name) != source(t, baseline, name) {
								t.Fatal("target/alias bytes changed", name)
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
	}
}
