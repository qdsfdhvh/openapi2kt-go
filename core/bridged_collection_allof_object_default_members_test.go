package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const collectionMemberBridges = `"RowsHops":{"properties":{"first":{"$ref":"#/components/schemas/RowsHops/properties/second"},"second":{"$ref":"#/components/schemas/Rows"}}},"RowsBridge":{"$ref":"#/components/schemas/RowsHops/properties/first"},"RowsOuter":{"$ref":"#/components/schemas/RowsBridge"},"CollectionGate":{"properties":{"value":{"$ref":"#/components/schemas/RowsOuter"}}},`

func TestBridgedCollectionAllOfMemberNullable(t *testing.T) {
	for _, ref := range []string{"RowsOuter", "RowsHops/properties/first", "CollectionGate/properties/value"} {
		t.Run(ref, func(t *testing.T) { testCollectionAllOfMemberNullable(t, ref) })
	}
}

func testCollectionAllOfMemberNullable(t *testing.T, ref string) {
	for _, mapRoot := range []bool{false, true} {
		for _, checked := range []bool{false, true} {
			for _, near := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, far := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
						collection := `{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Target"}}`
						if mapRoot {
							collection = `{"type":"object","nullable":true,"additionalProperties":{"$ref":"#/components/schemas/Target"}}`
						}
						bridges := strings.Replace(collectionMemberBridges, `"first":{"$ref":"#/components/schemas/RowsHops/properties/second"}`, `"first":{"$ref":"#/components/schemas/RowsHops/properties/second"`+near+`}`, 1)
						bridges = strings.Replace(bridges, `"second":{"$ref":"#/components/schemas/Rows"}`, `"second":{"$ref":"#/components/schemas/Rows"`+far+`}`, 1)
						input := collectionMemberDocument(allOfObjectMember, collection, bridges, `{"$ref":"#/components/schemas/`+ref+`"`+consumer+`}`, `{"member":null}`)
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
							t.Fatal(mapRoot, near, far, consumer, err)
						}
					}
				}
			}
		}
	}
}

func TestBridgedCollectionAllOfMemberDiagnosticsAndGuards(t *testing.T) {
	for _, ref := range []string{"RowsOuter", "RowsHops/properties/first"} {
		t.Run(ref, func(t *testing.T) { testCollectionAllOfMemberDiagnosticsAndGuards(t, ref) })
	}
}

func testCollectionAllOfMemberDiagnosticsAndGuards(t *testing.T, ref string) {
	collection := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`
	member := `{"$ref":"#/components/schemas/` + ref + `"}`
	definitions := `"Target":` + allOfObjectMember + `,"Rows":` + collection + `,` + collectionMemberBridges
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, tail, code string }{
			{`[{}]`, "/0", "E_SCHEMA_INVALID"},
			{`[{"label":false}]`, "/0/label", "E_SCHEMA_INVALID"},
			{`[{"z/~":1,"label":false}]`, "/0/z~1~0", "E_SCHEMA_INVALID"},
			{`[null]`, "/0", "E_SCHEMA_INVALID"},
			{`[{"label":"` + strings.Repeat("x", 32767) + `"}]`, "/0/label", "E_MODEL_LIMIT"},
		} {
			field := allOfMemberField(member, `{"member":`+tc.value+`}`)
			input := collectionMemberDocument(allOfObjectMember, collection, collectionMemberBridges, member, `{"member":`+tc.value+`}`)
			carrierDefaultFailure(t, input, checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
			carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"Holder":{"properties":`, `"Holder":{"required":["v"],"properties":`, 1)), checked, tc.code, "/components/schemas/Holder/properties/v/default/member"+tc.tail)
			discarded := schemas(`{` + definitions + `"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default/member"+tc.tail)
			patch := strings.TrimSuffix(string(carrierRequestDocument(strings.Replace(field, `{"properties":`, `{"nullable":true,"properties":`, 1), false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`
			carrierDefaultFailure(t, []byte(patch), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member"+tc.tail)
		}
		for _, hop := range []string{"first", "second"} {
			for _, value := range []string{`{}`, `{"member":[]}`, `{"member":null}`} {
				bridges := strings.Replace(collectionMemberBridges, `"`+hop+`":{`, `"`+hop+`":{"default":[{"label":"hidden"}],`, 1)
				input := collectionMemberDocument(allOfObjectMember, collection, bridges, `{"$ref":"#/components/schemas/`+ref+`","nullable":true}`, value)
				carrierDefaultFailure(t, input, checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/RowsHops/properties/"+hop+"/default")
			}
		}
	}
	cycle := strings.Replace(collectionMemberBridges, `"second":{"$ref":"#/components/schemas/Rows"}`, `"second":{"$ref":"#/components/schemas/RowsHops/properties/first"}`, 1)
	if _, err := core.GenerateWithDiagnostics(request(collectionMemberDocument(allOfObjectMember, collection, cycle, member, `{}`), "Dto")); err == nil {
		t.Fatal("cycle admitted")
	}
	counted := strings.Replace(collection, `"type":"array"`, `"type":"array","minItems":1`, 1)
	carrierDefaultFailure(t, collectionMemberDocument(allOfObjectMember, counted, collectionMemberBridges, member, `{"member":[]}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
}

func TestBridgedCollectionAllOfMemberContexts(t *testing.T) {
	for _, ref := range []string{"RowsOuter", "RowsHops/properties/first"} {
		t.Run(ref, func(t *testing.T) { testCollectionAllOfMemberContexts(t, ref) })
	}
}

func testCollectionAllOfMemberContexts(t *testing.T, ref string) {
	collection := `{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`
	object := `{"properties":{"member":{"$ref":"#/components/schemas/` + ref + `"}}}`
	value := `{"member":[{"label":"x"}]}`
	definitions := `"Target":` + allOfObjectMember + `,"Rows":` + collection + `,` + collectionMemberBridges + `"Envelope":` + object + `,`
	for _, field := range []string{
		`{"type":"array","items":` + object + `,"default":[` + value + `]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":` + value + `}}`,
		`{"allOf":[` + object + `,{"properties":{"n":{"type":"integer"}}}],"default":` + value + `}`,
		`{"$ref":"#/components/schemas/Envelope","default":` + value + `}`,
		allOfMemberField(`{"$ref":"#/components/schemas/`+ref+`","default":[{"label":"own"}]}`, `{}`),
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

func TestBridgedCollectionAllOfObjectDefaultMembers(t *testing.T) {
	for _, tc := range []struct{ collection, value string }{
		{`{"type":"array","items":{"$ref":"#/components/schemas/Target"}}`, `[{"label":"x"}]`},
		{`{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}`, `{"k":{"label":"x"}}`},
		{`{"type":"array","items":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Target"}}}`, `[{"k":{"label":"x"}}]`},
	} {
		for _, ref := range []string{"RowsBridge", "RowsOuter", "RowsHops/properties/first", "RowsHops/properties/second", "CollectionGate/properties/value"} {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					input := collectionMemberDocument(allOfObjectMember, tc.collection, collectionMemberBridges, `{"$ref":"#/components/schemas/`+ref+`"}`, `{"member":`+tc.value+`}`)
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
					for _, name := range []string{"RowsDto", "RowsBridgeDto", "RowsOuterDto", "RowsHopsDto", "CollectionGateDto", "TargetDto"} {
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
