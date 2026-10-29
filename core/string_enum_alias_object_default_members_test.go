package core_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"openapi-kt/core"
)

func stringEnumAliasMemberDocument(target string, internal bool, member, value string) []byte {
	if !internal {
		return scalarAliasMemberDocument(target, allOfMemberHops, member, value)
	}
	return schemas(`{"Source":{"properties":{"code":` + target + `}},"Alias":{"$ref":"#/components/schemas/Source/properties/code"},` + allOfMemberHops + `"Holder":{"properties":{"v":{"properties":{"member":` + member + `},"default":` + value + `}}}}`)
}

func TestStringEnumAliasObjectDefaultMembers(t *testing.T) {
	for _, target := range []string{`{"type":["string","null"],"enum":[null,"on","off","null"]}`, `{"nullable":true,"enum":[null,"on","off","null"]}`} {
		for _, internal := range []bool{false, true} {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					for _, value := range []string{`"on"`, `"\u006fn"`, `"null"`, `null`} {
						input := stringEnumAliasMemberDocument(target, internal, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+value+`}`)
						r := request(input, "Dto")
						r.Options.Validation, r.Options.ModelStyle = checked, style
						got, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal(target, internal, checked, style, value, err)
						}
						owner, classifier := "TargetDto", "TargetDto"
						if internal {
							owner, classifier = "SourceDto", "SourceDto.Code"
						}
						text := source(t, got, "HolderDto")
						if !strings.Contains(text, "public val member: org.example.models.models."+classifier+"?") || !strings.Contains(text, "openapi-kt model default") || strings.Contains(text, "Validation") {
							t.Fatal("lost actual enum classifier", text)
						}
						baseline := r
						baseline.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+value+`}`, "", 1))
						base, err := core.GenerateWithDiagnostics(baseline)
						if err != nil {
							t.Fatal(err)
						}
						for _, name := range []string{owner, "AliasDto", "HopsDto", "BridgeDto"} {
							if source(t, got, name) != source(t, base, name) {
								t.Fatal("changed source declaration", name)
							}
						}
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

func TestStringEnumAliasObjectDefaultMemberVocabularyAndNullable(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, internal := range []bool{false, true} {
			for _, value := range []string{`"unknown"`, `"o\u0301n"`, `1`, `true`, `[]`, `{}`, `null`} {
				input := stringEnumAliasMemberDocument(`{"type":"string","enum":["on","off"]}`, internal, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+value+`}`)
				// Diagnostics use the escaped consuming wire, not an alias or enum source.
				input = []byte(strings.ReplaceAll(string(input), `"member"`, `"a/~"`))
				carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/a~1~0")
			}
		}
		for _, far := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			for _, near := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
					hops := `"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second"` + near + `},"second":{"$ref":"#/components/schemas/Alias"` + far + `}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/first"},`
					input := scalarAliasMemberDocument(`{"type":["string","null"],"enum":[null,"on","null"]}`, hops, `{"$ref":"#/components/schemas/Bridge"`+consumer+`}`, `{"member":null}`)
					nullable := true
					for _, intent := range []string{far, near, consumer} {
						if intent != "" {
							nullable = intent == `,"nullable":true`
						}
					}
					if !nullable {
						carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
					} else {
						r := request(input, "Dto")
						r.Options.Validation = checked
						if _, err := core.GenerateWithDiagnostics(r); err != nil {
							t.Fatal("nearest nullable intent", err)
						}
					}
				}
			}
		}
	}
}

func TestStringEnumAliasObjectDefaultMemberSourceAndSuppression(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, extra := range []string{`,"format":"uuid"`, `,"minLength":1`, `,"pattern":"on"`, `,"x-native-type":"kotlin.String"`, `,"x-kotlin-value-class":true`, `,"default":"on"`, `,"const":"on"`} {
			for _, value := range []string{`{}`, `null`} {
				input := stringEnumAliasMemberDocument(`{"type":"string","enum":["on"]`+extra+`}`, false, `{"$ref":"#/components/schemas/Bridge"}`, value)
				r := request([]byte(strings.Replace(string(input), `"v":{"properties"`, `"v":{"nullable":true,"properties"`, 1)), "Dto")
				r.Options.Validation = checked
				if extra == `,"default":"on"` {
					independentEnumFactoryGraduation(t, r)
					continue
				}
				if checked && (extra == `,"minLength":1` || extra == `,"pattern":"on"`) {
					checkedStringEnumRootGraduation(t, r)
					continue
				}
				if extra == `,"format":"uuid"` || extra == `,"x-native-type":"kotlin.String"` {
					metadataStringEnumGraduation(t, r)
					continue
				}
				if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("source hidden by empty/null annotation", extra, err)
				}
			}
		}
		hops := strings.Replace(allOfMemberHops, `"first":{`, `"first":{"default":"on",`, 1)
		carrierDefaultFailure(t, scalarAliasMemberDocument(`{"type":"string","enum":["on"]}`, hops, `{"$ref":"#/components/schemas/Bridge"}`, `{}`), checked, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Hops/properties/first/default")
		definitions := `"Status":{"type":"string","enum":["on","off"]},"Alias":{"$ref":"#/components/schemas/Status"},`
		field := `{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"nullable":true,"default":{"member":"on"}}`
		for _, role := range []string{"required", "discarded", "patch"} {
			holder := `{"required":["v"],"properties":{"v":` + field + `}}`
			pointer := "/components/schemas/Holder/properties/v/default/member"
			if role == "discarded" {
				holder = `{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`
				pointer = "/components/schemas/Holder/allOf/0/properties/v/default/member"
			}
			input := schemas(`{` + definitions + `"Holder":` + holder + `}`)
			if role == "patch" {
				input = []byte(strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{` + strings.TrimSuffix(definitions, ",") + `}}}`)
				pointer = "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/member"
			}
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("valid suppressed annotation", role, err)
			}
			r.Document = []byte(strings.Replace(string(input), `"default":{"member":"on"}`, `"default":{"member":"off"}`, 1))
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("suppressed default identity", role, err)
			}
			carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"default":{"member":"on"}`, `"default":{"member":"unknown"}`, 1)), checked, "E_SCHEMA_INVALID", pointer)
		}
	}
}

func TestStringEnumAliasObjectDefaultMemberEnumLimit(t *testing.T) {
	for _, count := range []int{240, 241} {
		members := []string{`null`}
		for i := 1; i < count; i++ {
			members = append(members, `"v`+strconv.Itoa(i)+`"`)
		}
		input := stringEnumAliasMemberDocument(`{"type":["string","null"],"enum":[`+strings.Join(members, ",")+`]}`, true, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":"v1"}`)
		for _, checked := range []bool{false, true} {
			if count == 241 {
				carrierDefaultFailure(t, input, checked, "E_ENUM_LIMIT", "/components/schemas/Source/properties/code/enum")
			} else {
				r := request(input, "Dto")
				r.Options.Validation = checked
				if _, err := core.GenerateWithDiagnostics(r); err != nil {
					t.Fatal("raw enum boundary", err)
				}
			}
		}
	}
}

func TestStringEnumAliasObjectDefaultMemberContextsAndIdentity(t *testing.T) {
	definitions := `"Status":{"type":"string","enum":["on","off"]},"Alias":{"$ref":"#/components/schemas/Status"},`
	object := `{"properties":{"member":{"$ref":"#/components/schemas/Alias"}}}`
	for _, field := range []string{
		strings.TrimSuffix(object, "}") + `,"default":{"member":"on"}}`,
		`{"type":"array","items":` + object + `,"default":[{"member":"on"}]}`,
		`{"type":"object","additionalProperties":` + object + `,"default":{"k":{"member":"on"}}}`,
		`{"$ref":"#/components/schemas/Envelope","default":{"member":"on"}}`,
		`{"allOf":[` + object + `,{"properties":{"extra":{"type":"boolean"}}}],"default":{"member":"on"}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(`{`+definitions+`"Envelope":`+object+`,"Holder":{"properties":{"v":`+field+`}}}`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("object-default context", field, err)
			}
		}
	}
	// Canonical String equality composes with whole-object default identity.
	for _, value := range []string{`"\u006fn"`, `"off"`} {
		r := request(schemas(`{`+definitions+`"Alpha":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"default":{"member":"on"}}}},"Zeta":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"default":{"member":`+value+`}}}}}`), "Dto")
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(source(t, got, "ZetaDto"), "typealias ZetaDto =") != (value == `"\u006fn"`) {
			t.Fatal("effective default merge identity", value)
		}
	}
}
