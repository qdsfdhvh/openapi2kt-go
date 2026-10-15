package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestStringEnumNullMembersNativeCheckAndFailurePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/string-enum-null-members.json")))
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("String null enum check wrote")
	}
	p.change(t, func(d map[string]any) { schemas(d)["State"].(map[string]any)["enum"] = []any{"ready", nil, nil} })
	fails(t, p.generate(t), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Duplicate null changed owned output")
	}
}
