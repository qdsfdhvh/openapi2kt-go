package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/config"
)

func TestConfigurationFullTOMLNativeParity(t *testing.T) {
	dir, cfg := configFiles(t)
	if err := os.Rename(filepath.Join(dir, "Alpha.json"), filepath.Join(dir, "Alpha file.json")); err != nil {
		t.Fatal(err)
	}
	old := strings.ReplaceAll(string(read(t, cfg)), "Alpha.json", "Alpha file.json")
	put(t, cfg, []byte(old))
	control := result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	root := filepath.Join(dir, "generated")
	put(t, filepath.Join(root, "handwritten.txt"), []byte("synthetic handwritten file"))
	before := snapshot(t, root)
	multiline := string(read(t, filepath.Join(repo, "tests/fixtures/config/full-syntax.toml")))
	for _, text := range []string{
		multiline,
		strings.ReplaceAll(multiline, "\n", "\r\n"),
		`config_version=+1
defaults={output_dir='generated', package_name='org.example.config', validation=true}
specs=[{name='alpha', input="Alpha\u0020file.json"}, {name='beta', input='Beta.json'}]
`,
		`'config_version'=0b0_1
'defaults'."output_dir"='generated'
defaults.package_name='org.example.config'
defaults.validation=true
specs=[{name='alpha', input='Alpha file.json'}, {name='beta', input='Beta.json'}]
`,
	} {
		put(t, cfg, []byte(text))
		generated := result(t, invoke(t, t.TempDir(), "generate", "--config", cfg, "--diagnostic-format=json"), 0)
		if !reflect.DeepEqual(generated, control) || !reflect.DeepEqual(snapshot(t, root), before) {
			t.Fatal("syntax changed source/report/manifest or no-op mtimes")
		}
		result(t, invoke(t, t.TempDir(), "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 0)
		result(t, invoke(t, t.TempDir(), "validate", "--config", cfg, "--diagnostic-format=json"), 0)
		// Explicit CLI settings still replace default/spec settings.
		result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--model-name-suffix=Cli", "--check", "--diagnostic-format=json"), 3)
		fails(t, invoke(t, repo, "generate", "--config", cfg, "--model-name-suffix=Cli", "--diagnostic-format=json"), 1, "E_CONFIG_OVERRIDE")
		if !reflect.DeepEqual(snapshot(t, root), before) {
			t.Fatal("check/validate/override failure wrote output")
		}
	}
	if err := os.Remove(filepath.Join(dir, "Beta.json")); err != nil {
		t.Fatal(err)
	}
	result(t, invoke(t, t.TempDir(), "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 4, "E_INPUT_IO")
	if !reflect.DeepEqual(snapshot(t, root), before) {
		t.Fatal("partial or failed full generation lost retained owner/output")
	}
	put(t, cfg, []byte(`config_version=0o1
defaults={output_dir='generated', package_name='org.example.config', validation=true}
specs=[{name='alpha', input='Alpha file.json'}]
`))
	result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
	if !reflect.DeepEqual(snapshot(t, root), before) {
		t.Fatal("partial inventory removed beta")
	}
	result(t, invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json"), 3)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	if bytes.Contains(read(t, filepath.Join(root, ".openapi-kt-manifest.json")), []byte(`"beta"`)) {
		t.Fatal("full inventory did not reconcile removed owner")
	}
	put(t, cfg, []byte("config_version=+1\ndefaults.output_dir='generated'\nspecs=[]\n"))
	fails(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 1, "E_CONFIG_EMPTY")
	result(t, invoke(t, repo, "generate", "--config", cfg, "--reconcile", "--check", "--diagnostic-format=json"), 3)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--reconcile", "--diagnostic-format=json"), 0)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--reconcile", "--check", "--diagnostic-format=json"), 0)
	for name := range snapshot(t, root) {
		if strings.HasSuffix(name, ".kt") {
			t.Fatal("empty reconciliation left owned source")
		}
	}
	if string(read(t, filepath.Join(root, "handwritten.txt"))) != "synthetic handwritten file" {
		t.Fatal("reconcile lost handwritten output")
	}
}

func TestConfigurationFullTOMLFailureSafety(t *testing.T) {
	dir, cfg := configFiles(t)
	root := filepath.Join(dir, "generated")
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	put(t, filepath.Join(root, "handwritten.txt"), []byte("synthetic handwritten file"))
	before := snapshot(t, root)
	for _, tc := range []struct{ text, code, pointer string }{
		{"config_version=1\nspecs.'" + strings.Repeat("0", 30000) + "'=1\nspecs.'" + strings.Repeat("0", 30000) + "'=2", "E_CONFIG_DUPLICATE", "/specs"},
		{"config_version=1\nspecs.'123456789'=" + strings.Repeat("[", config.MaxDepth+1) + "0" + strings.Repeat("]", config.MaxDepth+1), "E_CONFIG_LIMIT", "/specs"},
		{"config_version=1\ndefaults={input='x', \"inpu\\u0074\"='SYNTHETIC_PRIVATE'}", "E_CONFIG_DUPLICATE", "/defaults"},
		{"config_version=1\n[defaults]\ninput='x'\n'input'='SYNTHETIC_PRIVATE'", "E_CONFIG_DUPLICATE", "/defaults/input"},
		{"config_version=1\n[defaults]\n['defaults']", "E_CONFIG_DUPLICATE", "/defaults"},
		{"config_version=1\ndefaults.'SYNTHETIC_PRIVATE'={value='hidden'}", "E_CONFIG_UNKNOWN_FIELD", "/defaults"},
		{"config_version=1\n[specs.download]\nurl='SYNTHETIC_PRIVATE'", "E_CONFIG_UNKNOWN_FIELD", "/specs"},
		{"config_version=1\nspecs=[{name='alpha', download={url='SYNTHETIC_PRIVATE'}}]", "E_CONFIG_UNKNOWN_FIELD", "/specs/0"},
		{"config_version=1\ndefaults.input=2026-01-01", "E_CONFIG_INVALID", "/defaults/input"},
		{"config_version=1\ndefaults.validation=nan", "E_CONFIG_INVALID", "/defaults/validation"},
		{"config_version=1\nspecs=[{name='alpha'}, {name='beta',input='-'}]", "E_CONFIG_INVALID", "/specs/1/input"},
		{"config_version=1\ndefaults.input=\"\\uD800\"", "E_CONFIG_SYNTAX", "/defaults/input"},
		{"config_version=1\ndefaults.exclude_apis=" + strings.Repeat("[", config.MaxDepth+1) + "'SYNTHETIC_PRIVATE'" + strings.Repeat("]", config.MaxDepth+1), "E_CONFIG_LIMIT", "/defaults/exclude_apis"},
		{"config_version=1\ndefaults.exclude_apis=[" + strings.Repeat("'SYNTHETIC_PRIVATE',", config.MaxNodes) + "]", "E_CONFIG_LIMIT", ""},
	} {
		put(t, cfg, []byte(tc.text))
		for _, args := range [][]string{{"generate"}, {"generate", "--check"}, {"generate", "--spec", "alpha"}, {"validate"}} {
			args = append(args, "--config", cfg, "--diagnostic-format=json")
			out := invoke(t, repo, args...)
			r := fails(t, out, 1, tc.code)
			if r.Diagnostics[0].Pointer == nil || *r.Diagnostics[0].Pointer != tc.pointer || bytes.Contains(out.Stdout, []byte("SYNTHETIC_PRIVATE")) || bytes.Contains(out.Stdout, []byte(dir)) || !reflect.DeepEqual(snapshot(t, root), before) {
				t.Fatal("unsafe diagnostic/location or failure changed output", r)
			}
		}
	}
}
