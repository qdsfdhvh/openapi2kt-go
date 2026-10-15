package cli_test

import (
	"openapi-kt/internal/testutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBooleanRootNullableNativeLifecycle(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			t.Run(style+map[bool]string{false: "/off", true: "/on"}[checked], func(t *testing.T) {
				p := newProject(t)
				input := read(t, filepath.Join(repo, "tests/fixtures/http/boolean-root-nullable.json"))
				control, err := testutil.BooleanRootControl(input)
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
				if len(generated.Warnings) != 1 || generated.Warnings[0].Code != "W_BOOLEAN_SCHEMA_PROJECTION" || generated.Warnings[0].Pointer == nil || *generated.Warnings[0].Pointer != "/components/schemas/ZAny" {
					t.Fatal(generated)
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
					t.Fatal("policy mutation")
				}
				validate := []string{"validate", "--input", p.Input, "--model-style=" + style, "--diagnostic-format=json"}
				if checked {
					validate = append(validate, "--validation")
				}
				result(t, invoke(t, "", validate...), 0)
				for _, profile := range []string{"false-root", "bad-nullable"} {
					put(t, p.Input, input)
					p.change(t, func(doc map[string]any) {
						if profile == "false-root" {
							schemas(doc)["ZAny"] = false
						} else {
							schemas(doc)["ANullable"].(map[string]any)["nullable"] = "true"
						}
					})
					for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings"}} {
						args := append(append([]string{}, flags...), extra...)
						failed := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
						pointer := "/components/schemas/ZAny"
						if profile == "bad-nullable" {
							pointer = "/components/schemas/ANullable/nullable"
						}
						if failed.Diagnostics[0].Pointer == nil || *failed.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
							t.Fatal(profile, failed)
						}
					}
				}
			})
		}
	}
}
