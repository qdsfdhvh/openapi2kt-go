package naming

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/testutil"
)

func TestOptionsAndReservedPaths(t *testing.T) {
	for _, p := range []string{"org.example.catalog", "org.example._internal", "Kotlin.example", "org.example.com0"} {
		if err := ValidateOptions(p, "Dto"); err != nil {
			t.Fatal(p, err)
		}
	}
	for _, p := range []string{"", "org..example", ".org", "org.", "org/../../escape", "org.when", "kotlin.example", "org.CON", "org.lpt9", "org.雪", "org._", "org.0a"} {
		if err := ValidateOptions(p, ""); err == nil || err.Code != "E_PACKAGE_INVALID" || err.Pointer != nil {
			t.Fatalf("unsafe package accepted: %q", p)
		}
	}
	for _, suffix := range []string{"", "Dto", "_Dto", "d1"} {
		if err := ValidateOptions("org.example", suffix); err != nil {
			t.Fatal(suffix, err)
		}
	}
	for _, suffix := range []string{"../Dto", "雪", "_", "0Dto", "Dto-X"} {
		if err := ValidateOptions("org.example", suffix); err == nil || err.Code != "E_NAME_INVALID" {
			t.Fatalf("unsafe suffix accepted: %q", suffix)
		}
	}
	for name, want := range map[string]bool{"con": true, "NUL": true, "COM1": true, "lpt9": true, "COM0": false, "COM10": false, "Null": false, "ConDto": false} {
		if ReservedPathSegment(name) != want {
			t.Fatal(name)
		}
	}
}

func TestModelAndPropertyNames(t *testing.T) {
	for _, tc := range []struct{ wire, class, property string }{
		{"display_label", "DisplayLabel", "display_label"},
		{"some-value", "SomeValue", "someValue"},
		{"URLValue", "URLValue", "URLValue"},
		{"9-lives", "N9Lives", "n9Lives"},
		{"when", "When", "`when`"},
		{"_a", "A", "_a"},
		{"雪-id", "Id", "id"},
		{"a/b~c", "ABC", "aBC"},
	} {
		class, err := ClassName(tc.wire, "/name")
		if err != nil || class != tc.class {
			t.Fatal(tc.wire, class, err)
		}
		property, err := PropertyName(tc.wire, "/name")
		if err != nil || property != tc.property {
			t.Fatal(tc.wire, property, err)
		}
	}
	for _, wire := range []string{"", "雪", "___", "-/*"} {
		for _, name := range []func(string, string) (string, *diagnostic.Diagnostic){ClassName, PropertyName} {
			_, err := name(wire, "/name")
			if err == nil || err.Code != "E_NAME_INVALID" || err.Pointer == nil || *err.Pointer != "/name" {
				t.Fatalf("invalid name accepted: %q", wire)
			}
		}
	}
}

func TestEnumMemberSpellings(t *testing.T) {
	for wire, want := range map[string]string{
		"hello-world": "HELLO_WORLD", "0value": "_0VALUE", "ab雪": "AB_",
		"": "VALUE", " \t\u00a0": "VALUE", "++": "PLUS_PLUS",
		"==": "EQUAL_EQUAL", "雪": "U96EA", "😀": "UD83D_UDE00", "_": "U005F",
		" " + "!" + "\u2000": "NOT", "\\/": "BACKSLASH_SLASH", "a__b": "A__B",
	} {
		if got := EnumMemberName(wire); got != want {
			t.Fatalf("%q: %q != %q", wire, got, want)
		}
	}
}

func TestKotlinLiteralEscapesAndPointerDecoding(t *testing.T) {
	if got := Literal("quote\" slash\\ $name\n\r\t\x00\x7f\u0085\u2028\u2029雪😀"); got != `"quote\" slash\\ \$name\n\r\t\u0000\u007f\u0085\u2028\u2029雪😀"` {
		t.Fatal(got)
	}
	for raw, encoded := range map[string]string{"": "", "a/b~c": "a~1b~0c", "~1": "~01", "雪": "雪"} {
		if Pointer("/base", raw) != "/base/"+encoded {
			t.Fatal("pointer escaping mismatch")
		}
		got, err := DecodePointerSegment(encoded, "/ref")
		if err != nil || got != raw {
			t.Fatal(raw, got, err)
		}
	}
	for _, encoded := range []string{"~", "a~2", "~~0"} {
		_, err := DecodePointerSegment(encoded, "/ref")
		if err == nil || err.Code != "E_SCHEMA_INVALID" || err.Pointer == nil || *err.Pointer != "/ref" {
			t.Fatal("invalid pointer escape accepted")
		}
	}
}

func TestFixtureOptionContracts(t *testing.T) {
	for _, c := range testutil.Corpus(t) {
		t.Run(c.ID, func(t *testing.T) {
			err := ValidateOptions(c.PackageName, c.ModelNameSuffix)
			if c.Expect.Code == "E_PACKAGE_INVALID" || c.ID == "suffix-traversal" {
				if err == nil || err.Code != c.Expect.Code || !reflect.DeepEqual(err.Pointer, c.Expect.Pointer) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal("option layer must not validate component names", err)
			}
		})
	}
}

func FuzzNamesAndPointers(f *testing.F) {
	for _, seed := range []string{"a/b~c", "", "雪😀", "9-lives", "when", "\x00", "_", "${value}"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, wire string) {
		if !utf8.ValidString(wire) || len(wire) > 4096 {
			t.Skip()
		}
		encoded := strings.TrimPrefix(Pointer("", wire), "/")
		decoded, err := DecodePointerSegment(encoded, "/ref")
		if err != nil || decoded != wire {
			t.Fatal("pointer failed round-trip", err)
		}
		name, err := ClassName(wire, "/name")
		if err == nil && (!identifier(name) || strings.ContainsAny(name, "/\\")) {
			t.Fatal("unsafe class name")
		}
		property, err := PropertyName(wire, "/name")
		if err == nil && !identifier(strings.Trim(property, "`")) {
			t.Fatal("unsafe property name")
		}
		if enum := EnumMemberName(wire); !identifier(enum) {
			t.Fatal("unsafe enum name", enum)
		}
		_ = Literal(wire)
	})
}
