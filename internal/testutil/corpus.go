// Package testutil contains test-only fixture inventory support. Generator
// packages must not import it; its filesystem reads are verification, not core IO.
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type Case struct {
	ID              string   `json:"id"`
	Group           string   `json:"group"`
	Description     string   `json:"description"`
	Input           string   `json:"input"`
	PackageName     string   `json:"package_name"`
	ModelNameSuffix string   `json:"model_name_suffix"`
	InputFormat     string   `json:"input_format"`
	SourceName      *string  `json:"source_name"`
	Validate        bool     `json:"validate"`
	Expect          Expected `json:"expect"`
}

type Expected struct {
	Kind    string            `json:"kind"`
	Files   map[string]string `json:"files"`
	Code    string            `json:"code"`
	Pointer *string           `json:"pointer"`
}

// RepoRoot locates the repository independently of the fixture layout and cwd.
func RepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate test repository")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func Root() string {
	return filepath.Join(RepoRoot(), "tests", "fixtures", "p1")
}

func Read(t testing.TB, relative string) []byte {
	t.Helper()
	if !safeRelative(relative) {
		t.Fatalf("unsafe fixture path: %q", relative)
	}
	data, err := os.ReadFile(filepath.Join(Root(), filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func safeRelative(p string) bool {
	if p == "" || strings.Contains(p, "\\") || path.IsAbs(p) {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// Corpus rejects drift in the inventory rather than silently skipping fixtures.
func Corpus(t testing.TB) []Case {
	t.Helper()
	var corpus struct {
		Version int    `json:"version"`
		Cases   []Case `json:"cases"`
	}
	dec := json.NewDecoder(bytes.NewReader(Read(t, "cases.json")))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&corpus); err != nil {
		t.Fatal(err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		t.Fatalf("invalid trailing corpus data: %v", err)
	}
	if corpus.Version != 1 || len(corpus.Cases) == 0 {
		t.Fatal("unsupported or empty corpus")
	}
	ids, refs := map[string]bool{}, map[string]bool{}
	register := func(p string) {
		if !safeRelative(p) || refs[p] {
			t.Fatalf("unsafe or duplicate fixture path: %q", p)
		}
		refs[p] = true
	}
	for _, c := range corpus.Cases {
		if c.ID == "" || ids[c.ID] || c.Group == "" || c.Description == "" {
			t.Fatalf("invalid/duplicate fixture id %q", c.ID)
		}
		ids[c.ID] = true
		register(c.Input)
		if !strings.HasSuffix(c.Input, ".json") {
			t.Fatal("corpus inputs must be JSON fixture files")
		}
		if c.SourceName != nil && (!safeRelative(*c.SourceName) || strings.Contains(*c.SourceName, "/")) {
			t.Fatal("invalid fixture source name")
		}
		if c.InputFormat != "auto" && c.InputFormat != "json" && c.InputFormat != "yaml" {
			t.Fatal("invalid fixture input format")
		}
		if !c.Validate && c.ModelNameSuffix == "" {
			t.Fatal("validate can only be skipped for suffix options")
		}
		switch c.Expect.Kind {
		case "success":
			if !strings.HasPrefix(c.Input, "valid/") || len(c.Expect.Files) == 0 || c.Expect.Code != "" || c.Expect.Pointer != nil {
				t.Fatal("invalid success expectation")
			}
			for output, snapshot := range c.Expect.Files {
				if !safeRelative(output) || !strings.HasPrefix(snapshot, "snapshots/") || !strings.HasSuffix(snapshot, ".kt") {
					t.Fatal("invalid snapshot expectation")
				}
				register(snapshot)
			}
		case "error":
			if !strings.HasPrefix(c.Input, "invalid/") || !strings.HasPrefix(c.Expect.Code, "E_") || len(c.Expect.Files) != 0 || c.Expect.Pointer != nil && *c.Expect.Pointer != "" && !strings.HasPrefix(*c.Expect.Pointer, "/") {
				t.Fatal("invalid error expectation")
			}
		default:
			t.Fatal("unknown fixture expectation")
		}
	}
	for _, dir := range []string{"valid", "invalid", "snapshots"} {
		err := filepath.WalkDir(filepath.Join(Root(), dir), func(p string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				t.Fatal("fixtures cannot depend on symlinks")
			}
			if !entry.IsDir() {
				rel, err := filepath.Rel(Root(), p)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)
				if !refs[rel] {
					t.Fatalf("orphaned fixture %q", rel)
				}
				delete(refs, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(refs) != 0 {
		t.Fatalf("missing fixture files: %v", refs)
	}
	return corpus.Cases
}
