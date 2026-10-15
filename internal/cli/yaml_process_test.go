package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestYAMLNativeFileStdinConfigParity(t *testing.T) {
	for _, flags := range [][]string{{}, {"--validation", "--model-style=plain_class"}, {"--hoist-shared-shapes", "--lock-hoisted-names"}} {
		p := newProject(t)
		put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/yaml/records.json")))
		want := result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic handwritten file"))
		before := snapshot(t, p.Output)
		yamlInput := read(t, filepath.Join(repo, "tests/fixtures/yaml/records.yaml"))
		file := filepath.Join(p.Root, "records.YmL")
		put(t, file, yamlInput)
		base := append([]string{"generate", "--input", file, "--output", p.Output, "--package-name=org.example.catalog", "--diagnostic-format=json"}, flags...)
		for _, tail := range [][]string{{"--check"}, {}} {
			got := result(t, invoke(t, t.TempDir(), append(append([]string{}, base...), tail...)...), 0)
			if !reflect.DeepEqual(want.Warnings, got.Warnings) || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("YAML file warnings/no-op/check differ from JSON")
			}
		}
		base[2] = "-"
		base = append(base, "--input-format=yaml")
		for _, tail := range [][]string{{"--check"}, {}} {
			got := result(t, invokeInput(t, p.Root, bytes.NewReader(yamlInput), append(append([]string{}, base...), tail...)...), 0)
			if !reflect.DeepEqual(want.Warnings, got.Warnings) || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("YAML stdin warnings/no-op/check differ from JSON")
			}
		}
		result(t, invokeInput(t, p.Root, bytes.NewReader(yamlInput), "validate", "--input=-", "--input-format=yaml", "--diagnostic-format=json"), 0)
		fails(t, invokeInput(t, p.Root, bytes.NewReader(yamlInput), "validate", "--input=-", "--diagnostic-format=json"), 1, "E_JSON_INVALID")
		// A one-owner config can switch formats without changing source bytes.
		cfg := filepath.Join(p.Root, "config.toml")
		configRoot := filepath.Join(p.Root, "configured")
		text := "config_version=1\n[defaults]\noutput_dir='configured'\npackage_name='org.example.catalog'\n[[specs]]\nname='records'\ninput='catalog.json'\n"
		put(t, cfg, []byte(text))
		result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
		configured := snapshot(t, configRoot)
		for _, format := range []string{"auto", "yaml"} {
			put(t, cfg, []byte(strings.ReplaceAll(text, "input='catalog.json'", "input='records.YmL'\ninput_format='"+format+"'")))
			for _, command := range []string{"generate", "validate"} {
				result(t, invoke(t, t.TempDir(), command, "--config", cfg, "--diagnostic-format=json"), 0)
			}
			result(t, invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
			if !reflect.DeepEqual(configured, snapshot(t, configRoot)) {
				t.Fatal("config YAML format changed source/manifest/mtime")
			}
		}
	}
}

func TestYAMLNativeCustomTagIdentityPreservesOutputs(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic handwritten file"))
	before := snapshot(t, p.Output)
	for _, value := range []string{
		"!<!!str> 1e400", "!<!!map> {}", "!<!!seq> []",
		"!<%21%21str> 1e400", "!<%21%21map> {}", "!<%21%21seq> []",
		"!%21str 1e400", "!%21map {}", "!%21seq []",
	} {
		t.Run(value, func(t *testing.T) {
			input := "openapi: 3.1.0\ncomponents: {schemas: {A: {type: object, properties: {v: {type: string}}}}}\nexample: " + value + "\n"
			file := filepath.Join(p.Root, "tag.yaml")
			put(t, file, []byte(input))
			for _, stream := range []bool{false, true} {
				for _, check := range []bool{false, true} {
					source := file
					if stream {
						source = "-"
					}
					args := []string{"generate", "--input", source, "--input-format=yaml", "--output", p.Output, "--package-name=org.example.catalog", "--diagnostic-format=json"}
					if check {
						args = append(args, "--check")
					}
					out := invokeInput(t, p.Root, strings.NewReader(input), args...)
					fails(t, out, 1, "E_YAML_UNSUPPORTED")
					if bytes.Contains(out.Stdout, []byte("1e400")) || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("custom tag rejection leaked payload or changed output")
					}
				}
			}
		})
	}
}

func TestYAMLNativeFailurePreservesOutputs(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic handwritten file"))
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ input, code string }{
		{"example: {synthetic_private_payload: 1, synthetic_private_payload: 2}", "E_YAML_INVALID"},
		{"1: value", "E_YAML_UNSUPPORTED"},
		{"example: &node [*node]", "E_YAML_ALIAS_CYCLE"},
		{"example: {<<: {a: 1}}", "E_YAML_UNSUPPORTED"},
		{"example: !private_tag value", "E_YAML_UNSUPPORTED"},
		{"%TAG !p! tag:example.test,2026:synthetic_private_payload/\n---\nexample: !p!value text", "E_YAML_UNSUPPORTED"},
		{"{}\n---\n{}", "E_YAML_INVALID"},
		{"example: 1e400", "E_YAML_UNSUPPORTED"},
		{"example: \"\xff\"", "E_YAML_INVALID"},
		{"example: " + strings.Repeat("[", 127) + "0" + strings.Repeat("]", 127), "E_YAML_LIMIT"},
		{strings.Repeat(" ", 8*1024*1024+1), "E_INPUT_LIMIT"},
	} {
		for _, stream := range []bool{false, true} {
			file := filepath.Join(p.Root, "invalid.yaml")
			put(t, file, []byte(tc.input))
			if stream {
				file = "-"
			}
			for _, check := range []bool{false, true} {
				args := []string{"generate", "--input", file, "--input-format=yaml", "--output", p.Output, "--package-name=org.example.catalog", "--diagnostic-format=json"}
				if check {
					args = append(args, "--check")
				}
				out := invokeInput(t, p.Root, strings.NewReader(tc.input), args...)
				fails(t, out, 1, tc.code)
				if bytes.Contains(out.Stdout, []byte("synthetic_private_payload")) || bytes.Contains(out.Stdout, []byte("private_tag")) || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("YAML error leaked payload or mutated output")
				}
			}
		}
	}
	// A valid YAML document denied by warning policy also preserves old output.
	put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures/yaml/records.yaml")))
	warningDenied(t, p.generate(t, "--input-format=yaml", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("YAML warning rejection mutated output")
	}
	// Full config generates earlier owners in memory but writes none if a later
	// YAML owner fails; partial selection does not read that invalid input.
	dir, cfg := configFiles(t)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	root := filepath.Join(dir, "generated")
	configured := snapshot(t, root)
	text := string(read(t, cfg))
	put(t, cfg, []byte(strings.ReplaceAll(text, "input = 'Beta.json'", "input = 'Beta.yaml'")))
	put(t, filepath.Join(dir, "Beta.yaml"), []byte("example: &a [*a]"))
	put(t, filepath.Join(dir, "Alpha.json"), []byte(configDocument("Changed")))
	for _, tail := range [][]string{{}, {"--check"}} {
		fails(t, invoke(t, repo, append([]string{"generate", "--config", cfg, "--diagnostic-format=json"}, tail...)...), 1, "E_YAML_ALIAS_CYCLE")
		if !reflect.DeepEqual(configured, snapshot(t, root)) {
			t.Fatal("config YAML failure wrote earlier owners")
		}
	}
	result(t, invoke(t, repo, "validate", "--config", cfg, "--spec=alpha", "--diagnostic-format=json"), 0)
}
