package cli_test

import (
	"openapi-kt/internal/testutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBooleanHTTPNativeLifecycle(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			t.Run(style+map[bool]string{false: "/off", true: "/on"}[checked], func(t *testing.T) {
				p := newProject(t)
				input := read(t, filepath.Join(repo, "tests/fixtures/http/boolean-http.json"))
				control, e := testutil.BooleanHTTPControl(input)
				if e != nil {
					t.Fatal(e)
				}
				flags := []string{"--model-style=" + style, "--model-name-suffix=Dto"}
				if checked {
					flags = append(flags, "--validation")
				}
				put(t, p.Input, control)
				result(t, p.generate(t, flags...), 0)
				put(t, p.Input, input)
				generated := result(t, p.generate(t, flags...), 0)
				if len(generated.Warnings) != 11 {
					t.Fatal(generated)
				}
				put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned"))
				before := snapshot(t, p.Output)
				for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning=W_BOOLEAN_SCHEMA_PROJECTION", "--check"}} {
					args := append(append([]string{}, flags...), extra...)
					result(t, p.generate(t, args...), 0)
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("noop/check mutation")
					}
				}
				warningDenied(t, p.generate(t, append(append([]string{}, flags...), "--deny-warnings")...))
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("deny mutation")
				}
				validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json"}
				if checked {
					validate = append(validate, "--validation")
				}
				result(t, invoke(t, "", validate...), 0)
				p.change(t, func(doc map[string]any) {
					doc["paths"].(map[string]any)["/echo"].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)["201"] = map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": false}}}
				})
				for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings"}} {
					args := append(append([]string{}, flags...), extra...)
					failed := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
					if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != "/paths/~1echo/post/responses/201/content/application~1json/schema" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal(failed)
					}
				}
			})
		}
	}
}
