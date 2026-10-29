package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Keep the former refusal input as a positive case and prove that its child
// annotation is independent of the enclosing factory and generated output.
func inlineEnumChildGraduation(t *testing.T, rq core.GenerationRequest, annotation, invalid, pointer string) {
	t.Helper()
	document := string(rq.Document)
	if !strings.Contains(document, annotation) {
		t.Fatal("missing original child annotation", annotation)
	}
	rq.Document = []byte(strings.Replace(document, annotation, "", 1))
	control, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("unannotated control rejected", err)
	}
	rq.Document = []byte(document)
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(got, control) {
		t.Fatal("valid child changed generated output", err)
	}
	rq.Document = []byte(strings.Replace(document, annotation, invalid, 1))
	report, failure := core.GenerateWithDiagnostics(rq)
	ds, ok := failure.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
		*ds[0].Pointer != pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("invalid child default was not rejected at its source", failure)
	}
}

func TestInlineEnumCollectionChildDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, body, leaf, annotation string
		checked                      bool
	}{
		{
			"string list root",
			`{"Rows":{"type":"array","items":{"enum":["one","two"]},"default":["one"]}}`,
			`"enum":["one","two"]`, `"two"`, false,
		},
		{
			"checked string map field",
			`{"Holder":{"properties":{"words":{"additionalProperties":{"enum":["one","two","x"],"minLength":2},"default":{"key":"one"}}}}}`,
			`"enum":["one","two","x"],"minLength":2`, `"two"`, true,
		},
		{
			"checked nested root",
			`{"Rows":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2}},"default":[{"key":"one"}]}}`,
			`"enum":["one","two","x"],"minLength":2`, `"two"`, true,
		},
		{
			"borrowed checked element",
			`{"Owner":{"properties":{"rows":{"type":"array","items":{"enum":["one","x"],"minLength":2}}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/rows/items"}}`,
			`"enum":["one","x"],"minLength":2`, `"one"`, true,
		},
		{
			"native boolean map",
			`{"Rows":{"additionalProperties":{"type":"boolean","enum":[true,false]},"default":{"key":true}}}`,
			`"enum":[true,false]`, `false`, false,
		},
		{
			"checked native floating list",
			`{"Rows":{"type":"array","items":{"type":"number","format":"float","enum":[1.25,2.5],"minimum":1},"default":[1.25]}}`,
			`"enum":[1.25,2.5],"minimum":1`, `2.5`, true,
		},
		{
			"nullable string element",
			`{"Holder":{"properties":{"words":{"type":"array","items":{"type":["string","null"],"enum":["one","two"]}}}}}`,
			`"enum":["one","two"]`, `null`, false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := request(schemas(tc.body), "Dto")
			rq.Options.Validation = tc.checked
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("unannotated collection rejected", err)
			}
			rq.Document = schemas(strings.Replace(tc.body, tc.leaf, tc.leaf+`,"default":`+tc.annotation, 1))
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("independent inline child changed generated output", err)
			}
		})
	}
}

func TestInlineEnumCollectionChildDefaultContexts(t *testing.T) {
	for _, tc := range []struct {
		name, body, leaf, value string
		checked                 bool
	}{
		{
			"checked unique string list",
			`{"Rows":{"type":"array","uniqueItems":true,"items":{"enum":["one","two"],"minLength":2},"default":["one"]}}`,
			`"enum":["one","two"],"minLength":2`, `"two"`, true,
		},
		{
			"unique native list",
			`{"Rows":{"type":"array","uniqueItems":true,"items":{"type":"integer","enum":[1,2]},"default":[1]}}`,
			`"enum":[1,2]`, `2`, true,
		},
		{
			"forward reference through object factory",
			`{"Owner":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}},"default":{"rows":{"key":["one"]}}},"Rows":{"additionalProperties":{"type":"array","items":{"enum":["one","two"]}},"default":{"key":["one"]}}}`,
			`"enum":["one","two"]`, `"two"`, false,
		},
		{
			"forward borrowed checked leaf",
			`{"Alias":{"$ref":"#/components/schemas/Owner/properties/rows/items"},"Owner":{"properties":{"rows":{"type":"array","items":{"enum":["one","two"],"minLength":2}}}}}`,
			`"enum":["one","two"],"minLength":2`, `"one"`, true,
		},
		{
			"float membership before mapping",
			`{"Rows":{"type":"array","items":{"type":"number","format":"float","enum":[1.00000001,2.5]}}}`,
			`"enum":[1.00000001,2.5]`, `1.00000001`, true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := request(schemas(tc.body), "Dto")
			rq.Options.Validation = tc.checked
			control, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("unannotated context rejected", err)
			}
			rq.Document = schemas(strings.Replace(tc.body, tc.leaf, tc.leaf+`,"default":`+tc.value, 1))
			got, err := core.GenerateWithDiagnostics(rq)
			if ds, ok := err.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
				t.Fatalf("independent child rejected at %s: %v", *ds[0].Pointer, err)
			}
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("independent child changed context output", err)
			}
		})
	}
}

func TestInlineEnumCollectionChildDefaultOperations(t *testing.T) {
	doc := `{
	  "openapi":"3.1.0",
	  "paths":{"/rows":{"post":{
	    "requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"array","items":{"enum":["one","two"]}}}}},
	    "responses":{"200":{"content":{"application/json":{"schema":{"additionalProperties":{"type":"integer","enum":[1,2]}}}}}}
	  }}}
	}`
	for _, checked := range []bool{false, true} {
		rq := request([]byte(doc), "Dto")
		rq.Options.Validation = checked
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("unannotated operation rejected", checked, err)
		}
		annotated := strings.Replace(doc, `"enum":["one","two"]`, `"enum":["one","two"],"default":"one"`, 1)
		annotated = strings.Replace(annotated, `"enum":[1,2]`, `"enum":[1,2],"default":2`, 1)
		rq.Document = []byte(annotated)
		got, err := core.GenerateWithDiagnostics(rq)
		if ds, ok := err.(core.Diagnostics); ok && len(ds) > 0 && ds[0].Pointer != nil {
			t.Fatalf("operation child rejected at %s: %v", *ds[0].Pointer, err)
		}
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("operation child annotation changed output", checked, err)
		}
		for _, tc := range []struct{ input, pointer string }{
			{strings.Replace(doc, `"enum":["one","two"]`, `"enum":["one","two"],"default":"other"`, 1), "/paths/~1rows/post/requestBody/content/application~1json/schema/items/default"},
			{strings.Replace(doc, `"enum":[1,2]`, `"enum":[1,2],"default":3`, 1), "/paths/~1rows/post/responses/200/content/application~1json/schema/additionalProperties/default"},
		} {
			carrierDefaultFailure(t, []byte(tc.input), checked, "E_SCHEMA_INVALID", tc.pointer)
		}
	}
}

func TestInlineEnumCollectionChildDefaultBoundaries(t *testing.T) {
	for _, body := range []string{
		`{"Rows":{"type":"array","prefixItems":[{"enum":["one"],"default":"one"}],"items":false}}`,
		`{"Rows":{"type":"array","prefixItems":[{"enum":["one"]}],"items":{"enum":["one"],"default":"one"}}}`,
		`{"Rows":{"type":"object","properties":{"x":{"type":"string"}},"additionalProperties":{"enum":["one"],"default":"one"}}}`,
		`{"Rows":{"type":"array","items":{"type":"string","default":"one"}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("unsupported child role admitted", checked, body, err)
			}
		}
	}
	for _, outer := range []string{`{}`, `null`} {
		body := `{"Rows":{"nullable":true,"additionalProperties":{"enum":["one","two"],"default":"other"},"default":` + outer + `}}`
		carrierDefaultFailure(t, schemas(body), false, "E_SCHEMA_INVALID", "/components/schemas/Rows/additionalProperties/default")
	}
	body := `{"Holder":{"properties":{"a/~":{"type":"array","items":{"enum":["one"],"default":"other"}}}}}`
	carrierDefaultFailure(t, schemas(body), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/items/default")
	body = `{"Rows":{"type":"array","items":{"type":"number","format":"float","enum":[1.00000001],"default":1}}}`
	carrierDefaultFailure(t, schemas(body), true, "E_SCHEMA_INVALID", "/components/schemas/Rows/items/default")
}

func TestInlineEnumCollectionChildDefaultLimits(t *testing.T) {
	for _, size := range []int{32750, 32768} {
		word := strings.Repeat("a", size)
		body := `{"Rows":{"type":"array","items":{"enum":["` + word + `"],"default":"` + word + `"}}}`
		report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto"))
		if size == 32750 && err != nil {
			t.Fatal("default below 32 KiB rejected", err)
		}
		if size == 32768 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("default escaped 32 KiB limit", err)
		}
	}
	for _, count := range []int{10, 50} {
		var body strings.Builder
		body.WriteString(`{"Rows":`)
		for i := 0; i < count; i++ {
			body.WriteString(`{"type":"array","items":`)
		}
		body.WriteString(`{"enum":["one"],"default":"one"}`)
		for i := 0; i < count; i++ {
			body.WriteByte('}')
		}
		body.WriteByte('}')
		control := strings.Replace(body.String(), `,"default":"one"`, ``, 1)
		_, controlErr := core.GenerateWithDiagnostics(request(schemas(control), "Dto"))
		report, err := core.GenerateWithDiagnostics(request(schemas(body.String()), "Dto"))
		if count == 10 && (controlErr != nil || err != nil) {
			t.Fatal("bounded collection rejected", controlErr, err)
		}
		if count == 50 && (controlErr == nil || err == nil || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("annotation bypassed depth limit", controlErr, err)
		}
	}
	for _, count := range []int{1, 18} {
		var body strings.Builder
		body.WriteByte('{')
		for i := 0; i < count; i++ {
			if i != 0 {
				body.WriteByte(',')
			}
			fmt.Fprintf(&body, `"Holder%d":{"properties":{`, i)
			for j := 0; j < 180; j++ {
				if j != 0 {
					body.WriteByte(',')
				}
				fmt.Fprintf(&body, `"v%d":{"type":"array","items":{"enum":["one"],"default":"one"}}`, j)
			}
			body.WriteString(`}}`)
		}
		body.WriteByte('}')
		report, err := core.GenerateWithDiagnostics(request(schemas(body.String()), "Dto"))
		if count == 1 && err != nil {
			t.Fatal("bounded defaults rejected", err)
		}
		if count == 18 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("defaults escaped cumulative work limit", err)
		}
	}
}

func TestInlineEnumCollectionChildDefaultFrozenNames(t *testing.T) {
	field := `{"type":"array","items":{"enum":["one","two"]}}`
	op := `{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":` + op + `,"/b":` + op + `}}`)
	for _, checked := range []bool{false, true} {
		rq := request(doc, "Dto")
		rq.Options.Validation = checked
		rq.Options.HoistSharedShapes = true
		rq.Options.LockHoistedNames = true
		rq.Options.DeduplicateModels = true
		rq.Options.MergeIdenticalModels = true
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || len(control.Artifacts.NameLock) == 0 {
			t.Fatal("unannotated locked generation", checked, err)
		}
		rq.PreviousNameLock = control.Artifacts.NameLock
		rq.Document = []byte(strings.ReplaceAll(string(doc), `"enum":["one","two"]`, `"enum":["one","two"],"default":"one"`))
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("independent child changed frozen identity", checked, err)
		}
	}
}

func TestInlineEnumCollectionChildDefaultValidation(t *testing.T) {
	for _, tc := range []struct {
		name, leaf, value, pointer string
		checked                    bool
	}{
		{"string membership", `{"enum":["one","two"]}`, `"future"`, "/components/schemas/Rows/items/default", false},
		{"string kind", `{"enum":["one","two"]}`, `1`, "/components/schemas/Rows/items/default", false},
		{"non-null string", `{"enum":["one","two"]}`, `null`, "/components/schemas/Rows/items/default", false},
		{"checked string", `{"enum":["one","x"],"minLength":2}`, `"x"`, "/components/schemas/Rows/items/default", true},
		{"native membership", `{"type":"integer","enum":[1,2]}`, `3`, "/components/schemas/Rows/items/default", false},
		{"native bounds", `{"type":"integer","enum":[1,2],"minimum":2}`, `1`, "/components/schemas/Rows/items/default", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := strings.TrimSuffix(tc.leaf, "}") + `,"default":` + tc.value + `}`
			for _, outer := range []string{"", `,"default":[]`, `,"nullable":true,"default":null`} {
				body := `{"Rows":{"type":"array","items":` + child + outer + `}}`
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = tc.checked
				report, failure := core.GenerateWithDiagnostics(rq)
				ds, ok := failure.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
					*ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("invalid child default lost validation or its source pointer", outer, failure)
				}
			}
		})
	}
}
