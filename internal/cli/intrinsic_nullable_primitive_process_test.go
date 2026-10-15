package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/internal/testutil"
)

func primitiveProject(t testing.TB, nullable bool) project {
	t.Helper()
	p := newProject(t)
	b := read(t, filepath.Join(repo, "tests/fixtures/http/intrinsic-nullable-primitives.json"))
	if !nullable {
		var err error
		b, err = testutil.NullablePrimitiveControl(b)
		if err != nil {
			t.Fatal(err)
		}
	}
	put(t, p.Input, b)
	return p
}
func TestIntrinsicNullablePrimitiveNativeControl(t *testing.T) {
	p := primitiveProject(t, false)
	result(t, p.generate(t), 0)
	result(t, p.generate(t, "--validation"), 0)
}
func TestIntrinsicNullablePrimitiveNativeDesiredSuccess(t *testing.T) {
	p := primitiveProject(t, true)
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
	p.change(t, func(doc map[string]any) { schemas(doc)["ZString"].(map[string]any)["minLength"] = -1 })
	fails(t, p.generate(t, "--validation"), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed source changed output")
	}
}
