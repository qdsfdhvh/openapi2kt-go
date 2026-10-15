package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestCommonDictionaryOriginalNodesBranchChecksAndProbeWarnings(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"additionalProperties":{"type":"string","minLength":2},"minProperties":1}},"required":["value"]},{"properties":{"kind":{"const":"b"},"value":{"additionalProperties":{"type":"string","minLength":3},"maxProperties":2}},"required":["value"]}]}}}}`)
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	a := &analyzer{root: root, components: map[string]*component{}, options: Options{PackageName: "org.example", Suffix: "Dto", Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	u := ds[0].(*ObjectUnion)
	common := commonProperty(u.Common, "value")
	if common == nil || common.Type.Kind != Map || common.Type.Collection != nil || common.Type.Element.Validation != nil {
		t.Fatal(common)
	}
	nodes := get(get(get(root, "components"), "schemas"), "Choice")
	for i, variant := range u.Variants {
		p := variant.Properties[0]
		original := get(get(get(nodes, "oneOf").Items()[i], "properties"), "value")
		if p.Type.Collection == nil || p.Type.Element.Validation == nil || p.Type.Element.Validation == common.Type.Element.Validation {
			t.Fatal("lost original branch checks", p)
		}
		if *p.Type.Element.Validation.MinLength != int64(i+2) {
			t.Fatal(p.Type)
		}
		if a.collectionSources[p.Type.Collection] != "/components/schemas/Choice/oneOf/"+string(rune('0'+i))+"/properties/value" {
			t.Fatal("original source pointer", a.collectionSources)
		}
		if get(original, "additionalProperties") == nil {
			t.Fatal("original node absent")
		}
	}
	// Common probing cannot append duplicate source warnings.
	warnings := len(a.warnings)
	f := &field{Wire: "value", Schema: get(get(get(nodes, "oneOf").Items()[0], "properties"), "value"), Pointer: "/components/schemas/Choice/oneOf/0/properties/value", Depth: 2}
	if _, e := a.commonType(f); e != nil || len(a.warnings) != warnings {
		t.Fatal(e, a.warnings)
	}
	bad, e := document.Load([]byte(`{"schema":{"additionalProperties":{"type":"string","format":"unsupported"}}}`), document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	f.Schema = get(bad, "schema")
	if _, e := a.commonType(f); e == nil {
		t.Fatal("replaced original field reused stale public projection")
	}
}
