package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func specialMapArray(format string) string {
	return `{"type":"array","uniqueItems":true,"items":{"type":"object","nullable":true,"additionalProperties":{"type":["string","null"],"format":"` + format + `"}}}`
}

func TestMapSpecialUniqueSources(t *testing.T) {
	// Preserve the exact previously rejected UUID Map input as a graduation.
	mapUniqueGraduation(t, schemas(`{"Value":{"type":"string"},"Holder":{"properties":{"v":{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"type":"string","format":"uuid"}}}}}}`))
	for _, format := range []string{"uuid", "date", "date-time"} {
		shape := specialMapArray(format)
		mapUniqueGraduation(t, fieldDoc(shape))
		rq := request(fieldDoc(shape), "Dto")
		rq.Options.Validation = true
		first, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, first, "RecordDto"), "`openapi-kt unique map items`(value)") {
			t.Fatal("Map check lost", format)
		}
		second, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("output drift", format, err)
		}
		falseReq := request(fieldDoc(strings.Replace(shape, `"uniqueItems":true`, `"uniqueItems":false`, 1)), "Dto")
		plainReq := request(fieldDoc(strings.Replace(shape, `"uniqueItems":true,`, "", 1)), "Dto")
		falseReq.Options.Validation, plainReq.Options.Validation = true, true
		falseResult, e1 := core.GenerateWithDiagnostics(falseReq)
		plainResult, e2 := core.GenerateWithDiagnostics(plainReq)
		if e1 != nil || e2 != nil || !reflect.DeepEqual(falseResult, plainResult) {
			t.Fatal("false no-op drift", format, e1, e2)
		}
		aliasFailure(t, schemas(`{"Holder":{"properties":{"v":{"type":"string"}},"additionalProperties":`+shape+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/additionalProperties/uniqueItems")
	}
}

func TestMapSpecialUniqueDefaultWireAndConstraints(t *testing.T) {
	// Membership/constraints and the emitted initializer keep the existing
	// mapped UUID policy, independently of equality key normalization.
	shape := `{"type":"array","uniqueItems":true,"items":{"type":"object","minProperties":1,"additionalProperties":{"type":"string","format":"uuid","minLength":36,"pattern":"^[0-9a-f-]+$"}},"default":[{"a":"550E8400E29B41D4A716446655440000"}]}`
	optional := func(s string) []byte {
		return []byte(strings.Replace(string(fieldDoc(s)), `,"required":["value"]`, "", 1))
	}
	rq := request(optional(shape), "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, got, "RecordDto"), `550E8400E29B41D4A716446655440000`) {
		t.Fatal("mapped UUID initializer lost", source(t, got, "RecordDto"))
	}
	for _, tc := range []struct{ old, replacement, suffix string }{
		{`"minLength":36`, `"minLength":37`, "/default/0/a"},
		{`"minProperties":1`, `"minProperties":2`, "/default/0"},
	} {
		rq.Document = optional(strings.Replace(shape, tc.old, tc.replacement, 1))
		failed, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(failed.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.suffix {
			t.Fatal("constraint Pointer lost", tc, err)
		}
	}
}

func TestMapSpecialUniqueDefaultKeysAndPointers(t *testing.T) {
	for _, tc := range []struct{ format, values, suffix string }{
		{"uuid", `[{"a":"550e8400-e29b-41d4-a716-446655440000","b":null},{"b":null,"a":"550E8400E29B41D4A716446655440000"}]`, "/1"},
		{"uuid", `[{"a":"550E8400-E29B-41D4-A716-446655440000"},{"a":"550e8400-e29b-41d4-a716-446655440000"}]`, "/1"},
		{"date", `[{"a":"2024-02-29"},{"a":"2024-02-29"}]`, "/1"},
		{"date-time", `[{"a":"2026-01-02T03:04:05Z","b":null},{"b":null,"a":"2026-01-02T05:04:05+02:00"}]`, "/1"},
		{"date-time", `[{"a":"1969-12-31T23:00:00.123Z"},{"a":"1970-01-01T01:00:00.123000000+02:00"}]`, "/1"},
		{"date-time", `[{"a":"2026-01-02T03:04:05.123000001Z"},{"a":"2026-01-02T03:04:05.123000002Z"}]`, ""},
		{"uuid", `[{},{"a":null},{"b":null},null,{"a":"550e8400-e29b-41d4-a716-446655440000"}]`, ""},
		{"date", `[{"a\"b":"2024-02-29"},{"a":"2024-02-29"}]`, ""},
		{"uuid", `[{"a":"invalid"}]`, "/0/a"},
		{"date", `[{"a":"2023-02-29"}]`, "/0/a"},
		{"date-time", `[{"a":"2026-01-02T03:04:05.1234567890Z"}]`, "/0/a"},
	} {
		shape := specialMapArray(tc.format)
		for _, role := range []string{"direct", "nested", "root", "alias", "field"} {
			body, pointer := "", ""
			annotation := strings.TrimSuffix(shape, "}") + `,"default":` + tc.values + `}`
			switch role {
			case "direct":
				body = `{"Record":{"properties":{"value":` + annotation + `}}}`
				pointer = "/components/schemas/Record/properties/value/default"
			case "nested":
				body = `{"Record":{"properties":{"value":{"type":"array","items":` + shape + `,"default":[` + tc.values + `]}}}}`
				pointer = "/components/schemas/Record/properties/value/default/0"
			case "root":
				body = `{"Rows":` + annotation + `}`
				pointer = "/components/schemas/Rows/default"
			case "alias":
				body = `{"Rows":` + shape + `,"Alias":{"$ref":"#/components/schemas/Rows","default":` + tc.values + `}}`
				pointer = "/components/schemas/Alias/default"
			case "field":
				body = `{"Rows":` + shape + `,"Record":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":` + tc.values + `}}}}`
				pointer = "/components/schemas/Record/properties/value/default"
			}
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if tc.suffix == "" {
				if err != nil {
					t.Fatal(tc, role, err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != pointer+tc.suffix {
				t.Fatal("mapped equality or original value Pointer lost", tc, role, err)
			}
		}
	}
}

func TestMapSpecialUniqueExcludedSources(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"string","format":"uuid","enum":["550e8400-e29b-41d4-a716-446655440000"]}`,
		`{"type":"string","format":"uuid","const":"550e8400-e29b-41d4-a716-446655440000"}`,
		`{"type":"string","x-native-type":"kotlin.time.Instant"}`,
		`{"type":"string","format":"uuid","x-kotlin-value-class":true}`,
		`{"$ref":"#/components/schemas/Id"}`,
		`{"anyOf":[{"type":"string","format":"date"},{"type":"null"}]}`,
	} {
		rq := request(schemas(`{"Id":{"type":"string","format":"uuid"},"Holder":{"properties":{"v":{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":`+leaf+`}}}}}`), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("excluded value source acquired equality", leaf, err)
		}
	}
}

func TestMapSpecialUniqueBudgets(t *testing.T) {
	for _, format := range []string{"uuid", "date", "date-time"} {
		shape := strings.Replace(specialMapArray(format), `"type":"object"`, `"type":"object","minProperties":0`, 1)
		for _, layers := range []int{15, 16} {
			field := shape
			for i := 1; i < layers; i++ {
				field = `{"type":"array","items":` + field + `}`
			}
			rq := request(fieldDoc(field), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if (err == nil) != (layers == 15) || err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
				t.Fatal("checked edge budget lost", format, layers, err)
			}
		}
		for _, width := range []int{240, 241} {
			fields := make([]string, width)
			for i := range fields {
				fields[i] = fmt.Sprintf(`"p%d":%s`, i, shape)
			}
			rq := request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if (err == nil) != (width == 240) || err != nil && (err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
				t.Fatal("slot budget lost", format, width, err)
			}
		}

		for _, hops := range []int{40, 50} {
			entries := []string{`"Rows":` + shape}
			next := "Rows"
			for i := 0; i < hops; i++ {
				key := fmt.Sprintf("Hop%d", i)
				entries = append(entries, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/%s","nullable":false}`, key, next))
				next = key
			}
			entries = append(entries, `"Holder":{"properties":{"first":{"$ref":"#/components/schemas/`+next+`"},"second":{"$ref":"#/components/schemas/`+next+`"}}}`)
			rq := request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if (err == nil) != (hops == 40) || err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
				t.Fatal("source/cache budget lost", format, hops, err)
			}
		}
	}
}
