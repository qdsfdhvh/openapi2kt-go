package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTemporalAliasObjectDefaultMembers(t *testing.T) {
	for _, format := range []string{"date", "date-time"} {
		value := `"2024-02-29"`
		actual := "kotlinx.datetime.LocalDate"
		if format == "date-time" {
			value = `"2024-02-29T03:04:05.123456789+02:00"`
			actual = "kotlin.time.Instant"
		}
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := scalarAliasMemberDocument(`{"type":"string","format":"`+format+`"}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+value+`}`)
				r := request(input, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(format, checked, style, err)
				}
				if text := source(t, got, "HolderDto"); !strings.Contains(text, "public val member: "+actual+"?") || !strings.Contains(text, "openapi-kt model default") {
					t.Fatal(text)
				}
				r.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+value+`}`, "", 1))
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"TargetDto", "AliasDto", "HopsDto", "BridgeDto"} {
					if source(t, got, name) != source(t, baseline, name) {
						t.Fatal("changed source declaration", name)
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

func TestTemporalAliasObjectDefaultMemberGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ format, value string }{
			{"date", `"2023-02-29"`}, {"date", `"2024-01-01T00:00:00Z"`}, {"date", `7`},
			{"date-time", `"2024-02-29"`}, {"date-time", `"2024-01-01T00:00:60Z"`},
			{"date-time", `"2024-01-01T00:00:00.1234567890Z"`}, {"date-time", `"2024-01-01T00:00:00+19:00"`},
		} {
			carrierDefaultFailure(t, scalarAliasMemberDocument(`{"type":"string","format":"`+tc.format+`"}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+tc.value+`}`), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
		}
		for _, intent := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			input := scalarAliasMemberDocument(`{"type":["string","null"],"format":"date"}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"`+intent+`}`, `{"member":null}`)
			if intent == `,"nullable":false` {
				carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
			} else if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestTemporalAliasObjectDefaultMemberSourceAndPresence(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, extra := range []string{`,"enum":["2024-02-29"]`, `,"x-native-type":"kotlinx.datetime.LocalDate"`, `,"x-kotlin-value-class":true`, `,"default":"2024-02-29"`, `,"minLength":2`} {
			r := request(scalarAliasMemberDocument(`{"type":"string","format":"date"`+extra+`}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{}`), "Dto")
			r.Options.Validation = checked
			if extra == `,"enum":["2024-02-29"]` {
				metadataStringEnumGraduation(t, r)
				continue
			}
			if got, err := core.GenerateWithDiagnostics(r); err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("source hidden by omission", extra, err)
			}
		}
		definitions := `"Day":{"type":"string","format":"date"},"Alias":{"$ref":"#/components/schemas/Day"},`
		field := `{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"nullable":true,"default":{"member":"2023-02-29"}}`
		for _, holder := range []string{
			`{"required":["v"],"properties":{"v":` + field + `}}`,
			`{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`,
		} {
			pointer := "/components/schemas/Holder/properties/v/default/member"
			if strings.Contains(holder, `"allOf"`) {
				pointer = "/components/schemas/Holder/allOf/0/properties/v/default/member"
			}
			carrierDefaultFailure(t, schemas(`{`+definitions+`"Holder":`+holder+`}`), checked, "E_SCHEMA_INVALID", pointer)
		}
		patch := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`
		carrierDefaultFailure(t, []byte(patch), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member")
	}
}
