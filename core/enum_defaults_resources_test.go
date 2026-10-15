package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestEnumDefaultsVocabularyAndSourceLimits(t *testing.T) {
	for _, count := range []int{240, 241} {
		values := make([]string, count)
		for i := range values {
			values[i] = fmt.Sprintf(`"v%d"`, i)
		}
		field := `{"enum":[` + strings.Join(values, ",") + `],"default":"v0"}`
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
		if (err == nil) != (count == 240) {
			t.Fatal(count, err)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT") {
			t.Fatal(got, err)
		}
	}
	for _, depth := range []int{47, 48} {
		field := `{"enum":["a","b"],"default":"a"}`
		for i := range depth {
			field = fmt.Sprintf(`{"properties":{"n%d":%s},"required":["n%d"]}`, i, field, i)
		}
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
		if (err == nil) != (depth == 47) {
			t.Fatal(depth, err)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal(got, err)
		}
	}
}
