package cli_test

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestImplicitStringEnumNullNativeCheckAndFailurePreservation(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			p := newProject(t)
			input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"State":{"enum":[null,"on","off","null"]},"Alias":{"$ref":"#/components/schemas/State"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Holder":{"properties":{"state":{"$ref":"#/components/schemas/Alias","default":"on"},"rows":{"type":"array","items":{"enum":[null,"on"]},"default":[null,"on"]}}}}}}`)
			put(t, p.Input, input)
			flags := []string{"--model-style", style, "--deduplicate-models", "--hoist-shared-shapes", "--merge-identical-models", "--lock-hoisted-names"}
			if checked {
				flags = append(flags, "--validation")
			}
			result(t, p.generate(t, append(flags, "--check")...), 3)
			if _, err := os.Stat(p.Output); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("read-only generation created an output")
			}
			result(t, p.generate(t, flags...), 0)
			before := snapshot(t, p.Output)
			result(t, p.generate(t, append(flags, "--check")...), 0)
			for _, mutate := range []func(map[string]any){
				func(d map[string]any) { schemas(d)["State"].(map[string]any)["nullable"] = false },
				func(d map[string]any) { schemas(d)["State"].(map[string]any)["enum"] = []any{"on", nil, nil} },
				func(d map[string]any) {
					schemas(d)["Holder"].(map[string]any)["properties"].(map[string]any)["state"].(map[string]any)["default"] = "future"
				},
			} {
				put(t, p.Input, input)
				p.change(t, mutate)
				fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid inferred vocabulary changed owned output or locks")
				}
			}
			put(t, p.Input, input)
			result(t, p.generate(t, append(flags, "--check")...), 0)
		}
	}
}
