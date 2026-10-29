package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestScalarReferenceChildFactoriesAndOuterDefaultsAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, rows, holder string
	}{
		{"list field", `"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}`, `"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[4]}}}`},
		{"list factory", `"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"},"default":[4]}`, `"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows"}}}`},
		{"map field", `"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}}`, `"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":{"key":4}}}}`},
		{"map factory", `"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"},"default":{"key":4}}`, `"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows"}}}`},
		{"owner factory", `"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}`, `"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows"}},"default":{"values":[4]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Leaf":{"type":"integer","default":1},` + tc.rows + `,` + tc.holder + `}`
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = true
				rq.Options.ModelStyle = style
				control, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					if ds, ok := err.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
						t.Fatalf("outer and component defaults rejected at %s: %v", *ds[0].Pointer, err)
					}
					t.Fatalf("outer and component defaults rejected: %v", err)
				}
				rq.Document = schemas(strings.Replace(body, `"$ref":"#/components/schemas/Leaf"`, `"$ref":"#/components/schemas/Leaf","default":2`, 1))
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatalf("child annotation changed independent defaults: %v", err)
				}
			}
		})
	}
}

func TestScalarReferenceOuterDefaultsAcrossScalarCodecs(t *testing.T) {
	for _, tc := range []struct {
		name, kind, target, child, outer string
	}{
		{"string", `"type":"string"`, `"target"`, `"child"`, `"outer"`},
		{"boolean", `"type":"boolean"`, `true`, `false`, `true`},
		{"int", `"type":"integer"`, `1`, `2`, `3`},
		{"long", `"type":"integer","format":"int64"`, `1`, `2`, `3`},
		{"float", `"type":"number","format":"float"`, `1.5`, `2.5`, `3.5`},
		{"double", `"type":"number","format":"double"`, `1.5`, `2.5`, `3.5`},
		{"uuid", `"type":"string","format":"uuid"`, `"123e4567-e89b-12d3-a456-426614174000"`, `"123e4567-e89b-12d3-a456-426614174001"`, `"123e4567-e89b-12d3-a456-426614174002"`},
		{"date", `"type":"string","format":"date"`, `"2024-02-29"`, `"2024-03-01"`, `"2024-03-02"`},
		{"instant", `"type":"string","format":"date-time"`, `"2024-03-01T12:00:00Z"`, `"2024-03-02T12:00:00Z"`, `"2024-03-03T12:00:00Z"`},
		{"bytes", `"type":"string","format":"byte"`, `[1]`, `[2]`, `[3]`},
		{"binary", `"type":"string","format":"binary"`, `[1]`, `[2]`, `[3]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Leaf":{` + tc.kind + `,"default":` + tc.target + `},` +
				`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"},"default":[` + tc.outer + `]},` +
				`"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[` + tc.outer + `]}},"default":{"values":[` + tc.outer + `]}}}`
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					rq := request(schemas(body), "Dto")
					rq.Options.Validation, rq.Options.ModelStyle = checked, style
					control, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal("independent factories rejected", checked, style, err)
					}
					rq.Document = schemas(strings.Replace(body, `"$ref":"#/components/schemas/Leaf"`, `"$ref":"#/components/schemas/Leaf","default":`+tc.child, 1))
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(got, control) {
						t.Fatal("child changed independent factories", checked, style, err)
					}
				}
			}
		})
	}
}

func TestScalarReferenceOuterAndAliasFactories(t *testing.T) {
	body := `{"Leaf":{"type":"integer","default":1},"Middle":{"$ref":"#/components/schemas/Leaf","default":2},` +
		`"Alias":{"$ref":"#/components/schemas/Middle","default":3},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":4},"default":[5]},` +
		`"RowsAlias":{"$ref":"#/components/schemas/Rows","default":[6]},` +
		`"Holder":{"properties":{"values":{"$ref":"#/components/schemas/RowsAlias","default":[7]}},"default":{"values":[8]}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("independent alias factories rejected", err)
	}
	again, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("independent alias factories are not deterministic", err)
	}
}

func TestScalarReferenceOuterDefaultsRejectInvalidOwners(t *testing.T) {
	for _, tc := range []struct {
		name, body, pointer string
	}{
		{"target", `{"Leaf":{"type":"integer","default":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":2},"default":[3]}}`, "/components/schemas/Leaf/default"},
		{"alias", `{"Leaf":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Leaf","default":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":2},"default":[3]}}`, "/components/schemas/Alias/default"},
		{"child", `{"Leaf":{"type":"integer","default":1},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":true},"default":[]}}`, "/components/schemas/Rows/items/default"},
		{"collection", `{"Leaf":{"type":"integer","default":1},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":2},"default":[true]}}`, "/components/schemas/Rows/default/0"},
		{"field", `{"Leaf":{"type":"integer","default":1},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":2}},"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[true]}}}}`, "/components/schemas/Holder/properties/values/default/0"},
		{"owner", `{"Leaf":{"type":"integer","default":1},"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":2}},"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows"}},"default":{"values":{"a/~":true}}}}`, "/components/schemas/Holder/default/values/a~1~0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := request(schemas(tc.body), "Dto")
			rq.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				if ok && len(ds) > 0 && ds[0].Pointer != nil {
					t.Fatalf("wrong diagnostic at %s: %v", *ds[0].Pointer, err)
				}
				t.Fatal("invalid annotation did not fail at its owner", err)
			}
		})
	}
}

func TestScalarReferenceInlineFieldAndOwnerDefaults(t *testing.T) {
	for _, tc := range []struct{ name, field, ownerDefault string }{
		{"inline list field", `"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":2},"default":[3]`, ""},
		{"inline map field", `"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":2},"default":{"field":3}`, ""},
		{"inline list owner", `"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":2}`, `,"default":{"values":[3]}`},
		{"inline map owner", `"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":2}`, `,"default":{"values":{"owner":3}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Leaf":{"type":"integer","default":1},"Holder":{"properties":{"values":{` + tc.field + `}}` + tc.ownerDefault + `}}`
			for _, checked := range []bool{false, true} {
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("inline outer default rejected", checked, err)
				}
				rq.Document = schemas(strings.Replace(body, `"$ref":"#/components/schemas/Leaf","default":2`, `"$ref":"#/components/schemas/Leaf"`, 1))
				control, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("inline outer default depended on the child annotation", checked, err)
				}
			}
		})
	}
}

func TestScalarReferenceOuterDefaultsRespectNearestNullable(t *testing.T) {
	body := `{"Leaf":{"type":["integer","null"],"default":null},"Alias":{"$ref":"#/components/schemas/Leaf","nullable":false,"default":1},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"default":[null,3]},` +
		`"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[null,4]}},"default":{"values":[null,5]}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("nearest nullable child rejected", err)
	}
	rq.Document = schemas(strings.Replace(body, `"nullable":true,"default":null`, `"nullable":false,"default":null`, 1))
	report, err := core.GenerateWithDiagnostics(rq)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
		*ds[0].Pointer != "/components/schemas/Rows/items/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("narrowed child did not reject its own null annotation", err)
	}
}

func TestScalarReferenceInlineOuterDefaultsValidateBeforePresenceSuppression(t *testing.T) {
	for _, required := range []bool{false, true} {
		body := `{"properties":{"rows":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Leaf","default":2},"default":[]}}`
		if required {
			body += `,"required":["rows"]`
		}
		body += `}`
		doc := `{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"integer","default":1}}},"paths":{"/update":{"patch":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + body + `}}},"responses":{"204":{}}}}}}`
		for _, checked := range []bool{false, true} {
			rq := request([]byte(doc), "Dto")
			rq.Options.Validation = checked
			annotated, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("inline request default rejected", required, checked, err)
			}
			rq.Document = []byte(strings.Replace(doc, `"$ref":"#/components/schemas/Leaf","default":2`, `"$ref":"#/components/schemas/Leaf"`, 1))
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(annotated, control) {
				t.Fatal("child changed request presence", required, checked, err)
			}
			rq.Document = []byte(strings.Replace(doc, `"default":2`, `"default":true`, 1))
			report, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			want := "/paths/~1update/patch/requestBody/content/application~1json/schema/properties/rows/items/default"
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != want || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("presence suppression hid invalid child", required, checked, err)
			}
		}
	}
}

func TestScalarReferenceOuterDefaultsKeepOptimizedNames(t *testing.T) {
	body := `{"Leaf":{"type":"integer","default":1},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"},"default":[3]},` +
		`"A":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[4]}}},` +
		`"B":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[4]}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(schemas(body), "Dto")
		rq.Options.ModelStyle, rq.Options.Validation = style, true
		rq.Options.DeduplicateModels, rq.Options.HoistSharedShapes = true, true
		rq.Options.MergeIdenticalModels, rq.Options.LockHoistedNames = true, true
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || len(control.Artifacts.NameLock) == 0 {
			t.Fatal("outer-default optimization rejected", style, err)
		}
		rq.PreviousNameLock = control.Artifacts.NameLock
		rq.Document = schemas(strings.Replace(body, `"$ref":"#/components/schemas/Leaf"`, `"$ref":"#/components/schemas/Leaf","default":2`, 1))
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("child changed optimized outer default or name lock", style, err)
		}
	}
}

func TestScalarReferenceOuterDefaultKeepsValueBudget(t *testing.T) {
	long := `"` + strings.Repeat("x", 32769) + `"`
	body := `{"Leaf":{"type":"string","default":"target"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"child"},"default":[` + long + `]}}`
	report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto"))
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil ||
		*ds[0].Pointer != "/components/schemas/Rows/default/0" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		if ok && len(ds) > 0 && ds[0].Pointer != nil {
			t.Fatalf("outer factory budget failed at %s: %v", *ds[0].Pointer, err)
		}
		t.Fatal("outer factory escaped its value budget", err)
	}
}

func TestScalarReferenceNestedCollectionFactoryGraph(t *testing.T) {
	body := `{"Leaf":{"type":"integer","default":1},` +
		`"Values":{"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":2},"default":{"inside":3}},` +
		`"ValuesAlias":{"$ref":"#/components/schemas/Values","default":{"alias":4}},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias"},"default":[{"row":5}]},` +
		`"RowsAlias":{"$ref":"#/components/schemas/Rows","default":[{"alias":6}]},` +
		`"Holder":{"properties":{"values":{"$ref":"#/components/schemas/RowsAlias","default":[{"field":7}]}},"default":{"values":[{"owner":8}]}}}`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation, rq.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("nested independent collection factories rejected", checked, style, err)
			}
			rq.Document = schemas(strings.Replace(body, `"$ref":"#/components/schemas/Leaf","default":2`, `"$ref":"#/components/schemas/Leaf"`, 1))
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("nested factory depended on child annotation", checked, style, err)
			}
		}
	}
}

func TestScalarReferenceNestedFactoriesValidateEachOwner(t *testing.T) {
	base := `{"Leaf":{"type":"integer","default":1},` +
		`"Values":{"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":2},"default":{"inner":3}},` +
		`"ValuesAlias":{"$ref":"#/components/schemas/Values","default":{"alias":4}},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias"},"default":[{"row":5}]},` +
		`"RowsAlias":{"$ref":"#/components/schemas/Rows","default":[{"alias":6}]},` +
		`"Holder":{"properties":{"values":{"$ref":"#/components/schemas/RowsAlias","default":[{"field":7}]}},"default":{"values":[{"owner":8}]}}}`
	for _, tc := range []struct{ old, replacement, pointer string }{
		{`"default":2`, `"default":true`, "/components/schemas/Values/additionalProperties/default"},
		{`"default":{"inner":3}`, `"default":{"inner":true}`, "/components/schemas/Values/default/inner"},
		{`"default":{"alias":4}`, `"default":{"alias":true}`, "/components/schemas/ValuesAlias/default/alias"},
		{`"default":[{"row":5}]`, `"default":[{"row":true}]`, "/components/schemas/Rows/default/0/row"},
		{`"default":[{"alias":6}]`, `"default":[{"alias":true}]`, "/components/schemas/RowsAlias/default/0/alias"},
		{`"default":[{"field":7}]`, `"default":[{"field":true}]`, "/components/schemas/Holder/properties/values/default/0/field"},
		{`"default":{"values":[{"owner":8}]}`, `"default":{"values":[{"owner":true}]}`, "/components/schemas/Holder/default/values/0/owner"},
	} {
		for _, checked := range []bool{false, true} {
			carrierDefaultFailure(t, schemas(strings.Replace(base, tc.old, tc.replacement, 1)), checked, "E_SCHEMA_INVALID", tc.pointer)
		}
	}
	// A collection-valued child validates its own literal independently.
	for _, checked := range []bool{false, true} {
		input := strings.Replace(base, `"items":{"$ref":"#/components/schemas/ValuesAlias"}`, `"items":{"$ref":"#/components/schemas/ValuesAlias","default":{}}`, 1)
		rq := request(schemas(input), "Dto")
		rq.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("independent collection child rejected", checked, err)
		}
		invalid := strings.Replace(input, `"default":{}`, `"default":{"bad":true}`, 1)
		carrierDefaultFailure(t, schemas(invalid), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default/bad")
	}
}

func TestScalarReferenceNestedFactoriesNearestNullable(t *testing.T) {
	body := `{"Leaf":{"type":["integer","null"]},` +
		`"Values":{"nullable":true,"additionalProperties":{"$ref":"#/components/schemas/Leaf"},"default":null},` +
		`"Alias":{"$ref":"#/components/schemas/Values","nullable":false,"default":{"a":null}},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":true},"default":[null,{"a":null}]}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("nearest nullable collection view rejected", checked, err)
		}
		narrowed := strings.Replace(body, `"items":{"$ref":"#/components/schemas/Alias","nullable":true}`, `"items":{"$ref":"#/components/schemas/Alias","nullable":false}`, 1)
		carrierDefaultFailure(t, schemas(narrowed), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows/default/0")
	}
}

func TestScalarReferenceNestedFactoriesStayBounded(t *testing.T) {
	for _, size := range []int{3, 30} {
		entries := []string{`"Leaf":{"type":"integer"}`}
		for i := size - 1; i >= 0; i-- {
			target := "Leaf"
			if i+1 < size {
				target = fmt.Sprintf("C%02d", i+1)
			}
			annotation := ""
			if i == 0 {
				annotation = `,"default":[]`
			}
			entries = append(entries, fmt.Sprintf(`"C%02d":{"type":"array","items":{"$ref":"#/components/schemas/%s"}%s}`, i, target, annotation))
		}
		report, err := core.GenerateWithDiagnostics(request(schemas("{"+strings.Join(entries, ",")+"}"), "Dto"))
		if size == 3 && err != nil {
			t.Fatal("short collection graph rejected", err)
		}
		if size == 30 && (err == nil || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("over-depth collection graph emitted artifacts", err)
		}
	}
	cycle := `{"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Loop"},"default":[]},"Loop":{"type":"array","items":{"$ref":"#/components/schemas/Rows"}}}`
	report, err := core.GenerateWithDiagnostics(request(schemas(cycle), "Dto"))
	if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("cyclic collection graph emitted artifacts", err)
	}
}

func TestScalarReferenceExistingFactoryBudgetIsPreserved(t *testing.T) {
	// The already supported direct factories share the original cumulative
	// source budget. New intermediate-collection proof must not charge them.
	for _, size := range []int{1050, 1100} {
		var body strings.Builder
		body.WriteString(`{"Leaf":{"type":"integer"}`)
		for i := 0; i < size; i++ {
			fmt.Fprintf(&body, `,"Rows%04d":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"},"default":[%d]}`, i, i)
		}
		body.WriteByte('}')
		report, err := core.GenerateWithDiagnostics(request(schemas(body.String()), "Dto"))
		if size == 1050 && err != nil {
			t.Fatal("previously admitted direct factories consumed extra source work", err)
		}
		if size == 1100 {
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("direct factories escaped the cumulative source limit", err)
			}
		}
	}
}
