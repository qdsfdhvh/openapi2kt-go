package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func temporalDefaultFailure(t *testing.T, input []byte, checked bool, code, pointer string) {
	t.Helper()
	r := request(input, "Dto")
	r.Options.Validation = checked
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("expected no artifacts/report", err, got)
	}
	d := err.(core.Diagnostics)[0]
	if d.Code != code || pointer != "" && (d.Pointer == nil || *d.Pointer != pointer) {
		t.Fatal(code, pointer, d)
	}
}

func TestReferencedTemporalScalarFieldDefaultsOriginalNegativeGraduation(t *testing.T) {
	// Exact two formerly rejected scalar-default source-role inputs, unchanged.
	for _, terminal := range []string{`{"type":"string","format":"date"}`, `{"type":"string","format":"date-time"}`} {
		for _, checked := range []bool{false, true} {
			r := request(scalarRefDoc(terminal, "null", `,"nullable":true`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "AHolderDto"), "= null") {
				t.Fatal(terminal, checked, err)
			}
		}
	}
}

func TestReferencedTemporalScalarFieldDefaultsGrammarAndNullability(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, format := range []string{"date", "date-time"} {
			terminal := `{"type":"string","format":"` + format + `","nullable":true}`
			valid := []string{`"2024-02-29"`, `"0000-01-01"`}
			invalid := []string{`"2023-02-29"`, `"2024-04-31"`, `"2024-2-29"`, `"2024-02-29T00:00:00Z"`}
			if format == "date-time" {
				valid = []string{`"2024-02-29T12:34:56.123456789Z"`, `"2024-02-29T00:00:00-18:00"`, `"2024-02-29T18:00:00+18:00"`, `"2024-02-29T12:34:56+02:30"`}
				invalid = []string{`"2023-02-29T00:00:00Z"`, `"2024-02-29T24:00:00Z"`, `"2024-02-29T12:34:60Z"`, `"2024-02-29T12:34:56.1234567890Z"`, `"2024-02-29T12:34:56,1Z"`, `"2024-02-29T12:34:56+18:01"`, `"2024-02-29T12:34:56-19:00"`, `"2024-02-29T12:34:56+00:60"`, `"2024-02-29t12:34:56z"`, `"2024-02-29"`}
			}
			for _, value := range valid {
				r := request(scalarRefDoc(terminal, value, ""), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || !strings.Contains(source(t, got, "AHolderDto"), value) {
					t.Fatal(value, err)
				}
			}
			for _, value := range append(invalid, `true`, `1`, `{}`, `[]`, `""`, `null`) {
				temporalDefaultFailure(t, scalarRefDoc(terminal, value, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/a~1~0/default")
			}
			r := request(scalarRefDoc(terminal, "null", `,"nullable":true`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("consumer true overrides internal false", err)
			}
		}
	}
}

func TestReferencedTemporalScalarFieldDefaultsSourceRolesRemainExcluded(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, terminal := range []string{
			`{"type":"string","format":"date","enum":["2024-02-29"]}`,
			`{"type":"string","format":"date-time","const":"2024-02-29T00:00:00Z"}`,
			`{"x-native-type":"date"}`, `{"type":"string","x-native-type":"date"}`,
			`{"type":"string","format":"date-time","x-native-type":"kotlin.time.Instant"}`,
			`{"type":"string","x-native-type":"kotlinx.datetime.LocalDate"}`,
			`{"type":"string","format":"date","x-kotlin-value-class":false}`,
			`{"type":"string","format":"date","default":"2024-02-29"}`,
			`{"oneOf":[{"type":"string","format":"date"},{"type":"null"}]}`,
		} {
			r := request(scalarRefDoc(terminal, "null", `,"nullable":true`), "Dto")
			r.Options.Validation = checked
			if metadataStringEnumFixture(terminal) {
				metadataStringEnumGraduation(t, r)
				continue
			}
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("source role", terminal, err)
			}
		}
		for _, value := range []string{"null", "[]", "{}"} {
			// These exact former collection-role inputs graduate independently of
			// the scalar role. Wrong JSON kind remains a value error at the field.
			doc := schemas(`{"Stamp":{"type":"string","format":"date-time"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Stamp"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","nullable":true,"default":` + value + `}}}}`)
			if value == "{}" {
				temporalDefaultFailure(t, doc, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
				continue
			}
			r := request(doc, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("collection own-default role graduation", value, err)
			}
		}
		for _, holder := range []string{
			`{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Day"}],"default":"2024-02-29"}}}`,
			`{"properties":{"v":{"oneOf":[{"$ref":"#/components/schemas/Day"},{"type":"null"}],"default":null}}}`,
		} {
			r := request(schemas(`{"Day":{"type":"string","format":"date"},"Holder":`+holder+`}`), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(holder, err)
			}
		}
		// Direct physical internal temporal leaves do not acquire ref permission.
		temporalDefaultFailure(t, schemas(`{"Owner":{"properties":{"day":{"type":"string","format":"date"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Owner/properties/day","default":"2024-02-29"}}}}`), checked, "E_REF_UNSUPPORTED", "/components/schemas/Holder/properties/v")
		for _, annotation := range []string{"null", `""`, "[]", "{}"} {
			for _, bound := range []string{`"minLength":0`, `"maximum":1`, `"pattern":".*"`} {
				temporalDefaultFailure(t, scalarRefDoc(`{"type":"string","format":"date-time","nullable":true,`+bound+`}`, annotation, `,"nullable":true`), checked, "E_SCHEMA_UNSUPPORTED", "")
			}
		}
	}
}

func TestReferencedTemporalScalarFieldDefaultsPresenceDiscardedAndBudgets(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, role := range []string{"optional", "required", "patch", "discarded"} {
			for _, value := range []string{`"2024-02-29"`, `"2023-02-29"`, `null`, `"` + strings.Repeat("a", 32767) + `"`} {
				field := `{"$ref":"#/components/schemas/Day","default":` + value + `}`
				holder := `{"properties":{"v":` + field + `}}`
				if role == "required" {
					holder = `{"required":["v"],"properties":{"v":` + field + `}}`
				}
				if role == "discarded" {
					holder = `{"allOf":[` + holder + `,{"properties":{"v":{"type":"string"}}}]}`
				}
				doc := schemas(`{"Day":{"type":"string","format":"date","nullable":true},"Holder":` + holder + `}`)
				if role == "patch" {
					doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Day":{"type":"string","format":"date","nullable":true}}},"paths":{"/p":{"patch":{"operationId":"patch","requestBody":{"content":{"application/json":{"schema":` + holder + `}}},"responses":{"204":{}}}}}}`)
				}
				r := request(doc, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if value == `"2023-02-29"` || len(value) > 32768 {
					code := "E_SCHEMA_INVALID"
					if len(value) > 32768 {
						code = "E_MODEL_LIMIT"
					}
					if err == nil || err.(core.Diagnostics)[0].Code != code || !reflect.DeepEqual(got, core.GenerationReport{}) {
						t.Fatal(role, code, err)
					}
				} else if err != nil {
					t.Fatal(role, value, err)
				} else {
					name := "HolderDto"
					if role == "patch" {
						name = "PatchInputDto"
					}
					text := source(t, got, name)
					if (role == "required" || role == "discarded" || role == "patch") && strings.Contains(text, "default date") {
						t.Fatal(role, text)
					}
					if role == "patch" && !strings.Contains(text, "Patch.Absent") {
						t.Fatal(text)
					}
				}
			}
		}
		for _, count := range []int{46, 47} {
			hops := []string{}
			for i := 0; i < count; i++ {
				target := "Stamp"
				if i > 0 {
					target = fmt.Sprintf("Hops/properties/h%d", i-1)
				}
				hops = append(hops, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
			}
			for _, own := range []bool{false, true} {
				annotation := ""
				if own {
					annotation = `,"default":"2024-02-29T00:00:00Z"`
				}
				doc := schemas(fmt.Sprintf(`{"Stamp":{"type":"string","format":"date-time","nullable":true},"Hops":{"properties":{%s}},"Holder":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/h%d"%s},"cached":{"$ref":"#/components/schemas/Hops/properties/h%d"%s}}}}`, strings.Join(hops, ","), count-1, annotation, count-1, annotation))
				r := request(doc, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if count == 46 && err != nil || count == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) > 0) {
					t.Fatal(count, own, err)
				}
			}
		}
		for _, count := range []int{240, 241} {
			fields, required := []string{}, []string{}
			for i := 0; i < count; i++ {
				fields = append(fields, fmt.Sprintf(`"v%d":{"$ref":"#/components/schemas/Day","default":"2024-02-29"}`, i))
				required = append(required, fmt.Sprintf(`"v%d"`, i))
			}
			r := request(schemas(`{"Day":{"type":"string","format":"date"},"Holder":{"required":[`+strings.Join(required, ",")+`],"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) > 0) {
				t.Fatal(count, err)
			}
		}
	}
}
