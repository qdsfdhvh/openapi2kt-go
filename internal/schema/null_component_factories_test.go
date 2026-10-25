package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestJsonNullComponentFactoryOriginalNodePermission(t *testing.T) {
	for _, definition := range []string{`{"type":"null","default":null}`, `{"enum":[null],"default":null}`} {
		root := loadComponentDocument(t, `{"Null":`+definition+`}`)
		v := get(get(get(root, "components"), "schemas"), "Null")
		before, _ := root.MarshalJSON()
		a := analyzer{options: Options{PackageName: "org.example.nullfactory"}}
		if err := a.markComponentDefaultSource(v, "/components/schemas/Null"); err != nil || !a.componentDefaultSchemas[v] {
			t.Fatal("original nullroot not marked", err)
		}
		if kind, _, err := schemaType(v, ""); err != nil || kind != "null" {
			t.Fatal("original null grammar lost", err)
		}
		if encoded, err := a.componentDefaultEncoded(&component{Schema: v, Pointer: "/components/schemas/Null", Name: "Null"}); err != nil || encoded != "null" {
			t.Fatal("nullroot default representation", encoded, err)
		}
		bytes, _ := v.MarshalJSON()
		clone, e := document.ParseObject(bytes, document.MaxDocumentBytes)
		if e != nil {
			t.Fatal(e)
		}
		if a.componentDefaultSchemas[clone] {
			t.Fatal("samebytes clone inherited source permission")
		}
		if has(v, "enum") {
			if _, err := a.nativeEnum("null", v, "/components/schemas/Null"); err != nil {
				t.Fatal(err)
			}
			if _, err := a.nativeEnum("null", clone, "/clone"); err == nil {
				t.Fatal("clone nullenum default authorized")
			}
			if _, err := nullOnlyEnumType(v, "/unowned"); err == nil {
				t.Fatal("global nullenum grammar widened")
			}
		} else {
			if _, err := scalarTypeFor(&a, "null", v, "/components/schemas/Null"); err != nil {
				t.Fatal(err)
			}
			if _, err := scalarTypeFor(&a, "null", clone, "/clone"); err == nil {
				t.Fatal("clone plainnull default authorized")
			}
		}
		plan, _, err := Analyze(root, Options{PackageName: "org.example.nullfactory", Suffix: "Dto"})
		if err != nil {
			t.Fatal(err)
		}
		alias, ok := componentDeclaration(plan.Models, "NullDto").(*ScalarAlias)
		if !ok || alias.Scalar != "kotlinx.serialization.json.JsonNull" || alias.ComponentDefault != "null" {
			t.Fatal("nullroot lost actual scalar alias", alias)
		}
		if len(plan.Models) != 1 {
			t.Fatal("nullroot invented nominal enum", plan.Models)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("nullfactory rewrote original schema")
		}
	}
}
