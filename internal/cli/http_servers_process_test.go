package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPServerSharedHelpersNamedOwners(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "servers.json")))
	config := filepath.Join(p.Root, "servers.toml")
	alpha := fmt.Sprintf("config_version=1\n[defaults]\ninput=%q\noutput_dir=%q\npackage_name='org.example.catalog'\n[[specs]]\nname='alpha'\napi_client_name='AlphaClient'\n", p.Input, p.Output)
	beta := "[[specs]]\nname='beta'\napi_client_name='BetaClient'\n"
	put(t, config, []byte(alpha+beta))
	result(t, invoke(t, "", "generate", "--config", config, "--diagnostic-format", "json"), 0)
	before := snapshot(t, p.Output)
	for _, extra := range [][]string{{"--check"}, {"--spec", "alpha"}, {"--spec", "beta", "--check"}} {
		result(t, invoke(t, "", append([]string{"generate", "--config", config, "--diagnostic-format", "json"}, extra...)...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("shared owner no-op/check changed files")
		}
	}
	put(t, config, []byte(alpha))
	result(t, invoke(t, "", "generate", "--config", config, "--diagnostic-format", "json"), 0)
	after := snapshot(t, p.Output)
	helpers := 0
	for name, state := range before {
		if strings.Contains(filepath.ToSlash(name), "/ServerDefault_") {
			helpers++
			if after[name] != state {
				t.Fatal("removing one owner changed its shared URL helper", name)
			}
		}
	}
	if helpers == 0 {
		t.Fatal("no shared URL helpers were generated")
	}
	if _, err := os.Stat(filepath.Join(p.Output, "org/example/catalog/BetaClient.kt")); !os.IsNotExist(err) {
		t.Fatal("removed owner's aggregate was retained", err)
	}
	result(t, invoke(t, "", "generate", "--config", config, "--check", "--diagnostic-format", "json"), 0)
}

func TestHTTPServerDefaultsNativeAndFailurePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "servers.json")))
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	result(t, p.generate(t), 0)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("server no-op/check/validate changed bytes or mtime")
	}
	for _, invalid := range []string{
		`{"openapi":"3.1.0","servers":[{"url":"https://{host}","variables":{"host":{"default":42}}}],"paths":{"/ping":{"get":{"responses":{"204":{}}}}}}`,
		`{"openapi":"3.1.0","paths":{"/ping":{"get":{"servers":[{"url":"https://{missing}"}],"responses":{"204":{}}}}}}`,
	} {
		put(t, p.Input, []byte(invalid))
		for _, extra := range [][]string{nil, {"--check"}, {"--exclude-api", "DefaultApi"}} {
			out := result(t, p.generate(t, extra...), 1)
			if len(out.Diagnostics) != 1 || len(out.Files) != 0 || out.Status != "error" {
				t.Fatal(out)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid server modified old output")
			}
		}
	}
}
