package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func collectionMemberDocument(target, collection, aliases, member, value string) []byte {
	return schemas(`{"Target":` + target + `,"Rows":` + collection + `,` + aliases + `"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
}

func TestReferencedCollectionAllOfMemberAliasesAndNullable(t *testing.T) {
	for _, mapRoot := range []bool{false, true} {
		for _, checked := range []bool{false, true} {
			for _, targetNullable := range []string{"", `,"nullable":true`} {
				for _, aliasNullable := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					for _, consumerNullable := range []string{"", `,"nullable":true`, `,"nullable":false`} {
						collection := `{"type":"array","items":{"$ref":"#/components/schemas/Bridge"}` + targetNullable + `}`
						value := `[{"label":"x"}]`
						if mapRoot {
							collection = `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Bridge"}` + targetNullable + `}`
							value = `{"k":{"label":"x"}}`
						}
						aliases := `"Alias":{"$ref":"#/components/schemas/Target"},` + allOfMemberHops + `"RowsAlias":{"$ref":"#/components/schemas/Rows"` + aliasNullable + `},"RowsOuter":{"$ref":"#/components/schemas/RowsAlias"},`
						for _, reference := range []string{"Rows", "RowsOuter"} {
							for _, supplied := range []string{"null", value} {
								nullable := targetNullable != ""
								if reference != "Rows" && aliasNullable != "" {
									nullable = aliasNullable == `,"nullable":true`
								}
								if consumerNullable != "" && (consumerNullable == `,"nullable":true` || reference != "Rows" || !mapRoot) {
									nullable = consumerNullable == `,"nullable":true`
								}
								input := collectionMemberDocument(allOfObjectMember, collection, aliases, `{"$ref":"#/components/schemas/`+reference+`"`+consumerNullable+`}`, `{"member":`+supplied+`}`)
								if supplied == "null" && !nullable {
									carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
									continue
								}
								r := request(input, "Dto")
								r.Options.Validation = checked
								if _, err := core.GenerateWithDiagnostics(r); err != nil {
									t.Fatal(mapRoot, targetNullable, aliasNullable, consumerNullable, reference, supplied, err)
								}
							}
						}
					}
				}
			}
		}
	}
	// Element intent is independent of the referenced container's root intent.
	target := strings.TrimSuffix(allOfObjectMember, "}") + `,"nullable":true}`
	for _, element := range []string{"", `,"nullable":false`} {
		collection := `{"type":"array","items":{"$ref":"#/components/schemas/Target"` + element + `}}`
		input := collectionMemberDocument(target, collection, "", `{"$ref":"#/components/schemas/Rows","nullable":true}`, `{"member":[null]}`)
		if element != "" {
			carrierDefaultFailure(t, input, false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/0")
			continue
		}
		if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReferencedCollectionAllOfMemberDiagnosticsAndGuards(t *testing.T) {
	collection := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`
	aliases := `"RowsAlias":{"$ref":"#/components/schemas/Rows"},`
	member := `{"$ref":"#/components/schemas/RowsAlias"}`
	definitions := `"Target":` + allOfObjectMember + `,"Rows":` + collection + `,` + aliases
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail, code string }{
			{`[{}]`, "/0", "E_SCHEMA_INVALID"},
			{`[{"label":false}]`, "/0/label", "E_SCHEMA_INVALID"},
			{`[{"z/~":1,"label":false}]`, "/0/z~1~0", "E_SCHEMA_INVALID"},
			{`[{"label":"x","child":{"flag":1}}]`, "/0/child/flag", "E_SCHEMA_INVALID"},
			{`[null]`, "/0", "E_SCHEMA_INVALID"},
			{`null`, "", "E_SCHEMA_INVALID"},
			{`[{"label":"` + strings.Repeat("x", 32767) + `"}]`, "/0/label", "E_MODEL_LIMIT"},
		} {
			field := allOfMemberField(member, `{"member":`+tc.value+`}`)
			input := schemas(`{` + definitions + `"Holder":{"properties":{"v":` + field + `}}}`)
			carrierDefaultFailure(t, input, checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
			carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1)), checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
			discarded := schemas(`{` + definitions + `"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default/member"+tc.tail)
			patch := strings.TrimSuffix(string(carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`
			carrierDefaultFailure(t, []byte(patch), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member"+tc.tail)
		}
		for _, value := range []string{`{}`, `{"member":[]}`, `{"member":null}`} {
			consumer := `{"$ref":"#/components/schemas/RowsAlias","nullable":true}`
			for _, bad := range []struct{ target, rows, aliases string }{
				{strings.TrimSuffix(allOfObjectMember, "}") + `,"default":{"label":"hidden"}}`, collection, aliases},
				{allOfObjectMember, strings.TrimSuffix(collection, "}") + `,"default":[]}`, aliases},
				{allOfObjectMember, collection, `"RowsAlias":{"$ref":"#/components/schemas/Rows","default":[]},`},
				{allOfObjectMember, strings.Replace(collection, `"items":{"$ref":`, `"items":{"default":{"label":"hidden"},"$ref":`, 1), aliases},
			} {
				if got, err := core.GenerateWithDiagnostics(request(collectionMemberDocument(bad.target, bad.rows, bad.aliases, consumer, value), "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
					t.Fatal("inherited annotation", bad, value, err)
				}
			}
		}
	}
	for _, rows := range []string{
		`{"type":"array","items":` + allOfObjectMember + `}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/Rows"}}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/OtherRows"}}`,
	} {
		extra := aliases + `"OtherRows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}},`
		if _, err := core.GenerateWithDiagnostics(request(collectionMemberDocument(allOfObjectMember, rows, extra, member, `{}`), "Dto")); err == nil {
			t.Fatal("bare/recursive/container reference expanded", rows)
		}
	}
	counted := strings.Replace(collection, `"type":"array"`, `"type":"array","minItems":1`, 1)
	carrierDefaultFailure(t, collectionMemberDocument(allOfObjectMember, counted, aliases, member, `{"member":[]}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
}

func TestReferencedCollectionAllOfMemberContexts(t *testing.T) {
	collection := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`
	object := `{"properties":{"member":{"$ref":"#/components/schemas/Rows"}}}`
	value := `{"member":[{"label":"x"}]}`
	definitions := `"Target":` + allOfObjectMember + `,"Rows":` + collection + `,"Envelope":` + object + `,`
	for _, field := range []string{
		`{"type":"array","items":` + object + `,"default":[` + value + `]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":` + value + `}}`,
		`{"allOf":[` + object + `,{"properties":{"n":{"type":"integer"}}}],"default":` + value + `}`,
		`{"$ref":"#/components/schemas/Envelope","default":` + value + `}`,
		allOfMemberField(`{"$ref":"#/components/schemas/Rows","default":[{"label":"own"}]}`, `{}`),
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

func TestReferencedCollectionAllOfObjectDefaultMembers(t *testing.T) {
	for _, internal := range []bool{false, true} {
		target, reference := allOfObjectMember, "Target"
		if internal {
			target = `{"properties":{"node":` + target + `}}`
			reference += "/properties/node"
		}
		for _, tc := range []struct{ collection, value string }{
			{`{"type":"array","items":{"$ref":"#/components/schemas/` + reference + `"}}`, `[{"label":"x"}]`},
			{`{"type":"object","additionalProperties":{"$ref":"#/components/schemas/` + reference + `"}}`, `{"k":{"label":"x"}}`},
			{`{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/` + reference + `"}}}`, `[{"k":{"label":"x"}}]`},
		} {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					input := collectionMemberDocument(target, tc.collection, "", `{"$ref":"#/components/schemas/Rows"}`, `{"member":`+tc.value+`}`)
					r := request(input, "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if internal {
						if err == nil || len(got.Artifacts.Files) != 0 {
							t.Fatal("internal terminal entered named collection role", err)
						}
						r.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+tc.value+`}`, "", 1))
						if _, err := core.GenerateWithDiagnostics(r); err != nil {
							t.Fatal("baseline internal typed collection", err)
						}
						continue
					}
					if err != nil {
						t.Fatal(internal, tc.collection, checked, style, err)
					}
					if text := source(t, got, "HolderDto"); !strings.Contains(text, "public val member: org.example.models.models.RowsDto?") || !strings.Contains(text, `\"label\":\"x\"`) {
						t.Fatal(text)
					}
					r.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+tc.value+`}`, "", 1))
					baseline, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
						t.Fatal("warnings", err)
					}
					for _, name := range []string{"RowsDto", "TargetDto"} {
						if source(t, got, name) != source(t, baseline, name) {
							t.Fatal("declaration drift", name)
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
