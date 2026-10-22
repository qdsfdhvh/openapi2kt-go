package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestPrimitiveTupleNativeWholeLifecycle(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			t.Run(style+map[bool]string{false: "/off", true: "/on"}[checked], func(t *testing.T) {
				p := newProject(t)
				input := read(t, filepath.Join(repo, "tests/fixtures/http/primitive-tuples.json"))
				control, err := testutil.PrimitiveTupleControl(input)
				if err != nil {
					t.Fatal(err)
				}
				flags := []string{"--model-name-suffix=Dto", "--model-style=" + style}
				if checked {
					flags = append(flags, "--validation")
				}
				put(t, p.Input, control)
				result(t, p.generate(t, flags...), 0)
				put(t, p.Input, input)
				got := result(t, p.generate(t, flags...), 0)
				if len(got.Warnings) == 0 {
					t.Fatal("missing tuple approximation warnings")
				}
				put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned input"))
				before := snapshot(t, p.Output)
				for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning=W_TUPLE_APPROXIMATION", "--check"}} {
					args := append(append([]string{}, flags...), extra...)
					result(t, p.generate(t, args...), 0)
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("no-op/check changed old files")
					}
				}
				validate := []string{"validate", "--input", p.Input, "--model-style=" + style, "--diagnostic-format=json"}
				if checked {
					validate = append(validate, "--validation")
				}
				result(t, invoke(t, "", validate...), 0)
				warningDenied(t, p.generate(t, append(flags, "--deny-warnings")...))
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("warning denial mutated output")
				}
				p.change(t, func(doc map[string]any) {
					schema := schemas(doc)["StringTuple"].(map[string]any)
					schema["items"] = map[string]any{"type": "string"}
					schema["prefixItems"].([]any)[1].(map[string]any)["minimum"] = 1
				})
				for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings"}} {
					args := append(append([]string{}, flags...), extra...)
					fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("discarded invalid member changed old output")
					}
				}
				put(t, p.Input, input)
				p.change(t, func(doc map[string]any) {
					schema := schemas(doc)["DateTuple"].(map[string]any)
					schema["items"] = map[string]any{"type": "string"}
					schema["prefixItems"].([]any)[1].(map[string]any)["minLength"] = 1
				})
				for _, extra := range [][]string{nil, {"--check"}} {
					args := append(append([]string{}, flags...), extra...)
					fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("discarded invalid temporal member changed old output")
					}
				}
			})
		}
	}
}
