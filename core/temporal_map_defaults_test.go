package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTemporalMapDefaultsTypedPresenceAndOriginalAdmission(t *testing.T) {
	// This exact empty date-map field was previously a rejected dictionary role.
	original := `{"type":"object","additionalProperties":{"type":"string","format":"date"},"default":{}}`
	input := schemas(`{"Holder":{"properties":{"empty":` + original + `,"dates":{"additionalProperties":{"type":["string","null"],"format":"date"},"default":{"a/~":"2024-02-29","nil":null}},"times":{"type":"object","additionalProperties":{"type":"string","format":"date-time"},"default":{"stamp":"2024-02-29T12:34:56.123456789+02:30"}},"required":{"type":"object","nullable":true,"additionalProperties":{"type":"string","format":"date"},"default":null}},"required":["required"]}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{
			"public val dates: kotlin.collections.Map<kotlin.String, kotlinx.datetime.LocalDate?>? =",
			"public val times: kotlin.collections.Map<kotlin.String, kotlin.time.Instant>? =",
			"public val required: kotlin.collections.Map<kotlin.String, kotlinx.datetime.LocalDate>?,",
			"`openapi-kt default map LocalDate`",
			"`openapi-kt default map LocalDate nullable`",
			"`openapi-kt default map Instant`",
		} {
			if !strings.Contains(text, marker) {
				t.Fatal("Temporal dictionary lost typed/presence/helper contract", marker, text)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Temporal dictionary generation drifted", err)
		}
	}
}

func TestTemporalMapDefaultsOriginalMemberDiagnosticsAndExcludedRoles(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"object","additionalProperties":{"type":"string","format":"date"},"default":{"a/~":"2023-02-29"}}`, "E_SCHEMA_INVALID", "/default/a~1~0"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date"},"default":{"first":"2024-02-29","second":"2024-13-01"}}`, "E_SCHEMA_INVALID", "/default/second"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date-time"},"default":{"a":0}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date"},"default":{"a":null}}`, "E_SCHEMA_INVALID", "/default/a"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date"},"default":null}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date"},"default":[]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date","minLength":1},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/minLength"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date-time","pattern":".*"},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/pattern"},
		{`{"type":"object","additionalProperties":{"type":"string","x-native-type":"date"},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/x-native-type"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date","enum":["2024-02-29"]},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/enum"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"date","default":"2024-02-29"},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/default"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = validation
			if tc.field == `{"type":"object","additionalProperties":{"type":"string","format":"date","enum":["2024-02-29"]},"default":{}}` {
				metadataStringEnumGraduation(t, r)
				continue
			}
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
	field := `{"type":"object","additionalProperties":{"type":"string","format":"date"},"default":{"day":"2024-02-29"}}`
	for _, input := range []string{
		`{"Holder":{"properties":{"value":{"type":"object","additionalProperties":` + field + `,"default":{}}}}}`,
		`{"Holder":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + field + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":` + strings.Replace(field, "2024-02-29", "invalid", 1) + `}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Temporal default expanded an excluded role or lost an original annotation", input, err)
		}
	}
	input := schemas(`{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":` + strings.Replace(field, "2024-02-29", "2024-03-01", 1) + `}}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "HolderDto"), `2024-03-01`) || strings.Contains(source(t, got, "HolderDto"), `2024-02-29`) {
		t.Fatal("Temporal dictionary lost the ordinary allOf default winner", err)
	}
}

func TestTemporalMapDefaultsKotlinInstantGrammar(t *testing.T) {
	for _, value := range []string{
		"2024-02-29T12:34:56.1234567891Z", "2024-02-29T12:34:56,123Z",
		"2024-02-29T1:34:56Z", "2024-02-29T12:34:56+24:00",
		"2024-02-29T12:34:56+01:60", "2024-02-29T12:34:56+18:01",
		"2024-02-29T12:34:56-19:00", "2024-02-29T12:34:56",
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(`{"type":"object","additionalProperties":{"type":"string","format":"date-time"},"default":{"a/~":"`+value+`"}}`), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default/a~1~0" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Temporal dictionary default cannot be decoded by Kotlin", value, err)
			}
		}
	}
	for _, value := range []string{
		"2024-02-29T12:34:56.123456789Z", "2024-02-29T12:34:56.100Z",
		"2024-02-29T12:34:56+00:00", "2024-02-29T12:34:56-00:00",
		"2024-02-29T12:34:56+18:00", "2024-02-29T12:34:56-18:00",
		"2024-02-29T12:34:56+17:59", "0000-01-01T00:00:00Z",
	} {
		if _, err := core.GenerateWithDiagnostics(request(fieldDoc(`{"type":"object","additionalProperties":{"type":"string","format":"date-time"},"default":{"stamp":"`+value+`"}}`), "Dto")); err != nil {
			t.Fatal("Rejected a supported temporal dictionary spelling", value, err)
		}
	}
}

func TestTemporalMapDefaultCountsAndLockIdentity(t *testing.T) {
	field := `{"type":"object","additionalProperties":{"type":"string","format":"date"},"minProperties":1,"maxProperties":2,"default":{"b":"2024-03-01","a":"2024-02-29"}}`
	r := request(fieldDoc(field), "Dto")
	if _, err := core.GenerateWithDiagnostics(r); err == nil {
		t.Fatal("Constrained default accepted without validation")
	}
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`{}`, `{"a":"2024-02-29","b":"2024-03-01","c":"2024-03-02"}`} {
		r.Document = fieldDoc(strings.ReplaceAll(field, `{"b":"2024-03-01","a":"2024-02-29"}`, value))
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Temporal dictionary default lost count validation", err)
		}
	}
	op := `{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	r.Document = []byte(`{"openapi":"3.1.0","paths":{"/a":` + op + `,"/b":` + op + `}}`)
	r.Options.HoistSharedShapes = true
	r.Options.LockHoistedNames = true
	r.Options.DeduplicateModels = true
	r.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil || len(got.Artifacts.NameLock) == 0 {
		t.Fatal("Temporal dictionary failed owned optimization", err)
	}
	r.PreviousNameLock = got.Artifacts.NameLock
	r.Document = []byte(strings.ReplaceAll(string(r.Document), `{"b":"2024-03-01","a":"2024-02-29"}`, `{"a":"2024-02-29","b":"2024-03-01"}`))
	again, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("Dictionary key order changed the frozen temporal default", err)
	}
	r.Document = []byte(strings.ReplaceAll(string(r.Document), "2024-02-29", "2024-03-02"))
	changed, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("Changed temporal default reused a frozen shape", err)
	}
}
