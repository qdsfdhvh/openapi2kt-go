package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestWrapperChainDefaultsStayOnEachOwner(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, tc := range []struct{ name, shape, target, middle, outer string }{
				{"object", `"type":"object","required":["z"],"properties":{"z":{"type":"string"}}`, `{"z":"target"}`, `{"z":"middle"}`, `{"z":"outer"}`},
				{"allof", `"allOf":[{"properties":{"z":{"type":"string"}},"required":["z"]}]`, `{"z":"target"}`, `{"z":"middle"}`, `{"z":"outer"}`},
				{"list", `"type":"array","items":{"type":"string"}`, `["target"]`, `["middle"]`, `["outer"]`},
				{"map", `"additionalProperties":{"type":"integer"}`, `{"z":1}`, `{"z":2}`, `{"z":3}`},
				{"enum", `"type":"string","enum":["a","b","c"]`, `"a"`, `"b"`, `"c"`},
				{"date", `"type":"string","format":"date"`, `"2024-01-01"`, `"2024-02-29"`, `"2024-12-31"`},
				{"bytes", `"type":"string","format":"byte"`, `[-128]`, `[0]`, `[127]`},
				{"raw", `"type":"object"`, `{"nil":null}`, `[1,null]`, `"outer"`},
				{"open", `"additionalProperties":true`, `{"z":1}`, `{"z":2}`, `{"z":3}`},
				{"scalar", `"type":"string"`, `"target"`, `"middle"`, `"outer"`},
				{"native", `"type":"integer","format":"int32","enum":[1,2,3]`, `1`, `2`, `3`},
			} {
				if !checked && (tc.name == "scalar" || tc.name == "native") {
					continue
				}
				t.Run(tc.name+"/"+string(style), func(t *testing.T) {
					// Middle widens nullability and outer narrows it again. Each annotation
					// is local; neither nearby annotation may replace another owner's value.
					body := `{"T":{` + tc.shape + `,"default":` + tc.target + `},"B":{"$ref":"#/components/schemas/T"},"M":{"$ref":"#/components/schemas/B","nullable":true,"default":` + tc.middle + `},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":` + tc.outer + `},"Plain":{"$ref":"#/components/schemas/N"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N"}}}}`
					r := request(schemas(body), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					report, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						details, _ := json.Marshal(err)
						t.Fatal(checked, string(details))
					}
					for _, name := range []string{"MDto", "NDto"} {
						if !strings.Contains(source(t, report, name), "public fun `"+name+" default`()") {
							t.Fatal("alias factory missing", name)
						}
					}
					if text := source(t, report, "PlainDto"); strings.Contains(text, "fun `PlainDto default`") {
						t.Fatal("annotation inherited onto unannotated alias", text)
					}
					for _, item := range []struct{ name, value string }{{"TDto", tc.target}, {"MDto", tc.middle}, {"NDto", tc.outer}} {
						var doc map[string]any
						if err := json.Unmarshal(schemas(body), &doc); err != nil {
							t.Fatal(err)
						}
						schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
						for _, name := range []string{"T", "M", "N"} {
							if name+"Dto" != item.name {
								delete(schemas[name].(map[string]any), "default")
							}
						}
						controlDoc, err := json.Marshal(doc)
						if err != nil {
							t.Fatal(err)
						}
						r.Document = controlDoc
						control, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal("single owner control", err)
						}
						if source(t, report, item.name) != source(t, control, item.name) {
							t.Fatal("another owner's default changed factory", item.name, item.value)
						}
						if source(t, report, "HolderDto") != source(t, control, "HolderDto") || source(t, report, "PlainDto") != source(t, control, "PlainDto") {
							t.Fatal("default changed consuming type/presence")
						}
					}
					r.Document = schemas(body)
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(report, again) {
						t.Fatal("independent default generation drift", err)
					}
				})
			}
		}
	}
}

func TestWrapperChainDefaultsValidateEachOriginalOwner(t *testing.T) {
	for _, tc := range []struct{ owner, value, tail string }{
		{"T", `{}`, ""}, {"M", `{}`, ""}, {"N", `{}`, ""},
		{"T", `{"z":1}`, "/z"}, {"M", `{"z":1}`, "/z"}, {"N", `{"z":1}`, "/z"},
	} {
		values := map[string]string{"T": `{"z":"target"}`, "M": `{"z":"middle"}`, "N": `{"z":"outer"}`}
		values[tc.owner] = tc.value
		r := request(schemas(`{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}},"default":`+values["T"]+`},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":`+values["M"]+`},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":`+values["N"]+`}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/"+tc.owner+"/default"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("another annotation hid invalid own default", tc, err)
		}
	}
}

func TestWrapperChainOwnDefaultsKeepNullAndBudgetRules(t *testing.T) {
	for _, tc := range []struct{ target, middle, outer string }{
		{`["target"]`, `null`, `[null,"outer"]`}, {`["target"]`, `["middle"]`, `null`},
	} {
		r := request(schemas(`{"T":{"type":"array","items":{"type":["string","null"]},"default":`+tc.target+`},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":`+tc.middle+`},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":`+tc.outer+`}}`), "Dto")
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("independent null role rejected", tc, err)
		}
	}
	for _, owner := range []string{"T", "M", "N"} {
		for _, value := range []string{`"` + strings.Repeat("a", 32768) + `"`, `[` + strings.Repeat("0,", 16384) + `0]`} {
			values := map[string]string{"T": "{}", "M": "{}", "N": "{}"}
			values[owner] = value
			r := request(schemas(`{"T":{"type":"object","default":`+values["T"]+`},"M":{"$ref":"#/components/schemas/T","default":`+values["M"]+`},"N":{"$ref":"#/components/schemas/M","default":`+values["N"]+`}}`), "Dto")
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/"+owner+"/default") || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("default owner lost its own resource budget", owner, err)
			}
		}
	}
	for _, body := range []string{
		`{"T":{"type":"string","x-kotlin-value-class":true,"default":"target"},"N":{"$ref":"#/components/schemas/T","default":"outer"}}`,
		`{"T":{"oneOf":[{"type":"string"},{"type":"integer"}],"default":"target"},"N":{"$ref":"#/components/schemas/T","default":"outer"}}`,
		`{"T":{"type":"array","items":{"type":"string","default":"child"},"default":[]},"N":{"$ref":"#/components/schemas/T","default":[]}}`,
		`{"N":{"$ref":"#/components/schemas/M","default":{}},"M":{"$ref":"#/components/schemas/N","default":{}}}`,
	} {
		r := request(schemas(body), "Dto")
		r.Options.Validation = true
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("independent annotations granted unsupported source", body, err)
		}
	}
}
