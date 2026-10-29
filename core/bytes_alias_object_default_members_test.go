package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBytesAliasObjectDefaultMembers(t *testing.T) {
	for _, format := range []string{"byte", "binary"} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, value := range []string{`[]`, `[-128,0,127]`, `null`} {
					input := scalarAliasMemberDocument(`{"type":["string","null"],"format":"`+format+`"}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+value+`}`)
					r := request(input, "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(format, checked, style, err)
					}
					if text := source(t, got, "HolderDto"); !strings.Contains(text, "public val member: kotlin.ByteArray?") || !strings.Contains(text, "openapi-kt model default") {
						t.Fatal(text)
					}
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("nondeterministic", err)
					}
					r.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+value+`}`, "", 1))
					base, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(err)
					}
					for _, name := range []string{"TargetDto", "AliasDto", "HopsDto", "BridgeDto"} {
						if source(t, got, name) != source(t, base, name) {
							t.Fatal("changed source declaration", name)
						}
					}
				}
			}
		}
	}
}

func TestBytesAliasObjectDefaultMemberGuards(t *testing.T) {
	for _, format := range []string{"byte", "binary"} {
		for _, checked := range []bool{false, true} {
			for _, tc := range []struct{ value, tail string }{
				{`"AA=="`, ""}, {`1`, ""}, {`[128]`, "/0"}, {`[-129]`, "/0"}, {`[0,1.5]`, "/1"}, {`[1e0]`, "/0"}, {`[-0]`, "/0"}, {`[null]`, "/0"}, {`["1"]`, "/0"}, {`null`, ""},
			} {
				input := scalarAliasMemberDocument(`{"type":"string","format":"`+format+`"}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+tc.value+`}`)
				carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member"+tc.tail)
			}
			for _, intent := range []string{`,"nullable":false`, `,"nullable":true`} {
				input := scalarAliasMemberDocument(`{"type":["string","null"],"format":"`+format+`"}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"`+intent+`}`, `{"member":null}`)
				if intent == `,"nullable":false` {
					carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
				} else {
					r := request(input, "Dto")
					r.Options.Validation = checked
					if _, err := core.GenerateWithDiagnostics(r); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}

func TestBytesAliasObjectDefaultMemberSourceAndPresence(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, extra := range []string{`,"enum":["x"]`, `,"x-native-type":"kotlin.ByteArray"`, `,"x-kotlin-value-class":true`, `,"default":[0]`, `,"minLength":2`} {
			r := request(scalarAliasMemberDocument(`{"type":"string","format":"byte"`+extra+`}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{}`), "Dto")
			r.Options.Validation = checked
			if extra == `,"enum":["x"]` {
				metadataStringEnumGraduation(t, r)
				continue
			}
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("source hidden by omission", extra, err)
			}
		}
		definitions := `"Blob":{"type":"string","format":"byte"},"Alias":{"$ref":"#/components/schemas/Blob"},`
		field := `{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"nullable":true,"default":{"member":[0,128]}}`
		for _, holder := range []string{
			`{"required":["v"],"properties":{"v":` + field + `}}`,
			`{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`,
		} {
			pointer := "/components/schemas/Holder/properties/v/default/member/1"
			if strings.Contains(holder, `"allOf"`) {
				pointer = "/components/schemas/Holder/allOf/0/properties/v/default/member/1"
			}
			carrierDefaultFailure(t, schemas(`{`+definitions+`"Holder":`+holder+`}`), checked, "E_SCHEMA_INVALID", pointer)
		}
		patch := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`
		carrierDefaultFailure(t, []byte(patch), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member/1")
	}
}
