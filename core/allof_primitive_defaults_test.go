package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func allOfDefaultsDocument(lower, upper, shape string) []byte {
	members := `{"properties":{"value":` + lower + `}},{"properties":{"value":` + upper + `}}`
	if shape == "reference" {
		members = `{"$ref":"#/components/schemas/Lower"},{"properties":{"value":` + upper + `}}`
	} else if shape == "nested" {
		members = `{"type":"object","allOf":[{"$ref":"#/components/schemas/Lower"}]},{"type":"object","allOf":[{"properties":{"value":` + upper + `}}]}`
	}
	return schemas(`{"Lower":{"properties":{"value":` + lower + `}},"Record":{"allOf":[` + members + `]}}`)
}

func TestAllOfPrimitiveDefaultsKeepWinnerAndOriginalConjunction(t *testing.T) {
	for _, shape := range []string{"flat", "reference", "nested"} {
		for _, tc := range []struct{ lower, upper, initializer, check string }{
			{`{"type":"string","minLength":2,"default":"longer"}`, `{"type":"string","maxLength":3,"default":"abc"}`, `kotlin.String? = "abc"`, "length >= 2"},
			{`{"type":"integer","minimum":2}`, `{"type":"integer","default":2}`, `kotlin.Int? = 2`, "value >= 2"},
			{`{"type":"integer","minimum":2,"multipleOf":2,"default":2}`, `{"type":"integer","maximum":4,"default":4}`, `kotlin.Int? = 4`, "multipleOf"},
			{`{"type":"integer","format":"int64","minimum":-9223372036854775808,"default":-9223372036854775808}`, `{"type":"integer","format":"int64","maximum":-9223372036854775806,"default":-9223372036854775808}`, `kotlin.Long? = (-9223372036854775807L - 1L)`, "value >= (-9223372036854775807L - 1L)"},
			{`{"type":"number","format":"float","exclusiveMinimum":0.300000001,"default":0.4}`, `{"type":"number","format":"float","maximum":2,"default":1}`, `kotlin.Float? = 1.0f`, "exclusiveMinimum"},
			{`{"type":"string","pattern":"^[a-z]+$","default":"ab"}`, `{"type":"string","maxLength":3,"default":"abc"}`, `kotlin.String? = "abc"`, "pattern"},
			{`{"type":["string","null"],"minLength":2,"default":"ab"}`, `{"type":["string","null"],"maxLength":3,"default":null}`, `kotlin.String? = null`, "length >= 2"},
			{`{"type":"string","minLength":2,"default":"ab"}`, `{"type":"string","maxLength":3}`, `kotlin.String? = null`, "length >= 2"},
		} {
			t.Run(shape+tc.initializer, func(t *testing.T) {
				r := request(allOfDefaultsDocument(tc.lower, tc.upper, shape), "Dto")
				r.Options.Validation = true
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				text := source(t, got, "RecordDto")
				if !strings.Contains(text, tc.initializer) || !strings.Contains(text, tc.check) {
					t.Fatal("Winning default/check lost", text)
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("Nondeterministic aggregate", err)
				}
				r.Options.Validation = false
				if _, err := core.Generate(r); err == nil {
					t.Fatal("Validation-off admitted bounded source")
				}
			})
		}
	}
}

func TestAllOfPrimitiveDefaultsRejectWinnerOutsideBorrowedChecks(t *testing.T) {
	for _, tc := range []struct{ lower, upper, code string }{
		{`{"type":"string","minLength":2}`, `{"type":"string","maxLength":3,"default":"a"}`, "E_SCHEMA_INVALID"},
		{`{"type":"string","maxLength":3}`, `{"type":"string","minLength":2,"default":"abcd"}`, "E_SCHEMA_INVALID"},
		{`{"type":"string","pattern":"^[a-z]+$"}`, `{"type":"string","maxLength":3,"default":"12"}`, "E_SCHEMA_INVALID"},
		{`{"type":"integer","minimum":2}`, `{"type":"integer","maximum":4,"default":1}`, "E_SCHEMA_INVALID"},
		{`{"type":"integer","minimum":2,"multipleOf":2}`, `{"type":"integer","maximum":4,"default":3}`, "E_SCHEMA_INVALID"},
		{`{"type":"number","format":"float","exclusiveMinimum":0.300000001}`, `{"type":"number","format":"float","maximum":2,"default":0.300000004}`, "E_SCHEMA_INVALID"},
		{`{"type":"number","exclusiveMinimum":0.300000000000000001}`, `{"type":"number","maximum":2,"default":0.300000000000000004}`, "E_SCHEMA_INVALID"},
		{`{"type":"number","multipleOf":0.25}`, `{"type":"number","maximum":2,"default":1}`, "E_SCHEMA_UNSUPPORTED"},
	} {
		r := request(allOfDefaultsDocument(tc.lower, tc.upper, "reference"), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/allOf/1/properties/value/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
	}
	// A discarded annotation must still satisfy its own original schema.
	r := request(allOfDefaultsDocument(`{"type":"string","minLength":2,"default":"a"}`, `{"type":"string","maxLength":3,"default":"abc"}`, "reference"), "Dto")
	r.Options.Validation = true
	_, err := core.GenerateWithDiagnostics(r)
	if err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Lower/properties/value/default" {
		t.Fatal("Discarded original default bypassed validation", err)
	}
}

func TestAllOfPrimitiveDefaultsKeepRequiredAndNullableRequestPresence(t *testing.T) {
	input := allOfDefaultsDocument(`{"type":"string","minLength":2}`, `{"type":"string","maxLength":3,"default":"abc"}`, "reference")
	input = []byte(strings.Replace(string(input), `"Record":{"allOf":`, `"Record":{"required":["value"],"allOf":`, 1))
	r := request(input, "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil || !strings.Contains(source(t, got, "RecordDto"), "val value: kotlin.String,") {
		t.Fatal("Required default became constructor initializer", err)
	}
	// The existing nested inline-request shape is admitted; outer HTTP root
	// allOf, field references and composition leaves retain their own guards.
	input = httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"properties":{"payload":{"allOf":[{"properties":{"value":{"type":["string","null"],"minLength":2}}},{"properties":{"value":{"type":["string","null"],"maxLength":3,"default":"abc"}}}]}}}}}},"responses":{"204":{}}}`)
	r = request(input, "Dto")
	r.Options.Validation = true
	got, err = core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range got.Artifacts.Files {
		found = found || strings.Contains(f.Content, "value: org.example.models.models.Patch<kotlin.String> = org.example.models.models.Patch.Absent")
	}
	if !found {
		t.Fatal("Request default replaced Patch.Absent", got)
	}
}
