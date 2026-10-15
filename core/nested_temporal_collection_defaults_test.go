package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedTemporalCollectionDefaultsBaselineAdmission(t *testing.T) {
	// The exact original discovery input executed RED before implementation.
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"days":{"type":"array","items":{"additionalProperties":{"type":"string","format":"date"}},"default":[{"a":"2024-02-29"}]}}}}}}`)
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(checked, err)
		}
		text := source(t, got, "HolderDto")
		if !strings.Contains(text, "kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlinx.datetime.LocalDate>>") || !strings.Contains(text, "`openapi-kt default tree List Map Scalar LocalDate`") {
			t.Fatal("lost actual temporal type or legal helper", text)
		}
	}
}

func TestNestedTemporalCollectionDefaultsVerbatimGraduation(t *testing.T) {
	// Former negative inputs are retained verbatim as positive contracts.
	for _, input := range [][]byte{
		fieldDoc(`{"type":"array","nullable":true,"items":{"additionalProperties":{"type":"string","format":"date"}},"default":null}`),
		schemas(`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"type":"string","format":"date"}},"default":[]}}}}`),
	} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(checked, err)
			}
		}
	}
}

func TestNestedTemporalCollectionDefaultsFiniteTypesPresenceAndDeterminism(t *testing.T) {
	for _, format := range []string{"date", "date-time"} {
		leaf, value, scalar := `{"type":["string","null"],"format":"date"}`, `"2024-02-29"`, "kotlinx.datetime.LocalDate"
		if format == "date-time" {
			leaf, value, scalar = `{"type":"string","nullable":true,"format":"date-time"}`, `"2024-02-29T12:34:56.123456789+18:00"`, "kotlin.time.Instant"
		}
		for _, tc := range []struct{ field, marker string }{
			{`{"type":"array","items":{"type":"array","nullable":true,"items":` + leaf + `},"default":[[` + value + `,null],null,[]]}`, "List<kotlin.collections.List<" + scalar + "?>?>"},
			{`{"type":"array","items":{"type":["object","null"],"additionalProperties":` + leaf + `},"default":[{"a/~":` + value + `,"$ref":null},null,{}]}`, "List<kotlin.collections.Map<kotlin.String, " + scalar + "?>?>"},
			{`{"additionalProperties":{"type":"array","nullable":true,"items":` + leaf + `},"default":{"z":null,"a":[` + value + `,null],"empty":[]}}`, "Map<kotlin.String, kotlin.collections.List<" + scalar + "?>?>"},
			{`{"additionalProperties":{"nullable":true,"additionalProperties":` + leaf + `},"default":{"z":null,"a":{"type":` + value + `,"/":null},"empty":{}}}`, "Map<kotlin.String, kotlin.collections.Map<kotlin.String, " + scalar + "?>?>"},
			{`{"type":"array","items":{"additionalProperties":{"type":"array","items":{"additionalProperties":` + leaf + `}}},"default":[{"a":[{"b":` + value + `}]}]}`, "List<kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.collections.Map<kotlin.String, " + scalar + "?>>>>"},
		} {
			for _, checked := range []bool{false, true} {
				r := request(fieldDoc(tc.field), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || !strings.Contains(source(t, got, "RecordDto"), tc.marker) {
					t.Fatal(format, tc, err)
				}
				text := source(t, got, "RecordDto")
				for _, line := range strings.Split(text, "\n") {
					if strings.Contains(line, "private fun `openapi-kt default tree") && strings.Contains(strings.Split(line, "`(source")[0], ".") {
						t.Fatal("illegal JVM helper name", line)
					}
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("generation drift", err)
				}
			}
		}
	}
	field := `{"type":"array","nullable":true,"items":{"additionalProperties":{"type":"string","format":"date"}},"default":[{"a":"2024-02-29"}]}`
	r := request(schemas(`{"Holder":{"properties":{"value":`+field+`},"required":["value"]}}`), "Dto")
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil || strings.Contains(source(t, got, "HolderDto"), " = `openapi-kt default tree") {
		t.Fatal("required initializer", err)
	}
	r.Document = []byte(`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`)
	got, err = core.GenerateWithDiagnostics(r)
	if err != nil || !strings.Contains(source(t, got, "PatchPatchInputDto"), "= org.example.models.models.Patch.Absent") {
		t.Fatal("Patch presence", err)
	}
}

func TestNestedTemporalCollectionDefaultsOriginalDiagnostics(t *testing.T) {
	for _, tc := range []struct{ leaf, annotation, tail string }{
		{`{"type":"string","format":"date"}`, `[{"z":{"a/~":"2023-02-29"},"a":{}}]`, "/default/0/z/a~1~0"},
		{`{"type":"string","format":"date"}`, `[{"z":{"bad":false},"a":{"bad":"invalid"}}]`, "/default/0/z/bad"},
		{`{"type":"string","format":"date"}`, `[{"a":{"nil":null}}]`, "/default/0/a/nil"},
		{`{"type":"string","format":"date-time"}`, `[{"a":{"v":"2024-02-29T12:34:56.1234567891Z"}}]`, "/default/0/a/v"},
		{`{"type":"string","format":"date-time"}`, `[{"a":{"v":"2024-02-29T12:34:56,123Z"}}]`, "/default/0/a/v"},
		{`{"type":"string","format":"date-time"}`, `[{"a":{"v":"2024-02-29T1:34:56Z"}}]`, "/default/0/a/v"},
		{`{"type":"string","format":"date-time"}`, `[{"a":{"v":"2024-02-29T12:34:56+18:01"}}]`, "/default/0/a/v"},
		{`{"type":"string","format":"date-time"}`, `[{"a":{"v":"2024-02-29T12:34:56-19:00"}}]`, "/default/0/a/v"},
		{`{"type":"string","format":"date-time"}`, `[{"a":{"v":"2024-02-29T12:34:56+01:60"}}]`, "/default/0/a/v"},
	} {
		field := `{"type":"array","items":{"additionalProperties":{"additionalProperties":` + tc.leaf + `}},"default":` + tc.annotation + `}`
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, err)
			}
		}
	}
	for _, value := range []string{"0000-01-01T00:00:00Z", "2024-02-29T12:34:56.123456789Z", "2024-02-29T12:34:56-18:00", "2024-02-29T12:34:56-00:00"} {
		field := nestedDefaultChain(3, `{"type":"string","format":"date-time"}`, `[[["`+value+`"]]]`)
		if _, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto")); err != nil {
			t.Fatal(value, err)
		}
	}
}

func TestNestedTemporalCollectionDefaultsSourceLimits(t *testing.T) {
	for _, edges := range []int{47, 48} {
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(nestedDefaultChain(edges, `{"type":"string","format":"date"}`, `[]`)), "Dto"))
		if edges == 47 && err != nil || edges == 48 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal("source48", edges, err)
		}
	}
	for _, edges := range []int{16, 17} {
		leaf := `{"type":"string","format":"date-time"}`
		for range edges {
			leaf = `{"type":"array","minItems":0,"items":` + leaf + `}`
		}
		r := request(fieldDoc(strings.TrimSuffix(leaf, "}")+`,"default":[]}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 16 && err != nil || edges == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal("checked16", edges, err)
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%d":%s`, i, nestedDefaultChain(2, `{"type":"string","format":"date"}`, `[["2024-02-29"]]`))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("slots240", count, err)
		}
	}
	// Temporal strings stay short; opaque dictionary keys exercise the byte cap.
	for _, size := range []int{32749, 32750} {
		field := `{"type":"array","items":{"additionalProperties":{"type":"string","format":"date"}},"default":[{"` + strings.Repeat("a", size) + `":"2024-02-29"}]}`
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
		if size == 32749 && err != nil || size == 32750 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("bytes32KiB", size, err)
		}
	}
	field := nestedDefaultChain(2, `{"type":"string","nullable":true,"format":"date"}`, `[[`+strings.Repeat("null,", 16382)+`null]]`)
	got, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
	if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("whole annotation limit", err)
	}
}
