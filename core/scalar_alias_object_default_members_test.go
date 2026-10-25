package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func scalarAliasMemberDocument(target, hops, member, value string) []byte {
	return schemas(`{"Target":` + target + `,"Alias":{"$ref":"#/components/schemas/Target"},` + hops + `"Holder":{"properties":{"v":{"properties":{"member":` + member + `},"default":` + value + `}}}}`)
}

func TestScalarAliasObjectDefaultMembers(t *testing.T) {
	for _, tc := range []struct{ target, value string }{
		{`{"type":"string"}`, `"text"`},
		{`{"type":"string","format":"uuid"}`, `"550e8400-e29b-41d4-a716-446655440000"`},
		{`{"type":"boolean"}`, `true`},
		{`{"type":"integer"}`, `7`},
		{`{"type":"integer","format":"int32"}`, `7`},
		{`{"type":"integer","format":"int64"}`, `9223372036854775807`},
		{`{"type":"number","format":"float"}`, `1.25`},
		{`{"type":"number","format":"double"}`, `1.25`},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				target := tc.target
				if !checked {
					target = strings.TrimSuffix(target, "}") + `,"nullable":true}`
				}
				input := scalarAliasMemberDocument(target, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+tc.value+`}`)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.target, checked, style, err)
				}
				if !strings.Contains(source(t, got, "HolderDto"), "openapi-kt model default") {
					t.Fatal("lost object member initializer")
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic", err)
				}
				r.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+tc.value+`}`, "", 1))
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"TargetDto", "AliasDto", "HopsDto", "BridgeDto"} {
					if source(t, got, name) != source(t, baseline, name) {
						t.Fatal("source declaration changed", name)
					}
				}
			}
		}
	}
}

func TestScalarAliasObjectDefaultMemberGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ target, value string }{
			{`{"type":"string"}`, `1`}, {`{"type":"integer","format":"int32"}`, `2147483648`},
			{`{"type":"boolean"}`, `"true"`}, {`{"type":"number","format":"float"}`, `1e100`},
		} {
			target := tc.target
			if !checked {
				target = strings.TrimSuffix(target, "}") + `,"nullable":true}`
			}
			carrierDefaultFailure(t, scalarAliasMemberDocument(target, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+tc.value+`}`), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
		}
	}
	for _, tc := range []struct{ target, value string }{
		{`{"type":"string","minLength":2}`, `"x"`},
		{`{"type":"integer","minimum":2}`, `1`},
	} {
		carrierDefaultFailure(t, scalarAliasMemberDocument(tc.target, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+tc.value+`}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
	}
	// A nearer explicit false narrows the terminal; a consumer true wins again.
	for _, far := range []string{"", `,"nullable":true`, `,"nullable":false`} {
		for _, near := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				hops := `"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second"` + near + `},"second":{"$ref":"#/components/schemas/Alias"` + far + `}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/first"},`
				nullable := true
				for _, intent := range []string{far, near, consumer} {
					if intent != "" {
						nullable = intent == `,"nullable":true`
					}
				}
				input := scalarAliasMemberDocument(`{"type":["string","null"],"format":"uuid"}`, hops, `{"$ref":"#/components/schemas/Bridge"`+consumer+`}`, `{"member":null}`)
				if !nullable {
					carrierDefaultFailure(t, input, true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
				} else if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	// Preserve historical enum/date/collection inputs as graduated member cases.
	// Hop/target defaults still keep their independent annotation roles.
	for _, target := range []string{`{"type":"string","enum":["x"]}`, `{"type":"string","format":"date"}`, `{"type":"array","items":{"type":"string"}}`} {
		if got, err := core.GenerateWithDiagnostics(request(scalarAliasMemberDocument(target, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{}`), "Dto")); err != nil || len(got.Artifacts.Files) == 0 {
			t.Fatal("enum/temporal/collection member graduation", target, err)
		}
	}
	hops := strings.Replace(allOfMemberHops, `"first":{`, `"first":{"default":"hidden",`, 1)
	carrierDefaultFailure(t, scalarAliasMemberDocument(`{"type":"string"}`, hops, `{"$ref":"#/components/schemas/Bridge"}`, `{}`), true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Hops/properties/first/default")
}

func TestScalarAliasObjectDefaultMemberContexts(t *testing.T) {
	definitions := `"Target":{"type":"string","nullable":true},"Alias":{"$ref":"#/components/schemas/Target"},`
	object := `{"properties":{"member":{"$ref":"#/components/schemas/Alias","nullable":false}}}`
	for _, field := range []string{
		strings.TrimSuffix(object, "}") + `,"default":{"member":"x"}}`,
		`{"type":"array","items":` + object + `,"default":[{"member":"x"}]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":{"member":"x"}}}`,
		`{"$ref":"#/components/schemas/Envelope","default":{"member":"x"}}`,
		`{"allOf":[` + object + `,{"properties":{"extra":{"type":"boolean"}}}],"default":{"member":"x"}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(`{`+definitions+`"Envelope":`+object+`,"Holder":{"properties":{"v":`+field+`}}}`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(field, err)
			}
		}
	}
	field := strings.TrimSuffix(object, "}") + `,"nullable":true,"default":{"member":null}}`
	for _, checked := range []bool{false, true} {
		for _, holder := range []string{
			`{"required":["v"],"properties":{"v":` + field + `}}`,
			`{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`,
		} {
			r := request(schemas(`{`+definitions+`"Holder":`+holder+`}`), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("suppressed default not checked", checked, err)
			}
		}
		patch := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`
		carrierDefaultFailure(t, []byte(patch), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member")
	}
}
