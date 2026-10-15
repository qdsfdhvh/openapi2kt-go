package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/internal/testutil"
)

func rootPrimitiveProject(t testing.TB, nullable bool) project {
	t.Helper()
	p := newProject(t)
	b := read(t, filepath.Join(repo, "tests/fixtures/http/root-nullable-primitives.json"))
	if !nullable {
		var err error
		b, err = testutil.NullableBuiltinAliasControl(b)
		if err != nil {
			t.Fatal(err)
		}
	}
	put(t, p.Input, b)
	return p
}
func TestRootNullablePrimitiveNativeControl(t *testing.T) {
	p := rootPrimitiveProject(t, false)
	result(t, p.generate(t), 0)
	result(t, p.generate(t, "--validation"), 0)
}
func TestRootNullablePrimitiveNativeDesiredSuccess(t *testing.T) {
	p := rootPrimitiveProject(t, true)
	for _, flag := range []bool{false, true} {
		args := []string{"--model-name-suffix=Dto"}
		if flag {
			args = append(args, "--validation")
		}
		result(t, p.generate(t, args...), 0)
		before := snapshot(t, p.Output)
		validate := []string{"validate", "--input", p.Input, "--package-name=org.example.catalog", "--diagnostic-format=json"}
		if flag {
			validate = append(validate, "--validation")
		}
		result(t, invoke(t, "", validate...), 0)
		result(t, p.generate(t, append(args, "--check")...), 0)
		result(t, p.generate(t, args...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("check/noop mutation")
		}
	}
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned input"))
	before := snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) { schemas(doc)["ZString"].(map[string]any)["nullable"] = "wrong-kind" })
	for _, args := range [][]string{nil, {"--validation"}, {"--check"}, {"--validation", "--check"}} {
		fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("failed source changed output")
		}
	}
}
