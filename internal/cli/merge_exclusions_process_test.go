package cli_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMergeExclusionNativeGrammarAndPreservation(t *testing.T) {
	p := mergeProject(t)
	result(t, mergeGenerate(t, p, "--merge-identical-models"), 0)
	before := snapshot(t, p.Output)
	for _, flags := range [][]string{
		{"--exclude-merge-model-pattern=.*"}, {"--exclude-merge-model-suffix=Dto"},
		{"--merge-identical-models", "--exclude-merge-model-pattern=(?=Alpha)"},
		{"--merge-identical-models", "--exclude-merge-model-pattern=(Alpha)\\1"},
		{"--merge-identical-models", "--exclude-merge-model-suffix=.Dto"},
	} {
		fails(t, mergeGenerate(t, p, flags...), 2, "E_USAGE")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("bad exclusion changed output")
		}
	}
	result(t, mergeGenerate(t, p, "--merge-identical-models", "--exclude-merge-model-pattern=org\\.example\\.merge\\.models\\.ZetaDto", "--exclude-merge-model-suffix=ChildZDto"), 0)
	for _, name := range []string{"ZetaDto", "ChildZDto"} {
		if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/merge/models", name+".kt"))), "data class "+name) {
			t.Fatal(name)
		}
	}
	before = snapshot(t, p.Output)
	result(t, mergeGenerate(t, p, "--merge-identical-models", "--exclude-merge-model-pattern=org\\.example\\.merge\\.models\\.ZetaDto", "--exclude-merge-model-suffix=ChildZDto", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("measured exclusions check changed outputs")
	}
}

func TestMergeExclusionConfigListReplacementAndWholeInventory(t *testing.T) {
	p := mergeProject(t)
	cfg := filepath.Join(p.Root, "exclusions.toml")
	text := fmt.Sprintf("config_version=1\n[defaults]\ninput=%q\noutput_dir=%q\npackage_name='org.example.merge'\nmodel_name_suffix='Dto'\nvalidation=true\nmerge_identical_models=true\nexclude_merge_model_patterns=['.*ZetaDto']\nexclude_merge_model_suffixes=['ChildZDto']\n[[specs]]\nname='a'\nexclude_merge_model_patterns=[]\nexclude_merge_model_suffixes=[]\n", p.Input, p.Output)
	put(t, cfg, []byte(text))
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, repo, args...), 0)
	if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/merge/models/ZetaDto.kt"))), "typealias ZetaDto") {
		t.Fatal("empty pattern did not reset")
	}
	result(t, invoke(t, repo, append(args, "--exclude-merge-model-pattern=.*ZetaDto", "--exclude-merge-model-suffix=ChildZDto")...), 0)
	for _, name := range []string{"ZetaDto", "ChildZDto"} {
		if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/merge/models", name+".kt"))), "data class "+name) {
			t.Fatal("CLI list replace", name)
		}
	}
	before := snapshot(t, p.Output)
	put(t, cfg, []byte(text+"[[specs]]\nname='b'\ninput='missing.json'\noutput_dir='other'\nexclude_merge_model_patterns=['(?=bad)']\n"))
	fails(t, invoke(t, repo, append(args, "--spec=a")...), 1, "E_OPTIONS")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("invalid inactive pattern was not preflighted")
	}
}
