package core_test

import (
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func internalCollectionMembers(collection, member, value string) []byte {
	return schemas(`{"Target":` + allOfObjectMember + `,"Owner":{"properties":{"rows":` + collection + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/rows"},"Outer":{"$ref":"#/components/schemas/Alias"},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Outer"}}},"Holder":{"properties":{"v":` + allOfMemberField(member, value) + `}}}`)
}

func TestInternalCollectionAllOfMemberNullable(t *testing.T) {
	for _, mapRoot := range []bool{false, true} {
		for _, checked := range []bool{false, true} {
			for _, rootNullable := range []bool{false, true} {
				for _, alias := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					for _, hop := range []string{"", `,"nullable":true`, `,"nullable":false`} {
						for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
							for _, ref := range []string{"Owner/properties/rows", "Alias", "Outer", "Hops/properties/first"} {
								collection := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}`
								if mapRoot {
									collection = `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}`
								}
								if rootNullable {
									collection += `,"nullable":true`
								}
								collection += `}`
								input := string(internalCollectionMembers(collection, `{"$ref":"#/components/schemas/`+ref+`"`+consumer+`}`, `{"member":null}`))
								input = strings.Replace(input, `"Alias":{"$ref":"#/components/schemas/Owner/properties/rows"}`, `"Alias":{"$ref":"#/components/schemas/Owner/properties/rows"`+alias+`}`, 1)
								input = strings.Replace(input, `"first":{"$ref":"#/components/schemas/Outer"}`, `"first":{"$ref":"#/components/schemas/Outer"`+hop+`}`, 1)
								nullable := rootNullable
								if ref != "Owner/properties/rows" && alias != "" {
									nullable = alias == `,"nullable":true`
								}
								if ref == "Hops/properties/first" && hop != "" {
									nullable = hop == `,"nullable":true`
								}
								if consumer != "" {
									nullable = consumer == `,"nullable":true`
								}
								if !nullable {
									carrierDefaultFailure(t, []byte(input), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
									continue
								}
								r := request([]byte(input), "Dto")
								r.Options.Validation = checked
								if _, err := core.GenerateWithDiagnostics(r); err != nil {
									t.Fatal(mapRoot, rootNullable, alias, hop, consumer, ref, err)
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestInternalCollectionAllOfMemberDiagnosticsAndGuards(t *testing.T) {
	collection := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`
	for _, ref := range []string{"Owner/properties/rows", "Alias", "Hops/properties/first"} {
		for _, checked := range []bool{false, true} {
			for _, tc := range []struct{ value, tail, code string }{
				{`[{}]`, "/0", "E_SCHEMA_INVALID"}, {`[{"label":false}]`, "/0/label", "E_SCHEMA_INVALID"},
				{`[{"z/~":1,"label":false}]`, "/0/z~1~0", "E_SCHEMA_INVALID"}, {`[null]`, "/0", "E_SCHEMA_INVALID"},
				{`null`, "", "E_SCHEMA_INVALID"}, {`[{"label":"` + strings.Repeat("x", 32767) + `"}]`, "/0/label", "E_MODEL_LIMIT"},
			} {
				field := allOfMemberField(`{"$ref":"#/components/schemas/`+ref+`"}`, `{"member":`+tc.value+`}`)
				input := internalCollectionMembers(collection, `{"$ref":"#/components/schemas/`+ref+`"}`, `{"member":`+tc.value+`}`)
				carrierDefaultFailure(t, input, checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
				carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1)), checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
				discarded := strings.Replace(string(input), `"Holder":{"properties":{"v":`+field+`}}`, `"Holder":{"allOf":[{"properties":{"v":`+field+`}},{"properties":{"v":{"type":"string"}}}]}`, 1)
				carrierDefaultFailure(t, []byte(discarded), checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default/member"+tc.tail)
				patch := strings.TrimSuffix(string(carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)), "}") + `,"components":{"schemas":{"Target":` + allOfObjectMember + `,"Owner":{"properties":{"rows":` + collection + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/rows"},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Alias"}}}}}}`
				carrierDefaultFailure(t, []byte(patch), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member"+tc.tail)
			}
			counted := strings.Replace(collection, `"type":"array"`, `"type":"array","minItems":1`, 1)
			if checked {
				carrierDefaultFailure(t, internalCollectionMembers(counted, `{"$ref":"#/components/schemas/`+ref+`"}`, `{"member":[]}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
			}
		}
	}
	for _, rows := range []string{
		`{"type":"array","items":` + allOfObjectMember + `}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/other"}}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/rows"}}`,
	} {
		input := string(internalCollectionMembers(rows, `{"$ref":"#/components/schemas/Alias"}`, `{}`))
		input = strings.Replace(input, `"Owner":{"properties":{`, `"Owner":{"properties":{"node":`+allOfObjectMember+`,"other":`+collection+`,`, 1)
		got, err := core.GenerateWithDiagnostics(request([]byte(input), "Dto"))
		if strings.Contains(rows, "Owner/properties/other") {
			if err != nil || len(got.Artifacts.Files) == 0 {
				t.Fatal("original finite internal container did not graduate", err)
			}
		} else if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("excluded leaf/container expanded", rows, err)
		}
	}
	// An original internal allOf terminal is now admitted through the shared
	// resolver seam; the consuming member keeps the retained collection type.
	terminal := string(internalCollectionMembers(`{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/node"}}`, `{"$ref":"#/components/schemas/Alias"}`, `{"member":[{"label":"x"}]}`))
	terminal = strings.Replace(terminal, `"Owner":{"properties":{`, `"Owner":{"properties":{"node":`+allOfObjectMember+`,`, 1)
	if got, err := core.GenerateWithDiagnostics(request([]byte(terminal), "Dto")); err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt model default ") {
		t.Fatal("internal allOf terminal not admitted", err)
	}
	// A wrong or hierarchy-owned internal source still fails.
	for _, bad := range []string{
		strings.Replace(terminal, `"node":`+allOfObjectMember, `"node":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"label":{"type":"string"}},"required":["label"]}]}`, 1),
	} {
		bad = strings.Replace(bad, `"schemas":{`, `"schemas":{"Base":{"type":"object","properties":{"kind":{"type":"string"}},"discriminator":{"propertyName":"kind"}},`, 1)
		if got, err := core.GenerateWithDiagnostics(request([]byte(bad), "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("hierarchy-owned internal terminal admitted", err)
		}
	}
	for _, value := range []string{`{}`, `{"member":[]}`, `{"member":null}`} {
		input := string(internalCollectionMembers(collection, `{"$ref":"#/components/schemas/Hops/properties/first","nullable":true}`, value))
		for _, bad := range []string{
			strings.Replace(input, `"rows":{"type":"array"`, `"rows":{"default":[],"type":"array"`, 1),
			strings.Replace(input, `"Alias":{"$ref":`, `"Alias":{"default":[],"$ref":`, 1),
			strings.Replace(input, `"first":{"$ref":`, `"first":{"default":[],"$ref":`, 1),
			strings.Replace(input, `"items":{"$ref":`, `"items":{"default":{"label":"hidden"},"$ref":`, 1),
		} {
			if got, err := core.GenerateWithDiagnostics(request([]byte(bad), "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("inherited annotation", value, err)
			}
		}
	}
}

func TestInternalCollectionAllOfMemberContexts(t *testing.T) {
	rows := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`
	member := `{"$ref":"#/components/schemas/Owner/properties/rows"}`
	value := `{"member":[{"label":"x"}]}`
	object := `{"properties":{"member":` + member + `}}`
	for _, field := range []string{
		`{"type":"array","items":` + object + `,"default":[` + value + `]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":` + value + `}}`,
		`{"allOf":[` + object + `,{"properties":{"n":{"type":"integer"}}}],"default":` + value + `}`,
		`{"$ref":"#/components/schemas/Envelope","default":` + value + `}`,
		allOfMemberField(`{"$ref":"#/components/schemas/Owner/properties/rows","default":[{"label":"own"}]}`, `{}`),
	} {
		for _, checked := range []bool{false, true} {
			input := string(internalCollectionMembers(rows, member, value))
			input = strings.Replace(input, `"Holder":`, `"Envelope":`+object+`,"Holder":`, 1)
			input = strings.Replace(input, allOfMemberField(member, value), field, 1)
			r := request([]byte(input), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(field, err)
			}
		}
	}
	for _, element := range []string{"", `,"nullable":false`} {
		target := strings.TrimSuffix(allOfObjectMember, "}") + `,"nullable":true}`
		rows := `{"type":"array","items":{"$ref":"#/components/schemas/Target"` + element + `}}`
		input := strings.Replace(string(internalCollectionMembers(rows, member, `{"member":[null]}`)), `"Target":`+allOfObjectMember, `"Target":`+target, 1)
		if element != "" {
			carrierDefaultFailure(t, []byte(input), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member/0")
			continue
		}
		if _, err := core.GenerateWithDiagnostics(request([]byte(input), "Dto")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInternalCollectionAllOfMemberLexicalScopes(t *testing.T) {
	rows := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`
	for _, tc := range []struct{ owner, path string }{
		{`{"properties":{"nested":{"properties":{"rows":` + rows + `}}}}`, "Owner/properties/nested/properties/rows"},
		{`{"type":"array","items":` + rows + `}`, "Owner/items"},
		{`{"type":"object","additionalProperties":` + rows + `}`, "Owner/additionalProperties"},
		{`{"allOf":[{"properties":{"rows":` + rows + `}},{"properties":{"n":{"type":"integer"}}}]}`, "Owner/allOf/0/properties/rows"},
	} {
		for _, checked := range []bool{false, true} {
			input := string(internalCollectionMembers(rows, `{"$ref":"#/components/schemas/Owner/properties/rows"}`, `{"member":[{"label":"x"}]}`))
			input = strings.Replace(input, `"Owner":{"properties":{"rows":`+rows+`}}`, `"Owner":`+tc.owner, 1)
			input = strings.ReplaceAll(input, "Owner/properties/rows", tc.path)
			r := request([]byte(input), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(tc.path, err)
			}
		}
	}
}

func TestInternalCollectionAllOfObjectDefaultMembers(t *testing.T) {
	for _, tc := range []struct{ collection, value string }{
		{`{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`, `[{"label":"x"}]`},
		{`{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}`, `{"k":{"label":"x"}}`},
		{`{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}}`, `[{"k":{"label":"x"}}]`},
	} {
		for _, ref := range []string{"Owner/properties/rows", "Alias", "Outer", "Hops/properties/first"} {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					input := internalCollectionMembers(tc.collection, `{"$ref":"#/components/schemas/`+ref+`"}`, `{"member":`+tc.value+`}`)
					r := request(input, "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(ref, tc.collection, checked, style, err)
					}
					if text := source(t, got, "HolderDto"); !strings.Contains(text, `\"label\":\"x\"`) {
						t.Fatal(text)
					}
					r.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+tc.value+`}`, "", 1))
					baseline, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
						t.Fatal("warning drift", err)
					}
					for _, name := range []string{"OwnerDto", "AliasDto", "OuterDto", "HopsDto", "TargetDto"} {
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
