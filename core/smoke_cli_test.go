package core_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"openapi-kt/core"
)

// This opt-in seam verifies actual CLI-produced Kotlin, not a second copy of
// the core output. The built executable must be this repository's Go command.
func exportThroughCLI(t testing.TB, root, directory string, rq core.GenerationRequest) bool {
	t.Helper()
	binary := os.Getenv("OPENAPI_KT_TEST_BINARY")
	if binary == "" {
		return false
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.Path != "openapi-kt/cmd/openapi-kt" || info.Main.Path != "openapi-kt" {
		t.Fatal("export requires a native openapi-kt Go binary", info, err)
	}
	expected, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(directory, err)
	}
	inputDir := filepath.Join(root, "inputs", directory)
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(rq.SourceName)
	if name == "." || name == "" {
		name = "input.json"
	}
	input := filepath.Join(inputDir, name)
	if err := os.WriteFile(input, rq.Document, 0644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "generated", directory)
	format := string(rq.Format)
	if format == "" {
		format = "auto"
	}
	args := []string{"generate", "--input", input, "--output", output, "--package-name", rq.Options.PackageName, "--model-name-suffix", rq.Options.ModelNameSuffix, "--input-format", format, "--diagnostic-format", "json"}
	if rq.Options.Validation {
		args = append(args, "--validation")
	}
	run := func(extra ...string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, binary, append(append([]string{}, args...), extra...)...)
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err != nil || stderr.Len() != 0 {
			t.Fatalf("CLI export %s: %v stdout=%s stderr=%s", directory, err, stdout.Bytes(), stderr.Bytes())
		}
		var report struct {
			Status                string
			Files                 []string
			Diagnostics, Warnings core.Diagnostics
		}
		decoder := json.NewDecoder(&stdout)
		if err := decoder.Decode(&report); err != nil {
			t.Fatal(err)
		}
		var extraJSON any
		if err := decoder.Decode(&extraJSON); err != io.EOF {
			t.Fatal("extra CLI stdout", err)
		}
		paths := make([]string, 0, len(expected.Artifacts.Files))
		for _, file := range expected.Artifacts.Files {
			paths = append(paths, file.Path)
		}
		if report.Status != "ok" || len(report.Diagnostics) != 0 || !reflect.DeepEqual(report.Files, paths) || (len(report.Warnings) != len(expected.Warnings) || len(expected.Warnings) > 0 && !reflect.DeepEqual([]core.Diagnostic(report.Warnings), expected.Warnings)) {
			t.Fatal("CLI/core report mismatch", directory, report, expected.Warnings)
		}
	}
	run()
	var manifest struct {
		Version uint32
		Files   map[string]string
	}
	manifestPath := filepath.Join(output, ".openapi-kt-manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != 1 || len(manifest.Files) != len(expected.Artifacts.Files) {
		t.Fatal("CLI manifest inventory", manifest)
	}
	mtimes := map[string]time.Time{}
	mtimes[manifestPath] = mustMtime(t, manifestPath)
	for _, file := range expected.Artifacts.Files {
		path := filepath.Join(output, filepath.FromSlash(file.Path))
		data, err := os.ReadFile(path)
		if err != nil || string(data) != file.Content {
			t.Fatal("actual CLI bytes differ from core", directory, file.Path, err)
		}
		sum := sha256.Sum256(data)
		if manifest.Files[file.Path] != hex.EncodeToString(sum[:]) {
			t.Fatal("CLI manifest hash mismatch")
		}
		mtimes[path] = mustMtime(t, path)
	}
	count := 0
	if err := filepath.WalkDir(output, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".kt" {
			count++
		}
		return nil
	}); err != nil || count != len(expected.Artifacts.Files) {
		t.Fatal("CLI extra/missing Kotlin files", count, err)
	}
	run("--check")
	run()
	for path, before := range mtimes {
		if !before.Equal(mustMtime(t, path)) {
			t.Fatal("CLI export check/noop changed mtime", directory)
		}
	}
	return true
}
func mustMtime(t testing.TB, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime()
}
