package core_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func patternSchema(t *testing.T, pattern any) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"type": "string", "x-kotlin-value-class": true, "pattern": pattern})
	if err != nil {
		t.Fatal(err)
	}
	return schemas(`{"Id":` + string(data) + `}`)
}

func TestPatternGuards(t *testing.T) {
	for _, tc := range []struct {
		pattern any
		code    string
	}{
		{nil, "E_SCHEMA_INVALID"}, {3, "E_SCHEMA_INVALID"}, {false, "E_SCHEMA_INVALID"},
		{"[", "E_SCHEMA_INVALID"}, {"(", "E_SCHEMA_INVALID"}, {"a{1001}", "E_SCHEMA_INVALID"},
		{"a\\", "E_SCHEMA_INVALID"}, {strings.Repeat("a", 1025), "E_SCHEMA_INVALID"},
		{strings.Repeat("(", 257) + "a" + strings.Repeat(")", 257), "E_SCHEMA_INVALID"},
		{`(?<=a)b`, "E_SCHEMA_UNSUPPORTED"}, {`(?!a)b`, "E_SCHEMA_UNSUPPORTED"},
		{`(a)\1`, "E_SCHEMA_UNSUPPORTED"}, {`(?i)abc`, "E_SCHEMA_UNSUPPORTED"},
		{`a++`, "E_SCHEMA_UNSUPPORTED"}, {`\p{L}`, "E_SCHEMA_UNSUPPORTED"}, {`\u1234`, "E_SCHEMA_UNSUPPORTED"},
	} {
		failure(t, patternSchema(t, tc.pattern), tc.code, "/components/schemas/Id/pattern")
	}
	for _, p := range []string{"", `^ab$`, `(?:ab|cd)+`, `[A-Z]{2}[0-9]+`, `\d\s\w`, `a+?`, `\++`, `\?+`, `🌿`, `[+*]`, `[a-z]`, `a{2,3}`, `\$\\"`} {
		r := report(t, patternSchema(t, p), "")
		if !strings.Contains(source(t, r, "Id"), "containsMatchIn") {
			t.Fatal(p, r)
		}
	}
	failure(t, schemas(`{"Id":{"type":"integer","x-kotlin-value-class":true,"pattern":"a"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Id/pattern")
	failure(t, schemas(`{"Id":{"type":"string","x-kotlin-value-class":false,"pattern":"a"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Id/pattern")
	failure(t, fieldDoc(`{"type":"string","pattern":"a"}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/pattern")
}

func TestPatternValidation(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/patterns.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CodeDto", "SearchDto", "RecordDto", "PatchInputDto"} {
		text := source(t, r, name)
		if !strings.Contains(text, "containsMatchIn") || !strings.Contains(text, "does not match pattern.") {
			t.Fatal(name, text)
		}
	}
	text := source(t, r, "CodeDto")
	if !strings.Contains(text, `"^[A-Z]{2}[0-9]+\$"`) {
		t.Fatal("pattern literal changed", text)
	}
	text = source(t, r, "RecordDto")
	if strings.Contains(text, "kotlin.text.Regex(") || !strings.Contains(text, ".toRegex().containsMatchIn") {
		t.Fatal("kotlin property shadows regex package", text)
	}
	if !strings.Contains(text, "this.kotlin") || !strings.Contains(text, "this.note?.let") {
		t.Fatal(text)
	}
}
