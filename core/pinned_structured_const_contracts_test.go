package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestPinnedStructuredConstFixtureProfilesAndOriginalWarnings(t *testing.T) {
	input, e := os.ReadFile("../tests/fixtures/http/pinned-structured-const.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for _, optimized := range []bool{false, true} {
				r := request(input, "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				r.Options.HoistSharedShapes = optimized
				r.Options.DeduplicateModels = optimized
				r.Options.MergeIdenticalModels = optimized
				r.Options.LockHoistedNames = optimized
				got, e := core.GenerateWithDiagnostics(r)
				if e != nil {
					t.Fatal(style, checked, optimized, e)
				}
				for _, name := range []string{"ChoiceDto", "RefChoiceDto", "InferredDto", "ProductDto", "SealedDto"} {
					code := source(t, got, name)
					if strings.Contains(code, `obj["payload"]?.jsonPrimitive`) || strings.Contains(code, `val tag = obj["payload"]`) || !strings.Contains(code, "public class ") {
						t.Fatal("Data literal became tag or singleton", name, code)
					}
				}
				alias := source(t, got, "ObjectAliasDto")
				if !strings.Contains(alias, "typealias ObjectAliasDto = kotlinx.serialization.json.JsonObject\n") {
					t.Fatal(alias)
				}
				if !strings.Contains(source(t, got, "AlphaDto"), "public val payload: kotlinx.serialization.json.JsonObject =") {
					t.Fatal("Inherited body lost")
				}
				locations := map[string]bool{}
				for _, w := range got.Warnings {
					if w.Code == "W_CONST_STRUCTURED_APPROXIMATION" {
						if w.Pointer == nil {
							t.Fatal(w)
						}
						locations[*w.Pointer] = true
					}
				}
				for _, p := range []string{"/components/schemas/ObjectValue/const", "/components/schemas/Choice/oneOf/0/properties/payload/const", "/components/schemas/Base/properties/payload/const"} {
					if !locations[p] {
						t.Fatal("Original source warning lost", p, locations)
					}
				}
				if locations["/components/schemas/RefChoice/oneOf/0/properties/payload/const"] || locations["/components/schemas/Alpha/properties/payload/const"] {
					t.Fatal("Invented consumer const source", locations)
				}
				r.PreviousNameLock = got.Artifacts.NameLock
				again, e := core.GenerateWithDiagnostics(r)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("Pinned profile/lock replay drift", e)
				}
			}
		}
	}
}

func TestPinnedStructuredConstGrammarAndActualTagBoundary(t *testing.T) {
	for _, tc := range []struct{ value, code, tail string }{
		{`{"type":"array","const":{}}`, "E_SCHEMA_INVALID", "const"},
		{`{"const":{},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"const":{},"format":"json"}`, "E_SCHEMA_UNSUPPORTED", "format"},
		{`{"const":[],"x-native-type":"example.Value"}`, "E_SCHEMA_UNSUPPORTED", "x-native-type"},
		{`{"const":{},"minProperties":0}`, "E_SCHEMA_UNSUPPORTED", "minProperties"},
		{`{"const":[],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "items"},
		{`{"const":{},"nullable":"bad"}`, "E_SCHEMA_INVALID", "nullable"},
	} {
		for _, checked := range []bool{false, true} {
			input := schemas(`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":` + tc.value + `}},{"properties":{"tag":{"const":"two"}}}]}}`)
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(r)
			if e == nil {
				t.Fatal(tc, checked)
			}
			d := e.(core.Diagnostics)[0]
			if d.Code != tc.code || d.Pointer == nil || *d.Pointer != "/components/schemas/Choice/oneOf/0/properties/payload/"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, e)
			}
		}
	}
	for _, value := range []string{`{"const":{}}`, `{"type":"array","const":[]}`, `{"$ref":"#/components/schemas/Value"}`} {
		input := schemas(`{"Value":{"const":{}},"Alpha":{"properties":{"tag":` + value + `}},"Beta":{"properties":{"tag":{"const":"two"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Alpha"},{"$ref":"#/components/schemas/Beta"}],"discriminator":{"propertyName":"tag","mapping":{"one":"Alpha","two":"Beta"}}}}`)
		got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Structured discriminator accepted", value, e)
		}
	}
}

func TestPinnedStructuredConstSerializedMemberLimit(t *testing.T) {
	for _, width := range []int{240, 241} {
		for _, reference := range []bool{false, true} {
			props := []string{`"tag":{"const":"one"}`}
			for i := 1; i < width; i++ {
				value := `{"const":{"literal":true}}`
				if reference {
					value = `{"$ref":"#/components/schemas/Value"}`
				}
				props = append(props, fmt.Sprintf(`"p%d":%s`, i, value))
			}
			input := schemas(`{"Value":{"const":{}},"Choice":{"oneOf":[{"properties":{` + strings.Join(props, ",") + `}},{"properties":{"tag":{"const":"two"}}}]}}`)
			got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
			if (e == nil) != (width == 240) || e != nil && (e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
				t.Fatal(width, reference, e)
			}
		}
	}
}

func TestPinnedStructuredConstKeepsAliasSourceHeightAndMixedDataWires(t *testing.T) {
	for _, hops := range []int{45, 46} {
		parts := []string{`"Value":{"const":{"$ref":"literal data"}}`}
		last := "Value"
		for i := range hops {
			name := fmt.Sprintf("Alias%d", i)
			parts = append(parts, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s"}`, name, last))
			last = name
		}
		parts = append(parts, fmt.Sprintf(`"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"$ref":"#/components/schemas/%s"}}},{"properties":{"tag":{"const":"two"},"payload":{"const":{}}}}]}`, last))
		got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if (e == nil) != (hops == 45) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Union reset alias source height", hops, e)
		}
	}
	input := schemas(`{"Choice":{"allOf":[{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"const":{}}}},{"properties":{"tag":{"const":"two"},"payload":{"const":"scalar"}}}]},{"properties":{"label":{"type":"string"}}}]}}`)
	got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(source(t, got, "ChoiceDto"), `obj["payload"]?.jsonPrimitive`) {
		t.Fatal("A scalar peer turned JSON data into a compound tag")
	}
}

func TestPinnedStructuredConstDataNamedTypeNeverBecomesFallbackTag(t *testing.T) {
	for _, common := range []bool{true, false} {
		beta := `{"const":"scalar"}`
		if common {
			beta = `{"const":{"n":2}}`
		}
		input := schemas(`{"Alpha":{"properties":{"type":{"const":{"n":1}}}},"Beta":{"properties":{"type":` + beta + `}},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Alpha"},{"$ref":"#/components/schemas/Beta"}]}}`)
		got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if e != nil {
			t.Fatal(e)
		}
		code := source(t, got, "ChoiceDto")
		if strings.Contains(code, `val tag = obj["type"]?.jsonPrimitive`) || !strings.Contains(code, "JsonContentPolymorphicSerializer") {
			t.Fatal("Structured type data became tag or conflicted with sealed discriminator", common, code)
		}
	}
}

func TestPinnedStructuredConstTypeDataInDefaultHierarchyIntermediate(t *testing.T) {
	input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"family","mapping":{"group":"Group","other":"Other"}}},"Group":{"oneOf":[{"$ref":"#/components/schemas/Alpha"},{"$ref":"#/components/schemas/Beta"}]},"Alpha":{"properties":{"type":{"const":{"n":1}}}},"Beta":{"properties":{"type":{"const":{"n":2}}}},"Other":{"properties":{"family":{"const":"other"}}}}`)
	got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, got, "GroupDto"), "JsonContentPolymorphicSerializer") {
		t.Fatal("Hierarchy type data conflicts with class discriminator", source(t, got, "GroupDto"))
	}
}

func TestPinnedStructuredConstTypeDataDoesNotInventAnUnpinnedCarrierTag(t *testing.T) {
	input := schemas(`{"Alpha":{"properties":{"type":{"const":{}},"labels":{"additionalProperties":{"type":"string"}}}},"Beta":{"properties":{"type":{"const":[]},"labels":{"additionalProperties":{"type":"string"}}}},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Alpha"},{"$ref":"#/components/schemas/Beta"}]}}`)
	got, e := core.GenerateWithDiagnostics(request(input, "Dto"))
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(source(t, got, "ChoiceDto"), `obj["labels"]?.jsonPrimitive`) {
		t.Fatal("Unpinned data carrier became an invented scalar tag")
	}
}
