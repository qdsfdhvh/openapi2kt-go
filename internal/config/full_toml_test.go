package config

import (
	"reflect"
	"strings"
	"testing"
)

// These are independently written, synthetic configurations of the same schema.
func TestFullTOMLGrammar(t *testing.T) {
	want := Config{Defaults: Settings{"input": "dir/alpha.json", "output_dir": "generated", "package_name": "org.example.config", "validation": true, "exclude_apis": []string{"First", "Second Name"}}, Specs: []Spec{{Name: "alpha", Settings: Settings{"model_name_suffix": "", "exclude_apis": []string{}, "validation": false}}}}
	for _, text := range []string{
		`"config_version" = +1
'defaults'."input" = "dir/\u0061lpha.json"
defaults.output_dir = 'generated'
defaults.package_name = 'org.example.config'
defaults.validation = true
defaults.exclude_apis = ['First', "Second\u0020Name",]
specs = [{name='alpha', model_name_suffix='', exclude_apis=[], validation=false}]
`,
		`config_version = 0x0_1
defaults = {input='dir/alpha.json', output_dir='generated', package_name='org.example.config', validation=true, exclude_apis=['First', 'Second Name']}
[["specs"]]
"na\u006de" = 'alpha'
model_name_suffix = ''
exclude_apis = []
validation = false
`,
		`config_version = 0o0_1
"defaults" = {
  input = 'dir/alpha.json',
  output_dir = 'generated', # TOML 1.1 inline-table newline/comment
  package_name = "org.example.config",
  validation = true,
  exclude_apis = ['First', "Second Name"],
}
specs = [
  {name='alpha', model_name_suffix='', exclude_apis=[], validation=false},
]
`,
		`config_version = 0b0_1
['defaults']
input = """
dir/\
  alpha.json"""
output_dir = '''
generated'''
package_name = '''org.example.config'''
validation = true
exclude_apis = [
  '''First''', # synthetic comment
  """Second Name""",
]
[['specs']]
name = """alpha"""
model_name_suffix = """""" # empty multiline string
exclude_apis = []
validation = false
`,
	} {
		for _, input := range []string{text, strings.ReplaceAll(text, "\n", "\r\n")} {
			got, err := Parse([]byte(input))
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("grammar parity: got %#v, diagnostic %v", got, err)
			}
		}
	}
}

func TestTOMLIntegerSpellings(t *testing.T) {
	for _, value := range []string{"1", "+1", "0x1", "0x0_1", "0o1", "0o0_1", "0b1", "0b0_1"} {
		if _, err := Parse([]byte("config_version=" + value)); err != nil {
			t.Errorf("integer %s: %v", value, err)
		}
	}
}

func TestTOMLClosedSchemaDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, text, code, pointer string }{
		{"version float", "config_version=1.0", "E_CONFIG_INVALID", "/config_version"},
		{"version date", "config_version=2026-01-01", "E_CONFIG_INVALID", "/config_version"},
		{"negative version", "config_version=-1", "E_CONFIG_VERSION", "/config_version"},
		{"integer version", "config_version=1_0", "E_CONFIG_VERSION", "/config_version"},
		{"quoted duplicate", "config_version=1\n\"config_version\"=1", "E_CONFIG_DUPLICATE", "/config_version"},
		{"escaped duplicate", "config_version=1\n[defaults]\ninput='x'\n\"inpu\\u0074\"='y'", "E_CONFIG_DUPLICATE", "/defaults/input"},
		{"dotted duplicate", "config_version=1\ndefaults.input='x'\ndefaults.'input'='y'", "E_CONFIG_DUPLICATE", "/defaults/input"},
		{"quoted table duplicate", "config_version=1\n[defaults]\n['defaults']", "E_CONFIG_DUPLICATE", "/defaults"},
		{"inline duplicate", "config_version=1\ndefaults={input='x', 'input'='y'}", "E_CONFIG_DUPLICATE", "/defaults"},
		{"inline table extension", "config_version=1\ndefaults={input='x'}\ndefaults.output_dir='y'", "E_CONFIG_DUPLICATE", "/defaults/output_dir"},
		{"dotted table redeclaration", "config_version=1\ndefaults.input='x'\n[defaults]", "E_CONFIG_DUPLICATE", "/defaults"},
		{"numeric unknown key", "config_version=1\nspecs.'123456789'=1\nspecs.'123456789'=2", "E_CONFIG_DUPLICATE", "/specs"},
		{"long numeric unknown key", "config_version=1\nspecs.'" + strings.Repeat("0", 30000) + "'=1\nspecs.'" + strings.Repeat("0", 30000) + "'=2", "E_CONFIG_DUPLICATE", "/specs"},
		{"numeric key budget", "config_version=1\nspecs.'123456789'=" + strings.Repeat("[", MaxDepth+1) + "0" + strings.Repeat("]", MaxDepth+1), "E_CONFIG_LIMIT", "/specs"},
		{"misplaced known root field", "config_version=1\ninput='hidden'", "E_CONFIG_UNKNOWN_FIELD", "/input"},
		{"misplaced version", "config_version=1\ndefaults.config_version=1", "E_CONFIG_UNKNOWN_FIELD", "/defaults/config_version"},
		{"unknown root", "config_version=1\n'SYNTHETIC_PRIVATE'='hidden'", "E_CONFIG_UNKNOWN_FIELD", ""},
		{"unknown nested", "config_version=1\ndefaults.'SYNTHETIC_PRIVATE'={value='hidden'}", "E_CONFIG_UNKNOWN_FIELD", "/defaults"},
		{"unknown table", "config_version=1\n[defaults.SYNTHETIC_PRIVATE]", "E_CONFIG_UNKNOWN_FIELD", "/defaults"},
		{"unknown quoted dot", "config_version=1\n['defaults.input']", "E_CONFIG_UNKNOWN_FIELD", ""},
		{"old download without inventory", "config_version=1\n[specs.download]\nurl='hidden'", "E_CONFIG_UNKNOWN_FIELD", "/specs"},
		{"old dotted download without inventory", "config_version=1\nspecs.download={url='hidden'}", "E_CONFIG_UNKNOWN_FIELD", "/specs"},
		{"old download", "config_version=1\n[[specs]]\nname='a'\n[specs.download]\nurl='hidden'", "E_CONFIG_UNKNOWN_FIELD", "/specs/0"},
		{"old inline download", "config_version=1\nspecs=[{name='a', download={url='hidden'}}]", "E_CONFIG_UNKNOWN_FIELD", "/specs/0"},
		{"stdin unselected", "config_version=1\nspecs=[{name='a'}, {name='b', input='-'}]", "E_CONFIG_INVALID", "/specs/1/input"},
		{"defaults type", "config_version=1\ndefaults=[]", "E_CONFIG_INVALID", "/defaults"},
		{"specs table type", "config_version=1\n[specs]\nname='a'", "E_CONFIG_INVALID", "/specs"},
		{"specs entry type", "config_version=1\nspecs=['a']", "E_CONFIG_INVALID", "/specs/0"},
		{"field nested table", "config_version=1\n[defaults.input]\nx='hidden'", "E_CONFIG_INVALID", "/defaults/input"},
		{"date string field", "config_version=1\ndefaults.input=2026-01-01T10:11:12Z", "E_CONFIG_INVALID", "/defaults/input"},
		{"local date field", "config_version=1\ndefaults.input=2026-01-01", "E_CONFIG_INVALID", "/defaults/input"},
		{"local time field", "config_version=1\ndefaults.input=10:11:12", "E_CONFIG_INVALID", "/defaults/input"},
		{"local datetime field", "config_version=1\ndefaults.input=2026-01-01T10:11:12", "E_CONFIG_INVALID", "/defaults/input"},
		{"float bool field", "config_version=1\ndefaults.validation=+inf", "E_CONFIG_INVALID", "/defaults/validation"},
		{"nested array", "config_version=1\ndefaults.exclude_apis=[['hidden']]", "E_CONFIG_INVALID", "/defaults/exclude_apis"},
		{"heterogeneous array", "config_version=1\ndefaults.exclude_apis=['x',1]", "E_CONFIG_INVALID", "/defaults/exclude_apis"},
		{"unknown defaults name", "config_version=1\ndefaults.name='hidden'", "E_CONFIG_UNKNOWN_FIELD", "/defaults/name"},
		{"array table conflict", "config_version=1\nspecs=[]\n[[specs]]", "E_CONFIG_DUPLICATE", "/specs"},
		{"spec duplicate", "config_version=1\n[[specs]]\nname='a'\ninput='x'\n'input'='y'", "E_CONFIG_DUPLICATE", "/specs/0/input"},
		{"unknown case", "config_version=1\ndefaults.Input='hidden'", "E_CONFIG_UNKNOWN_FIELD", "/defaults"},
		{"uppercase root", "Config_version=1", "E_CONFIG_UNKNOWN_FIELD", ""},
		{"unknown dotted plugin", "config_version=1\nplugins.generator={value='hidden'}", "E_CONFIG_UNKNOWN_FIELD", ""},
		{"old header metadata", "config_version=1\ndefaults.headers_from_env={Authorization='hidden'}", "E_CONFIG_UNKNOWN_FIELD", "/defaults"},
		{"empty quoted key", "config_version=1\ndefaults.''='hidden'", "E_CONFIG_UNKNOWN_FIELD", "/defaults"},
		{"BOM", "\ufeffconfig_version=1", "E_CONFIG_SYNTAX", ""},
		{"invalid escape", "config_version=1\n[defaults]\ninput=\"\\xGG\"", "E_CONFIG_SYNTAX", "/defaults/input"},
		{"leading zero", "config_version=01", "E_CONFIG_SYNTAX", "/config_version"},
		{"bad underscore", "config_version=1_", "E_CONFIG_SYNTAX", "/config_version"},
		{"overflow", "config_version=9223372036854775808", "E_CONFIG_SYNTAX", "/config_version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.text))
			if err == nil || err.Code != tc.code || err.Pointer == nil || *err.Pointer != tc.pointer || strings.Contains(err.Message, "hidden") || strings.Contains(err.Message, "SYNTHETIC_PRIVATE") || strings.Contains(*err.Pointer, "SYNTHETIC_PRIVATE") {
				t.Fatalf("want %s at %q, got %v", tc.code, tc.pointer, err)
			}
		})
	}
}
