package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func hierarchyFieldsInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/hierarchy-fields.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHierarchyFieldsFixedMissingAndVaryingAncestor(t *testing.T) {
	for _, suffix := range []string{"", "Dto"} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				req := request(hierarchyFieldsInput(t), suffix)
				req.Options.ModelStyle, req.Options.Validation = style, checked
				got, err := core.GenerateWithDiagnostics(req)
				if err != nil {
					t.Fatal(suffix, style, checked, err)
				}
				fixed := source(t, got, "FixedRoot"+suffix)
				if !strings.Contains(fixed, "public sealed class FixedRoot"+suffix+"Kind") || !strings.Contains(fixed, "UnknownFixedRoot"+suffix) || !strings.Contains(fixed, `tag == "fixed"`) {
					t.Fatal(fixed)
				}
				for _, tc := range []struct{ name, wire, value, owner string }{
					{"MissingFirst", "kind", "first/a~b", "MissingRoot"},
					{"MissingSecond", "kind", "second", "MissingRoot"},
					{"DeepFirst", "family", "group", "DeepRoot"},
					{"DeepFirst", "stage", "first", "DeepGroup"},
					{"DeepSecond", "family", "group", "DeepRoot"},
				} {
					leaf := source(t, got, tc.name+suffix)
					tagName := strings.ToUpper(tc.wire[:1]) + tc.wire[1:]
					want := "public override val " + tc.wire + ": org.example.models.models." + tc.owner + suffix + tagName + " ="
					if !strings.Contains(leaf, want) || !strings.Contains(leaf, `.Serializer.fromSerialName("`+tc.value+`")`) {
						t.Fatal(want, leaf)
					}
				}
				first := source(t, got, "VaryFirst"+suffix)
				if !strings.Contains(first, "public override val family: org.example.models.models.VaryRoot"+suffix+"Family,") || strings.Contains(first, "VaryRoot"+suffix+"Family?") {
					t.Fatal(first)
				}
				group := source(t, got, "VaryGroup"+suffix)
				if !strings.Contains(group, "public abstract override val family: org.example.models.models.VaryRoot"+suffix+"Family") {
					t.Fatal(group)
				}
				missing := source(t, got, "MissingRoot"+suffix)
				if strings.Contains(missing, "alias-only") {
					t.Fatal("secondary mapping became a new route", missing)
				}
				again, err := core.GenerateWithDiagnostics(req)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("not deterministic", err)
				}
			}
		}
	}
}

func TestHierarchyFieldsSynthesizedSlotAndSourceGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, count := range []int{239, 240} {
			fields, required := []string{}, []string{}
			for i := range count {
				wire := fmt.Sprintf("p%d", i)
				fields = append(fields, fmt.Sprintf(`%q:{"type":"boolean"}`, wire))
				required = append(required, fmt.Sprintf("%q", wire))
			}
			body := `{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"leaf":"Leaf"}}},"Leaf":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`
			req := request(schemas(body), "")
			req.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(req)
			if count == 239 {
				if err != nil {
					t.Fatal(count, checked, err)
				}
			} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Leaf" {
				t.Fatal(count, checked, got, err)
			}
		}
	}
	for _, tc := range []struct{ body, code, pointer string }{
		{`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"properties":{"id":{"type":"string"}},"required":["kind"]}}`, "E_SCHEMA_INVALID", "/components/schemas/Leaf"},
		{`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"a-b"}},"Leaf":{"properties":{"aB":{"type":"boolean"}}}}`, "E_NAME_COLLISION", "/components/schemas/Leaf/properties/aB"},
		{`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"allOf":[{"properties":{"id":{"type":"string","format":"unsupported"}}},{"properties":{"id":{"type":"string"}}}]}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Leaf/allOf/0/properties/id/format"},
	} {
		failure(t, schemas(tc.body), tc.code, tc.pointer)
	}
}

func TestHierarchyFieldsFixedConstOnlyAndEnumRequired(t *testing.T) {
	for _, field := range []string{`{"type":"string","const":"fixed"}`, `{"enum":["fixed"],"nullable":true}`} {
		input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"fixed":"Leaf"}}},"Leaf":{"properties":{"kind":` + field + `}}}`)
		got := report(t, input, "Dto")
		leaf := source(t, got, "LeafDto")
		if strings.Contains(field, "const") {
			if !strings.Contains(leaf, "public data object LeafDto") {
				t.Fatal(leaf)
			}
		} else if !strings.Contains(leaf, "public override val kind: org.example.models.models.RootDtoKind,") || strings.Contains(leaf, "RootDtoKind?") {
			t.Fatal(leaf)
		}
	}
	// Source required properties remain source-only even for generated tags.
	var doc map[string]any
	if err := json.Unmarshal(hierarchyFieldsInput(t), &doc); err != nil {
		t.Fatal(err)
	}
	leaf := doc["components"].(map[string]any)["schemas"].(map[string]any)["MissingFirst"].(map[string]any)
	leaf["required"] = []string{"id", "kind"}
	input, _ := json.Marshal(doc)
	failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/MissingFirst")
}
