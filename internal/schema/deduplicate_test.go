package schema

import (
	"reflect"
	"testing"
)

func dedupModel(name string) *Model {
	length := int64(1)
	child := &Model{ClassName: "Child", Properties: []Property{{Name: "id", WireName: "id", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}, Validation: &PrimitiveValidation{MinLength: &length}}}}
	return &Model{ClassName: name, Properties: []Property{{Name: "child", WireName: "child", Type: TypeRef{Kind: Named, Name: "org.example.api.models." + name + ".Child", SchemaSteps: 3}}}, Nested: []Declaration{child}}
}
func TestDeduplicateTypedOwnedScopeAndHints(t *testing.T) {
	a, b := dedupModel("Alpha"), dedupModel("Beta")
	b.Properties[0].Type.SchemaSteps = 7
	check := b.Nested[0].(*Model).Properties[0].Validation
	hints := EmissionHints{check: true}
	p := &Plan{Client: &Client{Models: []Declaration{b, a}}, EmissionHints: hints}
	before := b.Properties[0].Type.Name
	DeduplicateOperationModels(p, "org.example")
	alias, ok := p.Client.Models[0].(*ScalarAlias)
	if !ok || alias.Scalar != "org.example.api.models.Alpha" || p.Client.Models[1] != a {
		t.Fatal(p.Client.Models)
	}
	if !hints[a.Nested[0].(*Model).Properties[0].Validation] || b.Properties[0].Type.Name != before || b.Properties[0].Type.SchemaSteps != 7 {
		t.Fatal("hint lost or original view mutated")
	}
}
func TestDeduplicateSemanticDifferencesAndOpaqueNames(t *testing.T) {
	for _, kind := range []string{"wire", "default", "null", "patch", "codec", "constraint", "count", "constant", "external", "native", "foreignOwnedLike"} {
		t.Run(kind, func(t *testing.T) {
			a, b := dedupModel("Alpha"), dedupModel("Beta")
			switch kind {
			case "wire":
				b.Properties[0].WireName = "other"
			case "default":
				b.Properties[0].DefaultNull = true
			case "null":
				b.Properties[0].Type = b.Properties[0].Type.Null()
			case "patch":
				value := b.Properties[0].Type
				b.Properties[0].Type = TypeRef{Kind: PatchValue, Name: "org.example.models.Patch", Element: &value}
			case "codec":
				b.Properties[0].Type.Serializer = "org.example.ProvidedCodec"
			case "constraint":
				v := int64(2)
				b.Nested[0].(*Model).Properties[0].Validation.MinLength = &v
			case "count":
				v := int64(2)
				b.PropertyCount = &CollectionValidation{Minimum: &v, MinimumKey: "minProperties"}
			case "constant":
				b.Constants = []Constant{{Name: "kind", WireName: "kind", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}, Initializer: `"Beta"`}}
			case "external":
				a.Properties[0].Type.Name = "org.external.Alpha"
				b.Properties[0].Type.Name = "org.external.Beta"
			case "native":
				a.Properties[0].Type = TypeRef{Kind: Scalar, Name: "org.example.api.models.Alpha.Child"}
				b.Properties[0].Type = TypeRef{Kind: Scalar, Name: "org.example.api.models.Beta.Child"}
			case "foreignOwnedLike":
				a.Properties[0].Type.Name = "org.example.api.models.Alpha.NotDeclared"
				b.Properties[0].Type.Name = "org.example.api.models.Beta.NotDeclared"
			}
			p := &Plan{Client: &Client{Models: []Declaration{a, b}}, EmissionHints: EmissionHints{}}
			DeduplicateOperationModels(p, "org.example")
			if p.Client.Models[0] != a || p.Client.Models[1] != b {
				t.Fatal("different semantics folded", kind)
			}
		})
	}
}
func TestDeduplicateUnsupportedTreesAndComponentsUnchanged(t *testing.T) {
	a, b := dedupModel("Alpha"), dedupModel("Beta")
	a.Nested = append(a.Nested, &ScalarAlias{ClassName: "Native", Scalar: "kotlin.String"})
	b.Nested = append(b.Nested, &ScalarAlias{ClassName: "Native", Scalar: "kotlin.String"})
	p := &Plan{Models: []Declaration{a, b}, Client: &Client{Models: []Declaration{a, b}}, EmissionHints: EmissionHints{}}
	DeduplicateOperationModels(p, "org.example")
	if !reflect.DeepEqual(p.Models, p.Client.Models) {
		t.Fatal("component or excluded tree changed")
	}
}
