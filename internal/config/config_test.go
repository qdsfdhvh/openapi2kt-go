package config

import (
	"reflect"
	"strings"
	"testing"
)

const example = `config_version = 1 # top
[defaults]
validation = true
exclude_apis = [
 'First', # comment
 "Second\u0020Name",
]
model_name_suffix = 'Dto'
[[specs]]
name = 'alpha'
input = 'data.json'
validation = false
exclude_apis = []
model_name_suffix = ''
`

func TestParsePresenceAndSyntax(t *testing.T) {
	for _, text := range []string{example, strings.ReplaceAll(example, "\n", "\r\n")} {
		cfg, e := Parse([]byte(text))
		if e != nil || len(cfg.Specs) != 1 {
			t.Fatal(cfg, e)
		}
		s := Overlay(cfg.Defaults, cfg.Specs[0].Settings)
		if s["validation"] != false || s["model_name_suffix"] != "" || !reflect.DeepEqual(s["exclude_apis"], []string{}) {
			t.Fatal(s)
		}
		if !reflect.DeepEqual(cfg.Defaults["exclude_apis"], []string{"First", "Second Name"}) {
			t.Fatal(cfg)
		}
	}
}
func TestParseStrictFailures(t *testing.T) {
	for _, text := range []string{"", "config_version=2", "config_version=01", "config_version=1.0", "config_version=1\nconfig_version=1", "config_version=1\n[defaults]\n[defaults]", "config_version=1\n[unknown]", "config_version=1\n[defaults]\nfuture_option='secret'", "config_version=1\n[defaults]\nvalidation='false'", "config_version=1\n[[specs]]\nname='a'\n[[specs]]\nname='a'", "config_version=1\n[[specs]]\nname='../../unsafe'", "config_version=1\n[[specs]]\ninput='x'", "config_version=1\n[defaults]\nexclude_apis=[true]", "config_version=1\n[defaults]\ninput=\"\\xGG\"", "config_version=1\n[defaults]\ninput=\"\\uD800\"", "config_version=1\n[defaults]\ninput=\"\\U00110000\"", "config_version=1\r[defaults]", "config_version=1\n#\x00bad", string([]byte{255})} {
		t.Run(text, func(t *testing.T) {
			_, e := Parse([]byte(text))
			if e == nil || e.Pointer == nil || strings.Contains(e.Message, "secret") {
				t.Fatal(e)
			}
		})
	}
}
func TestParseNewBudgets(t *testing.T) {
	if _, e := Parse(make([]byte, MaxBytes+1)); e == nil || e.Code != "E_CONFIG_LIMIT" {
		t.Fatal(e)
	}
	text := "config_version=1\n"
	for i := range MaxSpecs + 1 {
		text += "[[specs]]\nname='a" + strings.Repeat("x", i) + "'\n"
	}
	if _, e := Parse([]byte(text)); e == nil || e.Code != "E_CONFIG_LIMIT" {
		t.Fatal(e)
	}
}
func FuzzParseConfig(f *testing.F) {
	for _, s := range []string{`"config_version"=+1
defaults.input="""
dir/\
  alpha.json"""
specs=[{name='alpha',exclude_apis=[]}]`, "config_version=1\nspecs.'123456789'=1\nspecs.'123456789'=2", "config_version=1\ndefaults={input='x',input='y'}", "config_version=1\ndefaults.validation=nan", "config_version=1\ndefaults.input=2026-01-01", "config_version=1\n[defaults]\nhoist_shared_shapes=true\nlock_hoisted_names=true\n[[specs]]\nname='a'\nlock_hoisted_names=false", "config_version=1\n[[specs]]\nname='a'\n[specs.download]\nlock_hoisted_names=true", "config_version=1\n[defaults]\nlock_hoisted_names='true'", "config_version=1\n[defaults]\nhoist_shared_shapes=true\nqualify_hoisted_names=true\n[[specs]]\nname='a'\nhoist_shared_shapes=false\nqualify_hoisted_names=false", "config_version=1\n[defaults]\ndeduplicate_models=true\n[[specs]]\nname='a'\ndeduplicate_models=false", "config_version=1\n[[specs]]\nname='a'\n[specs.download]\nurl='https://example.invalid/spec'\nheaders_from_env={Authorization='ENV_A'}", "config_version=1\n[defaults]\nruntime_package='org.example.shared.runtime'\n[[specs]]\nname='a'\nruntime_package=''", example, "config_version=1", "config_version=1\n[defaults]\ninput=\"\\U0001f600\"", "config_version=1\n[specs.download]", "config_version=1\n[[specs]]\nname='a'"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > MaxBytes+1 {
			t.Skip()
		}
		a, e := Parse(b)
		second, err := Parse(b)
		if !reflect.DeepEqual(a, second) || !reflect.DeepEqual(e, err) {
			t.Fatal("nondeterministic config decoding")
		}
	})
}
