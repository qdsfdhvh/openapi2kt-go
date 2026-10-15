package cli_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func invokeInput(t testing.TB, cwd string, input io.Reader, args ...string) invocation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, binary, args...)
	c.Dir = cwd
	c.Stdin = input
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if ctx.Err() != nil {
		t.Fatal("local input CLI timed out")
	}
	return invocation{code, stdout.Bytes(), stderr.Bytes()}
}

func TestLocalInputFileStdinParity(t *testing.T) {
	for _, fixture := range []string{"catalog.json", "http/root-named-dto-bridges.json"} {
		for _, flags := range [][]string{{}, {"--validation", "--model-style=plain_class"}, {"--validation", "--deduplicate-models", "--hoist-shared-shapes", "--qualify-hoisted-names", "--merge-identical-models"}} {
			p := newProject(t)
			put(t, p.Input, read(t, filepath.Join(repo, "tests/fixtures", fixture)))
			result(t, p.generate(t, flags...), 0)
			before := snapshot(t, p.Output)
			base := append([]string{"generate", "--input", "-", "--output", p.Output, "--package-name", "org.example.catalog", "--diagnostic-format=json"}, flags...)
			for _, tail := range [][]string{{"--check"}, {}} {
				args := append(append([]string{}, base...), tail...)
				result(t, invokeInput(t, p.Root, bytes.NewReader(read(t, p.Input)), args...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("stdin changed bytes/mode/mtime on unchanged input")
				}
			}
			result(t, invokeInput(t, p.Root, bytes.NewReader(read(t, p.Input)), "validate", "--input=-", "--diagnostic-format=json"), 0)
		}
	}
}

func TestLocalInputStdinFailurePreservation(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ input, format, code string }{
		{"", "json", "E_JSON_INVALID"}, {`{"synthetic_private_marker":`, "json", "E_JSON_INVALID"},
		{`{"openapi":"3.1.0","openapi":"3.1.0"}`, "json", "E_JSON_INVALID"},
		{`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"string"}}}} trailing`, "json", "E_JSON_INVALID"},
		{"example: &node [*node]", "yaml", "E_YAML_ALIAS_CYCLE"},
		{strings.Repeat(" ", 8*1024*1024+1), "json", "E_INPUT_LIMIT"},
	} {
		for _, check := range []bool{false, true} {
			args := []string{"generate", "--input=-", "--input-format", tc.format, "--output", p.Output, "--package-name", "org.example.catalog", "--diagnostic-format=json"}
			if check {
				args = append(args, "--check")
			}
			out := invokeInput(t, p.Root, strings.NewReader(tc.input), args...)
			fails(t, out, 1, tc.code)
			if bytes.Contains(out.Stdout, []byte("synthetic_private_marker")) || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("stdin failure leaked or mutated output")
			}
		}
	}
	// '-' is stdin only when exact; './-' retains ordinary local file semantics.
	put(t, filepath.Join(p.Root, "-"), []byte(configDocument("Literal")))
	out := filepath.Join(p.Root, "literal-output")
	r := result(t, invokeInput(t, p.Root, nil, "generate", "--input=./-", "--output", out, "--package-name=org.example.literal", "--diagnostic-format=json"), 0)
	if len(r.Files) != 1 || !strings.HasSuffix(r.Files[0], "/Literal.kt") {
		t.Fatal("literal dash file confused with stdin", r)
	}
}

func TestLocalInputStreamWaitsForEOF(t *testing.T) {
	for _, format := range []string{"json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			p := newProject(t)
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			defer writer.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			c := exec.CommandContext(ctx, binary, "generate", "--input=-", "--input-format="+format, "--output", p.Output, "--package-name=org.example.catalog", "--diagnostic-format=json")
			c.Stdin = reader
			var stdout, stderr bytes.Buffer
			c.Stdout = &stdout
			c.Stderr = &stderr
			if err := c.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- c.Wait() }()
			if _, err := writer.Write(read(t, p.Input)); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				t.Fatal("generated before EOF", err)
			case <-time.After(100 * time.Millisecond):
			}
			if _, err := os.Stat(p.Output); !os.IsNotExist(err) {
				t.Fatal("output before stream completed", err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err, string(stdout.Bytes()), string(stderr.Bytes()))
			}
			result(t, invocation{0, stdout.Bytes(), stderr.Bytes()}, 0)
		})
	}
}

func TestLocalInputStdinLimitsMetricsAndNameLock(t *testing.T) {
	p := newProject(t)
	data := read(t, p.Input)
	exact := append(append([]byte{}, data...), bytes.Repeat([]byte{' '}, 8*1024*1024-len(data))...)
	r := measured(t, invokeInput(t, p.Root, bytes.NewReader(exact), "validate", "--input=-", "--metrics", "--diagnostic-format=json"), 0, "read", "generate")
	if r.Metrics.InputBytes != int64(len(exact)) {
		t.Fatal("exact stream size", r.Metrics)
	}
	r = measured(t, invokeInput(t, p.Root, bytes.NewReader(append(exact, ' ')), "validate", "--input=-", "--metrics", "--diagnostic-format=json"), 1, "read", "generate")
	if r.Metrics.InputBytes != 8*1024*1024+1 || r.Metrics.PlannedFiles != 0 {
		t.Fatal("stream limit metrics", r.Metrics)
	}
	root := filepath.Join(p.Root, "locked")
	input := lockInput(t, p.Root)
	args := lockArgs(input, root)
	result(t, invoke(t, p.Root, args...), 0)
	before := snapshot(t, root)
	for i := range args {
		if args[i] == input {
			args[i] = "-"
		}
	}
	result(t, invokeInput(t, p.Root, bytes.NewReader(read(t, input)), append(args, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("stdin check changed name-locked state")
	}
}

func TestLocalInputDownloadRemovedAndNoFetch(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(configDocument("Network")))
	}))
	defer server.Close()
	p := newProject(t)
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	original := read(t, p.Input)
	for _, args := range [][]string{
		{"download", "--url", server.URL, "--output", p.Input, "--allow-loopback-http", "--diagnostic-format=json"},
		{"download", "--help", "--diagnostic-format=json"}, {"help", "download", "--diagnostic-format=json"},
	} {
		fails(t, invoke(t, p.Root, args...), 2, "E_USAGE")
	}
	for _, args := range [][]string{{"--help"}, {"generate", "--help"}, {"validate", "--help"}} {
		out := invoke(t, p.Root, args...)
		if out.Code != 0 || bytes.Contains(out.Stdout, []byte("download")) {
			t.Fatal("download still advertised", out)
		}
		if len(args) > 1 && !bytes.Contains(out.Stdout, []byte("stdin")) {
			t.Fatal("stdin missing from help", out)
		}
	}
	cfg := filepath.Join(p.Root, "config.toml")
	put(t, cfg, []byte("config_version=1\n[defaults]\noutput_dir='generated'\npackage_name='org.example.catalog'\n[[specs]]\nname='a'\ninput='catalog.json'\n[specs.download]\nurl='"+server.URL+"'\nheaders_from_env={Authorization='SYNTHETIC_MISSING_ENV'}\n"))
	for _, cmd := range []string{"generate", "validate"} {
		fails(t, invoke(t, p.Root, cmd, "--config", cfg, "--diagnostic-format=json"), 1, "E_CONFIG_UNKNOWN_FIELD")
	}
	put(t, cfg, []byte("config_version=1\n[defaults]\noutput_dir='generated'\npackage_name='org.example.catalog'\n[[specs]]\nname='a'\ninput='-'\n"))
	fails(t, invoke(t, p.Root, "generate", "--config", cfg, "--diagnostic-format=json"), 1, "E_CONFIG_INVALID")
	fails(t, invoke(t, p.Root, "validate", "--input", server.URL, "--diagnostic-format=json"), 4, "E_INPUT_IO")
	if requests.Load() != 0 || !bytes.Equal(read(t, p.Input), original) || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("removed downloader contacted network or changed files")
	}
}
