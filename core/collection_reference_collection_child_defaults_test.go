package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedCollectionChildDefaultIsIndependent(t *testing.T) {
	body := `{"Leaf":{"type":"integer","default":1},` +
		`"Values":{"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":2},"default":{"inner":3}},` +
		`"ValuesAlias":{"$ref":"#/components/schemas/Values","default":{"alias":4}},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias","default":{"child":5}},"default":[{"outer":6}]},` +
		`"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[{"field":7}]}},"default":{"values":[{"owner":8}]}}}`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation, rq.Options.ModelStyle = checked, style
			annotated, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				if ds, ok := err.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
					t.Fatal("independent collection child rejected", checked, style, *ds[0].Pointer, err)
				}
				t.Fatal("independent collection child rejected", checked, style, err)
			}
			rq.Document = schemas(strings.Replace(body, `,"default":{"child":5}`, "", 1))
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(annotated, control) {
				t.Fatal("child annotation changed independent factories", checked, style, err)
			}
		}
	}
}

func TestReferencedCollectionChildDefaultKindsAndPointers(t *testing.T) {
	for _, tc := range []struct{ name, target, valid, invalid, tail string }{
		{"map target", `"additionalProperties":{"$ref":"#/components/schemas/Leaf"}`, `{"key":2}`, `{"a/~":true}`, "/a~1~0"},
		{"list target", `"type":"array","items":{"$ref":"#/components/schemas/Leaf"}`, `[2]`, `[true]`, "/0"},
	} {
		for _, parent := range []struct{ name, schema, location string }{
			{"list", `"type":"array","items":`, "/items"},
			{"map", `"additionalProperties":`, "/additionalProperties"},
		} {
			t.Run(tc.name+"/"+parent.name, func(t *testing.T) {
				body := `{"Leaf":{"type":"integer"},"Target":{` + tc.target + `},` +
					`"TargetAlias":{"$ref":"#/components/schemas/Target"},` +
					`"Container":{` + parent.schema + `{"$ref":"#/components/schemas/TargetAlias","default":` + tc.valid + `}}}`
				for _, checked := range []bool{false, true} {
					rq := request(schemas(body), "Dto")
					rq.Options.Validation = checked
					annotated, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal("collection-valued child rejected", checked, err)
					}
					rq.Document = schemas(strings.Replace(body, `,"default":`+tc.valid, "", 1))
					control, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(annotated, control) {
						t.Fatal("child default altered output", checked, err)
					}
					invalid := strings.Replace(body, `"default":`+tc.valid, `"default":`+tc.invalid, 1)
					carrierDefaultFailure(t, schemas(invalid), checked, "E_SCHEMA_INVALID", "/components/schemas/Container"+parent.location+"/default"+tc.tail)
				}
			})
		}
	}
}

func TestReferencedCollectionChildDefaultNearestNullable(t *testing.T) {
	body := `{"Leaf":{"type":"integer"},"Target":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Leaf"}},` +
		`"Alias":{"$ref":"#/components/schemas/Target","nullable":false},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"default":[]}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("nearest nullable child rejected", checked, err)
		}
		narrowed := strings.Replace(body, `"nullable":true,"default":null`, `"nullable":false,"default":null`, 1)
		carrierDefaultFailure(t, schemas(narrowed), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default")
	}
}

func TestReferencedCollectionChildDefaultRetainsChecks(t *testing.T) {
	body := `{"Leaf":{"type":"integer","minimum":0},` +
		`"Target":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"},"minProperties":1,"maxProperties":1},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":{"a":0}}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("valid checked collection child rejected", err)
	}
	invalidLeaf := strings.Replace(body, `"default":{"a":0}`, `"default":{"a":-1}`, 1)
	carrierDefaultFailure(t, schemas(invalidLeaf), true, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default/a")
	invalidSize := strings.Replace(body, `"default":{"a":0}`, `"default":{"a":0,"b":1}`, 1)
	carrierDefaultFailure(t, schemas(invalidSize), true, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default")
}

func TestReferencedCollectionChildDefaultBeforePatchSuppression(t *testing.T) {
	for _, required := range []bool{false, true} {
		owner := `{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":{"child":2}}}}`
		if required {
			owner += `,"required":["rows"]`
		}
		owner += `}`
		doc := `{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"integer"},"Values":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}}}},"paths":{"/update":{"patch":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + owner + `}}},"responses":{"204":{}}}}}}`
		for _, checked := range []bool{false, true} {
			rq := request([]byte(doc), "Dto")
			rq.Options.Validation = checked
			annotated, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("request collection child rejected", required, checked, err)
			}
			rq.Document = []byte(strings.Replace(doc, `,"default":{"child":2}`, "", 1))
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(annotated, control) {
				t.Fatal("child changed Patch presence", required, checked, err)
			}
			invalid := strings.Replace(doc, `"default":{"child":2}`, `"default":{"child":true}`, 1)
			carrierDefaultFailure(t, []byte(invalid), checked, "E_SCHEMA_INVALID", "/paths/~1update/patch/requestBody/content/application~1json/schema/properties/rows/items/default/child")
		}
	}
}

func TestReferencedCollectionChildDefaultsAtMultipleLevels(t *testing.T) {
	// Names deliberately put the consuming declaration before both targets.
	body := `{"AOuter":{"type":"array","items":{"$ref":"#/components/schemas/BMiddle","default":[{"outerChild":4}]},"default":[[{"outer":5}]]},` +
		`"BMiddle":{"type":"array","items":{"$ref":"#/components/schemas/CValues","default":{"middleChild":3}},"default":[{"middle":6}]},` +
		`"CValues":{"additionalProperties":{"$ref":"#/components/schemas/DLeaf","default":2},"default":{"root":7}},` +
		`"DLeaf":{"type":"integer","default":1}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		annotated, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("independent nested child defaults rejected", checked, err)
		}
		control := strings.Replace(body, `,"default":[{"outerChild":4}]`, "", 1)
		control = strings.Replace(control, `,"default":{"middleChild":3}`, "", 1)
		rq.Document = schemas(control)
		baseline, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(annotated, baseline) {
			t.Fatal("nested annotations altered generated output", checked, err)
		}
		for _, tc := range []struct{ valid, invalid, pointer string }{
			{`"default":[{"outerChild":4}]`, `"default":[{"outerChild":false}]`, "/components/schemas/AOuter/items/default/0/outerChild"},
			{`"default":{"middleChild":3}`, `"default":{"middleChild":false}`, "/components/schemas/BMiddle/items/default/middleChild"},
			{`"default":2`, `"default":false`, "/components/schemas/CValues/additionalProperties/default"},
		} {
			invalid := strings.Replace(body, tc.valid, tc.invalid, 1)
			carrierDefaultFailure(t, schemas(invalid), checked, "E_SCHEMA_INVALID", tc.pointer)
		}
	}
}

func TestReferencedCollectionChildDefaultWithForwardScalarAliases(t *testing.T) {
	body := `{"AOuter":{"type":"array","items":{"$ref":"#/components/schemas/BValues","default":{"child":5}},"default":[{"outer":6}]},` +
		`"BValues":{"additionalProperties":{"$ref":"#/components/schemas/YAlias","default":4},"default":{"root":3}},` +
		`"YAlias":{"$ref":"#/components/schemas/ZLeaf","default":2},"ZLeaf":{"type":"integer","default":1}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	annotated, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("forward scalar alias and independent factories rejected", err)
	}
	rq.Document = schemas(strings.Replace(body, `,"default":{"child":5}`, "", 1))
	control, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(annotated, control) {
		t.Fatal("collection child changed forward scalar factories", err)
	}
	invalid := strings.Replace(body, `"default":2`, `"default":false`, 1)
	carrierDefaultFailure(t, schemas(invalid), true, "E_SCHEMA_INVALID", "/components/schemas/YAlias/default")
}

func TestReferencedCollectionChildDefaultRejectsCollectionCycles(t *testing.T) {
	body := `{"A":{"type":"array","items":{"$ref":"#/components/schemas/B","default":[]}},` +
		`"B":{"type":"array","items":{"$ref":"#/components/schemas/A","default":[]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("cyclic collection child emitted artifacts", checked, err)
		}
	}
}

func TestReferencedCollectionChildDefaultKeepsOtherRolesSeparate(t *testing.T) {
	// Keep the original empty collection annotation as a positive control.
	enumCollection := `{"Word":{"type":"string","enum":["a"]},"Target":{"type":"array","items":{"$ref":"#/components/schemas/Word"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":[]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(enumCollection), "Dto")
		rq.Options.Validation = checked
		inlineBuiltinChildGraduation(t, rq, `,"default":[]`, `,"default":["future"]`,
			"/components/schemas/Rows/items/default/0")
	}
	for _, body := range []string{
		`{"Target":{"properties":{"v":{"type":"integer"}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":{"v":1}}}}`,
		`{"Leaf":{"type":"integer"},"Owner":{"properties":{"values":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/values","default":{}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("unadmitted collection child role emitted artifacts", checked, body, err)
			}
		}
	}
}
