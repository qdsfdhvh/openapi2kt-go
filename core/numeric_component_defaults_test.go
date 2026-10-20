package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNumericComponentDefaults(t *testing.T) {
	for _, tc := range []struct{ kind, format, value, scalar string }{
		{"integer", "int32", "-2147483648", "kotlin.Int"},
		{"integer", "int32", "2147483647", "kotlin.Int"},
		{"integer", "int64", "-9223372036854775808", "kotlin.Long"},
		{"integer", "int64", "9223372036854775807", "kotlin.Long"},
		{"number", "float", "0.300000004", "kotlin.Float"},
		{"number", "float", "-0.0", "kotlin.Float"},
		{"number", "double", "1.7976931348623157e308", "kotlin.Double"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, nullable := range []string{"", `,"nullable":true`, `,"nullable":false`, "3.1"} {
					kind := `"` + tc.kind + `"`
					intent := nullable
					if nullable == "3.1" {
						kind, intent = `["`+tc.kind+`","null"]`, ""
					}
					for _, value := range []string{tc.value, "null"} {
						if value == "null" && nullable != "3.1" && nullable != `,"nullable":true` {
							continue
						}
						t.Run(tc.format+"/"+value+"/"+nullable+"/"+string(style), func(t *testing.T) {
							shape := `"type":` + kind + `,"format":"` + tc.format + `"` + intent
							body := `{"T":{` + shape + `,"default":` + value + `}}`
							if checked {
								body = strings.TrimSuffix(body, "}") + `,"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":` + tc.value + `},"UnchangedIntent":{"$ref":"#/components/schemas/T","default":` + value + `},"Plain":{"$ref":"#/components/schemas/N"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/N"}}}}`
							}
							r := request(schemas(body), "Dto")
							r.Options.Validation, r.Options.ModelStyle = checked, style
							report, err := core.GenerateWithDiagnostics(r)
							if err != nil {
								t.Fatal(err)
							}
							text := source(t, report, "TDto")
							encoded, _ := json.Marshal(value)
							if !strings.Contains(text, "public typealias TDto = "+tc.scalar) || !strings.Contains(text, "public fun `TDto default`(): TDto =") || !strings.Contains(text, string(encoded)) {
								t.Fatal("mapped scalar/original token lost", text)
							}
							again, err := core.GenerateWithDiagnostics(r)
							if err != nil || !reflect.DeepEqual(report, again) {
								t.Fatal("numeric default generation drift", err)
							}
							r.Document = schemas(strings.Replace(body, `,"default":`+value, "", 1))
							control, err := core.GenerateWithDiagnostics(r)
							if err != nil || !reflect.DeepEqual(report.Warnings, control.Warnings) {
								t.Fatal("numeric default changed source admission/warnings", err)
							}
							if checked {
								unchecked := request(schemas(`{"T":{`+shape+`,"default":`+value+`}}`), "Dto")
								unchecked.Options.ModelStyle = style
								uncheckedReport, err := core.GenerateWithDiagnostics(unchecked)
								if err != nil || source(t, uncheckedReport, "TDto") != text {
									t.Fatal("validation option changed unconstrained numeric factory", err)
								}
								for _, name := range []string{"MDto", "NDto", "UnchangedIntentDto", "PlainDto", "HolderDto"} {
									if source(t, report, name) != source(t, control, name) {
										t.Fatal("default inherited/changed another owner", name)
									}
								}
							}
						})
					}
				}
			}
		}
	}
}

func TestNumericComponentDefaultValueGuards(t *testing.T) {
	for _, tc := range []struct{ kind, format, extra, value string }{
		{"integer", "int32", "", "2147483648"},
		{"integer", "int32", "", "-2147483649"},
		{"integer", "int64", "", "9223372036854775808"},
		{"integer", "int64", "", "-9223372036854775809"},
		{"integer", "int64", "", "1.0"},
		{"integer", "int32", "", "1e0"},
		{"integer", "int32", "", "-0"},
		{"number", "float", "", "3.5e38"},
		{"number", "float", "", "-3.5e38"},
		{"number", "float", "", `"1.5"`},
		{"number", "double", "", "true"},
		{"integer", "int32", `,"minimum":0`, "-1"},
		{"integer", "int64", `,"multipleOf":2`, "3"},
		{"number", "float", `,"exclusiveMinimum":0.300000001`, "0.3"},
		{"number", "double", `,"maximum":1`, "1.5"},
	} {
		for _, owner := range []string{"T", "M", "N"} {
			for _, nullable := range []string{"", `,"nullable":true`} {
				values := map[string]string{"T": "", "M": "", "N": ""}
				values[owner] = `,"default":` + tc.value
				r := request(schemas(`{"T":{"type":"`+tc.kind+`","format":"`+tc.format+`"`+tc.extra+nullable+values["T"]+`},"M":{"$ref":"#/components/schemas/T"`+values["M"]+`},"N":{"$ref":"#/components/schemas/M"`+values["N"]+`}}`), "Dto")
				r.Options.Validation = true
				report, err := core.GenerateWithDiagnostics(r)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/"+owner+"/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("mapped numeric value escaped owner check", tc, owner, err)
				}
			}
		}
	}
}

func TestNumericComponentDefaultDoubleOverflowAtInput(t *testing.T) {
	r := request(schemas(`{"T":{"type":"number","format":"double","default":1e309}}`), "Dto")
	report, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_JSON_INVALID" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("non-finite double bypassed input boundary", err)
	}
}

func TestNumericComponentDefaultUncheckedValueGuards(t *testing.T) {
	for _, tc := range []struct{ kind, format, value string }{
		{"integer", "int32", "2147483648"},
		{"integer", "int64", "9223372036854775808"},
		{"integer", "int64", "1.0"},
		{"integer", "int32", "1e0"},
		{"integer", "int32", "-0"},
		{"number", "float", "3.5e38"},
		{"number", "double", "true"},
	} {
		for _, nullable := range []string{"", `,"nullable":true`} {
			r := request(schemas(`{"T":{"type":"`+tc.kind+`","format":"`+tc.format+`"`+nullable+`,"default":`+tc.value+`}}`), "Dto")
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/T/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("unchecked numeric default bypassed mapped type", tc, err)
			}
		}
	}
}

func TestNumericComponentNullDefaultsValidateSource(t *testing.T) {
	for _, tc := range []struct{ shape, keyword string }{
		{`"type":"integer","format":"int64","multipleOf":0`, "multipleOf"},
		{`"type":"number","format":"float","minimum":"bad"`, "minimum"},
		{`"type":"number","format":"double","maximum":true`, "maximum"},
	} {
		r := request(schemas(`{"T":{`+tc.shape+`,"nullable":true,"default":null}}`), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/T/"+tc.keyword || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("null numeric default hid malformed source", tc, err)
		}
	}
}

func TestNumericComponentDefaultsKeepRoleGates(t *testing.T) {
	for _, shape := range []string{
		`"type":"integer","format":"uint64"`,
		`"type":"number","format":"int32"`,
		`"type":"string","format":"uri"`,
		`"type":"integer","format":3`,
		`"type":"integer","format":"int64","x-kotlin-value-class":true`,
		`"type":"integer","format":"int64","x-native-type":"bigint"`,
		`"type":"number","format":"float","multipleOf":0.5`,
	} {
		r := request(schemas(`{"T":{`+shape+`,"nullable":true,"default":null}}`), "Dto")
		r.Options.Validation = true
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("numeric factory widened another role", shape, err)
		}
	}
	for _, shape := range []string{`"type":"integer","format":"int64"`, `"type":"number","format":"float"`} {
		for _, body := range []string{
			`{"T":{` + shape + `},"M":{"$ref":"#/components/schemas/T","default":1}}`,
			`{"T":{` + shape + `,"minimum":0,"default":1}}`,
			`{"T":{` + shape + `,"default":null}}`,
		} {
			if report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto")); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("numeric factory bypassed nullability/validation opt-in", body, err)
			}
		}
	}
}
