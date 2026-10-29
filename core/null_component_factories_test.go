package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestJsonNullComponentDefaultFactories(t *testing.T) {
	for _, definition := range []string{`{"type":"null","default":null}`, `{"enum":[null],"default":null}`} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Null":`+definition+`}`), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			if e != nil {
				t.Fatal("original JsonNull component own factory", definition, checked, e)
			}
			generated := source(t, got, "NullDto")
			if !strings.Contains(generated, "public fun `NullDto default`(): NullDto =") || strings.Contains(generated, "NullDto? = null") {
				t.Fatal("JsonNull factory must return its nonnullable singleton", generated)
			}
		}
	}
}

func TestJsonNullComponentWrapperAndFieldRoles(t *testing.T) {
	for _, definition := range []string{`{"type":"null","default":null}`, `{"enum":[null],"default":null}`} {
		for _, checked := range []bool{false, true} {
			input := schemas(`{"Null":` + definition + `,"Middle":{"$ref":"#/components/schemas/Null","default":null},"Alias":{"$ref":"#/components/schemas/Middle","default":null},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Null"}}}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("independent nullroot alias factory", definition, checked, e)
			}
		}
	}
}

func TestJsonNullComponentFactoryOriginalFailurePointers(t *testing.T) {
	for _, tc := range []struct{ definitions, pointer, code string }{
		{`{"Null":{"type":"null","default":"null"}}`, "/components/schemas/Null/default", "E_SCHEMA_INVALID"},
		{`{"Null":{"enum":[null],"default":false}}`, "/components/schemas/Null/default", "E_SCHEMA_INVALID"},
		{`{"Null":{"type":"null"},"Alias":{"$ref":"#/components/schemas/Null","default":0}}`, "/components/schemas/Alias/default", "E_SCHEMA_INVALID"},
		{`{"Null":{"enum":[null,null],"default":null}}`, "/components/schemas/Null/enum/1", "E_SCHEMA_INVALID"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(tc.definitions), "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			ds, ok := e.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || len(r.Artifacts.Files) != 0 {
				t.Fatal("original nullfactory failure", tc.definitions, checked, e)
			}
		}
	}
	for _, definition := range []string{`{"type":"null","nullable":true,"default":null}`, `{"type":"null","const":null,"default":null}`, `{"const":null,"default":null}`, `{"type":"null","format":"uuid","default":null}`} {
		rq := request(schemas(`{"Null":`+definition+`}`), "Dto")
		if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("unadmitted nullfactory source inferred", definition, e)
		}
	}
}

func TestJsonNullComponentFactoryNullableAliasBoundary(t *testing.T) {
	for _, definition := range []string{`{"type":"null"}`, `{"enum":[null]}`} {
		rq := request(schemas(`{"Value":`+definition+`,"Alias":{"$ref":"#/components/schemas/Value","nullable":true,"default":null}}`), "Dto")
		if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("nullable JsonNull alias factory inferred", definition, e)
		}
	}
}

func TestJsonNullComponentFactoryFormerRootBoundaries(t *testing.T) {
	for _, input := range []string{`{"Value":{"type":"null","default":null}}`, `{"Value":{"type":"null","enum":[null],"default":null}}`} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(input), "Dto")
			rq.Options.Validation = checked
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("exact former null-only root boundary", input, checked, e)
			}
		}
	}
}
