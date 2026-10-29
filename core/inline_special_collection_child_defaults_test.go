package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// A child annotation validates on its own and must not change the containing
// collection's initializer, generated API, or serialized representation.
func TestInlineSpecialCollectionChildDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, field, valid, invalid, suffix string
		checkedOnly                         bool
	}{
		{
			"UUID list", `{"type":"array","items":{"type":"string","format":"uuid"@},"default":[]}`,
			`"123E4567E89B12D3A456426614174000"`, `"not-a-uuid"`, "/items/default", false,
		},
		{
			"UUID map nullable", `{"type":"object","additionalProperties":{"type":["string","null"],"format":"uuid"@},"default":{}}`,
			`null`, `42`, "/additionalProperties/default", false,
		},
		{
			"date list", `{"type":"array","items":{"type":"string","format":"date"@},"default":[]}`,
			`"2024-02-29"`, `"2023-02-29"`, "/items/default", false,
		},
		{
			"date map", `{"type":"object","additionalProperties":{"type":"string","format":"date"@}}`,
			`"0000-01-01"`, `null`, "/additionalProperties/default", false,
		},
		{
			"instant map", `{"type":"object","additionalProperties":{"type":"string","format":"date-time"@},"default":{}}`,
			`"2024-02-29T12:34:56.123456789+02:30"`, `"2024-02-29T12:34:56.1234567890Z"`, "/additionalProperties/default", false,
		},
		{
			"instant list nullable", `{"type":"array","items":{"type":["string","null"],"format":"date-time"@},"default":[]}`,
			`null`, `"2024-02-29T12:34:56+18:01"`, "/items/default", false,
		},
		{
			"UUID canonical bounds", `{"type":"array","items":{"type":"string","format":"uuid","minLength":36,"maxLength":36,"pattern":"^123e"@},"default":[]}`,
			`"123E4567E89B12D3A456426614174000"`, `"223e4567-e89b-12d3-a456-426614174000"`, "/items/default", true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Holder":{"properties":{"value":` + tc.field + `}}}`
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, checked := range []bool{false, true} {
					if tc.checkedOnly && !checked {
						continue
					}
					rq := request(schemas(strings.Replace(body, "@", "", 1)), "Dto")
					rq.Options.ModelStyle = style
					rq.Options.Validation = checked
					control, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal("unannotated control rejected", style, checked, err)
					}
					rq.Document = schemas(strings.Replace(body, "@", `,"default":`+tc.valid, 1))
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(got, control) {
						t.Fatal("valid independent child annotation changed output", style, checked, err)
					}
					rq.Document = schemas(strings.Replace(body, "@", `,"default":`+tc.invalid, 1))
					got, err = core.GenerateWithDiagnostics(rq)
					ds, ok := err.(core.Diagnostics)
					want := "/components/schemas/Holder/properties/value" + tc.suffix
					if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
						*ds[0].Pointer != want || !reflect.DeepEqual(got, core.GenerationReport{}) {
						t.Fatal("invalid child must fail at its original default", style, checked, err)
					}
				}
			}
		})
	}
}

func TestInlineSpecialCollectionChildAndOwnerDefaultsAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, ownerDefault, annotation string
	}{
		{"omitted collection", `{}`, `"2024-02-29"`},
		{"owner null", `null`, `"2024-02-29"`},
		{"supplied collection", `{"value":["2024-03-01"]}`, `"2024-02-29"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Holder":{"type":["object","null"],"properties":{"value":{"type":"array","items":{"type":"string","format":"date"@}}},"default":` + tc.ownerDefault + `}}`
			for _, checked := range []bool{false, true} {
				rq := request(schemas(strings.Replace(body, "@", "", 1)), "Dto")
				rq.Options.Validation = checked
				rq.Document = schemas(strings.Replace(body, "@", `,"default":`+tc.annotation, 1))
				inlineBuiltinChildGraduation(t, rq, `,"default":`+tc.annotation,
					`,"default":"2023-02-29"`, "/components/schemas/Holder/properties/value/items/default")
			}
		})
	}
}

func TestInlineSpecialCollectionChildInstantGrammar(t *testing.T) {
	body := `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","format":"date-time","default":@}}}}}`
	for _, valid := range []string{
		`"0000-01-01T00:00:00Z"`,
		`"2024-02-29T12:34:56.123456789+18:00"`,
		`"2024-02-29T12:34:56-18:00"`,
	} {
		rq := request(schemas(strings.Replace(body, "@", valid, 1)), "Dto")
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("valid Instant annotation rejected", valid, err)
		}
	}
	for _, invalid := range []string{
		`"2023-02-29T12:34:56Z"`,
		`"2024-02-29T1:34:56Z"`,
		`"2024-02-29T12:34:56,1Z"`,
		`"2024-02-29T12:34:56.1234567890Z"`,
		`"2024-02-29T12:34:56+18:01"`,
		`"2024-02-29T12:34:56"`,
		`true`,
	} {
		for _, checked := range []bool{false, true} {
			carrierDefaultFailure(t, schemas(strings.Replace(body, "@", invalid, 1)), checked,
				"E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/items/default")
		}
	}
}

func TestInlineSpecialCollectionChildDefaultBoundaries(t *testing.T) {
	// The original referenced date child is now admitted independently; retain
	// it unchanged and keep an invalid calendar date as the counterexample.
	referenced := `{"Date":{"type":"string","format":"date"},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Date","default":"2024-02-29"}}}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(referenced), "Dto")
		rq.Options.Validation = checked
		inlineBuiltinChildGraduation(t, rq, `,"default":"2024-02-29"`, `,"default":"2023-02-29"`,
			"/components/schemas/Holder/properties/value/items/default")
	}
	for _, body := range []string{
		`{"Rows":{"type":"array","items":{"type":"string","format":"uuid","default":"123e4567-e89b-12d3-a456-426614174000"}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"type":"string","format":"date","default":"2024-02-29"}}}}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","x-native-type":"date","default":"2024-02-29T00:00:00Z"}}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("an excluded source acquired a child annotation role", checked, body, err)
			}
		}
	}
	for _, format := range []string{"uuid", "date", "date-time"} {
		body := `{"Holder":{"type":["object","null"],"properties":{"a/~":{"type":["array","null"],"items":{"type":"string","format":"` +
			format + `","default":"` + strings.Repeat("x", 32768) + `"},"default":null}},"required":["a/~"],"default":null}}`
		for _, checked := range []bool{false, true} {
			carrierDefaultFailure(t, schemas(body), checked, "E_MODEL_LIMIT",
				"/components/schemas/Holder/properties/a~1~0/items/default")
		}
	}
	// Temporal bounds stay unsupported even when the collection is empty.
	body := `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","format":"date","minLength":2,"default":"2024-02-29"},"default":[]}}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if report, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("temporal bounds escaped their existing rejection", err)
	}
}

func TestInlineSpecialCollectionChildDefaultsFrozenNames(t *testing.T) {
	field := `{"type":"array","items":{"type":"string","format":"date"}}`
	body := `{"A":{"properties":{"value":` + field + `}},"B":{"properties":{"value":` + field + `}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		rq.Options.HoistSharedShapes = true
		rq.Options.LockHoistedNames = true
		rq.Options.DeduplicateModels = true
		rq.Options.MergeIdenticalModels = true
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || len(control.Artifacts.NameLock) == 0 {
			t.Fatal("unannotated locked control rejected", checked, err)
		}
		rq.PreviousNameLock = control.Artifacts.NameLock
		rq.Document = schemas(strings.ReplaceAll(body, `"format":"date"`, `"format":"date","default":"2024-02-29"`))
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("child annotation changed optimized output or frozen names", checked, err)
		}
	}
}
