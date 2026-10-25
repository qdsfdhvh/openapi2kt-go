package schema

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestNullableRecursiveCoverOriginalIntentNode(t *testing.T) {
	cover := strings.ReplaceAll(jsonGraphCover("Value", "Value"), `"$ref":"#/components/schemas/Value"`, `"$ref":"#/components/schemas/Value","nullable":true`)
	root, e := document.ParseObject([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":`+cover+`,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}}}`), document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.nullableedges", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	child := get(get(a.components["Value"].Schema, "oneOf").Items()[4], "items")
	copy, e := document.ParseObject([]byte(`{"nullable":true}`), document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	for i, m := range child.Members() {
		if m.Name == "nullable" {
			child.Members()[i].Value = get(copy, "nullable")
			if a.validateInternalModels(ds) == nil {
				t.Fatal("same-byte recursive edge intent node replacement authorized")
			}
			child.Members()[i].Value = m.Value
			if err := a.validateInternalModels(ds); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("missing original edge intent")
}

func TestNullableRecursiveCoverOriginalWeightsAndAliasIntent(t *testing.T) {
	for _, n := range []int{24, 25} {
		entries := []string{}
		for i := 0; i < n; i++ {
			next := fmt.Sprint("R", (i+1)%n)
			c := jsonGraphCover(next, next)
			c = strings.ReplaceAll(c, `"$ref":"#/components/schemas/`+next+`"`, `"$ref":"#/components/schemas/`+next+`","nullable":true`)
			entries = append(entries, fmt.Sprintf(`"R%d":%s`, i, c))
		}
		a := jsonGraphAnalyzer(t, strings.Join(entries, ","))
		v := get(get(get(a.root, "components"), "schemas"), "R0")
		h, e := a.recursiveJSONRoot(v, "/components/schemas/R0")
		if n == 24 {
			if e != nil || h != 48 {
				t.Fatal("edge intent changed SCC work", h, e)
			}
		} else if e == nil || e.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("edge intent erased source work", h, e)
		}
	}
	a := jsonGraphAnalyzer(t, `"A":`+jsonGraphCover("ToB", "B")+`,"ToB":{"$ref":"#/components/schemas/B","nullable":true},"B":`+jsonGraphCover("ToA", "A")+`,"ToA":{"$ref":"#/components/schemas/A","nullable":false}`)
	schemas := get(get(a.root, "components"), "schemas")
	for _, name := range []string{"A", "B"} {
		h, e := a.recursiveJSONRoot(get(schemas, name), "/components/schemas/"+name)
		if e != nil || h != 6 {
			t.Fatal("transparent nullable alias erased SCC weight", name, h, e)
		}
	}
}
