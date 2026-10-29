package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedTemporalCollectionFieldDefaultsOriginalGraduation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		// Verbatim former referenced builtin-collection source-role date negative.
		r := request(referencedFieldDoc(`{"type":"array","items":{"type":"string","format":"date"}}`, "[]", ""), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
		// Verbatim former direct-nested temporal role negative; the consuming field
		// is a direct ref, so only the independent referenced role graduates.
		r.Document = schemas(`{"Rows":{"type":"array","items":{"type":"array","items":{"type":"string","format":"date"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[]}}}}`)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReferencedTemporalCollectionFieldDefaultsValuesAndNullLayers(t *testing.T) {
	for _, format := range []string{"date", "date-time"} {
		valid := []string{`"2024-02-29"`, `"0000-01-01"`}
		invalid := []string{`"2023-02-29"`, `"2024-04-31"`, `"2024-2-29"`}
		if format == "date-time" {
			valid = []string{`"2024-02-29T12:34:56.123456789Z"`, `"2024-02-29T00:00:00-18:00"`, `"2024-02-29T18:00:00+18:00"`}
			invalid = []string{`"2023-02-29T00:00:00Z"`, `"2024-02-29T12:34:60Z"`, `"2024-02-29T24:00:00Z"`, `"2024-02-29T12:34:56.1234567890Z"`, `"2024-02-29T12:34:56,1Z"`, `"2024-02-29T1:34:56Z"`, `"2024-02-29T12:34:56+18:01"`, `"2024-02-29T12:34:56-19:00"`, `"2024-02-29t12:34:56z"`}
		}
		terminal := `{"type":"array","items":{"additionalProperties":{"type":"string","format":"` + format + `","nullable":true}}}`
		for _, checked := range []bool{false, true} {
			for _, value := range valid {
				r := request(referencedFieldDoc(terminal, `[{"a/~":`+value+`}]`, ""), "Dto")
				r.Options.Validation = checked
				if _, err := core.GenerateWithDiagnostics(r); err != nil {
					t.Fatal(value, err)
				}
			}
			for _, value := range append(invalid, `true`, `1`, `{}`, `[]`, `""`) {
				temporalDefaultFailure(t, referencedFieldDoc(terminal, `[{"a/~":`+value+`}]`, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0/a~1~0")
			}
			// Source order wins over canonical map key order.
			temporalDefaultFailure(t, referencedFieldDoc(terminal, `[{"z":false,"a":true}]`, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default/0/z")
			for _, value := range []string{`null`, `[null]`} {
				temporalDefaultFailure(t, referencedFieldDoc(terminal, value, ""), checked, "E_SCHEMA_INVALID", "/components/schemas/AHolder/properties/value/default"+map[string]string{`[null]`: "/0"}[value])
			}
			r := request(referencedFieldDoc(terminal, `[{"a":null}]`, ""), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("leaf null", err)
			}
			r.Document = referencedFieldDoc(terminal, `null`, `,"nullable":true`)
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("nearest true", err)
			}
		}
	}
}

func TestReferencedTemporalCollectionFieldDefaultsExcludedSourceRolesAndNoDrop(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"string","format":"date","enum":["2024-02-29"]}`, `{"type":"string","format":"date-time","const":"2024-02-29T00:00:00Z"}`,
		`{"type":"string","x-native-type":"date"}`, `{"type":"string","x-native-type":"kotlin.time.Instant"}`, `{"type":"string","format":"date","x-native-type":"kotlinx.datetime.LocalDate"}`,
		`{"type":"string","format":"date","x-kotlin-value-class":false}`, `{"type":"string","x-kotlin-value-class":true}`,
		`{"type":"string","format":"date","default":"2024-02-29"}`, `{"oneOf":[{"type":"string","format":"date"},{"type":"null"}]}`,
		`{"type":"string","format":"date","minLength":0}`, `{"type":"string","format":"date-time","minimum":0}`, `{"type":"string","format":"date","pattern":".*"}`,
	} {
		for _, checked := range []bool{false, true} {
			for _, value := range []string{`[]`, `null`} {
				r := request(referencedFieldDoc(`{"type":"array","items":`+leaf+`}`, value, `,"nullable":true`), "Dto")
				r.Options.Validation = checked
				if metadataStringEnumFixture(leaf) {
					metadataStringEnumGraduation(t, r)
					continue
				}
				got, err := core.GenerateWithDiagnostics(r)
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("excluded source", leaf, value, checked, err)
				}
			}
		}
	}
	// Inline object elements with temporal properties now use the separate
	// referenced object collection role.
	for _, body := range []string{
		`{"Rows":{"type":"array","items":{"type":"string","format":"date"},"default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":[]}}}}`,
		`{"Rows":{"type":"array","items":{"type":"string","format":"date"}},"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":[]}}}}`,
		`{"Rows":{"type":"array","items":{"type":"string","format":"date"}},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Rows"}],"default":[]}}}}`,
		`{"Rows":{"type":"array","items":{"type":"string","format":"date"}},"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Rows","default":[false]}}}}}`,
		`{"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Rows"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":[]}}}}`,
		`{"Rows":{"type":"array","items":{"type":"string","format":"date"}},"Choice":{"oneOf":[{"properties":{"tag":{"const":"a"},"v":{"$ref":"#/components/schemas/Rows","default":[]}}},{"properties":{"tag":{"const":"b"}}}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("role", body, err)
			}
		}
	}
	// The original empty collection-valued child is now independently admitted.
	validChild := schemas(`{"Rows":{"type":"array","items":{"type":"string","format":"date"}},"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Rows","default":[]}}}}}`)
	for _, checked := range []bool{false, true} {
		r := request(validChild, "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("admitted temporal collection child", checked, err)
		}
	}
	for _, checked := range []bool{false, true} {
		// Even the no-default resolver rejects direct physical borrowed special leaves.
		for _, own := range []string{"", `,"default":[]`} {
			temporalDefaultFailure(t, schemas(`{"Owner":{"properties":{"rows":{"type":"array","items":{"type":"string","format":"date"}}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Owner/properties/rows"`+own+`}}}}`), checked, "E_REF_UNSUPPORTED", "/components/schemas/Holder/properties/v")
		}
	}
	temporalUniqueGraduation(t, referencedFieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"string","format":"date"}}`, `[]`, ""))
	// Validated false remains exactly effect-free, including default identity.
	r := request(referencedFieldDoc(`{"type":"array","items":{"type":"string","format":"date"}}`, `[]`, ""), "Dto")
	r.Options.Validation = true
	before, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Document = referencedFieldDoc(`{"type":"array","uniqueItems":false,"items":{"type":"string","format":"date"}}`, `[]`, "")
	after, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("explicit false drift", err)
	}
}

func TestReferencedTemporalCollectionFieldDefaultsValidateBeforeSuppression(t *testing.T) {
	for _, role := range []string{"optional", "required", "patch", "discarded"} {
		for _, checked := range []bool{false, true} {
			for _, value := range []string{`["2024-02-29"]`, `["2023-02-29"]`, `["` + strings.Repeat("a", 32765) + `"]`} {
				field := `{"$ref":"#/components/schemas/Rows","default":` + value + `}`
				holder := `{"properties":{"v":` + field + `}}`
				pointer := "/components/schemas/Holder/properties/v/default/0"
				if role == "required" {
					holder = `{"required":["v"],"properties":{"v":` + field + `}}`
				}
				if role == "discarded" {
					holder = `{"allOf":[` + holder + `,{"properties":{"v":{"type":"string"}}}]}`
					pointer = "/components/schemas/Holder/allOf/0/properties/v/default/0"
				}
				doc := schemas(`{"Rows":{"type":"array","nullable":true,"items":{"type":"string","format":"date"}},"Holder":` + holder + `}`)
				if role == "patch" {
					doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Rows":{"type":"array","nullable":true,"items":{"type":"string","format":"date"}}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":` + holder + `}}},"responses":{"204":{}}}}}}`)
					pointer = "/paths/~1p/patch/requestBody/content/application~1json/schema/properties/v/default/0"
				}
				if value != `["2024-02-29"]` {
					code := "E_SCHEMA_INVALID"
					if len(value) > 32768 {
						code = "E_MODEL_LIMIT"
					}
					temporalDefaultFailure(t, doc, checked, code, pointer)
					continue
				}
				r := request(doc, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(role, err)
				}
				name := "HolderDto"
				if role == "patch" {
					name = "PatchPInputDto"
				}
				if role != "optional" {
					for _, file := range got.Artifacts.Files {
						if strings.Contains(file.Content, "default tree") {
							t.Fatal(role, "suppressed initializer leaked", name)
						}
					}
				}
			}
		}
	}
	for _, checked := range []bool{false, true} {
		terminal := `{"type":"array","minItems":1,"items":{"additionalProperties":{"type":"string","format":"date"},"minProperties":1}}`
		temporalDefaultFailure(t, referencedFieldDoc(terminal, `[]`, ""), checked, map[bool]string{true: "E_SCHEMA_INVALID", false: "E_SCHEMA_UNSUPPORTED"}[checked], "")
		temporalDefaultFailure(t, referencedFieldDoc(terminal, `[{}]`, ""), checked, map[bool]string{true: "E_SCHEMA_INVALID", false: "E_SCHEMA_UNSUPPORTED"}[checked], "")
	}
}

func TestReferencedTemporalCollectionFieldDefaultsIndependentBudgets(t *testing.T) {
	for _, count := range []int{45, 46} {
		props := []string{}
		for i := 0; i < count; i++ {
			target := "Rows"
			if i > 0 {
				target = fmt.Sprintf("Hops/properties/h%d", i-1)
			}
			props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
		}
		doc := schemas(fmt.Sprintf(`{"Rows":{"type":"array","items":{"type":"string","format":"date"}},"Hops":{"properties":{%s}},"Holder":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/h%d","default":[]},"cached":{"$ref":"#/components/schemas/Hops/properties/h%d","default":[]}}}}`, strings.Join(props, ","), count-1, count-1))
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if count == 45 && err != nil || count == 46 && (err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("cached source48", count, err)
		}
	}
	for _, n := range []int{16, 17} {
		terminal := `{"type":"string","format":"date"}`
		for i := 0; i < n; i++ {
			terminal = `{"type":"array","minItems":0,"items":` + terminal + `}`
		}
		r := request(referencedFieldDoc(terminal, `[]`, ""), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if n == 16 && err != nil || n == 17 && (err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("checked16", n, err)
		}
	}
	for _, n := range []int{240, 241} {
		props := []string{}
		for i := 0; i < n; i++ {
			props = append(props, fmt.Sprintf(`"v%d":{"$ref":"#/components/schemas/Rows","default":["2024-02-29"]}`, i))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Rows":{"type":"array","items":{"type":"string","format":"date"}},"Holder":{"properties":{`+strings.Join(props, ",")+`}}}`), "Dto"))
		if n == 240 && err != nil || n == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("slots240", n, err)
		}
	}
	// Retain the independent shared node budget beside temporal byte probes;
	// all original nodes are preflighted before initializer suppression.
	for _, n := range []int{16382, 16383} {
		annotation := `[[` + strings.Repeat(`0,`, n-1) + `0]]`
		terminal := `{"type":"array","items":{"type":"array","items":{"type":"integer"}}}`
		r := request(referencedFieldDoc(terminal, annotation, ""), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if n == 16382 && err != nil || n == 16383 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("existing node cap", n, err)
		}
	}
	for _, size := range []int{32751, 32752} {
		annotation := `{"` + strings.Repeat("a", size) + `":"2024-02-29"}`
		r := request(referencedFieldDoc(`{"additionalProperties":{"type":"string","format":"date"}}`, annotation, ""), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if size == 32751 && err != nil || size == 32752 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("temporal map original/canonical bytes", size, err)
		}
	}
}
