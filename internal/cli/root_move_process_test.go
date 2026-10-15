package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootMoveNativeDirectVersionsReportAndPostMoveNoop(t *testing.T) {
	for _, locked := range []bool{false, true} {
		base := t.TempDir()
		src, dst := filepath.Join(base, "old"), filepath.Join(base, "new")
		input := filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")
		args := []string{"generate", "--input", input, "--output", src, "--package-name=org.example.moved", "--model-name-suffix=Dto", "--hoist-shared-shapes", "--qualify-hoisted-names", "--deduplicate-models", "--validation", "--diagnostic-format=json"}
		if locked {
			args = append(args, "--lock-hoisted-names")
		}
		result(t, invoke(t, base, args...), 0)
		put(t, filepath.Join(src, "Notes.kt"), []byte("// handwritten source"))
		if e := os.Mkdir(dst, 0700); e != nil {
			t.Fatal(e)
		}
		put(t, filepath.Join(dst, "Notes.kt"), []byte("// handwritten destination"))
		before := snapshot(t, src)
		dest := snapshot(t, dst)
		run := append(append([]string(nil), args...), "--move-output-from", src)
		for i, v := range run {
			if v == "--output" {
				run[i+1] = dst
				break
			}
		}
		checked := invoke(t, base, append(run, "--check")...)
		result(t, checked, 3)
		valid := invoke(t, base, lifecycleValidate(run)...)
		result(t, valid, 0)
		if !reflect.DeepEqual(before, snapshot(t, src)) || !reflect.DeepEqual(dest, snapshot(t, dst)) {
			t.Fatal("readonly changed roots")
		}
		applied := invoke(t, base, run...)
		result(t, applied, 0)
		var expected json.RawMessage
		for _, out := range []invocation{checked, valid, applied} {
			var r map[string]json.RawMessage
			if e := json.Unmarshal(out.Stdout, &r); e != nil {
				t.Fatal(e)
			}
			if len(r["output_root_migration"]) == 0 || bytes.Contains(out.Stdout, []byte(base)) {
				t.Fatal("missing/unsafe report", string(out.Stdout))
			}
			if expected == nil {
				expected = r["output_root_migration"]
			} else if !bytes.Equal(expected, r["output_root_migration"]) {
				t.Fatal("report varies")
			}
		}
		after := snapshot(t, dst)
		for path, old := range before {
			if path == ".openapi-kt.lock" || path == "Notes.kt" {
				continue
			}
			if !reflect.DeepEqual(old, after[path]) {
				t.Fatal("copied source differs", path)
			}
			if _, e := os.Lstat(filepath.Join(src, path)); !os.IsNotExist(e) {
				t.Fatal("owned source remains", path)
			}
		}
		if !reflect.DeepEqual(before["Notes.kt"], snapshot(t, src)["Notes.kt"]) || !reflect.DeepEqual(dest["Notes.kt"], after["Notes.kt"]) {
			t.Fatal("foreign file lost")
		}
		ordinary := append([]string(nil), args...)
		for i, v := range ordinary {
			if v == "--output" {
				ordinary[i+1] = dst
				break
			}
		}
		result(t, invoke(t, base, append(ordinary, "--check")...), 0)
		result(t, invoke(t, base, ordinary...), 0)
		if !reflect.DeepEqual(after, snapshot(t, dst)) {
			t.Fatal("ordinary postmove generation changed bytes/mtime")
		}
		identity := append(ordinary, "--move-output-from", dst)
		result(t, invoke(t, base, append(identity, "--check")...), 0)
		result(t, invoke(t, base, identity...), 0)
		if !reflect.DeepEqual(after, snapshot(t, dst)) {
			t.Fatal("identity wrote")
		}
	}
}

func TestRootMoveNativeFullSharedInventoryAndFailClosedSelection(t *testing.T) {
	for _, locked := range []bool{false, true} {
		base := t.TempDir()
		names := []string{"alpha", "beta"}
		lifecycleConfigInputs(t, base, names)
		cfg := lifecycleConfig(t, base, names, locked, false, true)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
		result(t, invoke(t, base, args...), 0)
		src, dst := filepath.Join(base, "generated"), filepath.Join(base, "new")
		run := append(append([]string(nil), args...), "--move-output-from", src, "--output", dst)
		// Full inventory output override is the explicit move binding, not emission options.
		before := snapshot(t, src)
		fails(t, invoke(t, base, append(run, "--spec=alpha")...), 2, "E_USAGE")
		full := read(t, cfg)
		cut := strings.Index(string(full), "[[specs]]\nname=\"beta\"")
		put(t, cfg, full[:cut])
		fails(t, invoke(t, base, run...), 4, "E_OUTPUT_ROOT_MOVE_INVALID")
		put(t, cfg, full)
		// Current config points to the target explicitly; changing the path alone
		// must never discover or clean the old root.
		put(t, cfg, []byte(strings.ReplaceAll(string(full), "output_dir='generated'", "output_dir='new'")))
		checked := invoke(t, base, append(run, "--check")...)
		result(t, checked, 3)
		result(t, invoke(t, base, lifecycleValidate(run)...), 0)
		if !reflect.DeepEqual(before, snapshot(t, src)) {
			t.Fatal("selection/readonly changed source")
		}
		result(t, invoke(t, base, run...), 0)
		after := snapshot(t, dst)
		for path, old := range before {
			if path != ".openapi-kt.lock" && !reflect.DeepEqual(old, after[path]) {
				t.Fatal("full inventory lost", path)
			}
		}
		result(t, invoke(t, base, append(args, "--check")...), 0)
		result(t, invoke(t, base, args...), 0)
		if !reflect.DeepEqual(after, snapshot(t, dst)) {
			t.Fatal("postmove source changed")
		}
	}
}

func TestRootMoveNativeParserHelpAndChangedCandidate(t *testing.T) {
	base := t.TempDir()
	src, dst := filepath.Join(base, "old"), filepath.Join(base, "new")
	input := filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json")
	args := []string{"generate", "--input", input, "--output", src, "--package-name=org.example.moved", "--validation", "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	before := snapshot(t, src)
	for _, extra := range [][]string{{"--move-output-from="}, {"--move-output-from", src, "--move-output-from", src}, {"--move-output-from", src, "--name-lock-action=reset"}, {"--move-output-from", src, "--rename-owner=alpha:beta"}} {
		fails(t, invoke(t, base, append(args, extra...)...), 2, "E_USAGE")
	}
	run := append(append([]string(nil), args...), "--move-output-from", src)
	for i, v := range run {
		if v == "--output" {
			run[i+1] = dst
			break
		}
	}
	changed := append(run, "--model-name-suffix=Drift")
	for _, call := range [][]string{changed, append(changed, "--check"), lifecycleValidate(changed)} {
		fails(t, invoke(t, base, call...), 4, "E_OUTPUT_ROOT_MOVE_CHANGED")
	}
	if !reflect.DeepEqual(before, snapshot(t, src)) {
		t.Fatal("changed candidate overwrote source")
	}
	for _, command := range []string{"generate", "validate"} {
		out := invoke(t, base, command, "--help")
		if out.Code != 0 || !bytes.Contains(out.Stdout, []byte("--move-output-from <SOURCE>")) {
			t.Fatal("help missing root move")
		}
	}
}
