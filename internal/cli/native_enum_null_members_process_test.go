package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNativeEnumNullMembersNativeCheckAndFailuresPreserveOutput(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/native-enum-null-members.json")))
	report := result(t, p.generate(t, "--validation"), 0)
	found := false
	for _, name := range report.Files {
		if strings.HasSuffix(name, "models/Code.kt") {
			found = strings.Contains(string(read(t, filepath.Join(p.Output, filepath.FromSlash(name)))), "typealias Code = kotlin.Int?")
		}
	}
	if !found {
		t.Fatal("nullable null-vocabulary export missing")
	}
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("check wrote")
	}
	fails(t, p.generate(t), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("default-off refusal wrote")
	}
	p.change(t, func(doc map[string]any) { schemas(doc)["Code"].(map[string]any)["enum"] = []any{1, nil, nil} })
	fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid original enum wrote")
	}
}
