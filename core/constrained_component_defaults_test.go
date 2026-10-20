package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestConstrainedScalarComponentDefaults(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, tc := range []struct{ name, shape, value, alias string }{
			{"text", `"type":"string","minLength":2,"maxLength":6,"pattern":"^[a-z]+$"`, `"given"`, "kotlin.String"},
			{"integer", `"type":"integer","minimum":0,"maximum":10,"multipleOf":2`, `4`, "kotlin.Int"},
			{"number", `"type":"number","exclusiveMinimum":0,"maximum":2`, `1.50`, "kotlin.Double"},
		} {
			for _, nullable := range []string{"", "true", "3.1"} {
				var target map[string]any
				if err := json.Unmarshal([]byte(`{`+tc.shape+`}`), &target); err != nil {
					t.Fatal(err)
				}
				if nullable == "true" {
					target["nullable"] = true
				} else if nullable == "3.1" {
					target["type"] = []string{target["type"].(string), "null"}
				}
				shape, err := json.Marshal(target)
				if err != nil {
					t.Fatal(err)
				}
				for _, value := range []string{tc.value, "null"} {
					if nullable == "" && value == "null" {
						continue
					}
					t.Run(tc.name+"/"+string(style)+"/"+nullable+"/"+value, func(t *testing.T) {
						body := `{"T":` + strings.TrimSuffix(string(shape), "}") + `,"default":` + value + `},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":` + tc.value + `},"Plain":{"$ref":"#/components/schemas/N"},"Holder":{"properties":{"target":{"$ref":"#/components/schemas/T"},"outer":{"$ref":"#/components/schemas/N"}}}}`
						r := request(schemas(body), "Dto")
						r.Options.Validation, r.Options.ModelStyle = true, style
						report, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							details, _ := json.Marshal(err)
							t.Fatal(string(details))
						}
						if text := source(t, report, "TDto"); !strings.Contains(text, "public typealias TDto = "+tc.alias) || !strings.Contains(text, "public fun `TDto default`(): TDto =") {
							t.Fatal(text)
						}
						again, err := core.GenerateWithDiagnostics(r)
						if err != nil || !reflect.DeepEqual(report, again) {
							t.Fatal("constrained factories drift", err)
						}
						r.Document = schemas(strings.Replace(body, `,"default":`+value, "", 1))
						control, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal("no own-default control", err)
						}
						for _, name := range []string{"MDto", "NDto", "PlainDto", "HolderDto"} {
							if source(t, report, name) != source(t, control, name) {
								t.Fatal("component default changed consumer/another factory", name)
							}
						}
						if !reflect.DeepEqual(report.Warnings, control.Warnings) {
							t.Fatal("factory changed warnings")
						}
					})
				}
			}
		}
	}
}

func TestConstrainedScalarDefaultValueGuards(t *testing.T) {
	for _, tc := range []struct{ shape, value, code string }{
		{`"type":"string","minLength":2`, `"a"`, "E_SCHEMA_INVALID"},
		{`"type":"string","maxLength":2`, `"long"`, "E_SCHEMA_INVALID"},
		{`"type":"string","pattern":"^[a-z]+$"`, `"UPPER"`, "E_SCHEMA_INVALID"},
		{`"type":"string","pattern":"^[a-z]+$"`, `"é"`, "E_SCHEMA_UNSUPPORTED"},
		{`"type":"integer","minimum":2`, `1`, "E_SCHEMA_INVALID"},
		{`"type":"integer","maximum":4`, `5`, "E_SCHEMA_INVALID"},
		{`"type":"integer","multipleOf":2`, `3`, "E_SCHEMA_INVALID"},
		{`"type":"number","exclusiveMinimum":1`, `1.0`, "E_SCHEMA_INVALID"},
		{`"type":"number","exclusiveMaximum":2`, `2.0`, "E_SCHEMA_INVALID"},
		{`"type":"string","minLength":1`, `null`, "E_SCHEMA_INVALID"},
	} {
		for _, owner := range []string{"T", "M", "N"} {
			values := map[string]string{"T": "", "M": "", "N": ""}
			values[owner] = `,"default":` + tc.value
			r := request(schemas(`{"T":{`+tc.shape+values["T"]+`},"M":{"$ref":"#/components/schemas/T"`+values["M"]+`},"N":{"$ref":"#/components/schemas/M"`+values["N"]+`}}`), "Dto")
			r.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/"+owner+"/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid constrained annotation lost its owner", tc, owner, err)
			}
		}
	}
}

func TestConstrainedScalarDefaultsKeepSourceAndOptInGates(t *testing.T) {
	for _, tc := range []struct{ shape, value string }{
		{`"type":"string","minLength":-1`, `null`},
		{`"type":"string","minLength":4,"maxLength":2`, `null`},
		{`"type":"string","pattern":"["`, `null`},
		{`"type":"integer","minimum":"bad"`, `null`},
		{`"type":"integer","multipleOf":0`, `null`},
		{`"type":"number","multipleOf":0.5`, `1.0`},
		{`"type":"boolean","minimum":0`, `true`},
		{`"type":"string","format":"date","minLength":1`, `"2024-02-29"`},
		{`"type":"string","x-kotlin-value-class":true,"minLength":1`, `"given"`},
	} {
		r := request(schemas(`{"T":{`+tc.shape+`,"nullable":true,"default":`+tc.value+`},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null}}`), "Dto")
		r.Options.Validation = true
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("null/default granted unsupported source", tc, err)
		}
	}
	for _, shape := range []string{`"type":"string","minLength":1`, `"type":"integer","maximum":10`, `"type":"number","minimum":0`} {
		r := request(schemas(`{"T":{`+shape+`,"default":null,"nullable":true}}`), "Dto")
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("component default bypassed validation opt-in", shape, err)
		}
	}
	for _, shape := range []string{`"type":"string","minLength":1`, `"type":"integer","minimum":0`, `"type":"number","maximum":2`} {
		r := request(schemas(`{"T":{`+shape+`},"M":{"$ref":"#/components/schemas/T","default":null}}`), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/M/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("bounded wrapper null became presence-based", shape, err)
		}
	}
}
