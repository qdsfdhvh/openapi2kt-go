package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"openapi-kt/internal/testutil"
)

// Seed ownership from unchanged legacy goldens, not this Go generator. The
// language switch must accept manifest v1 without adopting/rebuilding files.
func TestLegacyGoldenOwnedOutputsNeedNoManifestReset(t *testing.T) {
	count := 0
	for _, c := range testutil.Corpus(t) {
		if c.Expect.Kind != "success" {
			continue
		}
		// New schema and YAML acceptances added after the language cutover
		// cannot serve as evidence of unchanged legacy output compatibility.
		if c.ID == "writeonly-annotation" || c.ID == "dictionary-typed" || c.ID == "dictionary-open" || c.ID == "closed-object" || c.ID == "explicit-yaml" || c.ID == "auto-yaml-extension" {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			p := newProject(t)
			name := filepath.Base(c.Input)
			if c.SourceName != nil {
				name = *c.SourceName
			}
			p.Input = filepath.Join(p.Root, name)
			put(t, p.Input, testutil.Read(t, c.Input))
			if err := os.MkdirAll(p.Output, 0700); err != nil {
				t.Fatal(err)
			}
			manifest := struct {
				Version uint32            `json:"version"`
				Files   map[string]string `json:"files"`
			}{1, map[string]string{}}
			for relative, golden := range c.Expect.Files {
				path := filepath.Join(p.Output, filepath.FromSlash(relative))
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				data := testutil.Read(t, golden)
				put(t, path, data)
				stamp := time.Unix(1700000000, 123000000)
				if err := os.Chtimes(path, stamp, stamp); err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(data)
				manifest.Files[relative] = hex.EncodeToString(sum[:])
				count++
			}
			encoded, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			put(t, filepath.Join(p.Output, ".openapi-kt-manifest.json"), encoded)
			// Legacy coordination files are persistent and never truncated.
			put(t, filepath.Join(p.Output, ".openapi-kt.lock"), []byte("legacy coordination"))
			put(t, filepath.Join(p.Output, "Notes.kt"), []byte("handwritten"))
			before := snapshot(t, p.Output)
			args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name", c.PackageName, "--model-name-suffix", c.ModelNameSuffix, "--input-format", c.InputFormat, "--diagnostic-format", "json"}
			result(t, invoke(t, "", append(args, "--check")...), 0)
			result(t, invoke(t, "", args...), 0)
			if !reflect.DeepEqual(snapshot(t, p.Output), before) {
				t.Fatal("legacy v1 outputs/manifest/lock/handwritten bytes or mtimes changed")
			}
		})
	}
	if count != 48 {
		t.Fatal("legacy golden inventory drift", count)
	}
}
