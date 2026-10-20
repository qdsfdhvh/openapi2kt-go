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
			for _, nullable := range []string{"", `,"nullable":false`} {
				for _, value := range []string{"550e8400-e29b-41d4-a716-446655440000", "550E8400-E29B-41D4-A716-446655440000", "550e8400e29b41d4a716446655440000"} {
					encoded, _ := json.Marshal(value)
					shape := `"type":"string","format":"uuid"` + nullable
					body := `{"T":{` + shape + `,"default":` + string(encoded) + `}}`
					if checked {
						body = strings.TrimSuffix(body, "}") + `,"M":{"$ref":"#/components/schemas/T","default":"550e8400-e29b-41d4-a716-446655440001"},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":"550e8400-e29b-41d4-a716-446655440002"},"Plain":{"$ref":"#/components/schemas/N"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/N"}}}}`
					}
					r := request(schemas(body), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					report, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(err)
					}
					text := source(t, report, "TDto")
					literal, _ := json.Marshal(string(encoded))
					if !strings.Contains(text, "public typealias TDto = kotlin.uuid.Uuid") || !strings.Contains(text, "public fun `TDto default`(): TDto =") || !strings.Contains(text, string(literal)) {
						t.Fatal("Kotlin UUID factory/value changed", text)
					}
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(report, again) {
						t.Fatal("UUID factory generation drift", err)
					}
					r.Document = schemas(strings.Replace(body, `,"default":`+string(encoded), "", 1))
					control, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(report.Warnings, control.Warnings) {
						t.Fatal("UUID default changed source/warnings", err)
					}
					if checked {
						unchecked := request(schemas(`{"T":{`+shape+`,"default":`+string(encoded)+`}}`), "Dto")
						unchecked.Options.ModelStyle = style
						bare, err := core.GenerateWithDiagnostics(unchecked)
						if err != nil || source(t, bare, "TDto") != text {
							t.Fatal("UUID root factory depends on validation option", err)
						}
						for _, name := range []string{"MDto", "NDto", "PlainDto", "HolderDto"} {
							if source(t, report, name) != source(t, control, name) {
								t.Fatal("UUID default inherited/changed another owner", name)
							}
						}
					}
				}
			}
		}
	}
}

func TestUUIDComponentDefaultConstraints(t *testing.T) {
	for _, tc := range []struct{ bounds, value string }{
		{`"minLength":36`, `"550e8400-e29b-41d4-a716-446655440000"`},
		{`"maxLength":36`, `"550e8400e29b41d4a716446655440000"`},
		{`"pattern":"^550e8400-"`, `"550E8400-E29B-41D4-A716-446655440000"`},
	} {
		r := request(schemas(`{"T":{"type":"string","format":"uuid",`+tc.bounds+`,"default":`+tc.value+`},"M":{"$ref":"#/components/schemas/T","default":`+tc.value+`}}`), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("existing String constraint denied UUID factory", tc, err)
		}
	}
}

func TestUUIDComponentDefaultValueGuards(t *testing.T) {
	for _, tc := range []struct{ bounds, value, code string }{
		{"", "null", "E_SCHEMA_INVALID"},
		{"", "1", "E_SCHEMA_INVALID"},
		{"", "true", "E_SCHEMA_INVALID"},
		{"", "[]", "E_SCHEMA_INVALID"},
		{"", "{}", "E_SCHEMA_INVALID"},
		{`,"minLength":37`, `"550e8400-e29b-41d4-a716-446655440000"`, "E_SCHEMA_INVALID"},
		{`,"maxLength":35`, `"550e8400e29b41d4a716446655440000"`, "E_SCHEMA_INVALID"},
		{`,"pattern":"^650e"`, `"550e8400-e29b-41d4-a716-446655440000"`, "E_SCHEMA_INVALID"},
		{`,"pattern":"^[a-z]+$"`, `"é"`, "E_SCHEMA_INVALID"},
	} {
		for _, checked := range []bool{false, true} {
			if !checked && tc.bounds != "" {
				continue
			}
			for _, owner := range []string{"T", "M", "N"} {
				if !checked && owner != "T" {
					continue
				}
				values := map[string]string{"T": "", "M": "", "N": ""}
				values[owner] = `,"default":` + tc.value
				body := `{"T":{"type":"string","format":"uuid"` + tc.bounds + values["T"] + `}}`
				if checked {
					body = strings.TrimSuffix(body, "}") + `,"M":{"$ref":"#/components/schemas/T"` + values["M"] + `},"N":{"$ref":"#/components/schemas/M","nullable":false` + values["N"] + `}}`
				}
				r := request(schemas(body), "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/"+owner+"/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("UUID value escaped original owner check", tc, owner, err)
				}
			}
		}
	}
}

func TestUUIDComponentDefaultsKeepSourceGates(t *testing.T) {
	for _, body := range []string{
		`{"T":{"type":"string","format":"uuid","enum":["text"],"default":"text"}}`,
		`{"T":{"type":"string","format":"uuid","x-native-type":"org.example.Text","default":"text"}}`,
		`{"T":{"type":"string","format":"uuid","x-kotlin-value-class":true,"default":"text"}}`,
		`{"T":{"type":"string","format":"uri","default":"text"}}`,
		`{"T":{"type":"number","format":"uuid","default":1}}`,
	} {
		r := request(schemas(body), "Dto")
		r.Options.Validation = true
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("UUID factory widened another source role", body, err)
		}
	}
	for _, body := range []string{
		`{"T":{"type":"string","format":"uuid","minLength":1,"default":"text"}}`,
		`{"T":{"type":"string","format":"uuid"},"M":{"$ref":"#/components/schemas/T","default":"text"}}`,
	} {
		if report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto")); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("UUID factory bypassed validation opt-in", body, err)
		}
	}
	for _, tc := range []struct{ bounds, keyword string }{
		{`"minLength":-1`, "minLength"},
		{`"minLength":4,"maxLength":2`, "maxLength"},
		{`"pattern":"["`, "pattern"},
	} {
		r := request(schemas(`{"T":{"type":"string","format":"uuid",`+tc.bounds+`,"default":"text"}}`), "Dto")
		r.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/T/"+tc.keyword {
			t.Fatal("UUID source constraint lost keyword Pointer", tc, err)
		}
	}
}

func TestUUIDComponentDefaultPointerAndBudget(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ value, code string }{
			{"1", "E_SCHEMA_INVALID"},
			{`"` + strings.Repeat("a", 32769) + `"`, "E_MODEL_LIMIT"},
		} {
			r := request(schemas(`{"A/B~C":{"type":"string","format":"uuid","default":`+tc.value+`}}`), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/A~1B~0C/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("UUID default escaped Pointer/budget contract", tc.code, err)
			}
		}
	}
}
