package core_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"openapi-kt/core"
)

const nativeEnumMemberNamedHops = `"Bridge":{"$ref":"#/components/schemas/Alias"},`

func TestNativeEnumAliasObjectDefaultMembers(t *testing.T) {
	for _, tc := range []struct{ target, value, scalar string }{
		{`{"type":"integer","enum":[-2147483648,1]}`, `-2147483648`, "Int"},
		{`{"type":"integer","format":"int64","enum":[-9223372036854775808,9223372036854775807]}`, `9223372036854775807`, "Long"},
		{`{"type":"number","format":"float","enum":[1,1.5]}`, `1.5`, "Float"},
		{`{"type":"number","format":"double","enum":[1,1.25]}`, `1.25`, "Double"},
		{`{"type":"boolean","enum":[true,false]}`, `false`, "Boolean"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, hops := range []string{nativeEnumMemberNamedHops, allOfMemberHops} {
					if !checked && hops == allOfMemberHops {
						continue // Internal native enum references retain validation opt-in.
					}
					for _, value := range []string{tc.value, `null`} {
						input := scalarAliasMemberDocument(tc.target, hops, `{"$ref":"#/components/schemas/Bridge","nullable":true}`, `{"member":`+value+`}`)
						r := request(input, "Dto")
						r.Options.Validation, r.Options.ModelStyle = checked, style
						baseline := r
						baseline.Document = []byte(strings.Replace(string(input), `,"default":{"member":`+value+`}`, "", 1))
						base, err := core.GenerateWithDiagnostics(baseline)
						if err != nil {
							t.Fatal("no-default control", tc.target, checked, err)
						}
						got, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal("member default", tc.target, checked, err)
						}
						text := source(t, got, "HolderDto")
						if !strings.Contains(text, "public val member: kotlin."+tc.scalar+"?") || !strings.Contains(text, "openapi-kt model default") {
							t.Fatal("native ABI or initializer lost", text)
						}
						if source(t, got, "TargetDto") != source(t, base, "TargetDto") || !reflect.DeepEqual(got.Warnings, base.Warnings) {
							t.Fatal("source declaration or warnings changed")
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

func TestNativeEnumAliasObjectDefaultMemberValuesAndBounds(t *testing.T) {
	at := "/components/schemas/Holder/properties/v/default/member"
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ target, value string }{
			{`{"type":"integer","enum":[1,2]}`, `9`},
			{`{"type":"integer","enum":[1,2]}`, `1.0`},
			{`{"type":"integer","enum":[1,2]}`, `1e0`},
			{`{"type":"integer","enum":[0,1]}`, `-0`},
			{`{"type":"integer","enum":[1,2]}`, `2147483648`},
			{`{"type":"integer","enum":[1,2]}`, `"1"`},
			{`{"type":"integer","enum":[1,2]}`, `null`},
			{`{"type":"number","format":"float","enum":[1]}`, `1.000000001`},
			{`{"type":"number","format":"double","enum":[1.5]}`, `1.50000000000000001`},
			{`{"type":"boolean","enum":[true]}`, `false`},
			{`{"type":"boolean","enum":[true]}`, `1`},
			{`{"type":"boolean","enum":[true]}`, `[]`},
		} {
			input := scalarAliasMemberDocument(tc.target, nativeEnumMemberNamedHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+tc.value+`}`)
			carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", at)
		}
		for _, value := range []string{`1`, `1.0`, `1e0`} {
			r := request(scalarAliasMemberDocument(`{"type":"number","format":"float","enum":[1]}`, nativeEnumMemberNamedHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+value+`}`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("numeric equality before mapping", err)
			}
		}
	}
	for _, value := range []string{`1`, `3`} {
		input := scalarAliasMemberDocument(`{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2}`, allOfMemberHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":`+value+`}`)
		carrierDefaultFailure(t, input, true, "E_SCHEMA_INVALID", at)
	}
	input := scalarAliasMemberDocument(`{"type":"number","format":"float","enum":[1],"multipleOf":0.5}`, nativeEnumMemberNamedHops, `{"$ref":"#/components/schemas/Bridge","nullable":true}`, `{"member":null}`)
	carrierDefaultFailure(t, input, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Target/multipleOf")
}

func TestNativeEnumAliasObjectDefaultMemberNullableAndAdmission(t *testing.T) {
	for _, far := range []string{"", `,"nullable":true`, `,"nullable":false`} {
		for _, near := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				hops := `"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/second"` + near + `},"second":{"$ref":"#/components/schemas/Alias"` + far + `}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/first"},`
				input := scalarAliasMemberDocument(`{"type":["integer","null"],"enum":[null,1,2]}`, hops, `{"$ref":"#/components/schemas/Bridge"`+consumer+`}`, `{"member":null}`)
				nullable := true
				for _, intent := range []string{far, near, consumer} {
					if intent != "" {
						nullable = intent == `,"nullable":true`
					}
				}
				if !nullable {
					carrierDefaultFailure(t, input, true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member")
				} else {
					r := request(input, "Dto")
					r.Options.Validation = true
					if _, err := core.GenerateWithDiagnostics(r); err != nil {
						t.Fatal("nearest nullable intent", err)
					}
				}
			}
		}
	}
	// Existing validation-only source roles do not become accepted through defaults.
	for _, tc := range []struct{ target, hops string }{
		{`{"type":"integer","enum":[1,2]}`, allOfMemberHops},
		{`{"type":["integer","null"],"enum":[null,1,2]}`, nativeEnumMemberNamedHops},
		{`{"type":"integer","enum":[1,2],"minimum":2}`, nativeEnumMemberNamedHops},
	} {
		input := scalarAliasMemberDocument(tc.target, tc.hops, `{"$ref":"#/components/schemas/Bridge"}`, `{}`)
		for _, doc := range [][]byte{input, []byte(strings.Replace(string(input), `,"default":{}`, "", 1))} {
			if got, err := core.GenerateWithDiagnostics(request(doc, "Dto")); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("reference/validation gate widened", err)
			}
		}
	}
}

func TestNativeEnumAliasObjectDefaultMemberSourceAndSuppression(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, target := range []string{
			`{"type":"integer","enum":[1],"default":1}`,
			`{"type":"integer","enum":[1],"x-native-type":"kotlin.Int"}`,
			`{"type":"integer","enum":[1],"x-kotlin-value-class":true}`,
			`{"type":"integer","enum":[1],"const":1}`,
			`{"type":"integer","enum":[1],"format":"uuid"}`,
			`{"type":"integer","enum":[]}`,
			`{"type":"integer","enum":[1,1]}`,
		} {
			for _, value := range []string{`{}`, `null`} {
				input := scalarAliasMemberDocument(target, nativeEnumMemberNamedHops, `{"$ref":"#/components/schemas/Bridge"}`, value)
				r := request([]byte(strings.Replace(string(input), `"v":{"properties"`, `"v":{"nullable":true,"properties"`, 1)), "Dto")
				r.Options.Validation = checked
				if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("source hidden by omission/null", target, err)
				}
			}
		}
		definitions := `"Code":{"type":"integer","enum":[1,2]},"Alias":{"$ref":"#/components/schemas/Code"},`
		field := `{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"nullable":true,"default":{"member":1}}`
		for _, role := range []string{"required", "patch", "discarded"} {
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
			if checked && role == "discarded" {
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("discarded enforcing enum checks dropped")
				}
				continue
			}
			if err != nil {
				t.Fatal("valid suppressed annotation", role, err)
			}
			r.Document = []byte(strings.Replace(string(input), `"default":{"member":1}`, `"default":{"member":2}`, 1))
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("suppressed annotation identity", role, err)
			}
			carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"default":{"member":1}`, `"default":{"member":9}`, 1)), checked, "E_SCHEMA_INVALID", pointer)
		}
	}
}

func TestNativeEnumAliasObjectDefaultMemberLimitsAndIdentity(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			members := []string{}
			for i := 0; i < count; i++ {
				members = append(members, strconv.Itoa(i))
			}
			input := scalarAliasMemberDocument(`{"type":"integer","enum":[`+strings.Join(members, ",")+`]}`, nativeEnumMemberNamedHops, `{"$ref":"#/components/schemas/Bridge"}`, `{"member":1}`)
			if count == 241 {
				carrierDefaultFailure(t, input, checked, "E_MODEL_LIMIT", "/components/schemas/Target/enum")
			} else {
				r := request(input, "Dto")
				r.Options.Validation = checked
				if _, err := core.GenerateWithDiagnostics(r); err != nil {
					t.Fatal("raw enum boundary", err)
				}
			}
		}
		for _, value := range []string{`1e0`, `2`} {
			r := request(schemas(`{"Code":{"type":"number","format":"float","enum":[1,2]},"Alias":{"$ref":"#/components/schemas/Code"},"Alpha":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"default":{"member":1}}}},"Zeta":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Alias"}},"default":{"member":`+value+`}}}}}`), "Dto")
			r.Options.Validation, r.Options.MergeIdenticalModels = checked, true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(source(t, got, "ZetaDto"), "typealias ZetaDto =") != (value == `1e0`) {
				t.Fatal("effective numeric default identity", value)
			}
		}
	}
}
