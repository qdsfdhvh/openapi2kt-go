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
		`{"type":"array","items":{"$ref":"#/components/schemas/Rows"}}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/OtherRows"}}`,
	} {
		extra := aliases + `"OtherRows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}},`
		if _, err := core.GenerateWithDiagnostics(request(collectionMemberDocument(allOfObjectMember, rows, extra, member, `{}`), "Dto")); err == nil {
			t.Fatal("recursive/container reference expanded", rows)
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

func TestReferencedCollectionBareAllOfMemberLeaf(t *testing.T) {
	// A referenced collection with a bare inline ordinary allOf element leaf
	// borrows the same retained constructor as the consuming-field role. The
	// container declaration and warnings stay unchanged; only the annotation is added.
	for _, tc := range []struct{ name, collection string }{
		{"list", `{"type":"array","items":` + allOfObjectMember + `}`},
		{"map", `{"type":"object","additionalProperties":` + allOfObjectMember + `}`},
		{"mixed", `{"type":"array","items":{"type":"object","additionalProperties":` + allOfObjectMember + `}}`},
		{"nested", `{"type":"array","items":{"type":"array","items":` + allOfObjectMember + `}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := `[{"label":"x"}]`
			if tc.name == "map" {
				value = `{"k":{"label":"x"}}`
			} else if tc.name == "mixed" {
				value = `[{"k":{"label":"x"}}]`
			} else if tc.name == "nested" {
				value = `[[{"label":"x"}]]`
			}
			input := collectionMemberDocument(allOfObjectMember, tc.collection, "", `{"$ref":"#/components/schemas/Rows"}`, `{"member":`+value+`}`)
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					r := request(input, "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(checked, style, err)
					}
					if text := source(t, got, "HolderDto"); !strings.Contains(text, "`openapi-kt model default ") {
						t.Fatal(text)
					}
					// The bare leaf must still validate; a stray key fails at its member Pointer.
					bad := collectionMemberDocument(allOfObjectMember, tc.collection, "", `{"$ref":"#/components/schemas/Rows"}`, `{"member":[{"label":false}]}`)
					if tc.name == "map" {
						bad = collectionMemberDocument(allOfObjectMember, tc.collection, "", `{"$ref":"#/components/schemas/Rows"}`, `{"member":{"k":{"label":false}}}`)
					} else if tc.name == "mixed" {
						bad = collectionMemberDocument(allOfObjectMember, tc.collection, "", `{"$ref":"#/components/schemas/Rows"}`, `{"member":[{"k":{"label":false}}]}`)
					} else if tc.name == "nested" {
						bad = collectionMemberDocument(allOfObjectMember, tc.collection, "", `{"$ref":"#/components/schemas/Rows"}`, `{"member":[[{"label":false}]]}`)
					}
					br := request(bad, "Dto")
					br.Options.Validation = checked
					if got, err := core.GenerateWithDiagnostics(br); err == nil || len(got.Artifacts.Files) != 0 {
						t.Fatal("invalid bare leaf admitted", err)
					}
				}
			}
		})
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
					// The internal terminal keeps its original physical lexical owner; the
					// consuming member still uses the actual retained Rows collection type.
					if internal && !strings.Contains(source(t, got, "RowsDto"), "org.example.models.models.TargetDto.Node") {
						t.Fatal("internal terminal lost its lexical owner", source(t, got, "RowsDto"))
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

// An object-default property may reference a collection whose element resolves
// to a plain named ordinary DTO, matching the consuming-field role. The named
// collection may be referenced directly or through finite alias chains, and
// inline containers with a referenced element participate too.
func TestObjectDefaultCollectionPropertiesOfReferencedDTOs(t *testing.T) {
	const item = `{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"age":{"type":"integer","default":7}}}`
	definitions := `"Item":` + item + `,"ItemAlias":{"$ref":"#/components/schemas/Item"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"AliasRows":{"type":"array","items":{"$ref":"#/components/schemas/ItemAlias"}},"MapRows":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Item"}},"Owner":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}}}},`
	for _, tc := range []struct{ name, member, value string }{
		{"named list", `{"$ref":"#/components/schemas/Rows"}`, `[{"name":"a"}]`},
		{"named map", `{"$ref":"#/components/schemas/MapRows"}`, `{"k":{"name":"a"}}`},
		{"alias-chain element", `{"$ref":"#/components/schemas/AliasRows"}`, `[{"name":"a"}]`},
		{"inline list", `{"type":"array","items":{"$ref":"#/components/schemas/Item"}}`, `[{"name":"a"}]`},
		{"inline map", `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Item"}}`, `{"k":{"name":"a"}}`},
		{"internal collection", `{"$ref":"#/components/schemas/Owner/properties/rows"}`, `[{"name":"a"}]`},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := schemas(`{` + definitions + `"Holder":{"properties":{"v":` + allOfMemberField(tc.member, `{"member":`+tc.value+`}`) + `}}}`)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.name, checked, style, err)
				}
				if text := source(t, got, "HolderDto"); !strings.Contains(text, "`openapi-kt model default ") {
					t.Fatal(tc.name, text)
				}
			}
		}
	}
}

func TestObjectDefaultCollectionReferencedDTOGuards(t *testing.T) {
	const item = `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`
	base := func(member, value string) []byte {
		return schemas(`{"Item":` + item + `,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
	}
	for _, tc := range []struct{ value, tail string }{
		{`{"member":[{}]}`, "/member/0"},
		{`{"member":[{"name":false}]}`, "/member/0/name"},
		{`{"member":[{"extra/~":1}]}`, "/member/0/extra~1~0"},
		{`{"member":[null]}`, "/member/0"},
	} {
		for _, checked := range []bool{false, true} {
			carrierDefaultFailure(t, base(`{"$ref":"#/components/schemas/Rows"}`, tc.value), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default"+tc.tail)
		}
	}
	// A scalar/enum element chain is not a plain object target.
	scalar := schemas(`{"S":{"type":"string"},"SA":{"$ref":"#/components/schemas/S"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/SA"}},"Holder":{"properties":{"v":` + allOfMemberField(`{"$ref":"#/components/schemas/Rows"}`, `{"member":["x"]}`) + `}}}`)
	if got, err := core.GenerateWithDiagnostics(request(scalar, "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("scalar element chain gained object member role", err)
	}
}

// A plain-DTO-element collection reached through a finite named alias or an
// internal bridge uses the same member profile as a direct collection
// reference, matching the consuming-field role. Intermediate nullable intent
// and the consuming override still decide annotation null.
func TestObjectDefaultPlainDTOCollectionAliases(t *testing.T) {
	const item = `{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"count":{"type":"integer","default":2}}}`
	definitions := `"Item":` + item + `,"Rows":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Item"}},"RowsAlias":{"$ref":"#/components/schemas/Rows"},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/RowsAlias"},"second":{"$ref":"#/components/schemas/RowsAlias","nullable":false}}},"Entities":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}}}},"EntitiesAlias":{"$ref":"#/components/schemas/Entities/properties/rows"},`
	for _, tc := range []struct{ name, member string }{
		{"named alias", `{"$ref":"#/components/schemas/RowsAlias"}`},
		{"internal bridge", `{"$ref":"#/components/schemas/Hops/properties/first"}`},
		{"internal prefix narrowing", `{"$ref":"#/components/schemas/Hops/properties/second"}`},
		{"internal entity alias", `{"$ref":"#/components/schemas/EntitiesAlias"}`},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := schemas(`{` + definitions + `"Holder":{"properties":{"v":` + allOfMemberField(tc.member, `{"member":[{"name":"a"}]}`) + `}}}`)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.name, checked, style, err)
				}
				if text := source(t, got, "HolderDto"); !strings.Contains(text, "`openapi-kt model default ") {
					t.Fatal(tc.name, text)
				}
				// The terminal plain object still validates its own members.
				bad := schemas(`{` + definitions + `"Holder":{"properties":{"v":` + allOfMemberField(tc.member, `{"member":[{"name":false}]}`) + `}}}`)
				carrierDefaultFailure(t, bad, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/0/name")
			}
		}
	}
	// The intermediate explicit false narrows a nullable terminal: annotation
	// null then fails, while an explicit consumer true restores it.
	narrowed := schemas(`{` + definitions + `"Holder":{"properties":{"v":` + allOfMemberField(`{"$ref":"#/components/schemas/Hops/properties/second"}`, `{"member":null}`) + `}}}`)
	carrierDefaultFailure(t, narrowed, true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
	widened := schemas(`{` + definitions + `"Holder":{"properties":{"v":` + allOfMemberField(`{"$ref":"#/components/schemas/Hops/properties/second","nullable":true}`, `{"member":null}`) + `}}}`)
	if _, err := core.GenerateWithDiagnostics(request(widened, "Dto")); err != nil {
		t.Fatal("consumer override did not restore nullable alias", err)
	}
}
