package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const allOfMemberHops = `"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second"},"second":{"$ref":"#/components/schemas/Alias"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/first"},`

func chainedAllOfMemberDocument(target, hops, member, value string, internal bool) []byte {
	definitions := `"Target":` + target + `,"Alias":{"$ref":"#/components/schemas/Target"},`
	if internal {
		definitions = `"Owner":{"properties":{"node":` + target + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},`
	}
	return schemas(`{` + definitions + hops + `"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
}

func TestChainedAllOfObjectDefaultMemberNullable(t *testing.T) {
	target := strings.TrimSuffix(allOfObjectMember, "}") + `,"nullable":true}`
	for _, internal := range []bool{false, true} {
		for _, checked := range []bool{false, true} {
			for _, far := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, near := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
						hops := `"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second"` + near + `},"second":{"$ref":"#/components/schemas/Alias"` + far + `}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/first"},`
						input := chainedAllOfMemberDocument(target, hops, `{"$ref":"#/components/schemas/Bridge"`+consumer+`}`, `{"member":null}`, internal)
						nullable := true
						for _, intent := range []string{far, near, consumer} {
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
							t.Fatal(internal, far, near, consumer, err)
						}
					}
				}
			}
		}
	}
	// A named alias before the internal hops supplies the nearest explicit intent.
	hops := strings.Replace(allOfMemberHops, `"Bridge":{"$ref":"#/components/schemas/Hops/properties/first"}`, `"Bridge":{"$ref":"#/components/schemas/Hops/properties/first","nullable":false}`, 1)
	carrierDefaultFailure(t, chainedAllOfMemberDocument(target, hops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":null}`, true), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
	if _, err := core.GenerateWithDiagnostics(request(chainedAllOfMemberDocument(target, hops, `{"$ref":"#/components/schemas/Bridge","nullable":true}`, `{"member":null}`, true), "Dto")); err != nil {
		t.Fatal(err)
	}
}

func TestChainedAllOfObjectDefaultMemberGuards(t *testing.T) {
	member := `{"$ref":"#/components/schemas/Bridge"}`
	for _, internal := range []bool{false, true} {
		for _, checked := range []bool{false, true} {
			definitions := `"Target":` + allOfObjectMember + `,"Alias":{"$ref":"#/components/schemas/Target"},` + allOfMemberHops
			if internal {
				definitions = `"Owner":{"properties":{"node":` + allOfObjectMember + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},` + allOfMemberHops
			}
			for _, tc := range []struct{ value, tail, code string }{
				{`{"member":{"label":false}}`, "/member/label", "E_SCHEMA_INVALID"},
				{`{"member":{}}`, "/member", "E_SCHEMA_INVALID"},
				{`{"member":null}`, "/member", "E_SCHEMA_INVALID"},
				{`{"member":{"z/~":1,"label":false}}`, "/member/z~1~0", "E_SCHEMA_INVALID"},
				{`{"member":{"label":"x","child":{"flag":"bad"}}}`, "/member/child/flag", "E_SCHEMA_INVALID"},
				{`{"member":{"label":"` + strings.Repeat("x", 32767) + `"}}`, "/member/label", "E_MODEL_LIMIT"},
			} {
				input := chainedAllOfMemberDocument(allOfObjectMember, allOfMemberHops, member, tc.value, internal)
				carrierDefaultFailure(t, input, checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
				required := []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1))
				carrierDefaultFailure(t, required, checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
				field := allOfMemberField(member, tc.value)
				patch := strings.TrimSuffix(string(carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`
				carrierDefaultFailure(t, []byte(patch), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default"+tc.tail)
				discarded := schemas(`{` + definitions + `"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
				carrierDefaultFailure(t, discarded, checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default"+tc.tail)
			}
			for _, hop := range []string{"first", "second"} {
				for _, value := range []string{`{}`, `{"member":null}`, `{"member":{"label":"x"}}`} {
					hops := strings.Replace(allOfMemberHops, `"`+hop+`":{`, `"`+hop+`":{"default":{"label":"hidden"},`, 1)
					input := chainedAllOfMemberDocument(allOfObjectMember, hops, `{"$ref":"#/components/schemas/Bridge","nullable":true}`, value, internal)
					carrierDefaultFailure(t, input, checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Hops/properties/"+hop+"/default")
				}
			}
		}
	}
	// A finite transparent chain to a plain object terminal joins the member role.
	if got, err := core.GenerateWithDiagnostics(request(chainedAllOfMemberDocument(`{"properties":{"label":{"type":"string"}}}`, allOfMemberHops, member, `{}`, false), "Dto")); err != nil || len(got.Artifacts.Files) == 0 {
		t.Fatal("plain-object alias terminal not admitted", err)
	}
	for _, target := range []string{
		`{"type":"string"}`,
		`{"allOf":[{},{}]}`,
		`{"allOf":[{"properties":{"tag":{"const":"x"}}}]}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`,
		strings.TrimSuffix(allOfObjectMember, "}") + `,"default":{"label":"target"}}`,
	} {
		if got, err := core.GenerateWithDiagnostics(request(chainedAllOfMemberDocument(target, allOfMemberHops, member, `{}`, false), "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("excluded terminal admitted", target, err)
		}
	}
	cycle := strings.Replace(allOfMemberHops, `"second":{"$ref":"#/components/schemas/Alias"}`, `"second":{"$ref":"#/components/schemas/Hops/properties/first"}`, 1)
	if _, err := core.GenerateWithDiagnostics(request(chainedAllOfMemberDocument(allOfObjectMember, cycle, member, `{}`, false), "Dto")); err == nil {
		t.Fatal("reference cycle admitted")
	}
	bounded := strings.Replace(allOfObjectMember, `"type":"integer","default":2`, `"type":"integer","minimum":1,"default":2`, 1)
	carrierDefaultFailure(t, chainedAllOfMemberDocument(bounded, allOfMemberHops, member, `{"member":{"label":"x","count":0}}`, true), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/count")
}

func TestChainedAllOfObjectDefaultMemberContexts(t *testing.T) {
	object := `{"properties":{"member":{"$ref":"#/components/schemas/Bridge"}}}`
	value := `{"member":{"label":"x"}}`
	definitions := `"Owner":{"properties":{"node":` + allOfObjectMember + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},` + allOfMemberHops + `"Envelope":` + object + `,`
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

func TestChainedAllOfObjectDefaultMembers(t *testing.T) {
	for _, internal := range []bool{false, true} {
		for _, ref := range []string{"Hops/properties/first", "Bridge"} {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					input := chainedAllOfMemberDocument(allOfObjectMember, allOfMemberHops, `{"$ref":"#/components/schemas/`+ref+`"}`, `{"member":{"label":"initial"}}`, internal)
					r := request(input, "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(internal, ref, checked, style, err)
					}
					actual, owner := "TargetDto", "TargetDto"
					if internal {
						actual, owner = "OwnerDto.Node", "OwnerDto"
					}
					if text := source(t, got, "HolderDto"); !strings.Contains(text, "public val member: org.example.models.models."+actual+"?") || !strings.Contains(text, `\"member\":{\"label\":\"initial\"}`) {
						t.Fatal(text)
					}
					r.Document = []byte(strings.Replace(string(input), `,"default":{"member":{"label":"initial"}}`, "", 1))
					baseline, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
						t.Fatal("warnings changed", err)
					}
					for _, name := range []string{owner, "AliasDto", "BridgeDto", "HopsDto"} {
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
}
