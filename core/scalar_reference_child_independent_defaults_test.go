package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestScalarReferenceChildAndComponentDefaultsAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, components, use, child string
		checkedOnly                  bool
	}{
		{
			"direct integer",
			`"Leaf":{"type":"integer","default":1}`,
			`"$ref":"#/components/schemas/Leaf"`, `2`, false,
		},
		{
			"integer alias only",
			`"Leaf":{"type":"integer"},"Alias":{"$ref":"#/components/schemas/Leaf","default":1}`,
			`"$ref":"#/components/schemas/Alias"`, `2`, true,
		},
		{
			"integer alias chain",
			`"Leaf":{"type":"integer","default":1},"Middle":{"$ref":"#/components/schemas/Leaf","default":2},"Alias":{"$ref":"#/components/schemas/Middle","default":3}`,
			`"$ref":"#/components/schemas/Alias"`, `4`, true,
		},
		{
			"nullable target and child",
			`"Leaf":{"type":["integer","null"],"default":null},"Alias":{"$ref":"#/components/schemas/Leaf","nullable":false,"default":1}`,
			`"$ref":"#/components/schemas/Alias","nullable":true`, `null`, true,
		},
		{
			"string",
			`"Leaf":{"type":"string","default":"root"}`,
			`"$ref":"#/components/schemas/Leaf"`, `"child"`, false,
		},
		{
			"boolean",
			`"Leaf":{"type":"boolean","default":true}`,
			`"$ref":"#/components/schemas/Leaf"`, `false`, false,
		},
		{
			"long",
			`"Leaf":{"type":"integer","format":"int64","default":1}`,
			`"$ref":"#/components/schemas/Leaf"`, `2`, false,
		},
		{
			"float",
			`"Leaf":{"type":"number","format":"float","default":1.5}`,
			`"$ref":"#/components/schemas/Leaf"`, `2.5`, false,
		},
		{
			"double",
			`"Leaf":{"type":"number","format":"double","default":1.5}`,
			`"$ref":"#/components/schemas/Leaf"`, `2.5`, false,
		},
		{
			"uuid",
			`"Leaf":{"type":"string","format":"uuid","default":"123e4567-e89b-12d3-a456-426614174000"}`,
			`"$ref":"#/components/schemas/Leaf"`, `"123e4567-e89b-12d3-a456-426614174001"`, false,
		},
		{
			"date alias",
			`"Leaf":{"type":"string","format":"date","default":"2024-02-29"},"Alias":{"$ref":"#/components/schemas/Leaf","default":"2024-03-01"}`,
			`"$ref":"#/components/schemas/Alias"`, `"2024-03-02"`, false,
		},
		{
			"instant",
			`"Leaf":{"type":"string","format":"date-time","default":"2024-03-01T12:00:00Z"}`,
			`"$ref":"#/components/schemas/Leaf"`, `"2024-03-02T12:00:00Z"`, false,
		},
		{
			"bytes",
			`"Leaf":{"type":"string","format":"byte","default":[1,2]}`,
			`"$ref":"#/components/schemas/Leaf"`, `[3,4]`, false,
		},
		{
			"binary",
			`"Leaf":{"type":"string","format":"binary","default":[1,2]}`,
			`"$ref":"#/components/schemas/Leaf"`, `[3,4]`, false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, shape := range []string{
				`"Rows":{"type":"array","items":{` + tc.use + `}}`,
				`"Rows":{"type":"object","additionalProperties":{` + tc.use + `}}`,
			} {
				for _, checked := range []bool{false, true} {
					if tc.checkedOnly && !checked {
						continue
					}
					for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
						body := `{` + tc.components + `,` + shape + `}`
						rq := request(schemas(body), "Dto")
						rq.Options.Validation, rq.Options.ModelStyle = checked, style
						control, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal("unannotated collection with independent factories rejected", checked, style, err)
						}
						rq.Document = schemas(strings.Replace(body, tc.use, tc.use+`,"default":`+tc.child, 1))
						got, err := core.GenerateWithDiagnostics(rq)
						if err != nil || !reflect.DeepEqual(got, control) {
							t.Fatal("child annotation changed independent factories or collection output", checked, style, err)
						}
					}
				}
			}
		})
	}
}

func TestScalarReferenceChildAndComponentDefaultsRejectInvalidOwners(t *testing.T) {
	for _, tc := range []struct {
		name, body, pointer string
	}{
		{"target", `{"Leaf":{"type":"integer","default":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":2}}}`, "/components/schemas/Leaf/default"},
		{"alias", `{"Leaf":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Leaf","default":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":2}}}`, "/components/schemas/Alias/default"},
		{"child", `{"Leaf":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Leaf","default":2},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":true}}}`, "/components/schemas/Rows/items/default"},
		{"invalid date target", `{"Leaf":{"type":"string","format":"date","default":"2023-02-29"},"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Leaf","default":"2024-02-29"}}}`, "/components/schemas/Leaf/default"},
		{"invalid byte child", `{"Leaf":{"type":"string","format":"byte","default":[1]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":[128]}}}`, "/components/schemas/Rows/items/default/0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := request(schemas(tc.body), "Dto")
			rq.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid annotation lost its own source or returned artifacts", err)
			}
		})
	}
}

func TestScalarReferenceChildIndependentFactoryBudgets(t *testing.T) {
	body := `{"Leaf":{"type":"string","default":"` + strings.Repeat("x", 32769) + `"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"child"}}}`
	report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto"))
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil ||
		*ds[0].Pointer != "/components/schemas/Leaf/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("target factory escaped its own default budget", err)
	}
}

func TestScalarReferenceChildIndependentFactoryCumulativeBudget(t *testing.T) {
	for _, count := range []int{2, 270} {
		var body strings.Builder
		body.WriteString(`{"Leaf":{"type":"string","default":"` + strings.Repeat("x", 32750) + `"}`)
		for i := 0; i < count; i++ {
			fmt.Fprintf(&body, `,"Rows%d":{"type":"array","items":{"$ref":"#/components/schemas/Leaf","default":"child"}}`, i)
		}
		body.WriteByte('}')
		report, err := core.GenerateWithDiagnostics(request(schemas(body.String()), "Dto"))
		if count == 2 && err != nil {
			t.Fatal("bounded independent factory uses rejected", err)
		}
		if count == 270 {
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("repeated factory captures escaped cumulative byte budget", err)
			}
		}
	}
}

func TestScalarReferenceChildIndependentFactoriesKeepOptimizedNames(t *testing.T) {
	body := `{"Leaf":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Leaf","default":2},"A":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"B":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}}}`
	use := `"$ref":"#/components/schemas/Alias"`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(schemas(body), "Dto")
		rq.Options.ModelStyle = style
		rq.Options.Validation = true
		rq.Options.DeduplicateModels = true
		rq.Options.HoistSharedShapes = true
		rq.Options.MergeIdenticalModels = true
		rq.Options.LockHoistedNames = true
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || len(control.Artifacts.NameLock) == 0 {
			t.Fatal("unannotated factory control rejected", style, err)
		}
		rq.PreviousNameLock = control.Artifacts.NameLock
		rq.Document = schemas(strings.ReplaceAll(body, use, use+`,"default":3`))
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("independent child changed optimized output or name lock", style, err)
		}
	}
}

func TestScalarReferenceChildIndependentFactoriesInNestedAndHTTPUses(t *testing.T) {
	for _, tc := range []struct {
		name, document, use, pointer string
	}{
		{
			"nested",
			`{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"string","default":"root"},"Holder":{"properties":{"values":{"type":"array","items":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}}}}}}}}`,
			`"$ref":"#/components/schemas/Leaf"`, "/components/schemas/Holder/properties/values/items/additionalProperties/default",
		},
		{
			"http request and response",
			`{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"string","default":"root"}}},"paths":{"/values":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}}}},"responses":{"200":{"content":{"application/json":{"schema":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}}}}}}}}}}`,
			`"$ref":"#/components/schemas/Leaf"`, "/paths/~1values/post/requestBody/content/application~1json/schema/items/default",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, checked := range []bool{false, true} {
				rq := request([]byte(tc.document), "Dto")
				rq.Options.Validation = checked
				control, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("unannotated factory control rejected", checked, err)
				}
				rq.Document = []byte(strings.ReplaceAll(tc.document, tc.use, tc.use+`,"default":"child"`))
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("nested or HTTP child changed factory output", checked, err)
				}
				rq.Document = []byte(strings.Replace(tc.document, tc.use, tc.use+`,"default":true`, 1))
				report, err := core.GenerateWithDiagnostics(rq)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
					*ds[0].Pointer != tc.pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("invalid nested or HTTP child lost its source", checked, err)
				}
			}
		})
	}
}
