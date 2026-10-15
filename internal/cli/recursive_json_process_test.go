package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestRecursiveJSONNativeLifecycle(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			t.Run(style+map[bool]string{false: "/off", true: "/on"}[checked], func(t *testing.T) {
				p := newProject(t)
				input := read(t, filepath.Join(repo, "tests/fixtures/http/recursive-json.json"))
				control, err := testutil.RecursiveJSONControl(input)
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
				if len(generated.Warnings) != 1 || generated.Warnings[0].Code != "W_RECURSIVE_JSON_PROJECTION" || generated.Warnings[0].Pointer == nil || *generated.Warnings[0].Pointer != "/components/schemas/ZAny" {
					t.Fatal(generated)
				}
				put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned file"))
				before := snapshot(t, p.Output)
				for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning=W_RECURSIVE_JSON_PROJECTION", "--check"}} {
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
				for _, which := range []string{"unused-bound", "partial-cover", "nullable-alias"} {
					put(t, p.Input, input)
					p.change(t, func(d map[string]any) {
						root := schemas(d)["ZAny"].(map[string]any)
						switch which {
						case "unused-bound":
							root["oneOf"].([]any)[3].(map[string]any)["minimum"] = 1
						case "partial-cover":
							root["oneOf"] = root["oneOf"].([]any)[:5]
						case "nullable-alias":
							schemas(d)["AChain"].(map[string]any)["nullable"] = false
						}
					})
					if which == "nullable-alias" {
						// Keep the historical false-input. p.change marshals
						// map properties in a new order; compare its own generated
						// snapshot, while the canonical alias remains non-null.
						result(t, p.generate(t, flags...), 0)
						positive := snapshot(t, p.Output)
						result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
						if !reflect.DeepEqual(positive, snapshot(t, p.Output)) {
							t.Fatal("false alias check changed output")
						}
						continue
					}
					pointer := map[string]string{"unused-bound": "/components/schemas/ZAny/oneOf/3/minimum", "partial-cover": "/components/schemas/ZAny/oneOf", "nullable-alias": "/components/schemas/AChain"}[which]
					for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings"}} {
						args := append(append([]string{}, flags...), extra...)
						got := fails(t, p.generate(t, args...), 1, "E_SCHEMA_UNSUPPORTED")
						if got.Diagnostics[0].Pointer == nil || *got.Diagnostics[0].Pointer != pointer {
							t.Fatal(got)
						}
						if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
							t.Fatal("source failure changed owned/unowned/manifest bytes mode mtime")
						}
					}
				}
			})
		}
	}
}
