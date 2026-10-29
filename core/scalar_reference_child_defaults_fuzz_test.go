package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzScalarReferenceCollectionChildDefault(f *testing.F) {
	for _, value := range []string{"2024-02-29", "2023-02-29", "", "2024-2-29", "0000-01-01", "\"", "\u0000"} {
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
		doc := `{"Day":{"type":"string","format":"date"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Day","default":` + string(encoded) + `}}}`
		rq := request(schemas(doc), "Dto")
		rq.Options.Validation = checked
		report, failure := core.GenerateWithDiagnostics(rq)
		fieldDoc := `{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"day":{"$ref":"#/components/schemas/Day","default":` + string(encoded) + `}}}}`
		fieldRequest := request(schemas(fieldDoc), "Dto")
		fieldRequest.Options.Validation = checked
		_, fieldFailure := core.GenerateWithDiagnostics(fieldRequest)
		if (failure == nil) != (fieldFailure == nil) {
			t.Fatal("child and scalar field disagree on date validity", failure, fieldFailure)
		}
		if failure != nil {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != "/components/schemas/Rows/items/default" ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid child default escaped validation", failure)
			}
			return
		}
		rq.Document = schemas(`{"Day":{"type":"string","format":"date"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Day"}}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid child changed output", err)
		}
	})
}

func FuzzScalarReferenceOuterDateDefault(f *testing.F) {
	for _, value := range []string{"2024-02-29", "2023-02-29", "", "2024-2-29", "0000-01-01", "\"", "\u0000"} {
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
		doc := `{"Day":{"type":"string","format":"date","default":"2024-02-29"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Day","default":"2024-03-01"},"default":[` + string(encoded) + `]}}`
		rq := request(schemas(doc), "Dto")
		rq.Options.Validation = checked
		report, failure := core.GenerateWithDiagnostics(rq)
		fieldDoc := `{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"day":{"$ref":"#/components/schemas/Day","default":` + string(encoded) + `}}}}`
		fieldRequest := request(schemas(fieldDoc), "Dto")
		fieldRequest.Options.Validation = checked
		_, fieldFailure := core.GenerateWithDiagnostics(fieldRequest)
		if (failure == nil) != (fieldFailure == nil) {
			t.Fatal("outer factory and scalar field disagree on date validity", failure, fieldFailure)
		}
		if failure != nil {
			ds, ok := failure.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != "/components/schemas/Rows/default/0" ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid outer factory escaped validation", failure)
			}
			return
		}
		rq.Document = schemas(`{"Day":{"type":"string","format":"date","default":"2024-02-29"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Day"},"default":[` + string(encoded) + `]}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid child changed outer factory output", err)
		}
	})
}

func FuzzReferencedCollectionChildDefault(f *testing.F) {
	for _, value := range []string{`{}`, `{"x":1}`, `{"x":false}`, `[]`, `null`, `"x"`, `{"a/~":-7}`} {
		f.Add(value, false)
		f.Add(value, true)
	}
	f.Fuzz(func(t *testing.T, value string, checked bool) {
		if len(value) > 512 || !json.Valid([]byte(value)) {
			t.Skip()
		}
		prefix := `{"Leaf":{"type":"integer"},"Target":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}`
		childBody := prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":` + value + `}}}`
		rq := request(schemas(childBody), "Dto")
		rq.Options.Validation = checked
		report, childFailure := core.GenerateWithDiagnostics(rq)

		factoryBody := prefix + `,"default":` + value + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`
		factoryRequest := request(schemas(factoryBody), "Dto")
		factoryRequest.Options.Validation = checked
		_, factoryFailure := core.GenerateWithDiagnostics(factoryRequest)
		if (childFailure == nil) != (factoryFailure == nil) {
			t.Fatal("collection child and target factory disagree", childFailure, factoryFailure)
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
				t.Fatal("invalid collection child escaped validation", childFailure)
			}
			return
		}
		rq.Document = schemas(prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid collection child changed generated output", err)
		}
	})
}

func FuzzInlineScalarCollectionChildDefault(f *testing.F) {
	for _, value := range []string{`{}`, `{"x":1}`, `{"x":false}`, `[]`, `null`, `"x"`, `{"a/~":-7}`} {
		f.Add(value, false)
		f.Add(value, true)
	}
	f.Fuzz(func(t *testing.T, value string, checked bool) {
		if len(value) > 512 || !json.Valid([]byte(value)) {
			t.Skip()
		}
		prefix := `{"Target":{"additionalProperties":{"type":"integer"}`
		childBody := prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":` + value + `}}}`
		rq := request(schemas(childBody), "Dto")
		rq.Options.Validation = checked
		report, childFailure := core.GenerateWithDiagnostics(rq)

		factoryBody := prefix + `,"default":` + value + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`
		factoryRequest := request(schemas(factoryBody), "Dto")
		factoryRequest.Options.Validation = checked
		_, factoryFailure := core.GenerateWithDiagnostics(factoryRequest)
		if (childFailure == nil) != (factoryFailure == nil) {
			t.Fatal("inline scalar collection child and target factory disagree", childFailure, factoryFailure)
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
				t.Fatal("invalid inline scalar collection child escaped validation", childFailure)
			}
			return
		}
		rq.Document = schemas(prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid inline scalar collection child changed generated output", err)
		}
	})
}

func FuzzInlineTemporalCollectionChildDefault(f *testing.F) {
	for _, value := range []string{"2024-02-29", "2023-02-29", "", "2024-2-29", "0000-01-01", "\"", "\u0000"} {
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
		prefix := `{"Target":{"type":"array","items":{"type":"string","format":"date"}`
		childBody := prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":[` + string(encoded) + `]}}}`
		rq := request(schemas(childBody), "Dto")
		rq.Options.Validation = checked
		report, childFailure := core.GenerateWithDiagnostics(rq)

		factoryBody := prefix + `,"default":[` + string(encoded) + `]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`
		factoryRequest := request(schemas(factoryBody), "Dto")
		factoryRequest.Options.Validation = checked
		_, factoryFailure := core.GenerateWithDiagnostics(factoryRequest)
		if (childFailure == nil) != (factoryFailure == nil) {
			t.Fatal("inline temporal collection child and target factory disagree", childFailure, factoryFailure)
		}
		if childFailure != nil {
			ds, ok := childFailure.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != "/components/schemas/Rows/items/default/0" ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid inline temporal collection child escaped validation", childFailure)
			}
			return
		}
		rq.Document = schemas(prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid inline temporal collection child changed generated output", err)
		}
	})
}

func FuzzReferencedInlineEnumCollectionChildDefault(f *testing.F) {
	for _, value := range []string{"one", "two", "future", "", "\"", "\u0000"} {
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
		prefix := `{"Target":{"type":"array","items":{"type":"string","enum":["one","two"]}`
		childBody := prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":[` + string(encoded) + `]}}}`
		rq := request(schemas(childBody), "Dto")
		rq.Options.Validation = checked
		report, childFailure := core.GenerateWithDiagnostics(rq)

		factoryBody := prefix + `,"default":[` + string(encoded) + `]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`
		factoryRequest := request(schemas(factoryBody), "Dto")
		factoryRequest.Options.Validation = checked
		_, factoryFailure := core.GenerateWithDiagnostics(factoryRequest)
		if (childFailure == nil) != (factoryFailure == nil) {
			t.Fatal("inline enum collection child and target factory disagree", childFailure, factoryFailure)
		}
		if childFailure != nil {
			ds, ok := childFailure.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
				*ds[0].Pointer != "/components/schemas/Rows/items/default/0" ||
				!reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("invalid inline enum collection child escaped validation", childFailure)
			}
			return
		}
		rq.Document = schemas(prefix + `},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}}`)
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(report, control) {
			t.Fatal("valid inline enum collection child changed generated output", err)
		}
	})
}
