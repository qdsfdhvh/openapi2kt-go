package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestScalarTupleComponentDefaultFactory(t *testing.T) {
	for _, checked := range []bool{false, true} {
		input := schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"integer"},{"type":"integer"}],"default":[1,2]},"Wrapped":{"$ref":"#/components/schemas/Rows","default":[3]},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("scalar tuple factory rejected", checked, err)
		}
	}
}

func TestScalarTupleFactoryKeepsOriginalComponentWireNames(t *testing.T) {
	for _, name := range []string{"Rows%", "Rows~1Slash", "Rows/Slash"} {
		rq := request(schemas(`{"`+name+`":{"type":"array","prefixItems":[{"type":"integer"}],"default":[1]}}`), "Dto")
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("component wire identity changed", name, err)
		}
	}
}

func TestScalarTupleComponentDefaultGuards(t *testing.T) {
	for _, tc := range []struct{ schema, code, pointer string }{
		{`{"type":"array","prefixItems":[{"type":"integer"}],"default":[true]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","prefixItems":[{"type":"integer"}],"default":null}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","prefixItems":[{"type":"string","format":"date"}],"default":["2023-02-29"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","prefixItems":[{"type":"string","format":"date-time"}],"default":["2024-02-29T2:03:04Z"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","prefixItems":[{"type":"integer","default":1}],"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/default"},
	} {
		rq := request(schemas(`{"Rows":`+tc.schema+`}`), "Dto")
		if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows"+tc.pointer {
			t.Fatal("tuple root source/value exclusion changed", tc, err)
		}
	}
}

func TestScalarTupleComponentDefaultBoundedValues(t *testing.T) {
	for _, size := range []int{32764, 32765} {
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"string"}],"default":["`+strings.Repeat("x", size)+`"]}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(rq)
		// The budget includes quoted member bytes and enclosing JSON syntax.
		if size == 32764 && err != nil {
			t.Fatal("bounded tuple root literal rejected", err)
		}
		if size == 32765 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(report.Artifacts.Files) != 0) {
			t.Fatal("tuple factory escaped byte budget", err)
		}
	}
	for _, count := range []int{16383, 16384} {
		values := strings.TrimSuffix(strings.Repeat("0,", count), ",")
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"integer"}],"default":[`+values+`]}}`), "Dto")
		report, err := core.GenerateWithDiagnostics(rq)
		if count == 16383 && err != nil {
			t.Fatal("exact node budget rejected", err)
		}
		if count == 16384 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(report.Artifacts.Files) != 0) {
			t.Fatal("tuple factory escaped value budget", err)
		}
	}
}
