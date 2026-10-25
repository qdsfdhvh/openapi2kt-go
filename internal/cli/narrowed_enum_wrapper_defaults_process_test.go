package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNarrowedEnumWrapperDefaultsNativeCheckAndFailurePreservation(t *testing.T) {
	for _, tc := range []struct {
		name, schema, value string
		invalid             any
		native              bool
	}{
		{"string", `{"enum":["on",null,"off"],"default":null}`, `"on"`, "future", false},
		{"integer", `{"type":"integer","nullable":true,"enum":[1,null,2],"minimum":2,"default":null}`, `2`, 3, true},
	} {
		for _, style := range []string{"data_class", "plain_class"} {
			for _, checked := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%t", tc.name, style, checked), func(t *testing.T) {
					p := newProject(t)
					input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"State":` + tc.schema + `,"Middle":{"$ref":"#/components/schemas/State","nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Middle","nullable":false,"default":` + tc.value + `},"Again":{"$ref":"#/components/schemas/Narrow","nullable":true,"default":null},"Bare":{"$ref":"#/components/schemas/Narrow"}}},"paths":{"/a":{"get":{"operationId":"a","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"$ref":"#/components/schemas/Narrow"}}}}}}}}},"/b":{"get":{"operationId":"b","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"$ref":"#/components/schemas/Narrow"}}}}}}}}}}}`)
					put(t, p.Input, input)
					flags := []string{"--model-style", style, "--deduplicate-models", "--hoist-shared-shapes", "--merge-identical-models", "--lock-hoisted-names"}
					if checked {
						flags = append(flags, "--validation")
					}
					if tc.native && !checked {
						fails(t, p.generate(t, flags...), 1, "E_SCHEMA_UNSUPPORTED")
						if _, err := os.Stat(p.Output); !errors.Is(err, os.ErrNotExist) {
							t.Fatal("native validation-off refusal created output")
						}
						return
					}
					result(t, p.generate(t, append(flags, "--check")...), 3)
					if _, err := os.Stat(p.Output); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("read-only factory generation created output")
					}
					result(t, p.generate(t, flags...), 0)
					var lock struct{ Entries []json.RawMessage }
					if err := json.Unmarshal(read(t, filepath.Join(p.Output, ".openapi-kt-names.json")), &lock); err != nil || len(lock.Entries) == 0 {
						t.Fatal("optimizer did not exercise persisted allocations", err)
					}
					before := snapshot(t, p.Output)
					result(t, p.generate(t, flags...), 0)
					result(t, p.generate(t, append(flags, "--check")...), 0)
					if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("factory replay changed output or lock mtimes")
					}
					for _, bad := range []struct {
						component, key, pointer string
						value                   any
					}{
						{"Narrow", "default", "/components/schemas/Narrow/default", nil},
						{"Narrow", "default", "/components/schemas/Narrow/default", tc.invalid},
						{"State", "default", "/components/schemas/State/default", tc.invalid},
						{"State", "enum", "/components/schemas/State/enum/2", []any{"on", nil, nil}},
					} {
						if tc.native && bad.key == "enum" {
							bad.value = []any{1, nil, nil}
						}
						put(t, p.Input, input)
						p.change(t, func(d map[string]any) { schemas(d)[bad.component].(map[string]any)[bad.key] = bad.value })
						for _, args := range [][]string{flags, append(append([]string{}, flags...), "--check")} {
							failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
							if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
								t.Fatal("invalid wrapper/source changed Pointer, output or locks", bad, failure)
							}
						}
					}
					put(t, p.Input, input)
					result(t, p.generate(t, append(flags, "--check")...), 0)
				})
			}
		}
	}
}
