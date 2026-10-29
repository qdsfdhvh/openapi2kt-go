package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCollectionReferenceChildDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, leaf, shape, annotation string
		checked                       bool
	}{
		{"string list", `{"enum":["one","two"]}`, `{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}`, `"one"`, false},
		{"checked string map", `{"type":"string","enum":["one","two"],"minLength":2}`, `{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}}`, `"one"`, true},
		{"integer list", `{"type":"integer","enum":[1,2]}`, `{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}`, `1`, false},
		{"nullable string", `{"enum":["one","two"]}`, `{"type":"array","items":{"$ref":"#/components/schemas/Leaf","nullable":true}}`, `null`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := `{"Leaf":` + tc.leaf + `,"Rows":` + tc.shape + `}`
			with := strings.Replace(tc.shape, `"$ref":"#/components/schemas/Leaf"`, `"$ref":"#/components/schemas/Leaf","default":`+tc.annotation, 1)
			rq := request(schemas(base), "Dto")
			rq.Options.Validation = tc.checked
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("unannotated source rejected", err)
			}
			rq.Document = schemas(`{"Leaf":` + tc.leaf + `,"Rows":` + with + `}`)
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("valid child annotation rejected", err)
			}
			if !reflect.DeepEqual(got, control) {
				t.Fatal("child annotation changed generated artifacts or warnings")
			}
		})
	}
}

func TestCollectionReferenceChildDefaultValidation(t *testing.T) {
	for _, tc := range []struct {
		name, leaf, child, value, code string
		checked                        bool
	}{
		{"string membership", `{"enum":["one","two"]}`, `{"$ref":"#/components/schemas/Leaf"}`, `"future"`, "E_SCHEMA_INVALID", false},
		{"string kind", `{"enum":["one","two"]}`, `{"$ref":"#/components/schemas/Leaf"}`, `1`, "E_SCHEMA_INVALID", false},
		{"null without nullable", `{"enum":["one","two"]}`, `{"$ref":"#/components/schemas/Leaf"}`, `null`, "E_SCHEMA_INVALID", false},
		{"checked length", `{"type":"string","enum":["one","x"],"minLength":2}`, `{"$ref":"#/components/schemas/Leaf"}`, `"x"`, "E_SCHEMA_INVALID", true},
		{"native membership", `{"type":"integer","enum":[1,2]}`, `{"$ref":"#/components/schemas/Leaf"}`, `3`, "E_SCHEMA_INVALID", false},
		{"native kind", `{"type":"integer","enum":[1,2]}`, `{"$ref":"#/components/schemas/Leaf"}`, `"1"`, "E_SCHEMA_INVALID", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := strings.TrimSuffix(tc.child, "}") + `,"default":` + tc.value + `}`
			for _, outer := range []string{"", `,"default":[]`} {
				body := `{"Leaf":` + tc.leaf + `,"Rows":{"type":"array","items":` + child + outer + `}}`
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = tc.checked
				got, err := core.GenerateWithDiagnostics(rq)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil ||
					*ds[0].Pointer != "/components/schemas/Rows/items/default" ||
					!reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("invalid child annotation lost its original location", outer, err)
				}
			}
		})
	}
}

func TestCollectionReferenceChildDefaultIndependentSources(t *testing.T) {
	base := `{"Leaf":{"enum":["one","two"],"default":"one"},"Alias":{"$ref":"#/components/schemas/Leaf","default":"two"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":["one"]},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"},"unannotated":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}}}`
	annotated := strings.Replace(base, `"items":{"$ref":"#/components/schemas/Alias"}`, `"items":{"$ref":"#/components/schemas/Alias","default":"two"}`, 2)
	for _, checked := range []bool{false, true} {
		rq := request(schemas(base), "Dto")
		rq.Options.Validation = checked
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("independent unannotated control", err)
		}
		rq.Document = schemas(annotated)
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("child annotations inherited or changed source output", checked, err)
		}
		rq.Document = schemas(strings.Replace(annotated, `"default":"two"`, `"default":"future"`, 1))
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalid independent alias default was ignored", checked, err)
		}
	}
}

func TestCollectionReferenceChildDefaultContexts(t *testing.T) {
	for _, tc := range []struct {
		name, body, source, annotation string
		checked                        bool
	}{
		{
			"forward lexical alias in nested list and map",
			`{"Rows":{"type":"array","items":{"additionalProperties":{"$ref":"#/components/schemas/Alias"}},"default":[{"a":"one"}]},"Alias":{"$ref":"#/components/schemas/Word","default":"two"},"Word":{"enum":["one","two"],"default":"one"}}`,
			`"$ref":"#/components/schemas/Alias"`, `"two"`, false,
		},
		{
			"checked lexical alias and uniqueness",
			`{"Rows":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Alias"},"default":["one"]},"Alias":{"$ref":"#/components/schemas/Word"},"Word":{"enum":["one","two"],"minLength":2}}`,
			`"$ref":"#/components/schemas/Alias"`, `"two"`, true,
		},
		{
			"nullable alias narrowed at the item",
			`{"Word":{"type":["string","null"],"enum":["one","two"]},"Alias":{"$ref":"#/components/schemas/Word","nullable":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":false}}}`,
			`"$ref":"#/components/schemas/Alias","nullable":false`, `"one"`, false,
		},
		{
			"nullable item under non-null alias",
			`{"Word":{"enum":["one","two"]},"Alias":{"$ref":"#/components/schemas/Word","nullable":false},"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Alias","nullable":true}}}`,
			`"$ref":"#/components/schemas/Alias","nullable":true`, `null`, false,
		},
		{
			"native boolean map",
			`{"Word":{"type":"boolean","enum":[true,false]},"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Word"},"default":{"a":true}}}`,
			`"$ref":"#/components/schemas/Word"`, `false`, false,
		},
		{
			"native double list",
			`{"Word":{"type":"number","format":"double","enum":[1.25,2.5]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Word"}}}`,
			`"$ref":"#/components/schemas/Word"`, `2.5`, true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := request(schemas(tc.body), "Dto")
			rq.Options.Validation = tc.checked
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("original reference context rejected", err)
			}
			rq.Document = schemas(strings.Replace(tc.body, tc.source, tc.source+`,"default":`+tc.annotation, 1))
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				if ds, ok := err.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
					t.Fatalf("independent child annotation rejected at %s: %v", *ds[0].Pointer, err)
				}
				t.Fatal("independent child annotation rejected", err)
			}
			if !reflect.DeepEqual(got, control) {
				t.Fatal("child annotation changed generated artifacts or warnings")
			}
		})
	}
}

func TestCollectionReferenceChildDefaultInObjectDefaultGraph(t *testing.T) {
	for _, tc := range []struct{ name, owner string }{
		{"component factory", `"Owner":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}},"default":{"rows":{"key":["one"]}}}`},
		{"field factory", `"Owner":{"properties":{"inner":{"type":"object","properties":{"rows":{"$ref":"#/components/schemas/Rows"}},"default":{"rows":{"key":["one"]}}}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Word":{"enum":["one","two"],"default":"one"},"Alias":{"$ref":"#/components/schemas/Word","default":"two"},"Rows":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"default":{"key":["one"]}},` + tc.owner + `}`
			source := `"$ref":"#/components/schemas/Alias"`
			for _, checked := range []bool{false, true} {
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = checked
				control, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("unannotated object default graph rejected", checked, err)
				}
				rq.Document = schemas(strings.Replace(body, source, source+`,"default":"two"`, 1))
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("independent child changed object default graph", checked, err)
				}
				rq.Document = schemas(strings.Replace(body, source, source+`,"default":"future"`, 1))
				report, err := core.GenerateWithDiagnostics(rq)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
					*ds[0].Pointer != "/components/schemas/Rows/additionalProperties/items/default" ||
					!reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("object graph ignored invalid child annotation", checked, err)
				}
			}
		})
	}
}

func TestCollectionReferenceChildDefaultInOperations(t *testing.T) {
	doc := `{
	  "openapi":"3.1.0",
	  "components":{"schemas":{"Word":{"enum":["one","two"]}}},
	  "paths":{"/rows":{"post":{
	    "requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"array","items":{"$ref":"#/components/schemas/Word"}}}}},
	    "responses":{"200":{"content":{"application/json":{"schema":{"additionalProperties":{"$ref":"#/components/schemas/Word"}}}}}}
	  }}}
	}`
	for _, checked := range []bool{false, true} {
		rq := request([]byte(doc), "Dto")
		rq.Options.Validation = checked
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("unannotated operation rejected", checked, err)
		}
		source := `"$ref":"#/components/schemas/Word"`
		annotated := strings.Replace(doc, source, source+`,"default":"one"`, 2)
		rq.Document = []byte(annotated)
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("operation child default changed generated output", checked, err)
		}
		for _, tc := range []struct{ doc, pointer string }{
			{strings.Replace(doc, source, source+`,"default":"future"`, 1), "/paths/~1rows/post/requestBody/content/application~1json/schema/items/default"},
			{strings.Replace(doc, `"additionalProperties":{`+source, `"additionalProperties":{`+source+`,"default":"future"`, 1), "/paths/~1rows/post/responses/200/content/application~1json/schema/additionalProperties/default"},
		} {
			rq.Document = []byte(tc.doc)
			report, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid operation child default was ignored", checked, err)
			}
		}
	}
}

func TestCollectionReferenceChildDefaultDoesNotGrantOtherRoles(t *testing.T) {
	// Preserve the formerly rejected scalar example as a positive, and verify
	// that an invalid value still fails at the original child annotation.
	scalar := `{"Leaf":{"type":"string"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(scalar), "Dto")
		rq.Options.Validation = checked
		inlineBuiltinChildGraduation(t, rq, `,"default":"one"`, `,"default":true`,
			"/components/schemas/Rows/items/default")
	}
	// Preserve the exact former enum-reference collection refusal as a
	// positive; a word outside the original vocabulary still fails.
	collection := `{"Leaf":{"enum":["one"]},"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Inner","default":["one"]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(collection), "Dto")
		rq.Options.Validation = checked
		inlineBuiltinChildGraduation(t, rq, `,"default":["one"]`, `,"default":["future"]`,
			"/components/schemas/Rows/items/default/0")
	}
	for _, tc := range []struct{ name, body string }{
		{"tuple member", `{"Leaf":{"enum":["one"]},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Leaf","default":"one"}],"items":false}}`},
		{"tuple tail", `{"Leaf":{"enum":["one"]},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Leaf"}],"items":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`},
		{"discarded extra property", `{"Leaf":{"enum":["one"]},"Rows":{"type":"object","properties":{"x":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":"one"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(tc.body), "Dto")
				rq.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(rq)
				if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("unsupported child role was admitted", checked, err)
				}
			}
		})
	}
}

func TestCollectionReferenceChildDefaultBeforeEmptyOrNullContainer(t *testing.T) {
	for _, value := range []string{`{}`, `null`} {
		body := `{"Word":{"enum":["one","two"]},"Rows":{"nullable":true,"additionalProperties":{"$ref":"#/components/schemas/Word","default":"future"},"default":` + value + `}}`
		for _, checked := range []bool{false, true} {
			carrierDefaultFailure(t, schemas(body), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows/additionalProperties/default")
		}
	}
	body := `{"Word":{"enum":["one","two"]},"Holder":{"properties":{"a/~":{"type":"array","items":{"$ref":"#/components/schemas/Word","default":"future"}}}}}`
	carrierDefaultFailure(t, schemas(body), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/items/default")
}

func TestCollectionReferenceChildDefaultKeepsSourceValidation(t *testing.T) {
	for _, outer := range []string{`[]`, `null`} {
		body := `{"Word":{"enum":["one","one"]},"Rows":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Word","default":"one"},"default":` + outer + `}}`
		carrierDefaultFailure(t, schemas(body), false, "E_SCHEMA_INVALID", "/components/schemas/Word/enum/1")
	}
	for _, nullable := range []string{``, `,"nullable":false`} {
		body := `{"Word":{"type":["string","null"],"enum":["one","two"]},"Alias":{"$ref":"#/components/schemas/Word","nullable":false},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"` + nullable + `,"default":null}}}`
		carrierDefaultFailure(t, schemas(body), false, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default")
	}
}

func TestCollectionReferenceChildDefaultByteBudget(t *testing.T) {
	for _, size := range []int{32750, 32768} {
		word := strings.Repeat("a", size)
		body := `{"Word":{"enum":["` + word + `"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Word","default":"` + word + `"}}}`
		rq := request(schemas(body), "Dto")
		report, err := core.GenerateWithDiagnostics(rq)
		if size == 32750 && err != nil {
			t.Fatal("default below 32 KiB rejected", err)
		}
		if size == 32768 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("default escaped the 32 KiB budget", err)
		}
	}
}

func TestCollectionReferenceChildDefaultAliasDepth(t *testing.T) {
	for _, count := range []int{10, 50} {
		var body strings.Builder
		body.WriteString(`{"Word":{"enum":["one"]}`)
		for i := 0; i < count; i++ {
			target := "Word"
			if i+1 < count {
				target = fmt.Sprintf("Alias%d", i+1)
			}
			fmt.Fprintf(&body, `,"Alias%d":{"$ref":"#/components/schemas/%s"}`, i, target)
		}
		body.WriteString(`,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias0"}}}`)
		controlRequest := request(schemas(body.String()), "Dto")
		control, controlErr := core.GenerateWithDiagnostics(controlRequest)
		annotated := strings.Replace(body.String(), `"$ref":"#/components/schemas/Alias0"`, `"$ref":"#/components/schemas/Alias0","default":"one"`, 1)
		got, err := core.GenerateWithDiagnostics(request(schemas(annotated), "Dto"))
		if count == 10 && (controlErr != nil || err != nil || !reflect.DeepEqual(control, got)) {
			t.Fatal("finite alias chain changed", controlErr, err)
		}
		if count == 50 && (controlErr == nil || err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("annotation bypassed alias depth limit", controlErr, err)
		}
	}
}

func TestCollectionReferenceChildDefaultCumulativeWork(t *testing.T) {
	for _, count := range []int{1, 18} {
		var body strings.Builder
		body.WriteString(`{"Word":{"enum":["one"]}`)
		for i := 0; i < count; i++ {
			fmt.Fprintf(&body, `,"Holder%d":{"properties":{`, i)
			for j := 0; j < 180; j++ {
				if j != 0 {
					body.WriteByte(',')
				}
				fmt.Fprintf(&body, `"v%d":{"type":"array","items":{"$ref":"#/components/schemas/Word","default":"one"}}`, j)
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
