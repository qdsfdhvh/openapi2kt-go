package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestAllOfNumericOuterDefaultWithRetainedOriginalBounds(t *testing.T) {
	for _, tc := range []struct{ kind, format string }{
		{"integer", "int32"}, {"integer", "int64"}, {"number", "float"}, {"number", "double"},
	} {
		shape := `"allOf":[{"properties":{"value":{"type":"` + tc.kind + `","format":"` + tc.format + `","minimum":-10,"default":-5}}},{"properties":{"value":{"type":"` + tc.kind + `","format":"` + tc.format + `","maximum":10,"default":5}}}]`
		for _, own := range []bool{false, true} {
			annotation := ""
			if own {
				annotation = `,"default":{"value":0}`
			}
			rq := request(schemas(`{"Factory":{`+shape+annotation+`}}`), "Dto")
			rq.Options.Validation = true
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("accepted numeric conjunction lost outer default role", tc, own, err)
			}
		}
	}
}

func TestAllOfNumericOuterDefaultGuards(t *testing.T) {
	for _, tc := range []struct{ name, first, last, outer, want string }{
		{"earlier minimum", `"minimum":2`, `"maximum":10`, `{"value":1}`, "/default/value"},
		{"winning maximum", `"minimum":2`, `"maximum":10`, `{"value":11}`, "/default/value"},
		{"strict minimum", `"exclusiveMinimum":2`, `"maximum":10`, `{"value":2}`, "/default/value"},
		{"strict maximum", `"minimum":2`, `"exclusiveMaximum":10`, `{"value":10}`, "/default/value"},
		{"earlier factor", `"minimum":2,"multipleOf":3`, `"maximum":10`, `{"value":4}`, "/default/value"},
		{"invalid original own before override", `"minimum":2,"default":1`, `"maximum":10,"default":5`, `{"value":6}`, "/allOf/0/properties/value/default"},
		{"winner must satisfy earlier bound", `"minimum":2`, `"maximum":10,"default":1`, `{}`, "/allOf/1/properties/value/default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shape := `"allOf":[{"properties":{"value":{"type":"integer",` + tc.first + `}}},{"properties":{"value":{"type":"integer",` + tc.last + `}}}],"default":` + tc.outer
			rq := request(schemas(`{"Factory":{`+shape+`}}`), "Dto")
			rq.Options.Validation = true
			_, err := core.GenerateWithDiagnostics(rq)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Factory"+tc.want {
				t.Fatal("numeric default lost original constraints or Pointer", err)
			}
		})
	}
	for _, outer := range []string{`{}`, `{"value":6}`} {
		shape := `"allOf":[{"properties":{"value":{"type":"integer","minimum":2,"default":3}}},{"properties":{"value":{"type":"integer","maximum":10,"default":5}}}],"default":` + outer
		rq := request(schemas(`{"Factory":{`+shape+`}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("valid numeric outer/default omission rejected", err)
		}
		rq.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(rq); err == nil {
			t.Fatal("unchecked repeated numeric source gained role")
		}
	}
}

func TestAllOfNumericOuterDefaultExactLongAndMappedFloating(t *testing.T) {
	for _, tc := range []struct{ kind, format, minimum, maximum, valid, invalid string }{
		{"integer", "int64", "9007199254740993", "9007199254740995", "9007199254740993", "9007199254740992"},
		{"integer", "int64", "-9223372036854775808", "9223372036854775807", "-9223372036854775808", "9223372036854775808"},
		{"number", "float", "16777217", "16777220", "16777216", "16777214"},
		{"number", "double", "9007199254740993", "9007199254741000", "9007199254740992", "9007199254740990"},
	} {
		shape := `"allOf":[{"properties":{"value":{"type":"` + tc.kind + `","format":"` + tc.format + `","minimum":` + tc.minimum + `}}},{"properties":{"value":{"type":"` + tc.kind + `","format":"` + tc.format + `","maximum":` + tc.maximum + `}}}],"default":{"value":NUMBER}`
		for _, number := range []string{tc.valid, tc.invalid} {
			rq := request(schemas(`{"Factory":{`+strings.Replace(shape, "NUMBER", number, 1)+`}}`), "Dto")
			rq.Options.Validation = true
			_, err := core.GenerateWithDiagnostics(rq)
			if (err == nil) != (number == tc.valid) {
				t.Fatal(tc, number, err)
			}
			if err != nil && *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Factory/default/value" {
				t.Fatal("outer literal lost Pointer", err)
			}
		}
	}
}
