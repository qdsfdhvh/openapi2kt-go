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

func nullableCallerInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-nullable-caller-native-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootNullableCallerNativeAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(nullableCallerInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, name := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, r, name+suffix); !strings.Contains(s, "= org.example.scalarvalues.Money?") {
					t.Fatal(s)
				}
			}
			if s := source(t, r, "AFalse"+suffix); !strings.Contains(s, "= org.example.scalarvalues.Money") || strings.Contains(s, "?") {
				t.Fatal(s)
			}
			for name, typ := range map[string]string{"LMaybe": "kotlin.Long", "BytesMaybe": "kotlin.ByteArray", "ElementMaybe": "kotlinx.serialization.json.JsonElement", "ObjectMaybe": "kotlinx.serialization.json.JsonObject", "StringMaybe": "kotlin.String", "InstantMaybe": "kotlin.time.Instant"} {
				if s := source(t, r, name+suffix); !strings.Contains(s, "= "+typ+"?") {
					t.Fatal(s)
				}
			}
			holder := source(t, r, "Holder"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(holder, "val "+name+": org.example.scalarvalues.Money? = null") {
					t.Fatal(holder)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(holder, "val "+name+": org.example.scalarvalues.Money,") {
					t.Fatal(holder)
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			if !strings.Contains(patch, "val inherited: org.example.models.models.Patch<org.example.scalarvalues.Money>") || !strings.Contains(patch, "val requiredNull: org.example.scalarvalues.Money?,") {
				t.Fatal(patch)
			}
			if strings.Contains(holder, "LongAsStringSerializer") {
				t.Fatal("caller Long marker is not bigint", holder)
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal(api)
			}
		}
	}
}
func TestRootNullableCallerNativeAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(nullableCallerInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	targets := map[string]string{"AInherited": "ZMoney", "BNullable": "ZMoney", "ATrue": "ZMoney", "AFalse": "ZMoney", "LMaybe": "ZLong", "BytesMaybe": "ZBytes", "ElementMaybe": "ZElement", "ObjectMaybe": "ZObject", "StringMaybe": "ZString", "InstantMaybe": "ZInstant"}
	for n := range targets {
		delete(catalog, n)
	}
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if target, ok := targets[strings.TrimPrefix(ref, "#/components/schemas/")]; ok {
					v["$ref"] = "#/components/schemas/" + target
					if _, exists := v["nullable"]; !exists {
						v["nullable"] = ref != "#/components/schemas/AFalse"
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
			t.Fatal("whole control", flag, e)
		}
	}
}
func TestRootNullableCallerNativeAliasSourceAndGuards(t *testing.T) {
	for _, terminal := range []string{`{"x-native-type":"org.example.scalarvalues.Money"}`, `{"type":"string","x-native-type":"kotlinx.serialization.json.JsonElement"}`, `{"x-native-type":"kotlin.Long"}`} {
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
					b := rootSpecialChain(n, terminal, consumer)
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
	}
	for _, terminal := range []string{`{"x-native-type":"org.example.scalarvalues.Money","enum":["ok"]}`, `{"type":"string","x-native-type":"org.example.scalarvalues.Money","x-kotlin-value-class":true}`} {
		pointer := "/components/schemas/Z"
		if strings.Contains(terminal, `"enum"`) {
			pointer = "/components/schemas/A"
		} else if strings.Contains(terminal, `"x-kotlin-value-class"`) {
			pointer = "/components/schemas/Z/x-native-type"
		}
		input := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":` + terminal + `}`)
		if metadataStringEnumFixture(terminal) {
			metadataStringEnumGraduation(t, request(input, "Dto"))
			continue
		}
		_, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err == nil {
			t.Fatal("excluded caller-native profile accepted", terminal)
		}
		d := err.(core.Diagnostics)[0]
		if d.Pointer == nil {
			t.Fatal("missing pointer", d)
		}
		if d.Code != "E_SCHEMA_UNSUPPORTED" || *d.Pointer != pointer {
			t.Fatalf("profile %s: actual code %s pointer %s; expected %s", terminal, d.Code, *d.Pointer, pointer)
		}
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"x-native-type":true}}`), "E_SCHEMA_INVALID", "/components/schemas/Z/x-native-type")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"F":{"$ref":"#/components/schemas/A","nullable":false},"Z":{"x-native-type":"org.example.scalarvalues.Money"}`
	for _, target := range []string{"A", "F"} {
		pointer := "/components/schemas/C/allOf/0/$ref"
		if target == "F" {
			pointer = "/components/schemas/Z/x-native-type"
		}
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+target+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", pointer)
	}
}
func TestRootNullableCallerNativeAliasSlots(t *testing.T) {
	for _, n := range []int{240, 241} {
		var fields, required []string
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"x-native-type":"org.example.scalarvalues.Money"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}
