package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzCollectionReferenceChildDefault(f *testing.F) {
	for _, value := range []string{"one", "two", "", "future", "\"", "null", "\u0000", "一"} {
		f.Add(value, false)
		f.Add(value, true)
	}
	f.Fuzz(func(t *testing.T, value string, checked bool) {
		if len(value) > 256 {
			t.Skip()
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		doc := `{"Word":{"enum":["one","two"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Word","default":` + string(encoded) + `},"default":[]}}`
		rq := request(schemas(doc), "Dto")
		rq.Options.Validation = checked
		report, failure := core.GenerateWithDiagnostics(rq)
		if value == "one" || value == "two" {
			if failure != nil {
				t.Fatal("valid vocabulary member rejected", failure)
			}
			return
		}
		ds, ok := failure.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
			*ds[0].Pointer != "/components/schemas/Rows/items/default" ||
			!reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalid child default escaped validation", failure)
		}
	})
}

func FuzzReferencedEnumCollectionChildDefault(f *testing.F) {
	for _, value := range []string{`{}`, `{"x":"one"}`, `{"a/~":"two"}`, `{"x":"future"}`, `{"x":false}`, `[]`, `null`} {
		f.Add(value, false)
		f.Add(value, true)
	}
	f.Fuzz(func(t *testing.T, value string, checked bool) {
		if len(value) > 512 || !json.Valid([]byte(value)) {
			t.Skip()
		}
		prefix := `{"Word":{"enum":["one","two"]},"Target":{"additionalProperties":{"$ref":"#/components/schemas/Word"}`
		rq := request(schemas(prefix+`},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":`+value+`}}}`), "Dto")
		rq.Options.Validation = checked
		report, childFailure := core.GenerateWithDiagnostics(rq)
		factory := request(schemas(prefix+`,"default":`+value+`},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`), "Dto")
		factory.Options.Validation = checked
		_, factoryFailure := core.GenerateWithDiagnostics(factory)
		if (childFailure == nil) != (factoryFailure == nil) {
			t.Fatal("enum collection child and target factory disagree", childFailure, factoryFailure)
		}
		if childFailure != nil {
			ds, ok := childFailure.(core.Diagnostics)
			if ok && len(ds) > 0 && ds[0].Code == "E_JSON_INVALID" {
				other, otherOK := factoryFailure.(core.Diagnostics)
				if otherOK && len(other) > 0 && other[0].Code == "E_JSON_INVALID" && reflect.DeepEqual(report, core.GenerationReport{}) {
					return
				}
			}
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				!strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Rows/items/default") ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid enum collection child escaped validation", childFailure)
			}
			return
		}
		rq.Document = schemas(prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid enum collection child changed generated output", err)
		}
	})
}

func FuzzReferencedEnumIntermediateCollectionChildDefault(f *testing.F) {
	for _, value := range []string{`{}`, `{"x":[]}`, `{"x":["one"]}`, `{"a/~":["two"]}`, `{"x":["future"]}`, `{"x":[false]}`, `[]`, `null`} {
		f.Add(value, false)
		f.Add(value, true)
	}
	f.Fuzz(func(t *testing.T, value string, checked bool) {
		if len(value) > 512 || !json.Valid([]byte(value)) {
			t.Skip()
		}
		prefix := `{"Word":{"enum":["one","two"]},"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"}},"Target":{"additionalProperties":{"$ref":"#/components/schemas/Inner"}`
		rq := request(schemas(prefix+`},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":`+value+`}}}`), "Dto")
		rq.Options.Validation = checked
		report, childFailure := core.GenerateWithDiagnostics(rq)
		factory := request(schemas(prefix+`,"default":`+value+`},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`), "Dto")
		factory.Options.Validation = checked
		_, factoryFailure := core.GenerateWithDiagnostics(factory)
		if (childFailure == nil) != (factoryFailure == nil) {
			t.Fatal("intermediate child and target factory disagree", childFailure, factoryFailure)
		}
		if childFailure != nil {
			ds, ok := childFailure.(core.Diagnostics)
			if ok && len(ds) > 0 && ds[0].Code == "E_JSON_INVALID" {
				t.Skip()
			}
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				!strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Rows/items/default") ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid intermediate child escaped validation", childFailure)
			}
			return
		}
		rq.Document = schemas(prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid intermediate child changed generated output", err)
		}
	})
}
