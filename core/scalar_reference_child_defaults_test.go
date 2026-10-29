package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// An element annotation belongs to the reference itself. It validates without
// supplying a value to its enclosing collection or changing generated output.
func TestScalarReferenceCollectionChildDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, source, valid, invalid, invalidTail string
	}{
		{"string", `{"type":"string"}`, `"ready"`, `true`, ""},
		{"integer", `{"type":"integer"}`, `1`, `2147483648`, ""},
		{"long", `{"type":"integer","format":"int64"}`, `9223372036854775807`, `1.0`, ""},
		{"float", `{"type":"number","format":"float"}`, `0.5`, `1e300`, ""},
		{"double", `{"type":"number","format":"double"}`, `1.25`, `"1.25"`, ""},
		{"boolean", `{"type":"boolean"}`, `true`, `"true"`, ""},
		{"uuid", `{"type":"string","format":"uuid"}`, `"123E4567E89B12D3A456426614174000"`, `"bad"`, ""},
		{"date", `{"type":"string","format":"date"}`, `"2024-02-29"`, `"2023-02-29"`, ""},
		{"instant", `{"type":"string","format":"date-time"}`, `"2024-02-29T12:34:56.123456789Z"`, `"2024-02-29T12:34:56.1234567890Z"`, ""},
		{"byte", `{"type":"string","format":"byte"}`, `[-128,0,127]`, `[128]`, "/0"},
		{"binary", `{"type":"string","format":"binary"}`, `[-128,0,127]`, `[1.0]`, "/0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, collection := range []struct {
				name, schema, pointer string
			}{
				{"list", `{"type":"array","items":@}`, "/components/schemas/Rows/items/default"},
				{"map", `{"type":"object","additionalProperties":@}`, "/components/schemas/Rows/additionalProperties/default"},
			} {
				for _, checked := range []bool{false, true} {
					for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
						child := `{"$ref":"#/components/schemas/Leaf"}`
						body := `{"Leaf":` + tc.source + `,"Rows":` + strings.Replace(collection.schema, "@", child, 1) + `}`
						rq := request(schemas(body), "Dto")
						rq.Options.Validation, rq.Options.ModelStyle = checked, style
						control, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal("unannotated control rejected", collection.name, checked, style, err)
						}
						annotated := strings.Replace(body, child, strings.TrimSuffix(child, "}")+`,"default":`+tc.valid+`}`, 1)
						rq.Document = schemas(annotated)
						got, err := core.GenerateWithDiagnostics(rq)
						if err != nil || !reflect.DeepEqual(got, control) {
							t.Fatal("valid child annotation changed output", collection.name, checked, style, err)
						}
						rq.Document = schemas(strings.Replace(annotated, `"default":`+tc.valid, `"default":`+tc.invalid, 1))
						report, err := core.GenerateWithDiagnostics(rq)
						ds, ok := err.(core.Diagnostics)
						if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
							*ds[0].Pointer != collection.pointer+tc.invalidTail || !reflect.DeepEqual(report, core.GenerationReport{}) {
							t.Fatal("invalid child must fail at its original default", collection.name, checked, style, err)
						}
					}
				}
			}
		})
	}
}

func TestScalarReferenceCollectionChildDefaultContexts(t *testing.T) {
	for _, tc := range []struct {
		name, body, reference, valid, invalid, pointer string
		validation                                     bool
	}{
		{
			"named alias in nested collection",
			`{"Leaf":{"type":"integer","minimum":0},"Alias":{"$ref":"#/components/schemas/Leaf"},"Rows":{"type":"array","items":{"additionalProperties":{"$ref":"#/components/schemas/Alias"}}}}`,
			`"$ref":"#/components/schemas/Alias"`, `2`, `-1`,
			"/components/schemas/Rows/items/additionalProperties/default", true,
		},
		{
			"borrowed physical scalar",
			`{"Source":{"properties":{"value":{"type":"string","nullable":true}}},"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/value","nullable":false}}}}}`,
			`"$ref":"#/components/schemas/Source/properties/value","nullable":false`, `"ok"`, `null`,
			"/components/schemas/Holder/properties/rows/items/default", false,
		},
		{
			"nullable child",
			`{"Leaf":{"type":"boolean"},"Holder":{"properties":{"rows":{"additionalProperties":{"$ref":"#/components/schemas/Leaf","nullable":true}}}}}`,
			`"$ref":"#/components/schemas/Leaf","nullable":true`, `null`, `"false"`,
			"/components/schemas/Holder/properties/rows/additionalProperties/default", false,
		},
		{
			"independent field default",
			`{"Leaf":{"type":"integer"},"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"},"default":[3]}}}}`,
			`"$ref":"#/components/schemas/Leaf"`, `2`, `true`,
			"/components/schemas/Holder/properties/rows/items/default", false,
		},
		{
			"checked source",
			`{"Leaf":{"type":"string","minLength":2,"pattern":"^a"},"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}}}`,
			`"$ref":"#/components/schemas/Leaf"`, `"ab"`, `"x"`,
			"/components/schemas/Rows/additionalProperties/default", true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				rq := request(schemas(tc.body), "Dto")
				rq.Options.Validation, rq.Options.ModelStyle = tc.validation, style
				control, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("unannotated reference context rejected", style, err)
				}
				rq.Document = schemas(strings.Replace(tc.body, tc.reference, tc.reference+`,"default":`+tc.valid, 1))
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, control) {
					if ds, ok := err.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
						t.Log("diagnostic pointer", *ds[0].Pointer)
					}
					t.Fatal("child annotation changed existing context", style, err)
				}
				rq.Document = schemas(strings.Replace(tc.body, tc.reference, tc.reference+`,"default":`+tc.invalid, 1))
				report, err := core.GenerateWithDiagnostics(rq)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
					*ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("invalid child annotation lost its source", style, err)
				}
			}
		})
	}
}

func TestScalarReferenceCollectionChildNearestNullable(t *testing.T) {
	for _, tc := range []struct {
		name, body, use, valid, invalid string
		checked                         bool
	}{
		{
			"intrinsically nullable alias",
			`{"Leaf":{"type":["integer","null"]},"Alias":{"$ref":"#/components/schemas/Leaf"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}`,
			`"$ref":"#/components/schemas/Alias"`, `null`, `"bad"`, false,
		},
		{
			"nearest false narrows alias",
			`{"Leaf":{"type":["integer","null"]},"Alias":{"$ref":"#/components/schemas/Leaf","nullable":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":false}}}`,
			`"$ref":"#/components/schemas/Alias","nullable":false`, `1`, `null`, false,
		},
		{
			"nearest true widens byte alias",
			`{"Leaf":{"type":"string","format":"byte"},"Alias":{"$ref":"#/components/schemas/Leaf"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":true}}}`,
			`"$ref":"#/components/schemas/Alias","nullable":true`, `null`, `[128]`, false,
		},
		{
			"temporal alias narrows",
			`{"Leaf":{"type":["string","null"],"format":"date"},"Alias":{"$ref":"#/components/schemas/Leaf","nullable":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":false}}}`,
			`"$ref":"#/components/schemas/Alias","nullable":false`, `"2024-02-29"`, `null`, false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := request(schemas(tc.body), "Dto")
			rq.Options.Validation = tc.checked
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("unannotated nullable path rejected", err)
			}
			rq.Document = schemas(strings.Replace(tc.body, tc.use, tc.use+`,"default":`+tc.valid, 1))
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("nullable child annotation changed output", err)
			}
			rq.Document = schemas(strings.Replace(tc.body, tc.use, tc.use+`,"default":`+tc.invalid, 1))
			report, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != "/components/schemas/Rows/items/default" &&
					*ds[0].Pointer != "/components/schemas/Rows/items/default/0" ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid nearest nullable value", err)
			}
		})
	}
}

func TestScalarReferenceCollectionChildDefaultInOperations(t *testing.T) {
	doc := `{
	  "openapi":"3.1.0",
	  "components":{"schemas":{"Value":{"type":"integer"}}},
	  "paths":{"/rows":{"post":{
	    "requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"array","items":{"$ref":"#/components/schemas/Value"}}}}},
	    "responses":{"200":{"content":{"application/json":{"schema":{"additionalProperties":{"$ref":"#/components/schemas/Value"}}}}}}
	  }}}
	}`
	for _, checked := range []bool{false, true} {
		rq := request([]byte(doc), "Dto")
		rq.Options.Validation = checked
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("unannotated operation rejected", checked, err)
		}
		source := `"$ref":"#/components/schemas/Value"`
		rq.Document = []byte(strings.ReplaceAll(doc, source, source+`,"default":1`))
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("operation child annotation changed output", checked, err)
		}
		for _, tc := range []struct{ doc, pointer string }{
			{strings.Replace(doc, source, source+`,"default":true`, 1), "/paths/~1rows/post/requestBody/content/application~1json/schema/items/default"},
			{strings.Replace(doc, `"additionalProperties":{`+source, `"additionalProperties":{`+source+`,"default":true`, 1), "/paths/~1rows/post/responses/200/content/application~1json/schema/additionalProperties/default"},
		} {
			rq.Document = []byte(tc.doc)
			report, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid operation child was ignored", checked, err)
			}
		}
	}
}

func TestScalarReferenceCollectionChildDefaultInObjectGraphs(t *testing.T) {
	for _, tc := range []struct{ name, holder string }{
		{"field default", `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows","default":[3]}}}`},
		{"owner factory", `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}},"default":{"rows":[3]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Leaf":{"type":"integer"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}},` + tc.holder + `}`
			source := `"$ref":"#/components/schemas/Leaf"`
			for _, checked := range []bool{false, true} {
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = checked
				control, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("unannotated default graph rejected", checked, err)
				}
				rq.Document = schemas(strings.Replace(body, source, source+`,"default":2`, 1))
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("child changed an independent outer default", checked, err)
				}
				rq.Document = schemas(strings.Replace(body, source, source+`,"default":true`, 1))
				report, err := core.GenerateWithDiagnostics(rq)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
					*ds[0].Pointer != "/components/schemas/Rows/items/default" ||
					!reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("invalid child was hidden by an outer default", checked, err)
				}
			}
		})
	}
}

func TestScalarReferenceCollectionChildDefaultWithEmptyOuterValue(t *testing.T) {
	for _, tc := range []struct {
		name, outer string
	}{
		{"empty", `"default":[]`},
		{"null", `"nullable":true,"default":null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Leaf":{"type":"integer"},"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"},` + tc.outer + `}}}}`
			use := `"$ref":"#/components/schemas/Leaf"`
			for _, checked := range []bool{false, true} {
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = checked
				control, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("unannotated outer default rejected", checked, err)
				}
				rq.Document = schemas(strings.Replace(body, use, use+`,"default":2`, 1))
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("child changed outer default", checked, err)
				}
				rq.Document = schemas(strings.Replace(body, use, use+`,"default":true`, 1))
				report, err := core.GenerateWithDiagnostics(rq)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
					*ds[0].Pointer != "/components/schemas/Holder/properties/rows/items/default" ||
					!reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("empty or null outer default hid invalid child", checked, err)
				}
			}
		})
	}
}

func TestScalarReferenceCollectionChildDefaultRequiredAndPatch(t *testing.T) {
	for _, required := range []bool{false, true} {
		body := `{"properties":{"rows":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Leaf"},"default":[]}}`
		if required {
			body += `,"required":["rows"]`
		}
		body += `}`
		doc := `{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"integer"}}},"paths":{"/update":{"patch":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + body + `}}},"responses":{"204":{}}}}}}`
		use := `"$ref":"#/components/schemas/Leaf"`
		for _, checked := range []bool{false, true} {
			rq := request([]byte(doc), "Dto")
			rq.Options.Validation = checked
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("unannotated request rejected", required, checked, err)
			}
			rq.Document = []byte(strings.Replace(doc, use, use+`,"default":2`, 1))
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("child changed required or Patch output", required, checked, err)
			}
			rq.Document = []byte(strings.Replace(doc, use, use+`,"default":true`, 1))
			report, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != "/paths/~1update/patch/requestBody/content/application~1json/schema/properties/rows/items/default" ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("required or Patch suppression hid invalid child", required, checked, err)
			}
		}
	}
}

func TestScalarReferenceCollectionChildDefaultExcludedRoles(t *testing.T) {
	for _, body := range []string{
		`{"Leaf":{"type":"string","x-native-type":"kotlin.String"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`,
		`{"Leaf":{"type":"string","x-kotlin-value-class":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`,
		`{"Leaf":{"type":"string","const":"one"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`,
		`{"Leaf":{"type":"string"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Leaf","default":"one"}],"items":false}}`,
		`{"Leaf":{"type":"string"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Leaf"}],"items":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`,
		`{"Leaf":{"type":"string"},"Rows":{"type":"object","properties":{"x":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("an excluded source or child role was admitted", checked, body, err)
			}
		}
	}
}

func TestScalarReferenceCollectionChildDefaultBudgets(t *testing.T) {
	for _, size := range []int{32750, 32768} {
		body := `{"Leaf":{"type":"string"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"` + strings.Repeat("x", size) + `"}}}`
		report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto"))
		if size == 32750 && err != nil {
			t.Fatal("default below the byte limit rejected", err)
		}
		if size == 32768 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("default escaped the byte limit", err)
		}
	}
	for _, count := range []int{10, 50} {
		var body strings.Builder
		body.WriteString(`{"Leaf":{"type":"integer"}`)
		for i := 0; i < count; i++ {
			target := "Leaf"
			if i+1 < count {
				target = fmt.Sprintf("Alias%d", i+1)
			}
			fmt.Fprintf(&body, `,"Alias%d":{"$ref":"#/components/schemas/%s"}`, i, target)
		}
		body.WriteString(`,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias0","default":1}}}`)
		rq := request(schemas(body.String()), "Dto")
		rq.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(rq)
		if count == 10 && err != nil {
			t.Fatal("finite alias chain rejected", err)
		}
		if count == 50 && (err == nil || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("child annotation bypassed alias depth limit", err)
		}
	}
}

func TestScalarReferenceCollectionChildDefaultCumulativeWork(t *testing.T) {
	for _, count := range []int{1, 18} {
		var body strings.Builder
		body.WriteString(`{"Leaf":{"type":"integer"}`)
		for i := 0; i < count; i++ {
			fmt.Fprintf(&body, `,"Holder%d":{"properties":{`, i)
			for j := 0; j < 180; j++ {
				if j != 0 {
					body.WriteByte(',')
				}
				fmt.Fprintf(&body, `"v%d":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":1}}`, j)
			}
			body.WriteString(`}}`)
		}
		body.WriteByte('}')
		got, err := core.GenerateWithDiagnostics(request(schemas(body.String()), "Dto"))
		if count == 1 && err != nil {
			t.Fatal("bounded independent defaults rejected", err)
		}
		if count == 18 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("independent defaults escaped cumulative source work", err)
		}
	}
}

func TestScalarReferenceCollectionChildDefaultFrozenNames(t *testing.T) {
	body := `{"Leaf":{"type":"integer"},"A":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}}},"B":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}}}}`
	use := `"$ref":"#/components/schemas/Leaf"`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation, rq.Options.ModelStyle = checked, style
			rq.Options.DeduplicateModels = true
			rq.Options.HoistSharedShapes = true
			rq.Options.MergeIdenticalModels = true
			rq.Options.LockHoistedNames = true
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil || len(control.Artifacts.NameLock) == 0 {
				t.Fatal("unannotated locked control rejected", checked, style, err)
			}
			rq.PreviousNameLock = control.Artifacts.NameLock
			rq.Document = schemas(strings.ReplaceAll(body, use, use+`,"default":2`))
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("annotation changed optimized output or frozen names", checked, style, err)
			}
		}
	}
}
