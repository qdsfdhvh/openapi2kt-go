package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

var binary, repo string

// Tests build and execute the native Go command. No Cargo/Rust artifact or
// undeclared checkout participates; process scratch stays in the ignored repo.
func TestMain(m *testing.M) {
	repo = testutil.RepoRoot()
	base := filepath.Join(repo, ".pi", "go-migration", "process-tests")
	if err := os.MkdirAll(base, 0700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	scratch, err := os.MkdirTemp(base, "run-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.Setenv("TMPDIR", scratch)
	binary = filepath.Join(scratch, "openapi-kt")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	command := exec.Command("go", "build", "-o", binary, "./cmd/openapi-kt")
	command.Dir = repo
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	if data, err := command.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "native Go CLI build failed: %v\n%s", err, data)
		_ = os.RemoveAll(scratch)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(scratch)
	os.Exit(code)
}

type invocation struct {
	Code           int
	Stdout, Stderr []byte
}

func invoke(t testing.TB, cwd string, args ...string) invocation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir = cwd
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if ctx.Err() != nil {
		t.Fatal("CLI timed out")
	}
	return invocation{code, stdout.Bytes(), stderr.Bytes()}
}

type resultReport struct {
	Status                string
	Files                 []string
	Diagnostics, Warnings core.Diagnostics
}

func result(t testing.TB, output invocation, code int) resultReport {
	t.Helper()
	if output.Code != code || len(output.Stderr) != 0 {
		t.Fatalf("exit want %d got %d stdout=%s stderr=%s", code, output.Code, output.Stdout, output.Stderr)
	}
	var report resultReport
	decoder := json.NewDecoder(bytes.NewReader(output.Stdout))
	if err := decoder.Decode(&report); err != nil {
		t.Fatal("not JSON", err, string(output.Stdout))
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatal("not exactly one JSON document", err)
	}
	return report
}
func fails(t testing.TB, output invocation, exit int, code string) resultReport {
	t.Helper()
	report := result(t, output, exit)
	if report.Status != "error" || len(report.Files) != 0 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != code || len(report.Warnings) != 0 {
		t.Fatal(report)
	}
	return report
}

type project struct{ Root, Input, Output string }

func newProject(t testing.TB) project {
	t.Helper()
	root := t.TempDir()
	p := project{root, filepath.Join(root, "catalog.json"), filepath.Join(root, "generated")}
	data, err := os.ReadFile(filepath.Join(repo, "tests", "fixtures", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, data)
	return p
}
func put(t testing.TB, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func read(t testing.TB, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func (p project) generate(t testing.TB, extra ...string) invocation {
	return invoke(t, "", append([]string{"generate", "--input", p.Input, "--output", p.Output, "--package-name", "org.example.catalog", "--diagnostic-format", "json"}, extra...)...)
}
func (p project) change(t testing.TB, change func(map[string]any)) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(read(t, p.Input), &doc); err != nil {
		t.Fatal(err)
	}
	change(doc)
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, data)
}
func schemas(doc map[string]any) map[string]any {
	return doc["components"].(map[string]any)["schemas"].(map[string]any)
}

type snap struct {
	Bytes    string
	Modified time.Time
	Mode     os.FileMode
}

func snapshot(t testing.TB, root string) map[string]snap {
	t.Helper()
	out := map[string]snap{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[relative] = snap{string(read(t, path)), info.ModTime(), info.Mode()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestActualCLIWarningsChannelsControlsAndFailureSuppression(t *testing.T) {
	p := newProject(t)
	p.change(t, func(doc map[string]any) {
		doc["components"].(map[string]any)["schemas"] = map[string]any{"Record": map[string]any{"type": "object", "properties": map[string]any{"empty": map[string]any{"type": "object"}, "kind": map[string]any{"type": "string", "const": "synthetic_payload_marker"}}}}
	})
	difference := result(t, p.generate(t, "--check"), 3)
	if difference.Status != "different" || len(difference.Diagnostics) != 0 || len(difference.Warnings) != 1 || difference.Warnings[0].Code != "W_JSON_ELEMENT_APPROXIMATION" || *difference.Warnings[0].Pointer != "/components/schemas/Record/properties/empty" {
		t.Fatal(difference)
	}
	if _, err := os.Stat(p.Output); !os.IsNotExist(err) {
		t.Fatal("check created output")
	}
	generated := result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	root, _ := os.Stat(p.Output)
	if !reflect.DeepEqual(result(t, p.generate(t, "--check"), 0).Warnings, generated.Warnings) {
		t.Fatal("check warning drift")
	}
	validated := result(t, invoke(t, "", "validate", "--input", p.Input, "--package-name", "org.example.catalog", "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(validated.Warnings, generated.Warnings) {
		t.Fatal("validate warning drift")
	}
	after, _ := os.Stat(p.Output)
	if !reflect.DeepEqual(snapshot(t, p.Output), before) || !root.ModTime().Equal(after.ModTime()) {
		t.Fatal("check/validate mutated output")
	}
	encoded, _ := json.Marshal(generated)
	if bytes.Contains(encoded, []byte("synthetic_payload_marker")) || bytes.Contains(encoded, []byte(p.Root)) {
		t.Fatal("report echoed payload/path")
	}
	put(t, filepath.Join(p.Output, "org/example/catalog/models/Record.kt"), []byte("manual edit"))
	before = snapshot(t, p.Output)
	failed := fails(t, p.generate(t), 4, "E_OUTPUT_MODIFIED")
	if len(failed.Warnings) != 0 || !reflect.DeepEqual(snapshot(t, p.Output), before) {
		t.Fatal("failed output returned warnings or mutated tree")
	}
	p.change(t, func(doc map[string]any) {
		doc["components"].(map[string]any)["schemas"] = map[string]any{"Record": map[string]any{"type": "object", "properties": map[string]any{"empty\n\x1b[31m/~": map[string]any{"type": "object"}}}}
	})
	text := invoke(t, "", "validate", "--input", p.Input)
	if text.Code != 0 || string(text.Stdout) != "ok: 1 model file(s)\n" || !bytes.HasPrefix(text.Stderr, []byte("W_JSON_ELEMENT_APPROXIMATION at ")) || bytes.ContainsRune(text.Stderr, '\x1b') || !bytes.Contains(text.Stderr, []byte(`\n`)) || !bytes.Contains(text.Stderr, []byte("~1~0")) || bytes.Count(text.Stderr, []byte("\n")) != 1 {
		t.Fatalf("unsafe text streams %+v", text)
	}
}

func TestActualCLIWarningFreeShapeNoopCheckAndCrossCwd(t *testing.T) {
	p := newProject(t)
	p.change(t, func(doc map[string]any) {
		doc["components"].(map[string]any)["schemas"] = map[string]any{"Record": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}}}
	})
	generated := p.generate(t)
	result(t, generated, 0)
	var raw map[string]any
	if err := json.Unmarshal(generated.Stdout, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 3 || raw["warnings"] != nil || raw["status"] != "ok" {
		t.Fatal(raw)
	}
	before := snapshot(t, p.Output)
	other := filepath.Join(p.Root, "other-cwd")
	if err := os.Mkdir(other, 0700); err != nil {
		t.Fatal(err)
	}
	for _, check := range []bool{true, false} {
		args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name", "org.example.catalog", "--diagnostic-format", "json"}
		if check {
			args = append(args, "--check")
		}
		result(t, invoke(t, other, args...), 0)
		if !reflect.DeepEqual(snapshot(t, p.Output), before) {
			t.Fatal("no-op/cwd changed mtimes/content")
		}
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Record"].(map[string]any)["properties"].(map[string]any)["extra"] = map[string]any{"type": "string"}
	})
	if result(t, p.generate(t, "--check"), 3).Status != "different" || !reflect.DeepEqual(snapshot(t, p.Output), before) {
		t.Fatal("check mutated difference")
	}
	result(t, p.generate(t), 0)
	if !bytes.Contains(read(t, filepath.Join(p.Output, "org/example/catalog/models/Record.kt")), []byte("val extra:")) {
		t.Fatal("generation unchanged")
	}
}

func TestActualCLIInputBoundsInvalidInputFormatsIOAndUsage(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	data := read(t, p.Input)
	data = append(data, bytes.Repeat([]byte(" "), core.MaxDocumentBytes-len(data))...)
	put(t, p.Input, data)
	result(t, p.generate(t, "--check"), 0)
	put(t, p.Input, append(data, ' '))
	fails(t, p.generate(t), 1, "E_INPUT_LIMIT")
	put(t, p.Input, []byte("{ synthetic_private_payload"))
	out := p.generate(t)
	fails(t, out, 1, "E_JSON_INVALID")
	if bytes.Contains(out.Stdout, []byte("synthetic_private_payload")) {
		t.Fatal("payload leaked")
	}
	fails(t, p.generate(t, "--input-format", "yaml"), 1, "E_YAML_INVALID")
	fails(t, p.generate(t, "--not-a-real-flag"), 2, "E_USAGE")
	if err := os.Remove(p.Input); err != nil {
		t.Fatal(err)
	}
	fails(t, p.generate(t), 4, "E_INPUT_IO")
	if !reflect.DeepEqual(snapshot(t, p.Output), before) {
		t.Fatal("input/usage errors changed existing outputs")
	}
}

func TestActualCLIOwnershipStalePackageAndMissingFiles(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "Notes.kt"), []byte("handwritten"))
	model := filepath.Join(p.Output, "org/example/catalog/models/Item.kt")
	if err := os.Remove(model); err != nil {
		t.Fatal(err)
	}
	result(t, p.generate(t, "--check"), 3)
	result(t, p.generate(t), 0)
	put(t, model, []byte("manual"))
	before := snapshot(t, p.Output)
	for _, check := range []bool{true, false} {
		extra := []string{}
		if check {
			extra = append(extra, "--check")
		}
		fails(t, p.generate(t, extra...), 4, "E_OUTPUT_MODIFIED")
		if !reflect.DeepEqual(snapshot(t, p.Output), before) {
			t.Fatal("manual edit changed")
		}
	}
	p.change(t, func(doc map[string]any) { delete(schemas(doc), "Item") })
	fails(t, p.generate(t), 4, "E_OUTPUT_MODIFIED")
	// Restore through the known manifest hash, without adopting unowned files.
	// The original fixture model is regenerated independently into another root.
	seed := newProject(t)
	result(t, seed.generate(t), 0)
	put(t, model, read(t, filepath.Join(seed.Output, "org/example/catalog/models/Item.kt")))
	result(t, p.generate(t), 0)
	if _, err := os.Stat(model); !os.IsNotExist(err) {
		t.Fatal("stale file survived")
	}
	result(t, invoke(t, "", "generate", "--input", p.Input, "--output", p.Output, "--package-name", "org.example.renamed", "--diagnostic-format", "json"), 0)
	if _, err := os.Stat(filepath.Join(p.Output, "org/example/catalog/models/Payload.kt")); !os.IsNotExist(err) {
		t.Fatal("old package file survived")
	}
	if string(read(t, filepath.Join(p.Output, "Notes.kt"))) != "handwritten" {
		t.Fatal("unowned deleted")
	}
	if err := os.Remove(filepath.Join(p.Output, ".openapi-kt-manifest.json")); err != nil {
		t.Fatal(err)
	}
	before = snapshot(t, p.Output)
	for _, check := range []bool{true, false} {
		args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name", "org.example.renamed", "--diagnostic-format", "json"}
		if check {
			args = append(args, "--check")
		}
		fails(t, invoke(t, "", args...), 4, "E_OUTPUT_COLLISION")
		if !reflect.DeepEqual(snapshot(t, p.Output), before) {
			t.Fatal("unowned adopted")
		}
	}
}

func TestActualCLIHelpVersionAndClosedArgumentSurface(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"generate", "--help"}, {"validate", "--help"}, {"help", "generate"}, {"--version"}, {"-V"}} {
		out := invoke(t, "", args...)
		expectsConfig := len(args) == 2 && (args[0] == "generate" || args[0] == "validate" || args[0] == "help" && args[1] == "generate")
		if out.Code != 0 || len(out.Stderr) > 0 || bytes.Contains(out.Stdout, []byte("--config")) != expectsConfig {
			t.Fatal(args, out)
		}
	}
	// The source CLI intentionally removes the old downloader surface.
	if out := invoke(t, "", "--help"); bytes.Contains(out.Stdout, []byte("download")) {
		t.Fatal("removed download command still advertised")
	}
	for _, args := range [][]string{{"download", "--help"}, {"help", "download"}} {
		if out := invoke(t, "", args...); out.Code != 2 || len(out.Stdout) != 0 {
			t.Fatal("removed help command must fail with usage", args, out)
		}
	}
	if out := invoke(t, "", "--version"); string(out.Stdout) != "openapi-kt 0.2.0-alpha.1\n" {
		t.Fatal(string(out.Stdout))
	}
	for _, args := range [][]string{{"--diagnostic-format=json"}, {"generate", "--check=true", "--diagnostic-format", "json"}, {"validate", "--model-name-suffix", "Dto", "--diagnostic-format=json"}, {"generate", "--input-format", "YAML", "--diagnostic-format=json"}, {"validate", "--input=x", "--input=y", "--diagnostic-format=json"}, {"download", "--diagnostic-format=json"}} {
		fails(t, invoke(t, "", args...), 2, "E_USAGE")
	}
	out := invoke(t, "", "validate")
	if out.Code != 2 || len(out.Stdout) != 0 || len(out.Stderr) == 0 {
		t.Fatal("text usage stream")
	}
}

func TestActualCLIAbandonedStageBlocksGenerationAndCheck(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	if err := os.Mkdir(filepath.Join(p.Output, ".openapi-kt-stage-interrupted"), 0700); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, p.Output)
	for _, check := range []bool{true, false} {
		extra := []string{}
		if check {
			extra = append(extra, "--check")
		}
		fails(t, p.generate(t, extra...), 4, "E_RECOVERY_REQUIRED")
		if !reflect.DeepEqual(snapshot(t, p.Output), before) {
			t.Fatal("recovery evidence changed")
		}
	}
}
