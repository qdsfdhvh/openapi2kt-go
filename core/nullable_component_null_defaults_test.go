package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableComponentNullFactories(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, tc := range []struct{ name, shape, alias string }{
				{"String", `"type":"string"`, "kotlin.String?"},
				{"Boolean", `"type":"boolean"`, "kotlin.Boolean?"},
				{"Int", `"type":"integer"`, "kotlin.Int?"},
				{"Day", `"type":"string","format":"date"`, "kotlinx.datetime.LocalDate?"},
				{"Stamp", `"type":"string","format":"date-time"`, "kotlin.time.Instant?"},
				{"Byte", `"type":"string","format":"byte"`, "kotlin.ByteArray?"},
				{"Binary", `"type":"string","format":"binary"`, "kotlin.ByteArray?"},
				{"List", `"type":"array","items":{"type":"string"}`, "kotlin.collections.List<kotlin.String>?"},
				{"Map", `"type":"object","additionalProperties":{"type":"integer"}`, "kotlin.collections.Map<kotlin.String, kotlin.Int>?"},
				{"ByteList", `"type":"array","items":{"type":"string","format":"byte"}`, "kotlin.collections.List<kotlin.ByteArray>?"},
			} {
				t.Run(tc.name+"/"+string(style), func(t *testing.T) {
					body := `{"N":{` + tc.shape + `,"nullable":true,"default":null},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`
					r := request(schemas(body), "Dto")
					r.Options.ModelStyle = style
					r.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(checked, err)
					}
					text := source(t, got, "NDto")
					if !strings.Contains(text, "public typealias NDto = "+tc.alias) || !strings.Contains(text, "public fun `NDto default`(): NDto =") || !strings.Contains(text, `("null")`) {
						t.Fatal(text)
					}
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("nondeterministic", err)
					}
					r.Document = schemas(strings.Replace(body, `,"default":null`, "", 1))
					control, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal("no-default control", err)
					}
					if source(t, got, "HolderDto") != source(t, control, "HolderDto") || !reflect.DeepEqual(got.Warnings, control.Warnings) {
						t.Fatal("factory inherited or warning drift", checked)
					}
				})
			}
			for _, shape := range []string{`"type":["string","null"],"enum":["a","b"]`, `"enum":["a"],"nullable":true`} {
				r := request(schemas(`{"N":{`+shape+`,"default":null}}`), "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(checked, style, err)
				}
				text := source(t, got, "NDto")
				if !strings.Contains(text, "public fun `NDto default`(): NDto? = null") || strings.Contains(text, "openapi-kt component default") {
					t.Fatal("null enum must retain its classifier and need no helper", text)
				}
			}
		}
	}
}

func TestNullableComponentNullFactorySchemaIsNotSkipped(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, body := range []string{
			`{"T":{"type":"string"},"N":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/T"},"default":null}}`,
			`{"N":{"type":"string","nullable":true,"x-native-type":"org.example.Money","default":null}}`,
			`{"N":{"type":"string","nullable":true,"x-kotlin-value-class":true,"default":null}}`,
			`{"N":{"type":"string","nullable":true,"const":"a","default":null}}`,
			`{"N":{"type":"string","nullable":true,"enum":["a",1],"default":null}}`,
			`{"N":{"type":"array","nullable":true,"items":{"type":"string","default":"a"},"default":null}}`,
		} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("null bypassed source", checked, body, err)
			}
		}
	}
	for _, tc := range []struct{ body, pointer string }{
		{`{"N":{"type":"string","nullable":true,"minLength":-1,"default":null}}`, "/components/schemas/N/default"},
		{`{"N":{"type":"array","nullable":true,"minItems":-1,"items":{"type":"string"},"default":null}}`, "/components/schemas/N/default"},
	} {
		r := request(schemas(tc.body), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
			encoded, _ := json.Marshal(err)
			t.Fatal("original bad source lost", tc, string(encoded))
		}
	}
}

func TestNullableNativeEnumNullFactorySourceAndWarnings(t *testing.T) {
	for _, shape := range []string{
		`"type":"integer","format":"int32","enum":[1,2]`,
		`"type":"integer","format":"int64","enum":[1,2]`,
		`"type":"number","format":"float","enum":[1,2]`,
		`"type":"number","format":"double","enum":[1,2]`,
		`"type":"boolean","enum":[true,false]`,
	} {
		body := `{"N":{` + shape + `,"nullable":true,"default":null}}`
		r := request(schemas(body), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(shape, err)
		}
		if text := source(t, got, "NDto"); !strings.Contains(text, "public fun `NDto default`(): NDto =") || !strings.Contains(text, `("null")`) {
			t.Fatal(text)
		}
		nonNull := "1"
		if strings.Contains(shape, `"type":"boolean"`) {
			nonNull = "true"
		}
		// Factory warnings follow the existing non-null factory role. Bare
		// checked aliases have no factory approximation warning.
		r.Document = schemas(strings.Replace(body, `"default":null`, `"default":`+nonNull, 1))
		control, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got.Warnings, control.Warnings) || len(got.Warnings) != 1 {
			t.Fatal("native source warning drift", shape, err, got.Warnings, control.Warnings)
		}
		r.Document = schemas(body)
		r.Options.Validation = false
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("native nullable opt-in widened", shape, err)
		}
	}
	for _, shape := range []string{`"type":"string"`, `"type":"string","enum":["a"]`, `"type":"array","items":{"type":"string"}`, `"additionalProperties":{"type":"integer"}`} {
		failure(t, schemas(`{"N":{`+shape+`,"default":null}}`), "E_SCHEMA_INVALID", "/components/schemas/N/default")
	}
}

func TestNullableComponentNullFactoryVocabularyBudget(t *testing.T) {
	for _, native := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			members := make([]any, count)
			for i := range members {
				if native {
					members[i] = i
				} else {
					members[i] = fmt.Sprintf("v%d", i)
				}
			}
			encoded, err := json.Marshal(members)
			if err != nil {
				t.Fatal(err)
			}
			kind := `"type":"string"`
			if native {
				kind = `"type":"integer","format":"int32"`
			}
			r := request(schemas(`{"N":{`+kind+`,"enum":`+string(encoded)+`,"nullable":true,"default":null}}`), "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if count == 240 {
				if err != nil {
					t.Fatal(native, count, err)
				}
			} else {
				ds, ok := err.(core.Diagnostics)
				code := "E_ENUM_LIMIT"
				if native {
					code = "E_MODEL_LIMIT"
				}
				if !ok || len(ds) != 1 || ds[0].Code != code || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/enum" || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal(native, count, err)
				}
			}
		}
	}
}

func TestNullableComponentNullFactoryPreservesReferenceWrapperRoles(t *testing.T) {
	// Plain nullable scalar wrapper null was not a component-root own-default role.
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"T":{"type":"string","nullable":true},"N":{"$ref":"#/components/schemas/T","default":null}}`), "Dto")
		r.Options.Validation = checked
		if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("wrapper null permission widened", checked, err)
		}
	}
	// Previously admitted byte/temporal wrapper null keeps its helper/alias form.
	for _, format := range []string{"byte", "date"} {
		r := request(schemas(`{"T":{"type":"string","nullable":true,"format":"`+format+`"},"N":{"$ref":"#/components/schemas/T","default":null}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(format, err)
		}
		if text := source(t, got, "NDto"); !strings.Contains(text, `("null")`) || !strings.Contains(text, "private fun `openapi-kt component default") {
			t.Fatal("existing wrapper output changed", text)
		}
	}
}
