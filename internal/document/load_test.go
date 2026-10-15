package document

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"

	"openapi-kt/internal/testutil"
)

func mustLoad(t testing.TB, text string) *Value {
	t.Helper()
	value, err := Load([]byte(text), JSON, "input.json")
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestOrderedLookupExactNumbersAndMissingNull(t *testing.T) {
	text := `{"z":9223372036854775807,"a":-9223372036854775808,"u":18446744073709551615,"b":9007199254740993,"n":null,"f":1.0,"neg":-0.0,"exp":1e-99,"obj":{"second":false,"first":[]}}`
	root := mustLoad(t, text)
	var names []string
	for _, member := range root.Members() {
		names = append(names, member.Name)
		got, ok := root.Get(member.Name)
		if !ok || got != member.Value {
			t.Fatal("ordered member and indexed lookup disagree")
		}
	}
	if !reflect.DeepEqual(names, []string{"z", "a", "u", "b", "n", "f", "neg", "exp", "obj"}) {
		t.Fatal(names)
	}
	for name, token := range map[string]string{"z": "9223372036854775807", "a": "-9223372036854775808", "u": "18446744073709551615", "b": "9007199254740993", "f": "1.0", "neg": "-0.0", "exp": "1e-99"} {
		v, _ := root.Get(name)
		num, ok := v.Numeric()
		if !ok || string(num) != token {
			t.Fatalf("%s lost numeric token: %q", name, num)
		}
	}
	missing, exists := root.Get("missing")
	null, present := root.Get("n")
	if exists || missing.Kind() != Invalid || !present || null.Kind() != Null {
		t.Fatal("missing and null conflated")
	}
	encoded, err := json.Marshal(root)
	if err != nil || string(encoded) != text {
		t.Fatalf("round-trip: %s %v", encoded, err)
	}
}

func TestFormatDetectionAndNoYAMLFallback(t *testing.T) {
	for _, tc := range []struct {
		format InputFormat
		name   string
		ok     bool
	}{
		{Auto, "input.JSON", true}, {Auto, "input", true}, {"", "input", true},
		{Auto, ".yaml", true}, {Auto, "nested/.json", true},
		{Auto, "input.yaml", true}, {Auto, "input.YML", true},
		{Auto, "input.txt", false}, {Auto, "input.", false},
		{JSON, "input.yaml", true}, {YAML, "input.json", true}, {"invalid", "input.json", false},
	} {
		t.Run(string(tc.format)+"/"+tc.name, func(t *testing.T) {
			_, err := Load([]byte(`{}`), tc.format, tc.name)
			if (err == nil) != tc.ok {
				t.Fatalf("unexpected format result: %v", err)
			}
			if err != nil && (err.Code != "E_INPUT_FORMAT_UNSUPPORTED" || err.Pointer != nil || strings.Contains(err.Message, tc.name)) {
				t.Fatal("unsafe/wrong format diagnostic", err)
			}
		})
	}
	_, err := Load([]byte("openapi: 3.1.0"), JSON, "input.yaml")
	if err == nil || err.Code != "E_JSON_INVALID" {
		t.Fatal("JSON failure must not fall back to YAML")
	}
}

func TestInvalidJSONIsPayloadSafeAndNeverReturnsATree(t *testing.T) {
	for _, text := range []string{
		"", "{", `{"private-payload-marker":`, `{} {}`, `{} true`, `{} garbage`,
		`{"a":0,}`, `// comment\n{}`, `{"a":NaN}`, `{"a":Infinity}`, `{"a":1e400}`,
		`{"a":1,"a":2}`, `{"a":1,"\u0061":2}`, `{"nested":{"a":1,"a":2}}`,
		`{"example":{"a":1,"a":2}}`, `{"a":"\uD800"}`, `{"a":"\uDC00"}`,
		`{"a":"\uD800\u0041"}`, `{"\uDFFF":1}`, "{\"a\":\"\xff\"}",
		`{"a":"\uZZZZ"}`, `{"a":"\x20"}`, `{"a":"` + "\x00" + `"}`,
	} {
		t.Run(text, func(t *testing.T) {
			root, err := Load([]byte(text), JSON, "private-source-marker.json")
			if root != nil || err == nil || err.Code != "E_JSON_INVALID" || err.Pointer != nil {
				t.Fatalf("expected JSON failure, got %v %v", root, err)
			}
			if strings.Contains(err.Message, "private-payload-marker") || strings.Contains(err.Message, "private-source-marker") {
				t.Fatal("diagnostic leaks input")
			}
		})
	}
}

func TestValidUnicodeAndOpaquePayloads(t *testing.T) {
	root := mustLoad(t, `{"emoji":"\uD83D\uDE00","literal":"\\uD800","text":"雪\u2028\u0000","example":{"$ref":"https://example.test/opaque","type":false}}`)
	for key, want := range map[string]string{"emoji": "😀", "literal": `\uD800`, "text": "雪\u2028\x00"} {
		v, _ := root.Get(key)
		text, ok := v.Text()
		if !ok || text != want {
			t.Fatalf("%s: %q != %q", key, text, want)
		}
	}
}

func TestNonObjectRootPointer(t *testing.T) {
	for _, text := range []string{"[]", "null", "false", "1", `"text"`} {
		value, err := Load([]byte(text), JSON, "input.json")
		if value != nil || err == nil || err.Code != "E_DOCUMENT_INVALID" || err.Pointer == nil || *err.Pointer != "" {
			t.Fatalf("expected root diagnostic: %v", err)
		}
	}
}

func TestExactInputAndJSONDepthBoundaries(t *testing.T) {
	input := append([]byte(`{}`), bytes.Repeat([]byte{' '}, MaxDocumentBytes-2)...)
	if _, err := Load(input, JSON, "input.json"); err != nil {
		t.Fatal(err)
	}
	if value, err := Load(append(input, ' '), JSON, "input.json"); value != nil || err == nil || err.Code != "E_INPUT_LIMIT" {
		t.Fatal("missing input size bound")
	}
	// Format errors retain precedence over the byte limit, as before.
	if _, err := Load(append(input, ' '), "unsupported", "input.yaml"); err == nil || err.Code != "E_INPUT_FORMAT_UNSUPPORTED" {
		t.Fatal("wrong format/limit diagnostic precedence")
	}
	for _, depth := range []int{64, MaxJSONDepth - 1, MaxJSONDepth, 256} {
		input := `{"example":` + strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth) + "}"
		value, err := Load([]byte(input), JSON, "input.json")
		if depth < MaxJSONDepth {
			if err != nil {
				t.Fatalf("depth %d: %v", depth, err)
			}
		} else if value != nil || err == nil || err.Code != "E_JSON_INVALID" {
			t.Fatalf("depth %d: missing JSON depth limit", depth)
		}
	}
}

func TestEntireFixtureCorpusLoadingLayer(t *testing.T) {
	cases := testutil.Corpus(t)
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			name := c.Input
			if c.SourceName != nil {
				name = *c.SourceName
			}
			value, err := Load(testutil.Read(t, c.Input), InputFormat(c.InputFormat), name)
			switch c.Expect.Code {
			case "E_JSON_INVALID", "E_INPUT_FORMAT_UNSUPPORTED", "E_DOCUMENT_INVALID":
				if value != nil || err == nil || err.Code != c.Expect.Code || !reflect.DeepEqual(err.Pointer, c.Expect.Pointer) {
					t.Fatalf("wrong loader diagnostic: %v", err)
				}
			default:
				if err != nil || value.Kind() != Object {
					t.Fatalf("loader must not perform schema validation: %v", err)
				}
			}
		})
	}
}

func TestConcurrentLoads(t *testing.T) {
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			root := mustLoad(t, `{"x":9223372036854775807,"nested":{"b":true}}`)
			x, _ := root.Get("x")
			n, _ := x.Numeric()
			if string(n) != "9223372036854775807" {
				t.Error("concurrent load corrupted value")
			}
		})
	}
	wg.Wait()
}

func FuzzLoad(f *testing.F) {
	for _, seed := range []string{`{}`, `{"a":9223372036854775807}`, `{"a":null}`, `{"a":"\uD83D\uDE00"}`, `{"a":1,"a":2}`, `{"a":"\uD800"}`, "[[]]", `{"z":[false,"雪"],"a":-0.0}`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			t.Skip()
		}
		root, err := Load(data, JSON, "input.json")
		if err != nil {
			if root != nil {
				t.Fatal("failure returned partial tree")
			}
			return
		}
		encoded, marshalErr := json.Marshal(root)
		if marshalErr != nil || !json.Valid(encoded) {
			t.Fatal("accepted tree cannot be encoded", marshalErr)
		}
		second, loadErr := Load(encoded, JSON, "input.json")
		if loadErr != nil {
			t.Fatal("encoded tree cannot be loaded", loadErr)
		}
		encodedAgain, _ := json.Marshal(second)
		if !bytes.Equal(encoded, encodedAgain) {
			t.Fatal("non-deterministic ordered round trip")
		}
	})
}
