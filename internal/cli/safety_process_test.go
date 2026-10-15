package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestActualCheckDoesNotCreateCoordinationFileAndManifestHashesAreStable(t *testing.T) {
	p := newProject(t)
	generated := result(t, p.generate(t), 0)
	var manifest struct {
		Version uint32            `json:"version"`
		Files   map[string]string `json:"files"`
	}
	data := read(t, filepath.Join(p.Output, ".openapi-kt-manifest.json"))
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != 1 || len(manifest.Files) != len(generated.Files) {
		t.Fatal(manifest)
	}
	for _, name := range generated.Files {
		sum := sha256.Sum256(read(t, filepath.Join(p.Output, filepath.FromSlash(name))))
		if manifest.Files[name] != hex.EncodeToString(sum[:]) {
			t.Fatal("SHA256 ownership drift", name)
		}
	}
	if err := os.Remove(filepath.Join(p.Output, ".openapi-kt.lock")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, p.Output)
	meta, _ := os.Stat(p.Output)
	result(t, p.generate(t, "--check"), 0)
	after, _ := os.Stat(p.Output)
	if !reflect.DeepEqual(snapshot(t, p.Output), before) || !meta.ModTime().Equal(after.ModTime()) {
		t.Fatal("readonly check created lock/changed root")
	}
	if _, err := os.Stat(filepath.Join(p.Output, ".openapi-kt.lock")); !os.IsNotExist(err) {
		t.Fatal("check created lock")
	}
}

func TestActualGoProcessMalformedManifestSlotsAndUnsafePathsFailClosed(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	for _, tc := range []struct{ data, code string }{
		{"bad", "E_MANIFEST_INVALID"}, {`{"version":2,"files":{}}`, "E_MANIFEST_VERSION"},
		{`{"version":1,"files":{"../manual.kt":"` + strings.Repeat("0", 64) + `"}}`, "E_OUTPUT_PATH"},
		{`{"version":1,"files":{},"other":true}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"A.kt":"0","A.kt":"0"}}`, "E_MANIFEST_INVALID"},
		{`{"version":1,"files":{"A.kt":"` + strings.Repeat("g", 64) + `"}}`, "E_MANIFEST_INVALID"},
		{`{"version":9,"files":{"A.kt":true}}`, "E_MANIFEST_INVALID"},
	} {
		put(t, filepath.Join(p.Output, ".openapi-kt-manifest.json"), []byte(tc.data))
		before := snapshot(t, p.Output)
		for _, check := range []bool{true, false} {
			extra := []string{}
			if check {
				extra = append(extra, "--check")
			}
			fails(t, p.generate(t, extra...), 4, tc.code)
			if !reflect.DeepEqual(snapshot(t, p.Output), before) {
				t.Fatal("malformed manifest mutated output")
			}
		}
	}
}
