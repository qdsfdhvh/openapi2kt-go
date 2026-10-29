package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootMoveMultipleOwnersNativeRawLocksAndAllSelection(t *testing.T) {
	for _, all := range []bool{false, true} {
		base, cfg, src, dst, _ := multipleMoveFixture(t, "locked", false)
		m := multipleMoveState(t, src)
		for owner := range m.NameLocks {
			path := filepath.Join(src, namedLockPath(owner))
			raw := append(read(t, path), ' ', '\n')
			put(t, path, raw)
			sum := sha256.Sum256(raw)
			m.NameLocks[owner] = hex.EncodeToString(sum[:])
		}
		raw, e := json.MarshalIndent(m, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		put(t, filepath.Join(src, ".openapi-kt-manifest.json"), append(raw, ' ', '\n'))
		before := snapshot(t, src)
		args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", src, "--output", dst, "--move-owner", "b", "--move-owner=a"}
		if all {
			args = append(args, "--move-owner=c")
		}
		result(t, invoke(t, base, append(append([]string{}, args...), "--check")...), 3)
		result(t, invoke(t, base, lifecycleValidate(args)...), 0)
		if !reflect.DeepEqual(before, snapshot(t, src)) {
			t.Fatal("readonly raw metadata changed")
		}
		result(t, invoke(t, base, args...), 0)
		after := snapshot(t, dst)
		for _, owner := range []string{"a", "b", "c"} {
			root := after
			if owner == "c" && !all {
				root = snapshot(t, src)
			}
			if !reflect.DeepEqual(before[namedLockPath(owner)], root[namedLockPath(owner)]) {
				t.Fatal("noncanonical raw lock rewritten", owner)
			}
		}
		if all && !reflect.DeepEqual(before[".openapi-kt-manifest.json"], after[".openapi-kt-manifest.json"]) {
			t.Fatal("all selection must use full raw manifest preservation")
		}
	}
}

func TestRootMoveMultipleOwnersNativeHelpTextAndWarningPolicy(t *testing.T) {
	for _, cmd := range []string{"generate", "validate"} {
		out := invoke(t, repo, cmd, "--help")
		if out.Code != 0 || !bytes.Contains(out.Stdout, []byte("--move-owner <NAME>")) || !bytes.Contains(out.Stdout, []byte("repeatable, excludes --spec")) {
			t.Fatal("help", out)
		}
	}
	base := t.TempDir()
	names := []string{"a", "b", "c"}
	lifecycleConfigInputs(t, base, names)
	cfg := lifecycleConfig(t, base, names, false, false, false)
	warning := []byte(`{"openapi":"3.1.0","info":{"title":"Synthetic","version":"1"},"paths":{},"components":{"schemas":{"Record":{"type":"object","properties":{"empty":{"type":"object"}}}}}}`)
	put(t, filepath.Join(base, "c.json"), warning)
	result(t, invoke(t, base, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	src, dst := filepath.Join(base, "generated"), filepath.Join(base, "target")
	before := snapshot(t, src)
	text := strings.Replace(string(read(t, cfg)), "name=\"c\"", "name=\"c\"\ndeny_warnings=true", 1)
	put(t, cfg, []byte(text))
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json", "--move-output-from", src, "--output", dst, "--move-owner=a", "--move-owner=b"}
	for _, run := range [][]string{args, append(append([]string{}, args...), "--check"), lifecycleValidate(args)} {
		out := invoke(t, base, run...)
		r := result(t, out, 1)
		if len(r.Diagnostics) == 0 || r.Diagnostics[0].Code != "E_WARNING_DENIED" || len(r.Warnings) == 0 {
			t.Fatal("retained source policy bypassed", r)
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, src)) {
		t.Fatal("warning denial wrote")
	}
	if _, e := os.Lstat(dst); !os.IsNotExist(e) {
		t.Fatal("warning denial created target")
	}
	// Invocation allowlist applies to every regenerated source, retained c included.
	result(t, invoke(t, base, append(append([]string{}, args...), "--deny-warnings", "--allow-warning=W_JSON_ELEMENT_APPROXIMATION", "--check")...), 3)
	// Text exposes the same sorted membership; full source inputs remain visible.
	put(t, cfg, []byte(strings.Replace(text, "deny_warnings=true", "deny_warnings=false", 1)))
	plain := []string{}
	for _, arg := range args {
		if arg != "--diagnostic-format=json" {
			plain = append(plain, arg)
		}
	}
	out := invoke(t, base, append(plain, "--check")...)
	if out.Code != 3 || !bytes.Contains(out.Stdout, []byte(`moved owners="[a b]" retained owners="[c]"`)) {
		t.Fatal("text membership", out.Code, string(out.Stdout))
	}
}
