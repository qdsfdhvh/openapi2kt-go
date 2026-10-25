package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestBooleanValueNativeLifecycle(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			t.Run(style+map[bool]string{false: "/off", true: "/on"}[checked], func(t *testing.T) {
				p := newProject(t)
				input := read(t, filepath.Join(repo, "tests/fixtures/http/boolean-values.json"))
				control, err := testutil.BooleanValueControl(input)
				if err != nil {
					t.Fatal(err)
				}
				flags := []string{"--model-style=" + style, "--model-name-suffix=Dto"}
				if checked {
					flags = append(flags, "--validation")
				}
				put(t, p.Input, control)
				result(t, p.generate(t, flags...), 0)
				put(t, p.Input, input)
				generated := result(t, p.generate(t, flags...), 0)
				sources := map[string]bool{}
				for _, warning := range generated.Warnings {
					if warning.Code != "W_BOOLEAN_SCHEMA_PROJECTION" || warning.Pointer == nil || sources[*warning.Pointer] {
						t.Fatal("wrong/double source warning", warning)
					}
					sources[*warning.Pointer] = true
				}
				if len(sources) != 8 {
					t.Fatal("missing literal true sources", sources)
				}
				put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned input"))
				before := snapshot(t, p.Output)
				for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning=W_BOOLEAN_SCHEMA_PROJECTION", "--check"}} {
					args := append(append([]string{}, flags...), extra...)
					result(t, p.generate(t, args...), 0)
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("check/noop mutation")
					}
				}
				warningDenied(t, p.generate(t, append(append([]string{}, flags...), "--deny-warnings")...))
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("warning denial mutation")
				}
				validate := []string{"validate", "--input", p.Input, "--model-style=" + style, "--diagnostic-format=json"}
				if checked {
					validate = append(validate, "--validation")
				}
				result(t, invoke(t, "", validate...), 0)
				p.change(t, func(doc map[string]any) {
					schemas(doc)["Holder"].(map[string]any)["properties"].(map[string]any)["value"] = false
				})
				// Keep the historical literal-false input as an explicit positive
				// graduation; nested owner roles remain rejected below.
				result(t, p.generate(t, flags...), 0)
				holder := string(read(t, filepath.Join(p.Output, "org/example/catalog/models/HolderDto.kt")))
				if !strings.Contains(holder, "public sealed class FalseValue private constructor()") || !strings.Contains(holder, "HolderDto.FalseValue") {
					t.Fatal("original false field lost actual nested codec")
				}
				before = snapshot(t, p.Output)
				result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
				result(t, invoke(t, "", validate...), 0)
				warningDenied(t, p.generate(t, append(append([]string{}, flags...), "--deny-warnings")...))
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("graduated check/warning denial mutation")
				}
				p.change(t, func(doc map[string]any) {
					schemas(doc)["Holder"].(map[string]any)["properties"].(map[string]any)["value"] = map[string]any{"properties": map[string]any{"blocked": false}}
				})
				for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings"}} {
					args := append(append([]string{}, flags...), extra...)
					failed := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
					if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/value/properties/blocked" {
						t.Fatal(failed)
					}
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("false source failure mutation")
					}
				}
			})
		}
	}
}
