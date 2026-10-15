package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func warningProject(t *testing.T) project {
	t.Helper()
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "cli", "warning-policy.json")))
	return p
}
func TestWarningPolicyStructuralControl(t *testing.T) {
	p := warningProject(t)
	r := result(t, p.generate(t), 0)
	if len(r.Warnings) != 2 {
		t.Fatal(r)
	}
	for _, w := range r.Warnings {
		if w.Code != "W_ALL_OF_APPROXIMATION" && w.Code != "W_JSON_ELEMENT_APPROXIMATION" {
			t.Fatal(w)
		}
	}
}
func warningDenied(t *testing.T, output invocation) resultReport {
	t.Helper()
	r := result(t, output, 1)
	if r.Status != "error" || len(r.Files) != 0 || len(r.Diagnostics) == 0 {
		t.Fatal(r)
	}
	for _, d := range r.Diagnostics {
		if d.Code != "E_WARNING_DENIED" {
			t.Fatal(r)
		}
	}
	return r
}
func TestWarningPolicyNative(t *testing.T) {
	p := warningProject(t)
	base := result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "Notes.kt"), []byte("synthetic handwritten"))
	before := snapshot(t, p.Output)
	for _, check := range []bool{false, true} {
		args := []string{"--deny-warnings"}
		if check {
			args = append(args, "--check")
		}
		r := warningDenied(t, p.generate(t, args...))
		if len(r.Files) != 0 || len(r.Diagnostics) != 2 || !reflect.DeepEqual(r.Warnings, base.Warnings) {
			t.Fatal(r)
		}
		for i, d := range r.Diagnostics {
			if *d.Pointer != *r.Warnings[i].Pointer || !strings.Contains(d.Message, r.Warnings[i].Code) {
				t.Fatal(d, r.Warnings[i])
			}
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("denial changed owned/handwritten/mtime")
		}
	}
	if rejected := warningDenied(t, p.generate(t, "--deny-warnings", "--allow-warning=W_FUTURE_UNUSED")); len(rejected.Diagnostics) != 2 {
		t.Fatal("unused allow swallowed warnings", rejected)
	}
	for _, code := range []string{"W_ALL_OF_APPROXIMATION", "W_JSON_ELEMENT_APPROXIMATION"} {
		r := warningDenied(t, p.generate(t, "--deny-warnings", "--allow-warning", code))
		if len(r.Diagnostics) != 1 || len(r.Warnings) != 2 {
			t.Fatal(r)
		}
	}
	result(t, p.generate(t, "--deny-warnings", "--allow-warning=W_ALL_OF_APPROXIMATION", "--allow-warning", "W_JSON_ELEMENT_APPROXIMATION", "--allow-warning", "W_JSON_ELEMENT_APPROXIMATION", "--allow-warning=W_FUTURE_UNUSED"), 0)
	result(t, p.generate(t, "--deny-warnings", "--allow-warning=W_ALL_OF_APPROXIMATION", "--allow-warning=W_JSON_ELEMENT_APPROXIMATION", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("whitelist changed outputs/mtimes")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Raw"].(map[string]any)["description"] = "synthetic changed metadata"
		schemas(doc)["Added"] = map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}}}
	})
	result(t, p.generate(t, "--check"), 3)
	warningDenied(t, p.generate(t, "--check", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("denial/check difference wrote output")
	}
}
func TestWarningPolicyDenialNoOutputIO(t *testing.T) {
	p := warningProject(t)
	p.Output = filepath.Join(p.Root, "missing", "out")
	warningDenied(t, p.generate(t, "--deny-warnings"))
	if _, err := os.Stat(filepath.Dir(p.Output)); !os.IsNotExist(err) {
		t.Fatal("created output parent", err)
	}
	p.Output = filepath.Join(p.Root, "not-a-directory")
	put(t, p.Output, []byte("synthetic blocked path"))
	warningDenied(t, p.generate(t, "--deny-warnings"))
	if string(read(t, p.Output)) != "synthetic blocked path" {
		t.Fatal("modified blocked output")
	}
}
func TestWarningPolicyValidateAndText(t *testing.T) {
	p := warningProject(t)
	r := warningDenied(t, invoke(t, "", "validate", "--input", p.Input, "--deny-warnings", "--diagnostic-format=json"))
	if len(r.Warnings) != 2 {
		t.Fatal(r)
	}
	text := invoke(t, "", "validate", "--input", p.Input, "--deny-warnings")
	if text.Code != 1 || len(text.Stdout) != 0 || !strings.Contains(string(text.Stderr), "E_WARNING_DENIED") || !strings.Contains(string(text.Stderr), "W_ALL_OF_APPROXIMATION") {
		t.Fatal(text)
	}
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"id":{"type":"string"}}}}}}`))
	result(t, p.generate(t, "--deny-warnings"), 0)
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"type":"unimplemented"}}}}`))
	r = fails(t, p.generate(t, "--deny-warnings"), 1, "E_SCHEMA_UNSUPPORTED")
	if len(r.Warnings) != 0 {
		t.Fatal(r)
	}
}
