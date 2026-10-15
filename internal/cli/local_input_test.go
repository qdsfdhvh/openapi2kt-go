package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

type ownedInputProbe struct {
	source               io.Reader
	reads, bytes, closes int
}

func (p *ownedInputProbe) Read(dst []byte) (int, error) {
	p.reads++
	n, e := p.source.Read(dst)
	p.bytes += n
	return n, e
}
func (p *ownedInputProbe) Close() error { p.closes++; return nil }

type failedInput struct{}

func (failedInput) Read([]byte) (int, error) { return 0, errors.New("synthetic_private_reader_error") }

func TestLocalInputReaderOwnershipAndEarlyReturns(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}, {"generate", "--help"}, {"download", "--url=https://example.test", "--diagnostic-format=json"}, {"help", "download", "--diagnostic-format=json"}, {"generate", "--input=-", "--diagnostic-format=json"}} {
		reader := &ownedInputProbe{source: failedInput{}}
		var stdout, stderr bytes.Buffer
		RunWithInput(args, reader, &stdout, &stderr)
		if reader.reads != 0 || reader.closes != 0 {
			t.Fatal("early return consumed caller input", args, reader)
		}
	}
	data := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"string"}}}}`)
	reader := &ownedInputProbe{source: bytes.NewReader(data)}
	var stdout, stderr bytes.Buffer
	code := RunWithInput([]string{"validate", "--input=-", "--metrics", "--diagnostic-format=json"}, reader, &stdout, &stderr)
	var r report
	if e := json.Unmarshal(stdout.Bytes(), &r); e != nil {
		t.Fatal(e)
	}
	if code != 0 || reader.bytes != len(data) || reader.closes != 0 || r.Metrics == nil || r.Metrics.InputBytes != int64(len(data)) {
		t.Fatal("stream ownership/metrics", code, r, reader)
	}
	// File input must never consume an unrelated caller-provided stdin.
	reader = &ownedInputProbe{source: failedInput{}}
	stdout.Reset()
	stderr.Reset()
	path := filepath.Join("..", "..", "tests", "fixtures", "catalog.json")
	if code := RunWithInput([]string{"validate", "--input", path, "--diagnostic-format=json"}, reader, &stdout, &stderr); code != 0 || reader.reads != 0 || reader.closes != 0 {
		t.Fatal("file input touched stdin", code, reader, stdout.String())
	}
}

func inputOutputSnapshot(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[path] = data
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestLocalInputReadFailureAndBound(t *testing.T) {
	root := filepath.Join(t.TempDir(), "generated")
	data := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"string"}}}}`)
	args := []string{"generate", "--input=-", "--output", root, "--package-name=org.example.input", "--diagnostic-format=json"}
	var initialOut, initialErr bytes.Buffer
	if code := RunWithInput(args, bytes.NewReader(data), &initialOut, &initialErr); code != 0 {
		t.Fatal(code, initialOut.String(), initialErr.String())
	}
	if err := os.WriteFile(filepath.Join(root, "handwritten.txt"), []byte("synthetic handwritten file"), 0600); err != nil {
		t.Fatal(err)
	}
	before := inputOutputSnapshot(t, root)
	for _, reader := range []io.Reader{failedInput{}, io.MultiReader(bytes.NewReader([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"string"}}}}`)), failedInput{}), nil} {
		var stdout, stderr bytes.Buffer
		code := RunWithInput(args, reader, &stdout, &stderr)
		var r report
		if e := json.Unmarshal(stdout.Bytes(), &r); e != nil {
			t.Fatal(e)
		}
		if code != 4 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_INPUT_IO" || bytes.Contains(stdout.Bytes(), []byte("synthetic_private")) || !reflect.DeepEqual(before, inputOutputSnapshot(t, root)) {
			t.Fatal("reader error leaked or generated", code, r)
		}
	}
	reader := &ownedInputProbe{source: bytes.NewReader(bytes.Repeat([]byte{' '}, core.MaxDocumentBytes+4096))}
	var stdout, stderr bytes.Buffer
	code := RunWithInput([]string{"validate", "--input=-", "--metrics", "--diagnostic-format=json"}, reader, &stdout, &stderr)
	var r report
	if e := json.Unmarshal(stdout.Bytes(), &r); e != nil {
		t.Fatal(e)
	}
	if code != 1 || r.Diagnostics[0].Code != "E_INPUT_LIMIT" || reader.bytes != core.MaxDocumentBytes+1 || reader.closes != 0 || r.Metrics.InputBytes != int64(core.MaxDocumentBytes+1) {
		t.Fatal("unbounded read", code, reader, r)
	}
}
