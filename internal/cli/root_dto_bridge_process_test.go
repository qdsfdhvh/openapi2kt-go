package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootDTOBridgeNative(t *testing.T) {
	for _, kind := range []string{"named", "internal"} {
		for _, validation := range []bool{false, true} {
			t.Run(kind+"/validation", func(t *testing.T) {
				p := newProject(t)
				put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "root-"+kind+"-dto-bridges.json")))
				flags := []string{"--model-name-suffix", "Dto"}
				if validation {
					flags = append(flags, "--validation")
				}
				result(t, p.generate(t, flags...), 0)
				before := snapshot(t, p.Output)
				result(t, p.generate(t, append(flags, "--check")...), 0)
				result(t, p.generate(t, flags...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("bridge check/no-op mutated output")
				}
				p.change(t, func(doc map[string]any) {
					schemas(doc)["ZOriginalNodeAlias"].(map[string]any)["$ref"] = "#/components/schemas/Missing"
				})
				for _, checking := range []bool{false, true} {
					args := append([]string{}, flags...)
					if checking {
						args = append(args, "--check")
					}
					ds := fails(t, p.generate(t, args...), 1, "E_REF_MISSING")
					if *ds.Diagnostics[0].Pointer != "/components/schemas/ZOriginalNodeAlias/$ref" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("original forward alias diagnostic/failure preservation", ds)
					}
				}
			})
		}
	}
}
