package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

const repeatedStringDefaultShape = `"allOf":[{"properties":{"value":{"type":"string","minLength":2,"default":"longer"}}},{"properties":{"value":{"type":"string","maxLength":3,"default":"ab"}}}]`

func TestAllOfRepeatedStringOuterDefault(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, rootDefault := range []bool{false, true} {
			annotation := ""
			if rootDefault {
				annotation = `,"default":{"value":"abc"}`
			}
			rq := request(schemas(`{"Record":{`+repeatedStringDefaultShape+annotation+`}}`), "Dto")
			rq.Options.Validation = true
			rq.Options.ModelStyle = style
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("retained repeated String checks lost outer-default role", rootDefault, err)
			}
		}
	}
}

func TestAllOfRepeatedStringOuterDefaultGuards(t *testing.T) {
	for _, literal := range []string{`{"value":"a"}`, `{"value":"abcd"}`} {
		rq := request(schemas(`{"Record":{`+repeatedStringDefaultShape+`,"default":`+literal+`}}`), "Dto")
		rq.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(rq)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/default/value" {
			t.Fatal("outer default bypassed original conjunction", err)
		}
	}
	// The winning own annotation still satisfies every retained check.
	input := strings.Replace(repeatedStringDefaultShape, `"default":"ab"`, `"default":"a"`, 1)
	rq := request(schemas(`{"Record":{`+input+`,"default":{}}}`), "Dto")
	rq.Options.Validation = true
	_, err := core.GenerateWithDiagnostics(rq)
	if err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/allOf/1/properties/value/default" {
		t.Fatal("winning own default lost original Pointer", err)
	}
	// The exact former numeric refusal is now a positive numeric-default role.
	rq = request(schemas(`{"Record":{"allOf":[{"properties":{"value":{"type":"integer","minimum":2}}},{"properties":{"value":{"type":"integer","maximum":3}}}],"default":{}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("numeric conjunction lost empty outer-default role", err)
	}
	// Enum and unchecked duplicate groups keep their separate roles.
	for _, fields := range []string{
		`[{"properties":{"value":{"type":"string","enum":["ab","abc"]}}},{"properties":{"value":{"type":"string","enum":["ab"]}}}]`,
		`[{"properties":{"value":{"type":"string"}}},{"properties":{"value":{"type":"string"}}}]`,
	} {
		rq := request(schemas(`{"Record":{"allOf":`+fields+`,"default":{}}}`), "Dto")
		rq.Options.Validation = true
		if fields == `[{"properties":{"value":{"type":"string"}}},{"properties":{"value":{"type":"string"}}}]` {
			allOfDefaultOverridePositive(t, rq)
			continue
		}
		if _, err := core.GenerateWithDiagnostics(rq); err == nil {
			t.Fatal("unrelated duplicate source acquired role", fields)
		}
	}
}

func TestAllOfRepeatedStringPatternAndValidationGate(t *testing.T) {
	input := schemas(`{"Factory":{"allOf":[{"properties":{"value":{"type":"string","minLength":2,"pattern":"^a","default":"aaaaa"}}},{"properties":{"value":{"type":"string","maxLength":3,"default":"ab"}}}],"default":{"value":"abc"}}}`)
	rq := request(input, "Dto")
	if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 {
		t.Fatal("unchecked constraints gained an outer-default role", err)
	}
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("single retained pattern rejected", err)
	}
	rq.Document = []byte(strings.Replace(string(input), `"default":{"value":"abc"}`, `"default":{"value":"bbc"}`, 1))
	if _, err := core.GenerateWithDiagnostics(rq); err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Factory/default/value" {
		t.Fatal("discarded pattern was dropped", err)
	}
}
