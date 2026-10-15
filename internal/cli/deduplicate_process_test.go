package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func deduplicateArgs(root string) []string {
	return []string{"generate", "--input", filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json"), "--output", root, "--package-name=org.example.dedup", "--model-name-suffix=Dto", "--validation", "--diagnostic-format=json"}
}
func TestDeduplicateDefaultControl(t *testing.T) {
	root := t.TempDir()
	args := deduplicateArgs(root)
	result(t, invoke(t, repo, args...), 0)
	for _, name := range []string{"AlphaResponseDto", "BetaResponseDto", "GuardResponseDto", "AlphaInputDto", "BetaInputDto", "AlphaModeDto", "BetaModeDto"} {
		b, e := os.ReadFile(filepath.Join(root, "org/example/dedup/api/models", name+".kt"))
		if e != nil || strings.Contains(string(b), "public typealias ") {
			t.Fatal(name, e, string(b))
		}
	}
	before := snapshot(t, root)
	result(t, invoke(t, repo, append(args, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("default control mutated artifacts")
	}
}
func TestDeduplicateDesiredSuccess(t *testing.T) {
	root := t.TempDir()
	args := append(deduplicateArgs(root), "--deduplicate-models")
	result(t, invoke(t, repo, args...), 0)
	for _, tc := range []struct{ name, target string }{{"BetaResponseDto", "AlphaResponseDto"}, {"BetaInputDto", "AlphaInputDto"}, {"BetaModeDto", "AlphaModeDto"}} {
		b, e := os.ReadFile(filepath.Join(root, "org/example/dedup/api/models", tc.name+".kt"))
		if e != nil || !strings.Contains(string(b), "public typealias "+tc.name+" = org.example.dedup.api.models."+tc.target) {
			t.Fatal(tc, e, string(b))
		}
	}
	b, e := os.ReadFile(filepath.Join(root, "org/example/dedup/api/models/GuardResponseDto.kt"))
	if e != nil || strings.Contains(string(b), "public typealias ") {
		t.Fatal("distinct checks incorrectly merged", e, string(b))
	}
}
