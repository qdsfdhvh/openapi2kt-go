package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestWrapperChainDefaultNearestNullableIntent(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, tc := range []struct{ name, shape, value, nullCode string }{
				{"object", `"type":"object","properties":{"z":{"type":"string"}},"required":["z"]`, `{"z":"given"}`, "E_SCHEMA_INVALID"},
				{"allof", `"type":"object","allOf":[{"properties":{"z":{"type":"string"}},"required":["z"]}]`, `{"z":"given"}`, "E_SCHEMA_INVALID"},
				{"list", `"type":"array","items":{"type":["string","null"]}`, `[null,"given"]`, "E_SCHEMA_INVALID"},
				{"map", `"type":"object","additionalProperties":{"type":["integer","null"]}`, `{"nil":null,"a":1}`, "E_SCHEMA_INVALID"},
				{"enum", `"type":"string","enum":["a","b"]`, `"a"`, "E_SCHEMA_INVALID"},
				{"day", `"type":"string","format":"date"`, `"2024-02-29"`, "E_SCHEMA_INVALID"},
				{"bytes", `"type":"string","format":"byte"`, `[-128,127]`, "E_SCHEMA_INVALID"},
				{"raw", `"type":"object"`, `[1,null]`, "E_SCHEMA_UNSUPPORTED"},
				{"open", `"type":"object","additionalProperties":true`, `{"nil":null}`, "E_SCHEMA_UNSUPPORTED"},
				{"scalar", `"type":"string","minLength":1`, `"given"`, "E_SCHEMA_INVALID"},
				{"native", `"type":"integer","format":"int32","enum":[1,2]`, `1`, "E_SCHEMA_INVALID"},
			} {
				if !checked && (tc.name == "scalar" || tc.name == "native") {
					continue
				}
				for _, targetNullable := range []string{"none", "3.0", "3.1"} {
					var target map[string]any
					if err := json.Unmarshal([]byte(`{`+tc.shape+`}`), &target); err != nil {
						t.Fatal(err)
					}
					if targetNullable == "3.0" {
						target["nullable"] = true
					} else if targetNullable == "3.1" {
						target["type"] = []any{target["type"], "null"}
					}
					encoded, err := json.Marshal(target)
					if err != nil {
						t.Fatal(err)
					}
					for _, intent := range []struct {
						outer, near, far string
						want             bool
					}{
						{"", "true", "", true}, {"", "false", "true", false}, {"", "true", "false", true},
						{"false", "true", "", false}, {"true", "false", "", true}, {"false", "", "", false},
					} {
						t.Run(fmt.Sprintf("%s/%s/%s/%s-%s-%s/checked=%t", tc.name, style, targetNullable, intent.outer, intent.near, intent.far, checked), func(t *testing.T) {
							nullable := func(v string) string {
								if v == "" {
									return ""
								}
								return `,"nullable":` + v
							}
							for _, value := range []string{tc.value, "null"} {
								body := `{"T":` + string(encoded) + `,"B":{"$ref":"#/components/schemas/T"` + nullable(intent.far) + `},"M":{"$ref":"#/components/schemas/B"` + nullable(intent.near) + `},"N":{"$ref":"#/components/schemas/M"` + nullable(intent.outer) + `,"default":` + value + `},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`
								r := request(schemas(body), "Dto")
								r.Options.Validation, r.Options.ModelStyle = checked, style
								got, err := core.GenerateWithDiagnostics(r)
								if value == "null" && !intent.want {
									ds, ok := err.(core.Diagnostics)
									if !ok || len(ds) == 0 || ds[0].Code != tc.nullCode || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
										t.Fatal("narrowed null lost original rejection", err)
									}
									continue
								}
								if err != nil {
									t.Fatal(value, err)
								}
								text := source(t, got, "NDto")
								if !strings.Contains(text, "public fun `NDto default`(): NDto =") || !strings.Contains(text, "serializer<NDto>()") {
									t.Fatal(text)
								}
								r.Document = schemas(strings.Replace(body, `,"default":`+value, "", 1))
								control, err := core.GenerateWithDiagnostics(r)
								if err != nil {
									t.Fatal("no-default control", err)
								}
								for _, name := range []string{"TDto", "BDto", "MDto", "HolderDto"} {
									if source(t, got, name) != source(t, control, name) {
										t.Fatal("default changed original declaration/consumer", name)
									}
								}
								if !reflect.DeepEqual(got.Warnings, control.Warnings) {
									t.Fatal("warning drift", got.Warnings, control.Warnings)
								}
							}
						})
					}
				}
			}
		}
	}
}

func TestWrapperChainDefaultNullableGuards(t *testing.T) {
	for _, tc := range []struct{ shape, value, tail string }{
		{`"type":"string","minLength":1`, `""`, ""},
		{`"type":"integer","format":"int32","enum":[1,2]`, `3`, ""},
		{`"type":["array","null"],"minItems":1,"items":{"type":"string"}`, `[]`, ""},
		{`"type":"object","additionalProperties":{"type":"integer"}`, `{"a/~":false}`, "/a~1~0"},
	} {
		r := request(schemas(`{"T":{`+tc.shape+`},"M":{"$ref":"#/components/schemas/T","nullable":true},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":`+tc.value+`}}`), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("nullable chain dropped value constraints/Pointer", tc, err)
		}
	}
	for _, depth := range []int{44, 45} {
		r := request(schemas(`{"T":`+aliasDepthTree(depth, false)+`,"M":{"$ref":"#/components/schemas/T","nullable":true},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(r)
		if depth == 44 {
			if err != nil {
				t.Fatal(err)
			}
		} else if ds, ok := err.(core.Diagnostics); !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("narrowing erased cached chain height", err)
		}
	}
	for _, middle := range []string{`"nullable":"bad"`, `"nullable":false,"default":null`, `"nullable":true,"minLength":1`} {
		r := request(schemas(`{"N":{"$ref":"#/components/schemas/M","nullable":true,"default":null},"M":{"$ref":"#/components/schemas/T",`+middle+`},"T":{"properties":{"z":{"type":"string"}}}}`), "Dto")
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("outer override discarded invalid source", middle, err)
		}
	}
}

// Explicit nullable wrappers already promised retained primitive bounds. Check
// both the old widening role and newly admitted narrowing role before returning
// the annotation, rather than validating the scalar source alone.
func TestWrapperDefaultScalarBoundsAreCheckedForEitherIntent(t *testing.T) {
	for _, intent := range []string{"true", "false"} {
		r := request(schemas(`{"T":{"type":"string","minLength":1},"N":{"$ref":"#/components/schemas/T","nullable":`+intent+`,"default":""}}`), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/N/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalid constrained scalar annotation accepted", intent, err)
		}
	}
}
