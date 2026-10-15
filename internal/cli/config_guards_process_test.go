package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConfigurationFailBeforeOutputAndEmptyReconcile(t *testing.T) {
	for _, kind := range []string{"syntax", "unknown", "duplicate", "roots", "override", "selected", "metrics", "empty", "oversize", "secondSchema"} {
		t.Run(kind, func(t *testing.T) {
			dir, cfg := configFiles(t)
			result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
			root := filepath.Join(dir, "generated")
			before := snapshot(t, root)
			data, _ := os.ReadFile(cfg)
			text := string(data)
			wanted := "E_CONFIG_SYNTAX"
			extra := []string{}
			switch kind {
			case "syntax":
				text += "bad\n"
			case "unknown":
				text += "future_option='secret'\n"
				wanted = "E_CONFIG_UNKNOWN_FIELD"
			case "duplicate":
				text += "name='beta'\n"
				wanted = "E_CONFIG_DUPLICATE"
			case "roots":
				text += "output_dir='generated/nested'\n"
				wanted = "E_OUTPUT_ROOTS"
			case "override":
				extra = []string{"--model-name-suffix", "Dto"}
				wanted = "E_CONFIG_OVERRIDE"
			case "selected":
				extra = []string{"--spec", "missing"}
				wanted = "E_CONFIG_SPEC"
			case "metrics":
				text += "metrics=true\n"
				extra = []string{"--metrics"}
				wanted = "E_CONFIG_UNKNOWN_FIELD"
			case "empty":
				text = "config_version=1\n[defaults]\noutput_dir='generated'\n"
				wanted = "E_CONFIG_EMPTY"
			case "oversize":
				text = strings.Repeat(" ", 1024*1024+1)
				wanted = "E_CONFIG_LIMIT"
			case "secondSchema":
				if e := os.WriteFile(filepath.Join(dir, "Beta.json"), []byte("{broken"), 0600); e != nil {
					t.Fatal(e)
				}
				wanted = "E_JSON_INVALID"
			}
			if e := os.WriteFile(cfg, []byte(text), 0600); e != nil {
				t.Fatal(e)
			}
			args := append([]string{"generate", "--config", cfg, "--diagnostic-format=json"}, extra...)
			exit := 1
			if kind == "roots" {
				exit = 4
			}
			fails(t, invoke(t, repo, args...), exit, wanted)
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("failed config changed owned output")
			}
			if kind == "empty" {
				result(t, invoke(t, repo, "generate", "--config", cfg, "--reconcile", "--check", "--diagnostic-format=json"), 3)
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("reconcile check wrote")
				}
				result(t, invoke(t, repo, "generate", "--config", cfg, "--reconcile", "--diagnostic-format=json"), 0)
				var manifest struct {
					Owners map[string][]string
					Files  map[string]string
				}
				b, _ := os.ReadFile(filepath.Join(root, ".openapi-kt-manifest.json"))
				if e := json.Unmarshal(b, &manifest); e != nil || len(manifest.Files) != 0 || len(manifest.Owners) != 0 {
					t.Fatal(manifest, e)
				}
			}
		})
	}
}
func TestConfigurationSelectedOverridesAndNoWriteCheck(t *testing.T) {
	dir, cfg := configFiles(t)
	out := invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json")
	result(t, out, 3)
	if _, e := os.Stat(filepath.Join(dir, "generated")); !os.IsNotExist(e) {
		t.Fatal("check created directory", e)
	}
	r := result(t, invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--model-name-suffix", "Dto", "--package-name", "org.example.override", "--diagnostic-format=json"), 0)
	found := false
	for _, file := range r.Files {
		if strings.HasSuffix(file, "AlphaDto.kt") {
			found = true
		}
	}
	if !found {
		t.Fatal(r)
	}
	result(t, invoke(t, repo, "validate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 0)
}
