package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleScalarConstOriginalGuards(t *testing.T) {
	for _, tc := range []struct{ member, code, suffix string }{
		{`{"const":"fixed","default":"fixed"}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"const":"fixed","unknownWire":true}`, "E_SCHEMA_UNSUPPORTED", "/unknownWire"},
		{`{"type":"integer","const":"bad"}`, "E_SCHEMA_INVALID", "/const"},
		{`{"const":"fixed","enum":["fixed"]}`, "E_SCHEMA_UNSUPPORTED", "/enum"},
		{`{"type":"integer","const":1,"minimum":0}`, "E_SCHEMA_UNSUPPORTED", "/minimum"},
	} {
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			failure(t, schemas(`{"Rows":{"type":"array","prefixItems":[`+tc.member+`]`+tail+`,"default":[]}}`), tc.code, "/components/schemas/Rows/prefixItems/0"+tc.suffix)
		}
	}
}

func TestTupleScalarConstSignaturesAndDefaults(t *testing.T) {
	for _, tc := range []struct{ prefix, want string }{
		{`[{"const":"fixed"},{"const":"fixed"}]`, "List<kotlinx.serialization.json.JsonElement>"},
		{`[{"const":"fixed"},{"type":"string"}]`, "List<kotlinx.serialization.json.JsonElement>"},
		{`[{"$ref":"#/components/schemas/Only"},{"$ref":"#/components/schemas/Alias"}]`, "List<kotlin.String>"},
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Only":{"const":"fixed"},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":`+tc.prefix+`}}`), "Dto"))
		if err != nil || !strings.Contains(source(t, got, "RowsDto"), tc.want) {
			t.Fatal(tc, err)
		}
		found := false
		for _, w := range got.Warnings {
			if w.Code == "W_TUPLE_APPROXIMATION" && strings.Contains(w.Message, "const prefix membership") {
				found = true
			}
		}
		if !found {
			t.Fatal("lost const membership warning", got.Warnings)
		}
	}
}

func TestTupleScalarConstDepthAndWidth(t *testing.T) {
	for _, count := range []int{45, 46} {
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			leaf := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Only"}]` + tail + `}`
			for i := 1; i < count; i++ {
				leaf = `{"type":"array","items":` + leaf + `}`
			}
			for _, forward := range []bool{false, true} {
				parts := []string{`"Only":{"const":"fixed"}`, `"Rows":` + leaf, `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`}
				if forward {
					parts[0], parts[2] = parts[2], parts[0]
				}
				_, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
				if count == 45 && err != nil || count == 46 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
					t.Fatal("const erased source hops", count, tail, forward, err)
				}
			}
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%03d":{"type":"array","prefixItems":[{"const":"fixed"}],"default":["fixed"]}`, i)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("const tuple slot budget", width, err)
		}
	}
}

func TestTupleScalarConstOverlappingAliasBindings(t *testing.T) {
	for _, tuple := range []bool{false, true} {
		parts := []string{`"Only":{"const":"fixed"}`}
		next := "Only"
		for i := 0; i < 40; i++ {
			name := fmt.Sprintf("Alias%02d", i)
			parts = append(parts, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/%s"}`, name, next))
			next = name
		}
		for i := 0; i < 400; i++ {
			body := fmt.Sprintf(`"type":"array","items":{"$ref":"#/components/schemas/%s"}`, next)
			if tuple {
				body = fmt.Sprintf(`"type":"array","prefixItems":[{"$ref":"#/components/schemas/%s"}],"items":true`, next)
			}
			parts = append(parts, fmt.Sprintf(`"Rows%03d":{%s}`, i, body))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if !tuple && err != nil {
			t.Fatal("literal budget changed ordinary alias collections", err)
		}
		if tuple && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("overlapping literal catalog snapshots bypassed cumulative work cap", err)
		}
	}
}

func TestTupleScalarConstSources(t *testing.T) {
	for _, tc := range []struct{ member, tail, want string }{
		{`{"const":"fixed"}`, "", "List<kotlin.String>"},
		{`{"const":7}`, "", "List<kotlin.Int>"},
		{`{"const":true}`, "", "List<kotlin.Boolean>"},
		{`{"const":1.5}`, "", "List<kotlin.Double>"},
		{`{"type":"integer","format":"int64","const":7}`, "", "List<kotlin.Long>"},
		{`{"type":"number","format":"float","const":1.5}`, "", "List<kotlin.Float>"},
		{`{"type":"string","format":"uuid","const":"123e4567-e89b-12d3-a456-426614174000"}`, "", "List<kotlin.uuid.Uuid>"},
		{`{"type":"string","format":"date","const":"2026-01-01"}`, "", "List<kotlinx.datetime.LocalDate>"},
		{`{"type":"string","format":"date-time","const":"2026-01-01T00:00:00Z"}`, "", "List<kotlin.time.Instant>"},
		{`{"x-native-type":"bigint","const":"7"}`, "", "List<kotlin.Long>"},
		{`{"const":"fixed","nullable":true}`, "", "List<kotlin.String?>"},
		{`{"const":"fixed"}`, `,"items":{"type":"integer"}`, "List<kotlin.Int>"},
		{`{"const":"fixed"}`, `,"items":true`, "List<kotlinx.serialization.json.JsonElement>"},
		{`{"$ref":"#/components/schemas/Alias"}`, "", "List<kotlin.String>"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Only":{"const":"fixed"},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":[`+tc.member+`]`+tc.tail+`}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(tc, checked, err)
			}
			if !strings.Contains(source(t, got, "RowsDto"), tc.want) {
				t.Fatal(tc, source(t, got, "RowsDto"))
			}
			again, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("const source nondeterminism", e)
			}
		}
	}
}

func TestTupleScalarConstOwnDefaults(t *testing.T) {
	for _, tc := range []struct {
		member, value string
		valid         bool
	}{
		{`{"const":"fixed"}`, `["fixed"]`, true},
		{`{"const":"fixed"}`, `["other"]`, false},
		{`{"const":7}`, `[7]`, true},
		{`{"const":7}`, `[8]`, false},
		{`{"type":"number","format":"float","const":16777216}`, `[16777217]`, false},
		{`{"type":"number","format":"float","const":1.5}`, `[1.5]`, true},
		{`{"x-native-type":"bigint","const":"7"}`, `["7"]`, true},
		{`{"x-native-type":"bigint","const":"7"}`, `[7]`, false},
		{`{"type":"string","format":"date","const":"2026-01-01"}`, `["2026-01-01"]`, true},
		{`{"const":"fixed","nullable":true}`, `[null,"fixed"]`, true},
		{`{"const":"fixed"}`, `[null]`, false},
	} {
		for _, root := range []bool{false, true} {
			tuple := `{"type":"array","prefixItems":[` + tc.member + `],"default":` + tc.value + `}`
			owner := `"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`
			if !root {
				owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+owner+`}`), "Dto"))
			if (err == nil) != tc.valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, root, err)
			}
		}
	}
}

func TestTupleScalarConstReferenceDefaults(t *testing.T) {
	for _, value := range []string{`["fixed"]`, `["other"]`, `[null]`} {
		for _, nullable := range []bool{false, true} {
			for _, root := range []bool{false, true} {
				tuple := fmt.Sprintf(`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Second","nullable":%v}],"default":%s}`, nullable, value)
				owner := `"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`
				if !root {
					owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
				}
				input := schemas(`{"Only":{"const":"fixed"},"First":{"$ref":"#/components/schemas/Only"},"Second":{"$ref":"#/components/schemas/First"},` + owner + `}`)
				got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
				valid := value == `["fixed"]` || nullable && value == `[null]`
				if (err == nil) != valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal(value, nullable, root, err)
				}
			}
		}
	}
}
