package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestBytesReferenceChainsOriginalAuthorityAndCachedWork(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"string","format":"byte","nullable":true},"Alias":{"$ref":"#/components/schemas/Value"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias"},"false":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false},"again":{"$ref":"#/components/schemas/Bridge","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/false","nullable":true},"Again":{"$ref":"#/components/schemas/Hops/properties/again","nullable":false}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.bytes", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	c := a.components["Value"]
	for _, tc := range []struct {
		path, root string
		steps      int
		nullable   bool
	}{{"Hops/properties/value", "Bridge", 3, true}, {"Hops/properties/false", "Bridge", 4, false}, {"Hops/properties/again", "Again", 6, true}} {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(err)
		}
		if target.CanonicalBytes != c || target.Schema != c.Schema || target.Pointer != c.Pointer || target.Steps != tc.steps || target.ModelName != "" || target.Deferred != nil || len(a.components) != 5 {
			t.Fatal("canonical source/work identity", tc, target)
		}
		if flag, err := target.effectiveNullable(); err != nil || flag != tc.nullable {
			t.Fatal(tc, flag, err)
		}
		a.aliasPlanning = true
		no := false
		ty, err := a.referenceType(target, &no, 48-tc.steps, "/depth48")
		if err != nil || ty.Kind != Scalar || ty.Name != "kotlin.ByteArray" || ty.Slots() != 1 || ty.Validation != nil || ty.Serializer != "" || ty.SchemaSteps != tc.steps {
			t.Fatal(ty, err)
		}
		if _, err := a.referenceType(target, &no, 49-tc.steps, "/depth49"); err == nil {
			t.Fatal("false cache reset")
		}
		a.aliasPlanning = false
	}
	if a.componentType(a.components["Bridge"]).Kind != Nullable || a.componentType(a.components["Again"]).Kind != Scalar {
		t.Fatal("path-local nearest override lost")
	}
	cached, err := a.lookupSchemaReference("#/components/schemas/Again", "/cache")
	if err != nil || cached.Catalog != a.components["Again"] || cached.CanonicalBytes != c || cached.Steps != 7 {
		t.Fatal(cached, err)
	}
	missing := []Declaration{}
	var actual Declaration
	for _, d := range ds {
		if d.Name() == c.Name {
			actual = d
		} else {
			missing = append(missing, d)
		}
	}
	if err := a.validateBytesBindings(missing); err == nil {
		t.Fatal("missing source declaration authorized")
	}
	replacement := &ValueAlias{ClassName: c.Name, Type: a.componentType(c)}
	if err := a.validateBytesBindings(append(append([]Declaration{}, missing...), replacement)); err == nil {
		t.Fatal("same-name replacement authorized")
	}
	a.bytesOrigins[replacement] = a.bytesOrigins[actual]
	if err := a.validateBytesBindings(append(append([]Declaration{}, ds...), replacement)); err == nil {
		t.Fatal("multiple declarations authorized")
	}
	delete(a.bytesOrigins, replacement)
	if err := a.validateBytesBindings(append(append([]Declaration{}, ds...), actual)); err == nil {
		t.Fatal("repeat original declaration authorized")
	}
	saved := c.Schema
	copied, err := document.Load([]byte(`{"type":"string","format":"byte","nullable":true}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	c.Schema = copied
	if err := a.validateBytesBindings(ds); err == nil {
		t.Fatal("same-byte copied source authorized")
	}
	c.Schema = saved
	c.Scalar = "kotlin.Double"
	if err := a.validateBytesBindings(ds); err == nil {
		t.Fatal("layout changed")
	}
	c.Scalar = "kotlin.ByteArray"
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutation")
	}
}
