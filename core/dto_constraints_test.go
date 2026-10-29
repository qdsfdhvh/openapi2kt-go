package core_test

import (
	"fmt"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDTOBoundedSlots(t *testing.T) {
	for _, n := range []int{120, 121} {
		fields := make([]string, n)
		required := make([]string, n)
		for i := range fields {
			name := fmt.Sprintf("p%d", i)
			fields[i] = fmt.Sprintf(`"%s":{"type":"integer","format":"int64","minimum":0,"maximum":1}`, name)
			required[i] = fmt.Sprintf(`"%s"`, name)
		}
		input := schemas(`{"Wide":{"type":"object","properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		req := request(input, "")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if n == 120 {
			if err != nil || strings.Count(source(t, r, "Wide"), "require(value >= 0L)") != 120 {
				t.Fatal(r, err)
			}
		} else {
			if err == nil || len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
				t.Fatal(r, err)
			}
		}
	}
}

func TestDTOValidationGuards(t *testing.T) {
	for _, tc := range []struct{ field, code, pointer string }{
		{`{"type":"string","minLength":-1}`, "E_SCHEMA_INVALID", "minLength"},
		{`{"type":"integer","minimum":2147483648}`, "E_SCHEMA_INVALID", "minimum"},
		{`{"type":"number","minimum":2,"maximum":1}`, "E_SCHEMA_INVALID", "maximum"},
		{`{"type":"string","pattern":"(?<=x).*"}`, "E_SCHEMA_UNSUPPORTED", "pattern"},
		{`{"type":"array","items":{"type":"string"},"minLength":1}`, "E_SCHEMA_UNSUPPORTED", "items"},
		{`{"type":"object","minimum":0}`, "E_SCHEMA_UNSUPPORTED", "minimum"},
		{`{"maxLength":2}`, "E_SCHEMA_UNSUPPORTED", "maxLength"},
		{`{"type":"string","format":"date","minLength":1}`, "E_SCHEMA_UNSUPPORTED", "minLength"},
		{`{"type":"integer","x-native-type":"bigint","minimum":0}`, "E_SCHEMA_UNSUPPORTED", "x-native-type"},
		{`{"type":"string","const":"a","minLength":1}`, "E_SCHEMA_UNSUPPORTED", "minLength"},
		{`{"type":"array","items":{"type":"integer","default":0}}`, "E_SCHEMA_UNSUPPORTED", "items/default"},
		{`{"type":"object","additionalProperties":{"type":"string","default":"a"}}`, "E_SCHEMA_UNSUPPORTED", "additionalProperties/default"},
	} {
		req := request(fieldDoc(tc.field), "")
		req.Options.Validation = true
		if tc.field == `{"type":"array","items":{"type":"integer","default":0}}` {
			inlineBuiltinChildGraduation(t, req, `,"default":0`, `,"default":true`, "/components/schemas/Record/properties/value/items/default")
			continue
		}
		if tc.field == `{"type":"object","additionalProperties":{"type":"string","default":"a"}}` {
			inlineBuiltinChildGraduation(t, req, `,"default":"a"`, `,"default":true`, "/components/schemas/Record/properties/value/additionalProperties/default")
			continue
		}
		r, err := core.GenerateWithDiagnostics(req)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Record/properties/value/"+tc.pointer || len(r.Artifacts.Files) != 0 {
			t.Fatalf("%s: %v", tc.field, err)
		}
	}
	for _, input := range [][]byte{
		schemas(`{"Id":{"type":"string","x-kotlin-value-class":false,"minLength":1}}`),
		httpDoc(`{"parameters":[{"name":"p","in":"query","schema":{"type":"string","minLength":1}}],"responses":{}}`),
		httpDoc(`{"parameters":[{"name":"p","in":"query","schema":{"type":"array","items":{"type":"object","properties":{"v":{"type":"string","minLength":1}}}}}],"responses":{}}`),
	} {
		req := request(input, "")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("dropped bounds", r, err)
		}
	}
	// Option does not rewrite a schema without constraints.
	input := fieldDoc(`{"type":"string"}`)
	req := request(input, "")
	before := report(t, input, "")
	req.Options.Validation = true
	after, err := core.GenerateWithDiagnostics(req)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("unbounded output changed", err)
	}
}

func TestDTOConstraints(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/dto-constraints.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"PlainDto", "InlineChecksInputDto", "InlineChecksResponseDto", "WatchDataMessageDto"} {
		text := source(t, r, name)
		if !strings.Contains(text, "init {") || !strings.Contains(text, "isLowSurrogate") {
			t.Fatal(name, text)
		}
	}
	text := source(t, r, "PlainDto")
	for _, token := range []string{"this.name", "this.length", "this.`when`", "value >= -2", "value >= -9223372036854775806L", "value.isFinite()"} {
		if !strings.Contains(text, token) {
			t.Fatal(token, text)
		}
	}
	text = source(t, r, "InlineChecksInputDto")
	if !strings.Contains(text, "this.name.valueOrNull?.let") {
		t.Fatal(text)
	}
}
