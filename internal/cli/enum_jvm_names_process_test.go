package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEnumJVMNamesNativeCheckAndFailurePreservation(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", style, checked), func(t *testing.T) {
				p := newProject(t)
				wire, _ := json.Marshal(strings.Repeat(" ", 32766))
				input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"State":{"enum":[null,` + string(wire) + `]},"Narrow":{"$ref":"#/components/schemas/State","nullable":false,"default":` + string(wire) + `}}},"paths":{"/a":{"get":{"operationId":"a","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"$ref":"#/components/schemas/Narrow"}}}}}}}}},"/b":{"get":{"operationId":"b","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"$ref":"#/components/schemas/Narrow"}}}}}}}}}}}`)
				put(t, p.Input, input)
				flags := []string{"--model-style", style, "--deduplicate-models", "--hoist-shared-shapes", "--merge-identical-models", "--lock-hoisted-names"}
				if checked {
					flags = append(flags, "--validation")
				}
				result(t, p.generate(t, append(flags, "--check")...), 3)
				if _, err := os.Stat(p.Output); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("read-only enum name check created output")
				}
				result(t, p.generate(t, flags...), 0)
				var lock struct{ Entries []json.RawMessage }
				if err := json.Unmarshal(read(t, filepath.Join(p.Output, ".openapi-kt-names.json")), &lock); err != nil || len(lock.Entries) == 0 {
					t.Fatal("optimizer did not persist real allocations", err)
				}
				before := snapshot(t, p.Output)
				result(t, p.generate(t, flags...), 0)
				result(t, p.generate(t, append(flags, "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("safe same-byte vocabulary changed output or lock mtimes")
				}
				for _, n := range []int{10923, 32766} {
					p.change(t, func(d map[string]any) {
						s := schemas(d)
						s["State"].(map[string]any)["enum"] = []any{nil, strings.Repeat("/", n)}
						s["Narrow"].(map[string]any)["default"] = strings.Repeat("/", n)
					})
					for _, args := range [][]string{flags, append(append([]string{}, flags...), "--check")} {
						failure := fails(t, p.generate(t, args...), 1, "E_MODEL_LIMIT")
						if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/State/enum/1" {
							t.Fatal("overflow lost original enum member Pointer", failure)
						}
						if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
							t.Fatal("overflow replaced old files, locks or mtimes")
						}
					}
				}
				put(t, p.Input, input)
				result(t, p.generate(t, append(flags, "--check")...), 0)
			})
		}
	}
}
