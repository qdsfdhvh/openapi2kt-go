package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func nullableEnumInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-nullable-enum-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootNullableEnumAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(nullableEnumInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				s := source(t, r, n+suffix)
				if !strings.Contains(s, "= org.example.models.models.ZStatus"+suffix+"?") {
					t.Fatal(s)
				}
			}
			if s := source(t, r, "AFalse"+suffix); !strings.Contains(s, "= org.example.models.models.ZStatus"+suffix) || strings.Contains(s, "?") {
				t.Fatal(s)
			}
			holder := source(t, r, "Holder"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(holder, "val "+name+": org.example.models.models.ZStatus"+suffix+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(holder, "val "+name+": org.example.models.models.ZStatus"+suffix+",") {
					t.Fatal(holder)
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.Patch<org.example.models.models.ZStatus"+suffix+">") {
					t.Fatal(patch)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.ZStatus"+suffix+"? = null") {
					t.Fatal(patch)
				}
			}
			if !strings.Contains(patch, "val requiredNull: org.example.models.models.ZStatus"+suffix+"?,") {
				t.Fatal(patch)
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("nullability doesn't change erasure", api)
			}
		}
	}
}
func TestRootNullableEnumAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(nullableEnumInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	intent := map[string]bool{"AInherited": true, "BNullable": true, "ATrue": true, "AFalse": false}
	for n := range intent {
		delete(catalog, n)
	}
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if flag, ok := intent[strings.TrimPrefix(ref, "#/components/schemas/")]; ok {
					v["$ref"] = "#/components/schemas/ZStatus"
					if _, has := v["nullable"]; !has {
						v["nullable"] = flag
					}
				}
			}
			for _, child := range v {
				visit(child)
			}
		case []any:
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(doc)
	b, _ := json.Marshal(doc)
	for _, flag := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("whole existing target/nullability control", flag, e)
		}
	}
}

func TestRootNullableEnumAliasBudgetsAndExclusions(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, typed := range []bool{false, true} {
			limit := 48
			if typed {
				limit = 46
			}
			for _, n := range []int{limit, limit + 1} {
				consumer := ""
				if typed {
					consumer = fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, n-1)
				}
				b := rootSpecialChain(n, `{"enum":["ready","done"]}`, consumer)
				b = []byte(strings.Replace(string(b), `"A00":{"$ref":"#/components/schemas/Z"}`, `"A00":{"$ref":"#/components/schemas/Z","nullable":true}`, 1))
				req := request(b, "Dto")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if n == limit {
					if e != nil {
						t.Fatal(e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, r, e)
				}
			}
		}
	}
	for _, n := range []int{240, 241} {
		var values, fields []string
		for i := range n {
			values = append(values, fmt.Sprintf(`"v%d"`, i))
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"}`, i))
		}
		input := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"enum":[` + strings.Join(values, ",") + `]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(r, e)
		}
		input = schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"enum":["ready"]},"Holder":{"properties":{` + strings.Join(fields, ",") + `}}}`)
		r, e = core.GenerateWithDiagnostics(request(input, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(r, e)
		}
	}
	for _, terminal := range []string{`{"type":"string","format":"date","enum":["ready"]}`, `{"type":"string","x-native-type":"kotlin.String","enum":["ready"]}`, `{"type":"integer","enum":[1]}`, `{"type":"string","const":"ready"}`} {
		input := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":` + terminal + `}`)
		if terminal == `{"type":"integer","enum":[1]}` {
			// Original numeric input now has a validation-only nullable alias.
			r := request(input, "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "ADto"), "typealias ADto = kotlin.Int?") {
				t.Fatal(got, err)
			}
			failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/A")
			continue
		}
		if metadataStringEnumFixture(terminal) {
			metadataStringEnumGraduation(t, request(input, "Dto"))
			continue
		}
		aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/A")
	}
	report(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"enum":["ready"],"nullable":true}}`), "Dto") // Intrinsic plain enum nullability graduated independently.
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"enum":["ready"]},"Choice":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Choice/oneOf/0")
}
