package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInternalNativeEnumNativeChecksAndFailurePreserveOriginalOutputs(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/internal-native-enums.json")))
	report := result(t, p.generate(t, "--validation"), 0)
	found := false
	for _, name := range report.Files {
		if strings.HasSuffix(name, "DefaultApi.kt") {
			code := string(read(t, filepath.Join(p.Output, filepath.FromSlash(name))))
			found = strings.Contains(code, "does not belong to enum.") && strings.Contains(code, "value % 2 == 0")
		}
	}
	if !found {
		t.Fatal("borrowed leaf checks missing")
	}
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check wrote")
	}
	p.change(t, func(doc map[string]any) {
		source := schemas(doc)["Source"].(map[string]any)
		properties := source["properties"].(map[string]any)
		properties["code"].(map[string]any)["enum"] = []any{1, 1}
	})
	fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("bad original leaf wrote outputs")
	}
}
