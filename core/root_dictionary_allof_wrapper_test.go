package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRootDictionaryAllOfReferenceWrapper(t *testing.T) {
	input := schemas(`{
		"AView":{"allOf":[{"$ref":"#/components/schemas/ZTextMap"}]},
		"Holder":{"properties":{"value":{"$ref":"#/components/schemas/AView"}},"required":["value"]},
		"ZTextMap":{"type":"object","additionalProperties":{"type":"string"}}
	}`)
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(input, "Dto")
			rq.Options.Validation, rq.Options.ModelStyle = validation, style
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatalf("validation=%t style=%s: %v", validation, style, err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatalf("generation must be deterministic: %v", err)
			}
			if text := source(t, got, "AViewDto"); !strings.Contains(text, "public typealias AViewDto = org.example.models.models.ZTextMapDto") {
				t.Fatal(text)
			}
			if text := source(t, got, "HolderDto"); !strings.Contains(text, "value: org.example.models.models.ZTextMapDto") {
				t.Fatal(text)
			}
		}
	}
}

func TestRootDictionaryAllOfNeutralMembersMatchSingleReference(t *testing.T) {
	views := []string{
		`{},REF`,
		`REF,{}`,
		`{"description":"notes"},REF,{"title":"view","$comment":"note"}`,
	}
	intents := []struct{ target, member, outer string }{
		{"", "", ""},
		{`,"nullable":true`, "", ""},
		{`,"nullable":true`, `,"nullable":false`, ""},
		{"", `,"nullable":false`, `,"nullable":true`},
	}
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, intent := range intents {
				ref := `{"$ref":"#/components/schemas/ZAlias"` + intent.member + `}`
				leaf := `{"type":"string"}`
				if validation {
					leaf = `{"type":"string","minLength":1}`
				}
				document := func(members string) []byte {
					return schemas(`{
						"AView":{"allOf":[` + members + `]` + intent.outer + `},
						"Holder":{"properties":{"value":{"$ref":"#/components/schemas/AView"}},"required":["value"]},
						"ZAlias":{"$ref":"#/components/schemas/ZMap"},
						"ZMap":{"additionalProperties":` + leaf + intent.target + `,"default":{"key":"ok"}}
					}`)
				}
				rq := request(document(ref), "Dto")
				rq.Options.Validation, rq.Options.ModelStyle = validation, style
				want, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatalf("single reference: validation=%t style=%s: %v", validation, style, err)
				}
				for _, view := range views {
					rq.Document = document(strings.Replace(view, "REF", ref, 1))
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatalf("neutral siblings %s: validation=%t style=%s: %v", view, validation, style, err)
					}
					if !reflect.DeepEqual(got.Artifacts, want.Artifacts) || !reflect.DeepEqual(got.Warnings, want.Warnings) {
						t.Fatalf("neutral siblings %s changed output: validation=%t style=%s", view, validation, style)
					}
				}
			}
		}
	}
}

func TestRootDictionaryAllOfNonNeutralMembersRemainRejected(t *testing.T) {
	for _, member := range []string{
		`{"nullable":true}`, `{"nullable":false}`, `{"type":"object"}`,
		`{"minProperties":1}`, `{"default":{}}`, `{"additionalProperties":true}`,
		`{"$ref":"#/components/schemas/ZMap"}`, `{"readOnly":true}`,
		`{"description":null}`, `true`,
	} {
		for _, validation := range []bool{false, true} {
			input := schemas(`{"AView":{"allOf":[{"$ref":"#/components/schemas/ZMap"},` + member + `]},"ZMap":{"additionalProperties":{"type":"string"}}}`)
			rq := request(input, "")
			rq.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatalf("non-neutral sibling accepted: %s validation=%t", member, validation)
			}
		}
	}
}

func TestRootDictionaryAllOfNullableAndAliases(t *testing.T) {
	input := schemas(`{
		"AInherit":{"allOf":[{"$ref":"#/components/schemas/ZMap"}]},
		"BNarrow":{"allOf":[{"$ref":"#/components/schemas/ZMap","nullable":false}]},
		"COuter":{"allOf":[{"$ref":"#/components/schemas/ZMap","nullable":false}],"nullable":true},
		"DChain":{"allOf":[{"$ref":"#/components/schemas/BNarrow"}]},
		"EAlias":{"$ref":"#/components/schemas/DChain"},
		"Holder":{"properties":{
			"inherited":{"$ref":"#/components/schemas/AInherit"},
			"narrowed":{"$ref":"#/components/schemas/EAlias"}
		}},
		"ZMap":{"type":"object","nullable":true,"additionalProperties":{"type":["string","null"]}}
	}`)
	for _, validation := range []bool{false, true} {
		rq := request(input, "")
		rq.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatalf("validation=%t: %v at %s", validation, err, *err.(core.Diagnostics)[0].Pointer)
		}
		for name, want := range map[string]string{
			"AInherit": "org.example.models.models.ZMap?",
			"BNarrow":  "kotlin.collections.Map<kotlin.String, kotlin.String?>",
			"COuter":   "org.example.models.models.ZMap?",
			"DChain":   "kotlin.collections.Map<kotlin.String, kotlin.String?>",
			"EAlias":   "kotlin.collections.Map<kotlin.String, kotlin.String?>",
		} {
			if text := source(t, got, name); !strings.Contains(text, want) {
				t.Fatalf("%s missing %q:\n%s", name, want, text)
			}
		}
		if text := source(t, got, "Holder"); !strings.Contains(text, "inherited: org.example.models.models.ZMap?") ||
			!strings.Contains(text, "narrowed: kotlin.collections.Map<kotlin.String, kotlin.String?>") {
			t.Fatal(text)
		}
	}
}

func dictionaryAllOfWrapperChain(n int) []byte {
	entries := []string{`"ZMap":{"type":"object","additionalProperties":{"type":"string"}}`}
	for i := range n {
		target := "ZMap"
		if i > 0 {
			target = fmt.Sprintf("A%02d", i-1)
		}
		entries = append(entries, fmt.Sprintf(`"A%02d":{"allOf":[{"$ref":"#/components/schemas/%s"}]}`, i, target))
	}
	return schemas("{" + strings.Join(entries, ",") + "}")
}

func TestRootDictionaryAllOfSourceDepth(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, n := range []int{23, 24} {
			rq := request(dictionaryAllOfWrapperChain(n), "")
			rq.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(rq)
			if n == 23 {
				if err != nil {
					t.Fatalf("23 wrapper nodes plus dictionary edge fit: %v", err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatalf("24 wrapper nodes must retain source work: %v", err)
			}
		}
	}
}

func TestRootDictionaryAllOfExistingRejections(t *testing.T) {
	for _, tc := range []struct{ schemas, pointer string }{
		{`{"A":{"allOf":[{"type":"object","additionalProperties":{"type":"string"}}]}}`, "/components/schemas/A/allOf/0/additionalProperties"},
		{`{"A":{"allOf":[{"$ref":"#/components/schemas/Z"},{"type":"object"}]},"Z":{"type":"object","additionalProperties":{"type":"string"}}}`, "/components/schemas/Z/additionalProperties"},
		{`{"A":{"allOf":[{"$ref":"#/components/schemas/Z"}]},"Z":{"type":"array","items":{"type":"string"}}}`, "/components/schemas/A/allOf"},
	} {
		failure(t, schemas(tc.schemas), "E_SCHEMA_UNSUPPORTED", tc.pointer)
	}
}

func TestRootDictionaryAllOfRetainsTargetsAndIndependentDefaults(t *testing.T) {
	input := schemas(`{
		"AMap":{"allOf":[{"$ref":"#/components/schemas/ZMap"}],"description":"view"},
		"BAlias":{"$ref":"#/components/schemas/AMap"},
		"COpen":{"allOf":[{"$ref":"#/components/schemas/ZOpen"}]},
		"DEnum":{"allOf":[{"$ref":"#/components/schemas/ZEnum"}]},
		"EPeer":{"allOf":[{"$ref":"#/components/schemas/ZPeer"}]},
		"Holder":{"properties":{
			"optional":{"$ref":"#/components/schemas/BAlias"},
			"required":{"$ref":"#/components/schemas/AMap"}
		},"required":["required"]},
		"ZMap":{"type":"object","additionalProperties":{"type":"string"},"default":{"key":"ok"}},
		"ZOpen":{"additionalProperties":true,"default":{"key":"value"}},
		"ZEnum":{"additionalProperties":{"type":"string","enum":["one","two"]}},
		"ZPeer":{"additionalProperties":{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}}
	}`)
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(input, "Dto")
			rq.Options.Validation, rq.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatalf("checked=%t style=%s: %v at %s", checked, style, err, *err.(core.Diagnostics)[0].Pointer)
			}
			for _, name := range []string{"AMapDto", "BAliasDto", "COpenDto", "DEnumDto", "EPeerDto"} {
				if text := source(t, got, name); strings.Contains(text, " default`") {
					t.Fatalf("target factory leaked into %s:\n%s", name, text)
				}
			}
			for _, name := range []string{"ZMapDto", "ZOpenDto"} {
				if text := source(t, got, name); !strings.Contains(text, "fun `"+name+" default`()") {
					t.Fatalf("target factory missing from %s:\n%s", name, text)
				}
			}
			if text := source(t, got, "HolderDto"); !strings.Contains(text, "optional: org.example.models.models.ZMapDto? = null") ||
				!strings.Contains(text, "required: org.example.models.models.ZMapDto") {
				t.Fatalf("target factory changed field presence:\n%s", text)
			}
			if text := source(t, got, "ZPeerDtoValue"); !strings.Contains(text, "val id: kotlin.String") {
				t.Fatalf("dictionary lost nominal value peer:\n%s", text)
			}
		}
	}
}

func TestRootDictionaryAllOfTransparentCyclesRemainRejected(t *testing.T) {
	for _, input := range [][]byte{
		schemas(`{"A":{"allOf":[{"$ref":"#/components/schemas/A"}]}}`),
		schemas(`{"A":{"allOf":[{"$ref":"#/components/schemas/B"}]},"B":{"$ref":"#/components/schemas/A"}}`),
		schemas(`{"A":{"allOf":[{"$ref":"#/components/schemas/Z"}]},"Z":{"additionalProperties":{"$ref":"#/components/schemas/A"}}}`),
	} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatalf("transparent cycle accepted: checked=%t input=%s", checked, input)
			}
		}
	}
}

func TestRootDictionaryAllOfCheckedCollectionDepth(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, edges := range []int{16, 17} {
			leaf := `{"type":"string"}`
			if checked {
				leaf = `{"type":"string","minLength":1}`
			}
			for i := 1; i < edges; i++ {
				leaf = `{"type":"array","items":` + leaf + `}`
			}
			input := schemas(`{"AView":{"allOf":[{"$ref":"#/components/schemas/ZMap"}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/AView"}}},"ZMap":{"additionalProperties":` + leaf + `}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if !checked || edges == 16 {
				if err != nil {
					t.Fatalf("checked=%t edges=%d: %v", checked, edges, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatalf("checked edge %d escaped the depth limit: %v", edges, err)
			}
		}
	}
}

func TestRootDictionaryAllOfCumulativeCaptureBudget(t *testing.T) {
	for _, count := range []int{110, 120} {
		entries := []string{`"ZMap":{"description":"` + strings.Repeat("a", 72*1024) + `","additionalProperties":{"type":"string"}}`}
		for i := 0; i < count; i++ {
			entries = append(entries, fmt.Sprintf(`"A%03d":{"allOf":[{"$ref":"#/components/schemas/ZMap"}]}`, i))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas("{"+strings.Join(entries, ",")+"}"), "Dto"))
		if count == 110 {
			if err != nil {
				t.Fatalf("bounded capture rejected: %v", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatalf("overlapping target capture escaped the byte budget: %v", err)
		}
	}
}

func TestRootDictionaryAllOfNeutralMemberCaptureBudget(t *testing.T) {
	for _, count := range []int{128, 16384} {
		members := strings.Repeat(`{},`, count) + `{"$ref":"#/components/schemas/ZMap"}`
		input := schemas(`{"AView":{"allOf":[` + members + `]},"ZMap":{"additionalProperties":{"type":"string"}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 128 {
			if err != nil {
				t.Fatalf("bounded neutral members rejected: %v", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatalf("neutral source members escaped the capture budget: %v", err)
		}
	}
}
