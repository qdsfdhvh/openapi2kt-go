package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestYAMLOrderedTreeParity(t *testing.T) {
	for _, tc := range []struct{ yaml, json string }{
		{"z: 9223372036854775807\na: -9223372036854775808\nu: 18446744073709551615\nb: 9007199254740993\nf: 1.0\nneg: -0.0\ne: 1e-99\nn: null\n", `{"z":9223372036854775807,"a":-9223372036854775808,"u":18446744073709551615,"b":9007199254740993,"f":1.0,"neg":-0.0,"e":1e-99,"n":null}`},
		{"a: &node {second: false, first: [yes, on, 2024-02-29]}\nb: *node\nc: !!str 012\nd: TRUE\ne: ~\nf:\n", `{"a":{"second":false,"first":["yes","on","2024-02-29"]},"b":{"second":false,"first":["yes","on","2024-02-29"]},"c":"012","d":true,"e":null,"f":null}`},
		{"'<<': literal\ntext: |\n  snow 雪\n  line\nfold: >-\n  first\n  second\nemoji: \"\\U0001F600\"\n", `{"<<":"literal","text":"snow 雪\nline\n","fold":"first second","emoji":"😀"}`},
		{"text: '%TAG !h! literal'\n", `{"text":"%TAG !h! literal"}`},
		{"text: [0x_, 0o_, 0b_, 1e_, 1e+_, _123]\n0x_: payload\n", `{"text":["0x_","0o_","0b_","1e_","1e+_","_123"],"0x_":"payload"}`},
		{"a: '0xFFFFFFFFFFFFFFFFF'\nb: !!str +1e400\nc: !!float 1\nd: _text\n", `{"a":"0xFFFFFFFFFFFFFFFFF","b":"+1e400","c":1,"d":"_text"}`},
		{"!!map {str: !!str true, int: !!int 1, float: !!float 1.0, seq: !!seq [], map: !!map {}, null: !!null null}", `{"str":"true","int":1,"float":1.0,"seq":[],"map":{},"null":null}`},
	} {
		// Quote the null key: unquoted null is a non-string key.
		tc.yaml = strings.ReplaceAll(tc.yaml, ", null:", ", 'null':")
		root, err := Load([]byte(tc.yaml), YAML, "private-source.yaml")
		if err != nil {
			t.Fatal(err)
		}
		want := mustLoad(t, tc.json)
		if !reflect.DeepEqual(root, want) {
			got, _ := json.Marshal(root)
			t.Fatalf("ordered tree mismatch: %s != %s", got, tc.json)
		}
	}
	input, e := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests/fixtures/yaml/records.yaml"))
	if e != nil {
		t.Fatal(e)
	}
	control, e := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests/fixtures/yaml/records.json"))
	if e != nil {
		t.Fatal(e)
	}
	root, err := Load(input, YAML, "records.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := mustLoad(t, string(control))
	if !reflect.DeepEqual(root, want) {
		t.Fatal("fixture ordered tree differs")
	}
}

func TestYAMLNegativeProfile(t *testing.T) {
	for _, tc := range []struct{ input, code string }{
		{"", "E_YAML_INVALID"}, {"# only comment", "E_YAML_INVALID"},
		{"a: [", "E_YAML_INVALID"}, {"a: 1\na: 2", "E_YAML_INVALID"},
		{"a: 1\n'\\u0061': 2\na: 3", "E_YAML_INVALID"},
		{"a: 1\n\"\\u0061\": 2", "E_YAML_INVALID"},
		{"example: {private_payload: 1, private_payload: 2}", "E_YAML_INVALID"},
		{"a: *private_payload", "E_YAML_INVALID"},
		{"a: &node []\nb: &node []", "E_YAML_INVALID"},
		{"a: &node [*node]", "E_YAML_ALIAS_CYCLE"},
		{"a: &node {b: {c: *node}}", "E_YAML_ALIAS_CYCLE"},
		{"a: 1\n---\na: 2", "E_YAML_INVALID"}, {"{}\n---", "E_YAML_INVALID"},
		{"{}\n...\nprivate_payload", "E_YAML_INVALID"},
		{"? [a, b]\n: value", "E_YAML_UNSUPPORTED"},
		{"1: value", "E_YAML_UNSUPPORTED"}, {"true: value", "E_YAML_UNSUPPORTED"},
		{"null: value", "E_YAML_UNSUPPORTED"}, {"2024-02-29: value", "E_YAML_UNSUPPORTED"},
		{"a: &key text\n*key: value", "E_YAML_UNSUPPORTED"},
		{"base: &base {a: 1}\nmerged: {<<: *base}", "E_YAML_UNSUPPORTED"},
		{"a: !!merge '<<'", "E_YAML_UNSUPPORTED"},
		{"a: !private_payload value", "E_YAML_UNSUPPORTED"},
		{"a: !!binary YQ==", "E_YAML_UNSUPPORTED"}, {"a: !!set {b: null}", "E_YAML_UNSUPPORTED"},
		{"a: !!timestamp 2024-02-29", "E_YAML_UNSUPPORTED"},
		{"a: !!bool yes", "E_YAML_UNSUPPORTED"}, {"a: !!null not-null", "E_YAML_UNSUPPORTED"},
		{"a: !!int 'bad'", "E_YAML_UNSUPPORTED"},
		{"a: .inf", "E_YAML_UNSUPPORTED"}, {"a: .nan", "E_YAML_UNSUPPORTED"},
		{"a: 0x20", "E_YAML_UNSUPPORTED"}, {"a: 0o20", "E_YAML_UNSUPPORTED"},
		{"a: 012", "E_YAML_UNSUPPORTED"}, {"a: +1", "E_YAML_UNSUPPORTED"},
		{"a: 1_000", "E_YAML_UNSUPPORTED"}, {"a: .5", "E_YAML_UNSUPPORTED"},
		{"a: 1.", "E_YAML_UNSUPPORTED"}, {"a: 1e400", "E_YAML_UNSUPPORTED"},
		{"a: !!int 1.5", "E_YAML_UNSUPPORTED"}, {"a: !!int '1e2'", "E_YAML_UNSUPPORTED"},
		{"a: 0xFFFFFFFFFFFFFFFFF", "E_YAML_UNSUPPORTED"},
		{"a: 0o7777777777777777777777", "E_YAML_UNSUPPORTED"},
		{"a: +1e400", "E_YAML_UNSUPPORTED"}, {"a: 1_0e400", "E_YAML_UNSUPPORTED"},
		{"a: 01e400", "E_YAML_UNSUPPORTED"}, {"a: .5e400", "E_YAML_UNSUPPORTED"},
		{"a: 1.e400", "E_YAML_UNSUPPORTED"},
		{"a: +_1e2", "E_YAML_UNSUPPORTED"}, {"a: +_1e400", "E_YAML_UNSUPPORTED"},
		{"+_1e400: payload", "E_YAML_UNSUPPORTED"},
		{"a: -_1e400", "E_YAML_UNSUPPORTED"},
		{"a: 0_xFF", "E_YAML_UNSUPPORTED"}, {"a: 0_xFFFFFFFFFFFFFFFFF", "E_YAML_UNSUPPORTED"},
		{"0_xFFFFFFFFFFFFFFFFF: payload", "E_YAML_UNSUPPORTED"},
		{"a: 0_o7777777777777777777777", "E_YAML_UNSUPPORTED"},
		{"a: 0_b" + strings.Repeat("1", 65), "E_YAML_UNSUPPORTED"},
		{"a: 0b+1", "E_YAML_UNSUPPORTED"},
		{"a: 0b+" + strings.Repeat("1", 65), "E_YAML_UNSUPPORTED"},
		{"0b+" + strings.Repeat("1", 65) + ": payload", "E_YAML_UNSUPPORTED"},
		{"a: 0o-7777777777777777777777", "E_YAML_UNSUPPORTED"},
		{"example: {+1e400: payload}", "E_YAML_UNSUPPORTED"},
		{"example: {0xFFFFFFFFFFFFFFFFF: payload}", "E_YAML_UNSUPPORTED"},
		{"a: \"\xff\"", "E_YAML_INVALID"}, {"a: \"\\uD800\"", "E_YAML_INVALID"},
		{"a: \"\\uDC00\"", "E_YAML_INVALID"}, {"a: \"\\U00110000\"", "E_YAML_INVALID"},
		{"a: \x00", "E_YAML_INVALID"},
		{"[]", "E_DOCUMENT_INVALID"}, {"null", "E_DOCUMENT_INVALID"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			root, err := Load([]byte(tc.input), YAML, "private_source.yaml")
			if root != nil || err == nil || err.Code != tc.code {
				t.Fatalf("want %s, got %v %v", tc.code, root, err)
			}
			if strings.Contains(err.Message, "private_payload") || strings.Contains(err.Message, "private_source") {
				t.Fatal("payload or filename leaked")
			}
		})
	}
}

func TestYAMLTagIdentityPreservesStandardTags(t *testing.T) {
	for _, tc := range []struct{ yaml, json string }{
		{"a: !<tag:yaml.org,2002:str> 1e400", `{"a":"1e400"}`},
		{"a: !<%74ag%3Ayaml.org%2C2002%3Astr> 1e400", `{"a":"1e400"}`},
		{"a: !!%73tr 1e400", `{"a":"1e400"}`},
		{"a: !<tag:yaml.org,2002:map> {b: !<tag:yaml.org,2002:seq> []}", `{"a":{"b":[]}}`},
		{"a: &anchor # separation\n  !!str 1e400\nb: *anchor", `{"a":"1e400","b":"1e400"}`},
		{"a: !!str &anchor 1e400\nb: *anchor", `{"a":"1e400","b":"1e400"}`},
		{"a: ! text\nb: '!<!!str> 1e400'\nc: |\n  !<!!map> {}\n", `{"a":"text","b":"!<!!str> 1e400","c":"!<!!map> {}\n"}`},
		{"雪: text\na: !!str 1e400", `{"雪":"text","a":"1e400"}`},
		{"{雪: text, a: !!str 1e400}", `{"雪":"text","a":"1e400"}`},
		{"!<tag:yaml.org,2002:map> {a: !!str 1e400}", `{"a":"1e400"}`},
	} {
		got, err := Load([]byte(tc.yaml), YAML, "source.yaml")
		if err != nil || !reflect.DeepEqual(got, mustLoad(t, tc.json)) {
			t.Fatalf("standard tag/string handling changed: %q: %v", tc.yaml, err)
		}
	}
	for _, br := range []string{"\n", "\r\n", "\r", "\u0085", "\u2028", "\u2029"} {
		for _, prefix := range []string{"", "\ufeff"} {
			good := prefix + "雪: text" + br + "a: &n # separation" + br + "  !!str 1e400" + br + "b: *n"
			if _, err := Load([]byte(good), YAML, "source.yaml"); err != nil {
				t.Fatalf("standard tag source marks: %q: %v", br, err)
			}
			bad := strings.Replace(good, "!!str", "!<%21%21str>", 1)
			if root, err := Load([]byte(bad), YAML, "source.yaml"); root != nil || err == nil || err.Code != "E_YAML_UNSUPPORTED" {
				t.Fatalf("custom tag escaped source marks: %q: %v", br, err)
			}
		}
	}
}

func TestYAMLTagDirectivePreflight(t *testing.T) {
	inputs := []string{
		"%TAG !h! tag:example.test,2026:" + strings.Repeat("x", 65536) + "\n---\nexample:\n" + strings.Repeat("  - !h!thing value\n", 128),
		// This conservative profile also rejects percent-TAG at column zero
		// in quoted continuations; indent or escape the percent instead.
		"example: \"first\n%TAG !h! literal\nlast\"",
	}
	for _, br := range []string{"\n", "\r\n", "\r", "\u0085", "\u2028", "\u2029"} {
		inputs = append(inputs, "# header"+br+"\ufeff%TAG !h! tag:yaml.org,2002:"+br+"---"+br+"a: !h!str value")
	}
	for _, input := range inputs {
		root, err := Load([]byte(input), YAML, "private_source.yaml")
		if root != nil || err == nil || err.Code != "E_YAML_UNSUPPORTED" || err.Message != "YAML tag directives are unsupported." {
			t.Fatal("tag directive reached the upstream URI expansion parser", err)
		}
	}
}

func TestYAMLSizeDepthAndExpansionBudgets(t *testing.T) {
	input := append([]byte("{}\n#"), bytes.Repeat([]byte{' '}, MaxDocumentBytes-4)...)
	if _, err := Load(input, YAML, "source.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(append(input, ' '), YAML, "source.yaml"); err == nil || err.Code != "E_INPUT_LIMIT" {
		t.Fatal("missing source size bound", err)
	}
	for _, depth := range []int{126, 127, 10_001} {
		text := "a: " + strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth)
		root, err := Load([]byte(text), YAML, "source.yaml")
		if depth == 126 {
			if err != nil {
				t.Fatal(err)
			}
		} else if root != nil || err == nil || (err.Code != "E_YAML_LIMIT" && depth <= 10_000) {
			t.Fatal("missing source depth bound", depth, err)
		}
	}
	// Each source is shallow, but the alias use grows the expanded depth.
	var chain strings.Builder
	chain.WriteString("a0: &a0 [0]\n")
	for i := 1; i < 127; i++ {
		fmt.Fprintf(&chain, "a%d: &a%d [*a%d]\n", i, i, i-1)
	}
	if _, err := Load([]byte(chain.String()), YAML, "source.yaml"); err == nil || err.Code != "E_YAML_LIMIT" {
		t.Fatal("missing expanded depth bound", err)
	}
	// Exponential expansion must stop before building the complete tree.
	chain.Reset()
	chain.WriteString("a0: &a0 [0]\n")
	for i := 1; i < 20; i++ {
		fmt.Fprintf(&chain, "a%d: &a%d [*a%d, *a%d]\n", i, i, i-1, i-1)
	}
	if _, err := Load([]byte(chain.String()), YAML, "source.yaml"); err == nil || err.Code != "E_YAML_LIMIT" {
		t.Fatal("missing expanded node bound", err)
	}
	// A single anchored scalar can amplify bytes without many nodes.
	text := "a: &a '" + strings.Repeat("x", MaxDocumentBytes/2) + "'\nb: *a\n"
	if _, err := Load([]byte(text), YAML, "source.yaml"); err == nil || err.Code != "E_YAML_LIMIT" {
		t.Fatal("missing expanded byte bound", err)
	}
	text = "a: [" + strings.Repeat("0,", maxYAMLNodes-4) + "0]"
	if _, err := Load([]byte(text), YAML, "source.yaml"); err != nil {
		t.Fatal("exact source/expanded node budget", err)
	}
	text = strings.Replace(text, "]", ",0]", 1)
	if _, err := Load([]byte(text), YAML, "source.yaml"); err == nil || err.Code != "E_YAML_LIMIT" {
		t.Fatal("missing source node bound", err)
	}
}

func FuzzYAMLLoad(f *testing.F) {
	for _, seed := range []string{"{}", "a: &a [*a]", "a: &a {b: [1, true, null]}\nc: *a", "a: 1\na: 2", "a: 1e400", "a: '\\uD800'", "a: !!binary YQ==", "a: !<!!str> 1e400", "a: &n # comment\n !!str 1e400", "{雪: text, a: !<%21%21seq> []}"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			t.Skip()
		}
		root, err := Load(data, YAML, "source.yaml")
		if err != nil {
			if root != nil {
				t.Fatal("partial tree on failure")
			}
			return
		}
		encoded, e := json.Marshal(root)
		if e != nil {
			t.Fatal(e)
		}
		second, d := Load(encoded, JSON, "source.json")
		if d != nil || !reflect.DeepEqual(root, second) {
			t.Fatal("accepted YAML cannot round-trip through shared JSON tree", d)
		}
	})
}
