package schema

import (
	"fmt"
	"strings"
	"testing"
)

func TestWrapperDefaultTargetBoundsAndOriginalNodes(t *testing.T) {
	for _, edges := range []int{2, 48, 49} {
		fields := []string{`"T":{"type":"object"}`}
		for i := 0; i < edges; i++ {
			name, target := fmt.Sprintf("A%d", i), fmt.Sprintf("A%d", i+1)
			if i == edges-1 {
				target = "T"
			}
			fields = append(fields, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s"}`, name, target))
		}
		root := loadComponentDocument(t, `{`+strings.Join(fields, ",")+`}`)
		before, _ := root.MarshalJSON()
		schemas := get(get(root, "components"), "schemas")
		a := &analyzer{componentSchemas: schemas}
		pointer, target, ok := a.referenceWrapperDefaultTarget(&component{Schema: get(schemas, "A0")})
		if edges <= MaxSchemaDepth && (!ok || pointer != "/components/schemas/T" || target != get(schemas, "T")) || edges > MaxSchemaDepth && ok {
			t.Fatal("chain target lost original identity or exceeded hop cap", edges, pointer, ok)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("target traversal changed source nodes")
		}
	}
}

func TestChainedWrapperDefaultRetainsAliasOwnership(t *testing.T) {
	root := loadComponentDocument(t, `{"T":{"additionalProperties":{"additionalProperties":{"type":"integer"}}},"B":{"$ref":"#/components/schemas/T"},"M":{"$ref":"#/components/schemas/B"},"N":{"$ref":"#/components/schemas/M","default":{"z":{"z":1,"a":2},"a":{}}}}`)
	before, _ := root.MarshalJSON()
	for _, checked := range []bool{false, true} {
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked})
		if failure != nil {
			t.Fatal(failure)
		}
		for _, name := range []string{"TDto", "BDto", "MDto", "NDto"} {
			alias, ok := componentDeclaration(plan.Models, name).(*ValueAlias)
			if !ok {
				t.Fatal("chain alias disappeared", name)
			}
			want := ""
			if name == "NDto" {
				want = `{"a":{},"z":{"a":2,"z":1}}`
			}
			if alias.ComponentDefault != want {
				t.Fatal("chain inherited annotation or lost canonical value", name, alias.ComponentDefault)
			}
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("chain factory changed original schemas/default data")
		}
	}
}
