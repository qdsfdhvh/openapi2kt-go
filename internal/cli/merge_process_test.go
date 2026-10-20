package cli_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mergeProject(t testing.TB) project {
	t.Helper()
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/http/merge-models.json")))
	return p
}
func mergeGenerate(t testing.TB, p project, extra ...string) invocation {
	t.Helper()
	return invoke(t, "", append([]string{"generate", "--input", p.Input, "--output", p.Output, "--package-name=org.example.merge", "--model-name-suffix=Dto", "--validation", "--diagnostic-format=json"}, extra...)...)
}
func TestMergeNativeLifecycle(t *testing.T) {
	p := mergeProject(t)
	result(t, mergeGenerate(t, p), 0)
	original := snapshot(t, p.Output)
	result(t, mergeGenerate(t, p, "--merge-identical-models=false"), 0)
	if !reflect.DeepEqual(original, snapshot(t, p.Output)) {
		t.Fatal("default false changed bytes/mtime")
	}
	result(t, mergeGenerate(t, p, "--merge-identical-models"), 0)
	alias := read(t, filepath.Join(p.Output, "org/example/merge/models/ZetaDto.kt"))
	if !strings.Contains(string(alias), "typealias ZetaDto = org.example.merge.models.AlphaDto") {
		t.Fatal(string(alias))
	}
	for _, pair := range [][2]string{{"RecursiveZDto", "RecursiveADto"}, {"CheckedRecursiveZDto", "CheckedRecursiveADto"}} {
		code := string(read(t, filepath.Join(p.Output, "org/example/merge/models", pair[0]+".kt")))
		if !strings.Contains(code, "typealias "+pair[0]+" = org.example.merge.models."+pair[1]) {
			t.Fatal("Native owned-self alias missing", pair, code)
		}
	}
	before := snapshot(t, p.Output)
	for _, extra := range [][]string{{"--check"}, nil} {
		result(t, mergeGenerate(t, p, append([]string{"--merge-identical-models"}, extra...)...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("noop/check mutated")
		}
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--validation", "--merge-identical-models", "--diagnostic-format=json"), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned"))
	before = snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) { schemas(doc)["Alpha"].(map[string]any)["unsupportedWireKeyword"] = true })
	fails(t, mergeGenerate(t, p, "--merge-identical-models"), 1, "E_SCHEMA_UNSUPPORTED")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failure changed owned/unowned bytes/mode/mtime")
	}
}
func TestMergeNativeArguments(t *testing.T) {
	p := mergeProject(t)
	for _, extra := range [][]string{{"--merge-identical-models=yes"}, {"--merge-identical-models=true", "--merge-identical-models=false"}, {"--exclude-merge-model=org.example.merge.models.AlphaDto"}, {"--merge-identical-models", "--exclude-merge-model=.*"}} {
		fails(t, mergeGenerate(t, p, extra...), 2, "E_USAGE")
	}
	for _, flag := range []string{"--merge-identical-models", "--exclude-merge-model=org.example.X"} {
		fails(t, invoke(t, "", "download", flag, "--diagnostic-format=json"), 2, "E_USAGE")
	}
	result(t, mergeGenerate(t, p, "--merge-identical-models", "--exclude-merge-model=org.example.merge.models.AlphaDto", "--exclude-merge-model=org.example.merge.models.ChildZDto", "--exclude-merge-model=org.example.merge.models.RecursiveZDto"), 0)
	for _, name := range []string{"AlphaDto", "ChildZDto", "RecursiveZDto"} {
		if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/merge/models", name+".kt"))), "data class "+name) {
			t.Fatal("excluded root changed")
		}
	}
}
func TestMergeConfigPresenceAndPreIO(t *testing.T) {
	p := mergeProject(t)
	file := filepath.Join(p.Root, "config.toml")
	text := fmt.Sprintf("config_version=1\n[defaults]\ninput=%q\noutput_dir=%q\npackage_name='org.example.merge'\nmodel_name_suffix='Dto'\nvalidation=true\nmerge_identical_models=true\nexclude_merge_models=['org.example.merge.models.AlphaDto']\n[[specs]]\nname='a'\nexclude_merge_models=[]\n", p.Input, p.Output)
	put(t, file, []byte(text))
	command := []string{"generate", "--config", file, "--diagnostic-format=json"}
	result(t, invoke(t, "", command...), 0)
	if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/merge/models/ZetaDto.kt"))), "typealias ZetaDto") {
		t.Fatal("empty list didn't override defaults")
	}
	before := snapshot(t, p.Output)
	result(t, invoke(t, "", append(command, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("config check mutated")
	}
	result(t, invoke(t, "", append(command, "--merge-identical-models=false")...), 0)
	if !strings.Contains(string(read(t, filepath.Join(p.Output, "org/example/merge/models/ZetaDto.kt"))), "data class ZetaDto") {
		t.Fatal("explicit false lost")
	}
	before = snapshot(t, p.Output)
	put(t, file, []byte(text+"[[specs]]\nname='b'\ninput='missing.json'\nmerge_identical_models=false\nexclude_merge_models=['org.example.merge.models.ZetaDto']\n"))
	fails(t, invoke(t, "", append(command, "--spec=a")...), 1, "E_OPTIONS")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("inactive invalid inventory mutated output")
	}
}
