package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUUIDComponentDefaults(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, shape := range []string{`"type":"string"`, `"type":"string","nullable":false`, `"type":"string","nullable":true`, `"type":["string","null"]`} {
				for _, value := range []string{`"123e4567-e89b-12d3-a456-426614174000"`, `"ordinary text\n雪"`, `""`, `null`} {
					if value == "null" && !strings.Contains(shape, `true`) && !strings.Contains(shape, `[`) {
						continue
					}
					body := `{"T":{` + shape + `,"format":"uuid","default":` + value + `},"Holder":{"required":["required"],"properties":{"required":{"$ref":"#/components/schemas/T"},"optional":{"$ref":"#/components/schemas/T"}}}}`
					r := request(schemas(body), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					report, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						details, _ := json.Marshal(err)
						t.Fatal(shape, value, string(details))
					}
					encoded, _ := json.Marshal(value)
					text := source(t, report, "TDto")
					if !strings.Contains(text, "public typealias TDto = kotlin.String") || !strings.Contains(text, "public fun `TDto default`(): TDto =") || !strings.Contains(text, string(encoded)) {
						t.Fatal("UUID factory changed String mapping or annotation", text)
					}
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(report, again) {
						t.Fatal("UUID factory is nondeterministic", err)
					}
					r.Document = schemas(strings.Replace(body, `,"default":`+value, "", 1))
					control, err := core.GenerateWithDiagnostics(r)
					if err != nil || source(t, report, "HolderDto") != source(t, control, "HolderDto") || !reflect.DeepEqual(report.Warnings, control.Warnings) {
						t.Fatal("UUID component annotation changed consumer presence or warnings", err)
					}
				}
			}
		}
	}
}

func TestUUIDComponentDefaultAliasesAndBounds(t *testing.T) {
	body := `{"T":{"type":"string","format":"uuid","minLength":2,"maxLength":6,"pattern":"^[a-z]+$","default":"given"},"M":{"$ref":"#/components/schemas/T","default":"middle"},"N":{"$ref":"#/components/schemas/M","default":"outer"},"Plain":{"$ref":"#/components/schemas/N"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/N"}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		r := request(schemas(body), "Dto")
		r.Options.Validation, r.Options.ModelStyle = true, style
		report, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			details, _ := json.Marshal(err)
			t.Fatal(string(details))
		}
		for _, name := range []string{"TDto", "MDto", "NDto"} {
			if !strings.Contains(source(t, report, name), "public fun `"+name+" default`(): "+name+" =") {
				t.Fatal("alias lost its independent default", name)
			}
		}
		r.Document = schemas(strings.Replace(body, `,"default":"given"`, "", 1))
		control, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"MDto", "NDto", "PlainDto", "HolderDto"} {
			if source(t, report, name) != source(t, control, name) {
				t.Fatal("UUID target default changed another declaration", name)
			}
		}
	}
}

func TestUUIDComponentDefaultGuards(t *testing.T) {
	for _, tc := range []struct{ extra, value, code, tail string }{
		{"", `1`, "E_SCHEMA_INVALID", "default"},
		{"", `true`, "E_SCHEMA_INVALID", "default"},
		{"", `[]`, "E_SCHEMA_INVALID", "default"},
		{"", `{}`, "E_SCHEMA_INVALID", "default"},
		{"", `null`, "E_SCHEMA_INVALID", "default"},
		{`,"minLength":2`, `"x"`, "E_SCHEMA_INVALID", "default"},
		{`,"maxLength":2`, `"long"`, "E_SCHEMA_INVALID", "default"},
		{`,"pattern":"^[a-z]+$"`, `"UPPER"`, "E_SCHEMA_INVALID", "default"},
		{`,"minLength":-1`, `"xx"`, "E_SCHEMA_INVALID", "minLength"},
		{`,"pattern":"["`, `"xx"`, "E_SCHEMA_INVALID", "pattern"},
		{"", `"` + strings.Repeat("a", 32768) + `"`, "E_MODEL_LIMIT", "default"},
	} {
		r := request(schemas(`{"T":{"type":"string","format":"uuid"`+tc.extra+`,"default":`+tc.value+`}}`), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/T/"+tc.tail || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("UUID default bypassed value/source/budget checks", tc.extra, tc.code, tc.tail, err)
		}
	}
	for _, owner := range []string{"M", "N"} {
		for _, value := range []string{`null`, `"x"`, `3`} {
			values := map[string]string{"M": "", "N": ""}
			values[owner] = `,"default":` + value
			r := request(schemas(`{"T":{"type":"string","format":"uuid","minLength":2},"M":{"$ref":"#/components/schemas/T"`+values["M"]+`},"N":{"$ref":"#/components/schemas/M"`+values["N"]+`}}`), "Dto")
			r.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/"+owner+"/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("UUID alias default lost owner or constraints", owner, value, err)
			}
		}
	}
	for _, body := range []string{
		`{"T":{"type":"string","format":"uuid","minLength":1,"default":"x"}}`,
		`{"T":{"type":"string","format":"uuid"},"M":{"$ref":"#/components/schemas/T","default":"x"}}`,
	} {
		if report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto")); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("UUID factory bypassed existing validation opt-in", err)
		}
	}
	for _, shape := range []string{
		`"type":"string","format":"uuid","enum":["a"]`,
		`"type":"string","format":"uuid","x-kotlin-value-class":true`,
		`"type":"string","format":"uuid","x-native-type":"org.example.Id"`,
	} {
		r := request(schemas(`{"T":{`+shape+`,"default":"a"}}`), "Dto")
		r.Options.Validation = true
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("UUID factory widened an excluded source role", shape, err)
		}
	}
}
