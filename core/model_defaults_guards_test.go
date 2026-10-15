package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestModelDefaultsValidateOriginalLiteralsAndRejectedRoles(t *testing.T) {
	// Preserve the original enum-default input and prove its new typed result.
	graduated, e := core.GenerateWithDiagnostics(request(fieldDoc(`{"type":"string","enum":["ready"],"default":"ready"}`), "Dto"))
	if e != nil || !strings.Contains(source(t, graduated, "RecordDto"), "val value: Value,") || !strings.Contains(source(t, graduated, "RecordDto"), `serialName: kotlin.String = "ready"`) {
		t.Fatal("original enum default required field lost", graduated, e)
	}
	for _, field := range []string{
		`{"type":"string","default":true}`,
		`{"type":"integer","format":"int32","default":2147483648}`,
		`{"type":"integer","default":1.5}`,
		`{"type":"number","format":"float","default":1e300}`,
		`{"type":"boolean","default":"true"}`,
		`{"type":"string","default":null}`,
		`{"type":"string","format":"date","default":"invalid"}`,
		`{"type":["string","null"],"format":"byte","default":null}`,
		`{"type":["string","null"],"format":"invalid","default":null}`,
		`{"type":"array","items":{"type":"string"},"default":[]}`,
		`{"properties":{"id":{"type":"string"}},"default":{}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
		if field == `{"type":"array","items":{"type":"string"},"default":[]}` {
			if err != nil || !strings.Contains(source(t, got, "RecordDto"), "val value: kotlin.collections.List<kotlin.String>,") {
				t.Fatal("Original array default input lost required/no-initializer admission", got, err)
			}
			continue
		}
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("invalid/unsupported original default was dropped", field, got, err)
		}
	}
}

func TestModelDefaultsPrimitiveBoundaryLiterals(t *testing.T) {
	for _, tc := range []struct{ field, want string }{
		{`{"type":"integer","format":"int64","default":-9223372036854775808}`, "(-9223372036854775807L - 1L)"},
		{`{"type":"number","format":"float","default":1.25}`, "1.25f"},
		{`{"type":"string","default":"$quoted\nline"}`, `"\$quoted\nline"`},
		{`{"type":["string","null"],"default":null}`, "null"},
		{`{"type":"string","format":"date","default":"2026-10-12"}`, "`openapi-kt default date`(\"2026-10-12\")"},
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Record":{"properties":{"value":`+tc.field+`}}}`), "Dto"))
		if err != nil || !strings.Contains(source(t, got, "RecordDto"), " = "+tc.want+",") {
			t.Fatal(tc, got, err)
		}
	}
}
