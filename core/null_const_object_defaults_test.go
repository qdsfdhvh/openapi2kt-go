package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullConstObjectDefaultMembers(t *testing.T) {
	for _, field := range []string{`{"type":"null","const":null}`, `{"const":null}`, `{"const":null,"nullable":true}`} {
		for _, value := range []string{`{"id":"a"}`, `{"id":"a","fixed":null}`} {
			input := schemas(`{"Holder":{"properties":{"payload":{"properties":{"id":{"type":"string"},"fixed":` + field + `},"required":["id"],"default":` + value + `}}}}`)
			rq := request(input, "Dto")
			r, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !strings.Contains(source(t, r, "HolderDto"), "JsonNull") {
				t.Fatal("originalnullconst objectdefault member rejected", field, value, e)
			}
		}
	}
}

func TestNullConstObjectDefaultFormerBoundary(t *testing.T) {
	field := `{"type":"object","properties":{"tag":{"const":null}},"default":{"tag":null}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(`{"Holder":{"properties":{"value":`+field+`}}}`), "Dto")
		rq.Options.Validation = checked
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("exact former nullconst-only objectdefault case", checked, e)
		}
	}
}

func TestNullConstObjectDefaultNamedCollectionAllOfAndFactory(t *testing.T) {
	for _, nullable := range []string{"false", "true"} {
		input := schemas(`{"Item":{"properties":{"id":{"type":"string"},"fixed":{"const":null,"nullable":` + nullable + `}},"required":["id"]},"Combined":{"allOf":[{"properties":{"fixed":{"type":"null","const":null}}},{"properties":{"id":{"type":"string"}},"required":["id"]}]},"Seed":{"properties":{"id":{"type":"string"},"fixed":{"const":null}},"required":["id"],"default":{"id":"seed","fixed":null}},"Holder":{"properties":{"payload":{"$ref":"#/components/schemas/Item","default":{"id":"a","fixed":null}},"combined":{"$ref":"#/components/schemas/Combined","default":{"id":"c","fixed":null}},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"id":"r","fixed":null}]}}}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("actual ordinary object context rejected nullconst member", nullable, checked, e)
			}
		}
	}
}

func TestNullConstObjectDefaultInvalidValueRequiredAndUniquePointers(t *testing.T) {
	for _, value := range []string{`"null"`, `false`, `0`, `{}`, `[]`} {
		input := schemas(`{"Holder":{"properties":{"payload":{"properties":{"id":{"type":"string"},"fixed":{"const":null,"nullable":true}},"required":["id"],"default":{"id":"a","fixed":` + value + `}}}}}`)
		rq := request(input, "Dto")
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/payload/default/fixed" || len(r.Artifacts.Files) != 0 {
			t.Fatal("nullconst mismatch lost member Pointer", value, e)
		}
	}
	for _, unique := range []bool{false, true} {
		field := `{"properties":{"id":{"type":"string"},"fixed":{"const":null}},"required":["id","fixed"]}`
		values := `[{"id":"a"}]`
		pointer := "/components/schemas/Holder/properties/rows/default/0"
		keyword := ""
		if unique {
			field = `{"properties":{"id":{"type":"string"},"fixed":{"const":null}},"required":["id"]}`
			values = `[{"id":"a"},{"id":"a","fixed":null}]`
			keyword = `,"uniqueItems":true`
			pointer = "/components/schemas/Holder/properties/rows/default/1"
		}
		input := schemas(`{"Item":` + field + `,"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}` + keyword + `,"default":` + values + `}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("nullconst required/constructor key Pointer", unique, e)
		}
	}
}
