package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestModelDefaultsKeepSourceDepthAndConstructorLimits(t *testing.T) {
	for _, depth := range []int{47, 48} {
		value := `{"type":"string","default":"ready"}`
		for i := range depth {
			value = fmt.Sprintf(`{"properties":{"n%d":%s},"required":["n%d"]}`, i, value, i)
		}
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(value), "Dto"))
		if (err == nil) != (depth == 47) {
			t.Fatal(depth, err)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal(got, err)
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"type":"integer","format":"int64","default":7}`, i)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Record":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if (err == nil) != (count == 240) {
			t.Fatal(count, err)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal(got, err)
		}
	}
}
