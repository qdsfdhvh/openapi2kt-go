package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInlineScalarCollectionReferenceChildDefaultIsIndependent(t *testing.T) {
	for _, tc := range []struct{ name, target, valid, invalid, tail string }{
		{"list", `"type":"array","items":{"type":"integer"}`, `[3]`, `[false]`, "/0"},
		{"map", `"additionalProperties":{"type":"integer"}`, `{"child":3}`, `{"child":false}`, "/child"},
	} {
		for _, parent := range []struct{ name, schema, location string }{
			{"list", `"type":"array","items":`, "/items"},
			{"map", `"additionalProperties":`, "/additionalProperties"},
		} {
			t.Run(tc.name+"/"+parent.name, func(t *testing.T) {
				body := `{"Values":{` + tc.target + `,"default":` + tc.valid + `},` +
					`"Alias":{"$ref":"#/components/schemas/Values","default":` + tc.valid + `},` +
					`"Rows":{` + parent.schema + `{"$ref":"#/components/schemas/Alias","default":` + tc.valid + `}}}`
				for _, checked := range []bool{false, true} {
					for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
						rq := request(schemas(body), "Dto")
						rq.Options.Validation, rq.Options.ModelStyle = checked, style
						annotated, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal("inline scalar collection child rejected", checked, style, err)
						}
						rq.Document = schemas(strings.Replace(body, `,"default":`+tc.valid+`}}}`, `}}}`, 1))
						control, err := core.GenerateWithDiagnostics(rq)
						if err != nil || !reflect.DeepEqual(annotated, control) {
							t.Fatal("child annotation changed generated output", checked, style, err)
						}
						invalid := strings.Replace(body, `"default":`+tc.valid+`}}}`, `"default":`+tc.invalid+`}}}`, 1)
						carrierDefaultFailure(t, schemas(invalid), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows"+parent.location+"/default"+tc.tail)
					}
				}
			})
		}
	}
}

func TestInlineScalarCollectionChildDefaultNestedNullableAndChecks(t *testing.T) {
	body := `{"AOuter":{"additionalProperties":{"$ref":"#/components/schemas/BAlias","nullable":true,"default":[{"child":2}]}},"BAlias":{"$ref":"#/components/schemas/CValues","nullable":false},` +
		`"CValues":{"type":"array","items":{"additionalProperties":{"type":"integer","minimum":0},"maxProperties":1},"minItems":1}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("nested checked collection child rejected", err)
	}
	for _, tc := range []struct{ valid, invalid, pointer string }{
		{`"default":[{"child":2}]`, `"default":[{"child":-1}]`, "/components/schemas/AOuter/additionalProperties/default/0/child"},
		{`"default":[{"child":2}]`, `"default":[{"a":1,"b":2}]`, "/components/schemas/AOuter/additionalProperties/default/0"},
		{`"default":[{"child":2}]`, `"default":[]`, "/components/schemas/AOuter/additionalProperties/default"},
	} {
		invalid := strings.Replace(body, tc.valid, tc.invalid, 1)
		carrierDefaultFailure(t, schemas(invalid), true, "E_SCHEMA_INVALID", tc.pointer)
	}
	nullable := strings.Replace(body, `"default":[{"child":2}]`, `"default":null`, 1)
	if _, err := core.GenerateWithDiagnostics(request(schemas(nullable), "Dto")); err == nil {
		t.Fatal("validation-off input with bounds unexpectedly accepted")
	}
	rq.Document = schemas(nullable)
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("nearest nullable collection child rejected", err)
	}
	narrow := strings.Replace(nullable, `"nullable":true,"default":null`, `"nullable":false,"default":null`, 1)
	carrierDefaultFailure(t, schemas(narrow), true, "E_SCHEMA_INVALID", "/components/schemas/AOuter/additionalProperties/default")
}

func TestInlineScalarCollectionChildDefaultSeparateRoles(t *testing.T) {
	for _, body := range []string{
		`{"Target":{"type":"array","items":{"type":"integer","default":1}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":[1]}}}`,
		`{"Target":{"type":"array","items":{"properties":{"value":{"type":"integer"}}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":[{"value":1}]}}}`,
		`{"Target":{"additionalProperties":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":{}}}}`,
		`{"Inner":{"type":"array","items":{"type":"integer"}},"Target":{"type":"array","items":{"$ref":"#/components/schemas/Inner"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":[[1]]}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("unadmitted child role emitted artifacts", checked, body, err)
			}
		}
	}
}

func TestInlineScalarCollectionChildDefaultValueBudget(t *testing.T) {
	body := `{"Target":{"type":"array","items":{"type":"string"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":["` + strings.Repeat("x", 32768) + `"]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		report, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("oversized child default escaped its budget", checked, err)
		}
	}
}

func TestInlineScalarCollectionChildDefaultRejectsAliasCycle(t *testing.T) {
	body := `{"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/A","default":[]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("cyclic alias emitted artifacts", checked, err)
		}
	}
}
