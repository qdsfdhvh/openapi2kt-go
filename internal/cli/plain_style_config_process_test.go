package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlainStyleConfigPresenceOverridesAndFailurePreservation(t *testing.T) {
	dir, cfg := configFiles(t)
	text, e := os.ReadFile(cfg)
	if e != nil {
		t.Fatal(e)
	}
	text = []byte(strings.Replace(string(text), "[defaults]", "[defaults]\nmodel_style='plain_class'", 1))
	put(t, cfg, text)
	root := filepath.Join(dir, "generated")
	args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
	result(t, invoke(t, repo, args...), 0)
	for _, name := range []string{"Alpha", "Beta"} {
		b, e := os.ReadFile(filepath.Join(root, "org/example/config/models/"+name+".kt"))
		if e != nil || !strings.Contains(string(b), "public class "+name+"(") || !strings.Contains(string(b), "override fun equals(") {
			t.Fatal(e, string(b))
		}
	}
	before := snapshot(t, root)
	fails(t, invoke(t, repo, append(args, "--model-style=data_class")...), 1, "E_CONFIG_OVERRIDE")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("multi override mutated tree")
	}
	result(t, invoke(t, repo, append(append([]string{}, args...), "--spec=alpha", "--model-style=")...), 0)
	alpha, e := os.ReadFile(filepath.Join(root, "org/example/config/models/Alpha.kt"))
	if e != nil || !strings.Contains(string(alpha), "public data class Alpha(") {
		t.Fatal(e, string(alpha))
	}
	beta, e := os.ReadFile(filepath.Join(root, "org/example/config/models/Beta.kt"))
	if e != nil || !strings.Contains(string(beta), "public class Beta(") {
		t.Fatal(e, string(beta))
	}
	before = snapshot(t, root)
	put(t, cfg, []byte(strings.Replace(string(text), "model_style='plain_class'", "model_style='invalid_sensitive'", 1)))
	fails(t, invoke(t, repo, args...), 1, "E_CONFIG_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("invalid configured style changed old tree")
	}
	for _, tail := range [][]string{{"--model-style=bad"}, {"--model-style=data_class", "--model-style=plain_class"}} {
		fails(t, invoke(t, repo, append([]string{"validate", "--input=missing.json", "--diagnostic-format=json"}, tail...)...), 2, "E_USAGE")
	}
	help := invoke(t, repo, "generate", "--help")
	if help.Code != 0 || !strings.Contains(string(help.Stdout), "--model-style") {
		t.Fatal(help)
	}
	fails(t, invoke(t, repo, "download", "--url=https://example.invalid", "--output=x", "--model-style=plain_class", "--diagnostic-format=json"), 2, "E_USAGE")
}
