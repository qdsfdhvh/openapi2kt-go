package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func aliasFailure(t *testing.T, input []byte, code, pointer string) {
	t.Helper()
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	diagnostics, ok := err.(core.Diagnostics)
	if !ok || len(r.Artifacts.Files) != 0 || len(diagnostics) != 1 {
		t.Fatalf("expected rejection without artifacts: %#v %v", r, err)
	}
	d := diagnostics[0]
	if d.Code != code || d.Pointer == nil || *d.Pointer != pointer {
		t.Fatalf("unexpected diagnostic: %#v, want %s %s", d, code, pointer)
	}
}

func TestScalarAliasGraphGuards(t *testing.T) {
	for _, tc := range []struct{ body, code, pointer string }{
		{`"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"}`, "E_REF_CYCLE", "/components/schemas/A/$ref"},
		{`"A":{"$ref":"#/components/schemas/Missing"}`, "E_REF_MISSING", "/components/schemas/A/$ref"},
		{`"A":{"$ref":"https://example.test/spec.json"}`, "E_REF_UNSUPPORTED", "/components/schemas/A/$ref"},
		{`"A":{"$ref":true}`, "E_SCHEMA_INVALID", "/components/schemas/A/$ref"},
		{`"A":{"$ref":"#/components/schemas/B"},"B":{"type":"string","minLength":-1}`, "E_SCHEMA_INVALID", "/components/schemas/B/minLength"},
	} {
		t.Run(tc.code+tc.body, func(t *testing.T) { aliasFailure(t, schemas("{"+tc.body+"}"), tc.code, tc.pointer) })
	}
	for _, reverse := range []bool{false, true} {
		parts := []string{`"Base":{"type":"integer","format":"int64","minimum":0}`}
		for i := 0; i < 48; i++ {
			target := "Base"
			if i > 0 {
				target = fmt.Sprintf("A%02d", i-1)
			}
			parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
		}
		if reverse {
			for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
				parts[i], parts[j] = parts[j], parts[i]
			}
		}
		req := request(schemas("{"+strings.Join(parts, ",")+"}"), "")
		req.Options.Validation = true
		if _, err := core.Generate(req); err != nil {
			t.Fatal("48-edge catalog", reverse, err)
		}
		body := "{" + strings.Join(parts, ",") + `,"Record":{"type":"object","properties":{"v":{"$ref":"#/components/schemas/A47"}}}}`
		aliasFailure(t, schemas(body), "E_SCHEMA_DEPTH", "/components/schemas/Record/properties/v")
	}
}

func TestAliasConstraintsCannotDisappearInRawProjections(t *testing.T) {
	prefix := `{"openapi":"3.1.0","components":{"schemas":{"Id":{"type":"string","minLength":1},"Link":{"$ref":"#/components/schemas/Id"},"Record":{"type":"object","properties":{"id":{"$ref":"#/components/schemas/Link"}}}}},"paths":`
	for _, paths := range []string{
		`{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"type":"object","properties":{"id":{"$ref":"#/components/schemas/Link"}}}}],"responses":{"204":{}}}}}`,
		`{"/v":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"array","items":{"type":"object","properties":{"r":{"$ref":"#/components/schemas/Record"}}}}}}},"responses":{"204":{}}}}}`,
		`{"/v":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"anyOf":[{"$ref":"#/components/schemas/Link"},{"type":"boolean"}]}}}}}}}}`,
		`{"/v":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"v":{"anyOf":[{"$ref":"#/components/schemas/Link"},{"type":"boolean"}]}}}}}},"responses":{"204":{}}}}}`,
	} {
		aliasFailure(t, []byte(prefix+paths+"}"), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Id/minLength")
	}
}
