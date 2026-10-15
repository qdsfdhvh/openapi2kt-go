package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumBoundsMappedIntersectionDefaultAndPointers(t *testing.T) {
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"type":"integer","enum":[1,2],"minimum":3}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":"integer","enum":[1,3],"multipleOf":2}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":"integer","enum":[1,2],"minimum":2,"default":1}`, "E_SCHEMA_INVALID", "default"},
		{`{"type":"number","format":"float","enum":[16777216,16777217],"exclusiveMinimum":16777216}`, "E_SCHEMA_INVALID", "enum"},
		{`{"type":"number","enum":[1,2],"multipleOf":0.5}`, "E_SCHEMA_UNSUPPORTED", "multipleOf"},
		{`{"type":"boolean","enum":[true],"minimum":0}`, "E_SCHEMA_UNSUPPORTED", "minimum"},
		{`{"type":"integer","enum":[1,2],"minimum":4,"maximum":3}`, "E_SCHEMA_INVALID", "maximum"},
	} {
		r := request(fieldDoc(tc.schema), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || !strings.HasSuffix(*err.(core.Diagnostics)[0].Pointer, "/"+tc.tail) || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
	}
	for _, schema := range []string{`{"type":["integer","null"],"enum":[1,2],"minimum":2,"default":null}`, `{"type":"number","format":"float","enum":[16777216,16777217],"minimum":16777217}`, `{"type":"integer","enum":[-2147483648,2],"minimum":-2147483648,"multipleOf":2}`} {
		r := request(fieldDoc(schema), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(schema, err)
		}
	}
	for _, version := range []string{"3.0.3", "3.1.0"} {
		bound := `"exclusiveMinimum":1`
		if version == "3.0.3" {
			bound = `"minimum":1,"exclusiveMinimum":true`
		}
		r := request([]byte(`{"openapi":"`+version+`","components":{"schemas":{"Record":{"properties":{"value":{"type":"integer","enum":[1,2],`+bound+`}}}}}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "RecordDto"), "value > 1") {
			t.Fatal(version, err)
		}
	}
}
