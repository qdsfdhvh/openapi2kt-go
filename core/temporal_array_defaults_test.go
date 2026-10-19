package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTemporalArrayDefaultsTypedPresenceAndOriginalAdmission(t *testing.T) {
	// Preserve the exact previously rejected empty-date-array field as an admission.
	original := `{"type":"array","items":{"type":"string","format":"date"},"default":[]}`
	input := schemas(`{"Holder":{"properties":{"empty":` + original + `,"dates":{"type":"array","items":{"type":"string","format":"date","nullable":true},"default":["2024-02-29",null]},"times":{"type":"array","items":{"type":"string","format":"date-time"},"default":["2024-02-29T12:34:56.123456789+02:30"]},"required":{"type":"array","nullable":true,"items":{"type":"string","format":"date"},"default":null}},"required":["required"]}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{
			"public val dates: kotlin.collections.List<kotlinx.datetime.LocalDate?>? =",
			"public val times: kotlin.collections.List<kotlin.time.Instant>? =",
			"public val required: kotlin.collections.List<kotlinx.datetime.LocalDate>?,",
			"`openapi-kt default array LocalDate`",
			"`openapi-kt default array LocalDate nullable`",
			"`openapi-kt default array Instant`",
		} {
			if !strings.Contains(text, marker) {
				t.Fatal("Temporal array default lost typed/presence/helper contract", marker, text)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Temporal default generation drifted", err)
		}
	}
}

func TestTemporalArrayDefaultsKeepInvalidMembersAndUnsupportedRoles(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"type":"string","format":"date"},"default":["2024-02-29","2023-02-29"]}`, "E_SCHEMA_INVALID", "/default/1"},
		{`{"type":"array","items":{"type":"string","format":"date"},"default":["2024-13-01"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","format":"date-time"},"default":["2024-02-29T12:00:00"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","format":"date-time"},"default":[0]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","format":"date"},"default":[null]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"string","format":"date"},"default":null}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","items":{"type":"string","format":"date","minLength":1},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/minLength"},
		{`{"type":"array","items":{"type":"string","format":"date"},"default":[],"uniqueItems":true}`, "E_SCHEMA_UNSUPPORTED", "/uniqueItems"},
		{`{"type":"array","items":{"type":"string","x-native-type":"date"},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/x-native-type"},
		{`{"type":"array","items":{"type":"string","format":"date","enum":["2024-02-29"]},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/enum"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
	for _, input := range []string{
		`{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Day"},"default":[]}}}}`,
		`{"Holder":{"allOf":[{"properties":{"value":{"type":"array","items":{"type":"string","format":"date"},"default":["invalid"]}}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Temporal defaults erased source or expanded excluded role", input, err)
		}
	}
}

func TestTemporalArrayDefaultsRejectInstantSpellingsThatKotlinCannotParse(t *testing.T) {
	for _, value := range []string{
		"2024-02-29T12:34:56.1234567891Z",
		"2024-02-29T12:34:56,123Z",
		"2024-02-29T1:34:56Z",
		"2024-02-29T12:34:56+24:00",
		"2024-02-29T12:34:56+01:60",
		"2024-02-29T12:34:56+18:01",
		"2024-02-29T12:34:56-19:00",
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(`{"type":"array","items":{"type":"string","format":"date-time"},"default":["`+value+`"]}`), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default/0" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Generated default can fail the Kotlin Instant codec", value, err)
			}
		}
	}
	for _, value := range []string{
		"2024-02-29T12:34:56.123456789Z", "2024-02-29T12:34:56.100Z",
		"2024-02-29T12:34:56+00:00", "2024-02-29T12:34:56-00:00",
		"2024-02-29T12:34:56+18:00", "2024-02-29T12:34:56-18:00",
		"2024-02-29T12:34:56+17:59", "0000-01-01T00:00:00Z",
	} {
		if _, err := core.GenerateWithDiagnostics(request(fieldDoc(`{"type":"array","items":{"type":"string","format":"date-time"},"default":["`+value+`"]}`), "Dto")); err != nil {
			t.Fatal("Rejected supported temporal spelling", value, err)
		}
	}
}

func TestTemporalArrayDefaultCountsAndLockIdentity(t *testing.T) {
	field := `{"type":"array","items":{"type":"string","format":"date"},"minItems":1,"maxItems":2,"default":["2024-02-29"]}`
	r := request(fieldDoc(field), "Dto")
	if _, err := core.GenerateWithDiagnostics(r); err == nil {
		t.Fatal("Constrained default accepted without validation")
	}
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err != nil {
		t.Fatal(err)
	}
	r.Document = fieldDoc(strings.ReplaceAll(field, `["2024-02-29"]`, `[]`))
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("Invalid temporal default count was lost", err)
	}
	op := `{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	r.Document = []byte(`{"openapi":"3.1.0","paths":{"/a":` + op + `,"/b":` + op + `}}`)
	r.Options.HoistSharedShapes = true
	r.Options.LockHoistedNames = true
	r.Options.DeduplicateModels = true
	r.Options.MergeIdenticalModels = true
	got, err = core.GenerateWithDiagnostics(r)
	if err != nil || len(got.Artifacts.NameLock) == 0 {
		t.Fatal("Temporal defaults failed owned optimization", err)
	}
	r.PreviousNameLock = got.Artifacts.NameLock
	again, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("Temporal default lock replay drifted", err)
	}
	r.Document = []byte(strings.ReplaceAll(string(r.Document), "2024-02-29", "2024-03-01"))
	changed, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("Changed temporal default reused frozen shape", err)
	}
}
