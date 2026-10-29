package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedEnumCollectionReferenceChildDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, leaf, first, second, invalid string
	}{
		{"string", `"type":"string","enum":["one","two"]`, `"one"`, `"two"`, `"future"`},
		{"integer", `"type":"integer","enum":[1,2]`, `1`, `2`, `3`},
		{"number", `"type":"number","enum":[1.0,2.0]`, `1.0`, `2.0`, `3.0`},
		{"boolean", `"type":"boolean","enum":[true,false]`, `true`, `false`, `0`},
	} {
		for _, parent := range []struct{ name, schema, location string }{
			{"list", `"type":"array","items":`, "/items"},
			{"map", `"additionalProperties":`, "/additionalProperties"},
		} {
			t.Run(tc.name+"/"+parent.name, func(t *testing.T) {
				body := `{"Word":{` + tc.leaf + `,"default":` + tc.first + `},` +
					`"WordAlias":{"$ref":"#/components/schemas/Word","default":` + tc.second + `},` +
					`"Values":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias","default":` + tc.first + `},"default":[` + tc.first + `]},` +
					`"ValuesAlias":{"$ref":"#/components/schemas/Values","default":[` + tc.second + `]},` +
					`"Rows":{` + parent.schema + `{"$ref":"#/components/schemas/ValuesAlias","default":[` + tc.first + `]}}}`
				annotation := `,"default":[` + tc.first + `]}}}`
				for _, checked := range []bool{false, true} {
					for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
						rq := request(schemas(strings.Replace(body, annotation, `}}}`, 1)), "Dto")
						rq.Options.Validation, rq.Options.ModelStyle = checked, style
						control, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal("unannotated control rejected", checked, style, err)
						}
						rq.Document = schemas(body)
						annotated, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal("enum-reference collection child rejected", checked, style, err)
						}
						if !reflect.DeepEqual(annotated, control) {
							t.Fatal("child annotation changed generated output", checked, style)
						}
						invalid := strings.Replace(body, annotation, `,"default":[`+tc.invalid+`]}}}`, 1)
						carrierDefaultFailure(t, schemas(invalid), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows"+parent.location+"/default/0")
					}
				}
			})
		}
	}
}

func TestReferencedEnumCollectionReferenceChildNullableAndChecks(t *testing.T) {
	body := `{"Word":{"type":"string","enum":["one","x"],"minLength":2},` +
		`"Values":{"type":"array","nullable":true,"minItems":1,"items":{"$ref":"#/components/schemas/Word"}},` +
		`"Alias":{"$ref":"#/components/schemas/Values","nullable":false},` +
		`"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("nullable referenced enum collection child rejected", err)
	}
	for _, tc := range []struct{ before, after, pointer string }{
		{`"nullable":true,"default":null`, `"nullable":false,"default":null`, "/components/schemas/Rows/additionalProperties/default"},
		{`"default":null`, `"default":[]`, "/components/schemas/Rows/additionalProperties/default"},
		{`"default":null`, `"default":["x"]`, "/components/schemas/Rows/additionalProperties/default/0"},
		{`"default":null`, `"default":["future"]`, "/components/schemas/Rows/additionalProperties/default/0"},
	} {
		carrierDefaultFailure(t, schemas(strings.Replace(body, tc.before, tc.after, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
	}
	rq.Document = schemas(strings.Replace(body, `"default":null`, `"default":["one"]`, 1))
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("valid checked referenced enum collection child rejected", err)
	}
	rq.Options.Validation = false
	if _, err := core.GenerateWithDiagnostics(rq); err == nil {
		t.Fatal("checked vocabulary admitted without validation")
	}
}

func TestReferencedNullableNativeEnumCollectionReferenceChild(t *testing.T) {
	body := `{"Code":{"type":"integer","nullable":true,"enum":[1,2],"default":null},` +
		`"Values":{"type":"array","items":{"$ref":"#/components/schemas/Code","default":null},"default":[null,1]},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":[null,2]}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	annotated, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("nullable native enum collection child rejected", err)
	}
	rq.Document = schemas(strings.Replace(body, `,"default":[null,2]`, "", 1))
	control, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(annotated, control) {
		t.Fatal("nullable native enum child changed output", err)
	}
	carrierDefaultFailure(t, schemas(strings.Replace(body, `"default":[null,2]`, `"default":[null,3]`, 1)),
		true, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default/1")
}

func TestReferencedEnumCollectionReferenceChildNestedAndIndependentAnnotations(t *testing.T) {
	body := `{"AOuter":{"type":"array","items":{"$ref":"#/components/schemas/BAlias","default":{"child":["two"]}}},` +
		`"BAlias":{"$ref":"#/components/schemas/CValues","default":{"alias":["one"]}},` +
		`"CValues":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/DWord","default":"one"}},"default":{"root":["two"]}},` +
		`"DWord":{"type":"string","enum":["one","two"],"default":"one"}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		annotated, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("nested enum reference annotations rejected", checked, err)
		}
		rq.Document = schemas(strings.Replace(body, `,"default":{"child":["two"]}`, "", 1))
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(annotated, control) {
			t.Fatal("nested child changed generated output", checked, err)
		}
		for _, tc := range []struct{ before, after, pointer string }{
			{`"default":{"child":["two"]}`, `"default":{"child":["future"]}`, "/components/schemas/AOuter/items/default/child/0"},
			{`"default":{"alias":["one"]}`, `"default":{"alias":["future"]}`, "/components/schemas/BAlias/default/alias/0"},
			{`"default":{"root":["two"]}`, `"default":{"root":["future"]}`, "/components/schemas/CValues/default/root/0"},
			{`"default":"one"`, `"default":"future"`, "/components/schemas/CValues/additionalProperties/items/default"},
		} {
			carrierDefaultFailure(t, schemas(strings.Replace(body, tc.before, tc.after, 1)), checked, "E_SCHEMA_INVALID", tc.pointer)
		}
	}
}

func TestReferencedEnumCollectionReferenceChildEmptyAcrossContainers(t *testing.T) {
	body := `{"Word":{"enum":["one"]},` +
		`"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"}},` +
		`"Values":{"type":"array","items":{"$ref":"#/components/schemas/Inner"}},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":[]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		inlineBuiltinChildGraduation(t, rq, `,"default":[]`, `,"default":[["future"]]`,
			"/components/schemas/Rows/items/default/0/0")
	}
}

func TestReferencedEnumCollectionReferenceChildAcrossNamedContainers(t *testing.T) {
	body := `{"Word":{"type":"string","enum":["one","two"]},` +
		`"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"}},` +
		`"Values":{"type":"array","items":{"$ref":"#/components/schemas/Inner"}},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":[["one"]]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(strings.Replace(body, `,"default":[["one"]]`, "", 1)), "Dto")
		rq.Options.Validation = checked
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("unannotated inter-container control rejected", checked, err)
		}
		rq.Document = schemas(body)
		annotated, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("valid inter-container child default rejected", checked, err)
		}
		if !reflect.DeepEqual(annotated, control) {
			t.Fatal("child annotation changed generated output", checked)
		}
		carrierDefaultFailure(t, schemas(strings.Replace(body, `[["one"]]`, `[["future"]]`, 1)),
			checked, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default/0/0")
	}
}

func TestReferencedEnumCollectionReferenceChildAcrossContainersIndependentDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, leaf, first, second, invalid string
	}{
		{"string", `"type":"string","enum":["one","two"]`, `"one"`, `"two"`, `"future"`},
		{"integer", `"type":"integer","enum":[1,2]`, `1`, `2`, `3`},
		{"boolean", `"type":"boolean","enum":[true,false]`, `true`, `false`, `0`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Word":{` + tc.leaf + `,"default":` + tc.first + `},` +
				`"WordAlias":{"$ref":"#/components/schemas/Word","default":` + tc.second + `},` +
				`"Inner":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias","default":` + tc.first + `},"default":[` + tc.second + `]},` +
				`"InnerAlias":{"$ref":"#/components/schemas/Inner","default":[` + tc.first + `]},` +
				`"Values":{"additionalProperties":{"$ref":"#/components/schemas/InnerAlias","default":[` + tc.second + `]},"default":{"root":[` + tc.first + `]}},` +
				`"ValuesAlias":{"$ref":"#/components/schemas/Values","default":{"alias":[` + tc.second + `]}},` +
				`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias","default":{"child":[` + tc.first + `]}}}}`
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					rq := request(schemas(strings.Replace(body, `,"default":{"child":[`+tc.first+`]}`, "", 1)), "Dto")
					rq.Options.Validation, rq.Options.ModelStyle = checked, style
					control, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal("unannotated control rejected", checked, style, err)
					}
					rq.Document = schemas(body)
					annotated, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(annotated, control) {
						t.Fatal("independent nested child changed output", checked, style, err)
					}
					for _, sample := range []struct{ before, after, pointer string }{
						{`"default":{"child":[` + tc.first + `]}`, `"default":{"child":[` + tc.invalid + `]}`, "/components/schemas/Rows/items/default/child/0"},
						{`"default":{"alias":[` + tc.second + `]}`, `"default":{"alias":[` + tc.invalid + `]}`, "/components/schemas/ValuesAlias/default/alias/0"},
						{`"default":{"root":[` + tc.first + `]}`, `"default":{"root":[` + tc.invalid + `]}`, "/components/schemas/Values/default/root/0"},
						{`"default":[` + tc.second + `]`, `"default":[` + tc.invalid + `]`, "/components/schemas/Inner/default/0"},
					} {
						carrierDefaultFailure(t, schemas(strings.Replace(body, sample.before, sample.after, 1)),
							checked, "E_SCHEMA_INVALID", sample.pointer)
					}
				}
			}
		})
	}
}

func TestReferencedEnumCollectionReferenceChildAcrossContainersRejectsInternalAndCycles(t *testing.T) {
	for _, body := range []string{
		`{"Word":{"enum":["one"]},"Owner":{"properties":{"inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"}}}},"Values":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/inner"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":[]}}}`,
		`{"A":{"type":"array","items":{"$ref":"#/components/schemas/B","default":[]}},"B":{"type":"array","items":{"$ref":"#/components/schemas/A","default":[]}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("unsupported internal or cyclic path emitted artifacts", checked, err)
			}
		}
	}
}

func TestReferencedEnumCollectionReferenceChildAcrossContainersNullable(t *testing.T) {
	body := `{"Word":{"type":"string","enum":["one","two"],"minLength":2},` +
		`"Inner":{"type":"array","nullable":true,"minItems":1,"items":{"$ref":"#/components/schemas/Word"}},` +
		`"InnerAlias":{"$ref":"#/components/schemas/Inner","nullable":false},` +
		`"Values":{"additionalProperties":{"$ref":"#/components/schemas/InnerAlias","nullable":true}},` +
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":{"child":null}}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	annotated, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("nullable intermediate child rejected", err)
	}
	rq.Document = schemas(strings.Replace(body, `,"default":{"child":null}`, "", 1))
	control, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(annotated, control) {
		t.Fatal("nullable intermediate annotation changed output", err)
	}
	for _, tc := range []struct{ before, after, pointer string }{
		{`"nullable":true}},"Rows"`, `"nullable":false}},"Rows"`, "/components/schemas/Rows/items/default/child"},
		{`"default":{"child":null}`, `"default":{"child":[]}`, "/components/schemas/Rows/items/default/child"},
		{`"default":{"child":null}`, `"default":{"child":["future"]}`, "/components/schemas/Rows/items/default/child/0"},
	} {
		carrierDefaultFailure(t, schemas(strings.Replace(body, tc.before, tc.after, 1)),
			true, "E_SCHEMA_INVALID", tc.pointer)
	}
}

func TestReferencedEnumCollectionReferenceChildSourceLimit(t *testing.T) {
	for _, tc := range []struct{ name, schemas, value string }{
		{"direct", `"Word":{"enum":["one"]},"Values":{"type":"array","items":{"$ref":"#/components/schemas/Word"}}`, `["one"]`},
		{"intermediate", `"Word":{"enum":["one"]},"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"}},"Values":{"type":"array","items":{"$ref":"#/components/schemas/Inner"}}`, `[["one"]]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, count := range []int{1, 18} {
				var body strings.Builder
				body.WriteString(`{` + tc.schemas)
				for i := 0; i < count; i++ {
					fmt.Fprintf(&body, `,"Holder%d":{"properties":{`, i)
					for j := 0; j < 180; j++ {
						if j != 0 {
							body.WriteByte(',')
						}
						fmt.Fprintf(&body, `"v%d":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":%s}}`, j, tc.value)
					}
					body.WriteString(`}}`)
				}
				body.WriteByte('}')
				report, err := core.GenerateWithDiagnostics(request(schemas(body.String()), "Dto"))
				if count == 1 && err != nil {
					t.Fatal("bounded enum reference children rejected", err)
				}
				if count == 18 {
					ds, ok := err.(core.Diagnostics)
					if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
						t.Fatal("enum reference children escaped cumulative limit", err)
					}
				}
			}
		})
	}
}
