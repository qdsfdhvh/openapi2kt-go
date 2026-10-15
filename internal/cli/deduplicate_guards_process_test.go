package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeduplicateFailedIOAndInputPreserveOldTree(t *testing.T) {
	root := t.TempDir()
	args := deduplicateArgs(root)
	result(t, invoke(t, repo, args...), 0)
	if e := os.WriteFile(filepath.Join(root, "handwritten.txt"), []byte("synthetic"), 0600); e != nil {
		t.Fatal(e)
	}
	owned := filepath.Join(root, "org/example/dedup/api/models/BetaResponseDto.kt")
	if e := os.WriteFile(owned, []byte("changed user content"), 0600); e != nil {
		t.Fatal(e)
	}
	before := snapshot(t, root)
	r := result(t, invoke(t, repo, append(args, "--deduplicate-models")...), 4)
	if r.Diagnostics[0].Code != "E_OUTPUT_MODIFIED" || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal(r, "modified owned file or handwritten data changed")
	}
	bad := filepath.Join(t.TempDir(), "input.json")
	if e := os.WriteFile(bad, []byte("{"), 0600); e != nil {
		t.Fatal(e)
	}
	failed := append([]string{}, args...)
	failed[2] = bad
	failed = append(failed, "--deduplicate-models")
	r = result(t, invoke(t, repo, failed...), 1)
	if r.Diagnostics[0].Code != "E_JSON_INVALID" || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal(r, "failed input changed output")
	}
}
