package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func temporalUniqueGraduation(t *testing.T, input []byte) {
	t.Helper()
	rq := request(input, "Dto")
	if _, err := core.GenerateWithDiagnostics(rq); err == nil {
		t.Fatal("temporal uniqueness requires validation")
	}
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("original temporal uniqueness input did not graduate", err)
	}
}

func TestTemporalUniqueTypedSources(t *testing.T) {
	for _, format := range []string{"date", "date-time"} {
		for _, leaf := range []string{`{"type":"string","format":"` + format + `","nullable":true}`, `{"$ref":"#/components/schemas/Time","nullable":true}`} {
			rq := request(schemas(`{"Time":{"type":"string","format":"`+format+`"},"Holder":{"properties":{"values":{"type":"array","items":`+leaf+`,"uniqueItems":true}}}}`), "Dto")
			rq.Options.Validation = true
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("temporal list uniqueness should generate", format, leaf, err)
			}
			if !strings.Contains(source(t, r, "HolderDto"), "`openapi-kt uniqueItems`(value)") {
				t.Fatal("temporal unique check lost", r)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("temporal uniqueness output drift", err)
			}
			rq.Options.Validation = false
			if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
				t.Fatal("validation opt-in lost", r, err)
			}
		}
	}
}

func TestTemporalUniqueDefaultEquality(t *testing.T) {
	for _, tc := range []struct {
		format, values string
		duplicate      bool
	}{
		{"date", `["2024-02-29","2024-02-29"]`, true},
		{"date", `["2024-02-29","2024-03-01"]`, false},
		{"date-time", `["2026-01-02T03:04:05Z","2026-01-02T05:04:05+02:00"]`, true},
		{"date-time", `["2026-01-02T03:04:05.123Z","2026-01-02T03:04:05.123000000Z"]`, true},
		{"date-time", `["2026-01-02T03:04:05.123000001Z","2026-01-02T03:04:05.123000002Z"]`, false},
		{"date-time", `["1969-12-31T23:00:00Z","1970-01-01T01:00:00+02:00"]`, true},
		{"date-time", `[null,null]`, true},
	} {
		for _, nested := range []bool{false, true} {
			field := `{"type":"array","items":{"type":"string","format":"` + tc.format + `","nullable":true},"uniqueItems":true,"default":` + tc.values + `}`
			if nested {
				field = `{"type":"array","items":{"type":"array","items":{"type":"string","format":"` + tc.format + `","nullable":true},"uniqueItems":true},"default":[` + tc.values + `]}`
			}
			rq := request(fieldDoc(field), "Dto")
			rq.Options.Validation = true
			r, err := core.GenerateWithDiagnostics(rq)
			if !tc.duplicate {
				if err != nil {
					t.Fatal(tc, nested, err)
				}
				continue
			}
			pointer := "/components/schemas/Record/properties/value/default/1"
			if nested {
				pointer = "/components/schemas/Record/properties/value/default/0/1"
			}
			if err == nil || len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != pointer {
				t.Fatal("temporal default equality differs from actual codecs", tc, nested, r, err)
			}
		}
	}
}

func TestTemporalUniqueDefaultRolesAndOriginalPointer(t *testing.T) {
	terminal := `{"type":"array","uniqueItems":true,"items":{"type":"string","format":"date-time"}}`
	duplicate := `["2026-01-02T03:04:05Z","2026-01-02T05:04:05+02:00"]`
	for _, tc := range []struct{ body, pointer string }{
		{`{"Value":` + strings.TrimSuffix(terminal, "}") + `,"default":` + duplicate + `}}`, "/components/schemas/Value/default/1"},
		{`{"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","default":` + duplicate + `}}`, "/components/schemas/Alias/default/1"},
		{`{"Value":` + terminal + `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":` + duplicate + `}}}}`, "/components/schemas/Holder/properties/value/default/1"},
	} {
		rq := request(schemas(tc.body), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
			t.Fatal("default role erased actual Instant equality or original value Pointer", tc, r, err)
		}
	}
}

func TestTemporalUniqueKeepsOtherSourceRolesExcluded(t *testing.T) {
	for _, leaf := range []string{`{"type":"string","x-native-type":"date"}`, `{"type":"string","x-native-type":"kotlin.time.Instant"}`, `{"type":"string","format":"date","enum":["2024-02-29"]}`, `{"type":"string","format":"date","const":"2024-02-29"}`, `{"type":"string","format":"byte"}`} {
		rq := request(fieldDoc(`{"type":"array","uniqueItems":true,"items":`+leaf+`}`), "Dto")
		rq.Options.Validation = true
		if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("type spelling acquired temporal equality authority", leaf, r, err)
		}
	}
	for _, edges := range []int{16, 17} {
		field := `{"type":"array","items":{"type":"string","format":"date-time"},"uniqueItems":true}`
		for i := 1; i < edges; i++ {
			field = `{"type":"array","items":` + field + `}`
		}
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (edges == 16) || err != nil && (len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("temporal checked edge budget lost", edges, r, err)
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"type":"array","items":{"type":"string","format":"date"},"uniqueItems":true}`, i)
		}
		rq := request(schemas(`{"Record":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (width == 240) || err != nil && (len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("temporal slot budget lost", width, r, err)
		}
	}
}
