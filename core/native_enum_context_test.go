package core_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestNativeStringEnumContexts(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "native-enum-values.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	for name, tokens := range map[string][]string{
		"VocabularyDto":  {"public sealed class VocabularyDto", "serialName: kotlin.String"},
		"RecordDto":      {"public sealed class Stamp", "public sealed class Money", "public sealed class ItemsInner", "public sealed class LookupValue"},
		"RecordInputDto": {"public sealed class Choice", "Patch<Choice>"},
		"DefaultApi":     {"input: kotlin.time.Instant", "input: kotlin.Long", "input: org.example.scalarvalues.Money", "vocabulary: org.example.models.api.models.BigVocabularyDto"},
	} {
		content := source(t, r, name)
		for _, token := range tokens {
			if !strings.Contains(content, token) {
				t.Fatal(name, "missing", token, content)
			}
		}
	}
	for _, file := range r.Artifacts.Files {
		if strings.Contains(file.Path, "LongAsStringSerializer") || strings.Contains(file.Content, "LongAsStringSerializer") {
			t.Fatal("vocabulary must not use Long serializer", file.Path)
		}
	}
	if !reflect.DeepEqual(r, report(t, input, "Dto")) {
		t.Fatal("nondeterministic native enum report")
	}
	req := request(input, "Dto")
	req.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.Generate(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range excluded.Files {
		if strings.Contains(file.Path, "/api/") || strings.Contains(file.Path, "/Patch") || strings.Contains(file.Path, "LongAsStringSerializer") {
			t.Fatal("excluded operation left helper/local model", file.Path)
		}
	}
}

func TestNativeStringEnumGuards(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"enum":["x"],"x-native-type":false}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"type":"string","enum":["x"],"x-native-type":""}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"enum":["x"],"x-native-type":"org.example.class"}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"enum":["x"],"x-native-type":"org.example.Money<Type>"}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"enum":["x","x"],"x-native-type":"date"}`, "E_SCHEMA_INVALID", "/enum/1"},
		{`{"enum":[1],"x-native-type":"bigint"}`, "E_SCHEMA_INVALID", "/enum/0"},
		{`{"enum":[],"x-native-type":"date"}`, "E_SCHEMA_INVALID", "/enum"},
		{`{"enum":["x"],"x-native-type":"date","pattern":"x"}`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`{"enum":["x"],"x-native-type":"date","format":"unknown"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"type":"integer","enum":[1],"x-native-type":"bigint"}`, "E_SCHEMA_UNSUPPORTED", "/x-native-type"},
		{`{"enum":["x"],"x-native-type":"date","description":123}`, "E_SCHEMA_INVALID", "/description"},
		{`{"enum":["x"],"x-native-type":"bigint","const":"x"}`, "E_SCHEMA_UNSUPPORTED", "/enum"},
	} {
		failure(t, fieldDoc(tc.field), tc.code, "/components/schemas/Record/properties/value"+tc.tail)
	}
	for _, n := range []int{240, 241} {
		values := make([]string, n)
		for i := range n {
			values[i] = fmt.Sprintf(`"v%d"`, i)
		}
		input := schemas(`{"Vocabulary":{"x-native-type":"bigint","enum":[` + strings.Join(values, ",") + `]}}`)
		if n == 240 {
			r := report(t, input, "Dto")
			if len(r.Artifacts.Files) != 1 || strings.Contains(source(t, r, "VocabularyDto"), "LongAsStringSerializer") {
				t.Fatal(r)
			}
		} else {
			failure(t, input, "E_ENUM_LIMIT", "/components/schemas/Vocabulary/enum")
		}
	}
	mixed := report(t, schemas(`{"Vocabulary":{"x-native-type":"bigint","enum":["x"]},"Record":{"type":"object","properties":{"plain":{"x-native-type":"bigint"},"vocabulary":{"$ref":"#/components/schemas/Vocabulary"},"wrapped":{"allOf":[{"$ref":"#/components/schemas/Vocabulary"}],"nullable":true}}}}`), "Dto")
	content := source(t, mixed, "RecordDto")
	if len(mixed.Artifacts.Files) != 3 || strings.Count(content, "@kotlinx.serialization.Serializable(with = org.example.models.models.LongAsStringSerializer::class)") != 1 {
		t.Fatal("only the ordinary bigint field needs its serializer", content)
	}
	// Nullable folds remain vocabulary types and own no Long helper.
	for _, field := range []string{
		`{"anyOf":[{"type":"string","enum":["x"],"x-native-type":"bigint"},{"type":"null"}]}`,
		`{"type":["string","null"],"enum":["x"],"x-native-type":"bigint"}`,
	} {
		r := report(t, fieldDoc(field), "Dto")
		if len(r.Artifacts.Files) != 1 || !strings.Contains(source(t, r, "RecordDto"), "public sealed class Value") {
			t.Fatal(r)
		}
	}
}
