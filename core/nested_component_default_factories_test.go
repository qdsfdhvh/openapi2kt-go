package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedComponentDefaultFactories(t *testing.T) {
	for _, tc := range []struct{ name, shape, value, kotlin string }{
		{"lists", `"type":"array","items":{"type":"array","items":{"type":"integer"}}`, `[[1],[]]`, "kotlin.collections.List<kotlin.collections.List<kotlin.Int>>"},
		{"list maps", `"type":"array","items":{"additionalProperties":{"type":"string"}}`, `[{"z":"last","a":"first"},{}]`, "kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.String>>"},
		{"map lists", `"additionalProperties":{"type":"array","nullable":true,"items":{"type":["boolean","null"]}}`, `{"nil":null,"rows":[true,null],"empty":[]}`, "kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.Boolean?>?>"},
		{"maps", `"type":"object","additionalProperties":{"additionalProperties":{"type":"number","format":"float"}}`, `{"z":{"z":1.5,"a":2.5},"a":{}}`, "kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, kotlin.Float>>"},
		{"deeper", `"type":["array","null"],"items":{"additionalProperties":{"type":"array","nullable":true,"items":{"type":"integer","format":"int64","nullable":true}}}`, `[{"rows":[9223372036854775807,null],"nil":null}]`, "kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.Long?>?>>?"},
		{"null root", `"type":"array","nullable":true,"items":{"type":"array","items":{"type":"string"}}`, `null`, "kotlin.collections.List<kotlin.collections.List<kotlin.String>>?"},
		{"empty root", `"type":"array","items":{"type":"array","items":{"type":"string"}}`, `[]`, "kotlin.collections.List<kotlin.collections.List<kotlin.String>>"},
		{"date", `"type":"array","items":{"additionalProperties":{"type":"string","format":"date","nullable":true}}`, `[{"a":null,"z":"2024-02-29"}]`, ""},
		{"instant", `"additionalProperties":{"type":"array","items":{"type":"string","format":"date-time"}}`, `{"rows":["2024-02-29T00:00:00Z"]}`, ""},
		{"bytes", `"type":"array","items":{"type":"array","items":{"type":"string","format":"byte","nullable":true}}`, `[[null,[-128,127]]]`, ""},
		{"string enum", `"type":"array","items":{"additionalProperties":{"type":"string","enum":["a","b"],"nullable":true}}`, `[{"a":null,"z":"a"}]`, ""},
		{"native enum", `"additionalProperties":{"type":"array","items":{"type":"integer","format":"int32","enum":[1,2],"nullable":true}}`, `{"rows":[null,2]}`, ""},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				t.Run(tc.name+"/"+string(style)+"/checked="+map[bool]string{false: "off", true: "on"}[checked], func(t *testing.T) {
					r := request(schemas(`{"N":{`+tc.shape+`,"default":`+tc.value+`},"Holder":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/N"},"required":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(err)
					}
					text := source(t, got, "NDto")
					if !strings.Contains(text, "public fun `NDto default`(): NDto =") || (tc.kotlin != "" && !strings.Contains(text, "public typealias NDto = "+tc.kotlin+"\n")) {
						t.Fatal("nested alias type/factory lost", text)
					}
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("nested factory generation drifted", err)
					}
					r.Document = []byte(strings.Replace(string(r.Document), `,"default":`+tc.value, "", 1))
					control, err := core.GenerateWithDiagnostics(r)
					if err != nil || source(t, control, "HolderDto") != source(t, got, "HolderDto") || !reflect.DeepEqual(control.Warnings, got.Warnings) {
						t.Fatal("nested factory changed consumer presence/warnings", err)
					}
					if strings.Contains(source(t, control, "NDto"), "fun `NDto default`") {
						t.Fatal("unannotated nested alias gained a factory")
					}
				})
			}
		}
	}
}

// These are the original nested field-profile root/target/alias exclusions.
// The component factory role now owns the annotation; the field still gains no
// initializer. A wrapper factory must leave its target's output untouched.
func TestNestedComponentDefaultFactoriesGraduatedRoles(t *testing.T) {
	for _, tc := range []struct{ shape, value string }{
		{`"type":"array","items":{"additionalProperties":{"type":"string"}}`, `[{"a":"one"}]`},
		{`"type":"array","items":{"type":"array","items":{"type":"string","format":"date"}}`, `[]`},
		{`"type":"array","items":{"additionalProperties":{"enum":["one"]}}`, `[]`},
		{`"type":"array","items":{"type":"array","items":{"type":"integer","enum":[1,2]}}`, `[[1]]`},
	} {
		for _, checked := range []bool{false, true} {
			for _, wrapped := range []bool{false, true} {
				body := `"N":{` + tc.shape + `,"default":` + tc.value + `}`
				if wrapped {
					body = `"T":{` + tc.shape + `},"N":{"$ref":"#/components/schemas/T","default":` + tc.value + `}`
				}
				r := request(schemas(`{`+body+`,"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || !strings.Contains(source(t, got, "NDto"), "public fun `NDto default`()") {
					t.Fatal("original root/wrapper case lost", tc, checked, wrapped, err)
				}
				r.Document = []byte(strings.Replace(string(r.Document), `,"default":`+tc.value, "", 1))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil || source(t, got, "HolderDto") != source(t, control, "HolderDto") || (wrapped && source(t, got, "TDto") != source(t, control, "TDto")) {
					t.Fatal("nested factory changed referencing field or wrapper target", err)
				}
			}
		}
	}
}
