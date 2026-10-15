package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlainStyleDefaultControl(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(repo, "tests/fixtures/model-semantics/plain-style-baseline.json")
	args := []string{"generate", "--input", input, "--output", root, "--package-name=org.example.plainprobe", "--model-name-suffix=Dto", "--diagnostic-format=json"}
	result(t, invoke(t, repo, args...), 0)
	data, e := os.ReadFile(filepath.Join(root, "org/example/plainprobe/models/SampleDto.kt"))
	if e != nil || !strings.Contains(string(data), "public data class SampleDto(") {
		t.Fatal(e, string(data))
	}
	before := snapshot(t, root)
	result(t, invoke(t, repo, append(args, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("default check changed artifacts")
	}
}
func TestPlainStyleNativeSuccess(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fresh")
	args := []string{"generate", "--input", filepath.Join(repo, "tests/fixtures/model-semantics/plain-style-baseline.json"), "--output", root, "--package-name=org.example.plainprobe", "--model-name-suffix=Dto", "--model-style=plain_class", "--diagnostic-format=json"}
	result(t, invoke(t, repo, append(append([]string{}, args...), "--check")...), 3)
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("check created plain output")
	}
	result(t, invoke(t, repo, args...), 0)
	data, e := os.ReadFile(filepath.Join(root, "org/example/plainprobe/models/SampleDto.kt"))
	if e != nil || !strings.Contains(string(data), "public class SampleDto(") || strings.Contains(string(data), "public data class") || !strings.Contains(string(data), "override fun equals(") || !strings.Contains(string(data), "override fun hashCode(") {
		t.Fatal(e, string(data))
	}
	before := snapshot(t, root)
	result(t, invoke(t, repo, append(args, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("plain check changed output")
	}
}
