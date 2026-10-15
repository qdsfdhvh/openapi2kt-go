package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLockedOwnerAdoptionNativeMergedDescriptorsAndYAML(t *testing.T) {
	for _, profile := range []string{"descriptor", "yaml"} {
		t.Run(profile, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "generated")
			input := filepath.Join(base, "input.json")
			flags := []string{"--package-name=org.example.dedup", "--model-name-suffix=Dto", "--hoist-shared-shapes", "--lock-hoisted-names", "--qualify-hoisted-names", "--validation", "--diagnostic-format=json"}
			options := "qualify_hoisted_names=true\nvalidation=true\n"
			if profile == "descriptor" {
				put(t, input, read(t, filepath.Join(repo, "tests/fixtures/name-lock-lifecycle/merge-descriptors.json")))
				flags = append(flags, "--deduplicate-models", "--merge-identical-models")
				options += "deduplicate_models=true\nmerge_identical_models=true\n"
			} else {
				input = filepath.Join(base, "input.yaml")
				put(t, input, read(t, filepath.Join(repo, "tests/fixtures/yaml/records.yaml")))
			}
			direct := append([]string{"generate", "--input", input, "--output", root}, flags...)
			result(t, invoke(t, base, direct...), 0)
			before := snapshot(t, root)
			cfg := filepath.Join(base, "adopt.toml")
			put(t, cfg, []byte("config_version=1\n[defaults]\noutput_dir='generated'\npackage_name='org.example.dedup'\nmodel_name_suffix='Dto'\nhoist_shared_shapes=true\nlock_hoisted_names=true\n"+options+"[[specs]]\nname='catalog'\ninput='"+filepath.Base(input)+"'\n"))
			run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
			checked := invoke(t, base, append(run, "--check")...)
			result(t, checked, 3)
			validated := invoke(t, base, lifecycleValidate(run)...)
			result(t, validated, 0)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("preflight changed consumer")
			}
			var a, b map[string]json.RawMessage
			if e := json.Unmarshal(checked.Stdout, &a); e != nil {
				t.Fatal(e)
			}
			if e := json.Unmarshal(validated.Stdout, &b); e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(a["locked_owner_adoptions"], b["locked_owner_adoptions"]) {
				t.Fatal("report mismatch")
			}
			if profile == "descriptor" {
				for _, text := range []string{`"classifier":"org.example.dedup.models.AlphaResponseDto"`, `"descriptor":"org.example.dedup.models.AnchorDto"`, `"descriptor":"org.example.dedup.models.ModeDto"`} {
					if !bytes.Contains(checked.Stdout, []byte(text)) {
						t.Fatal("actual merged descriptor absent", string(checked.Stdout))
					}
				}
			}
			result(t, invoke(t, base, run...), 0)
			after := snapshot(t, root)
			for path, file := range before {
				if filepath.Ext(path) == ".kt" && !reflect.DeepEqual(file, after[path]) {
					t.Fatal("source bytes/attrs changed", path)
				}
			}
			result(t, invoke(t, base, append(run, "--check")...), 0)
		})
	}
}
