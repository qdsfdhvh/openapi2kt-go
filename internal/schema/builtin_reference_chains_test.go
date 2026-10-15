package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestBuiltinReferenceChainsOriginalIdentityChecksAndCachedWork(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Text":{"type":["string","null"],"minLength":2},"Legacy":{"type":"integer","format":"int64","minimum":2},"LegacyAlias":{"$ref":"#/components/schemas/Legacy"},"TextAlias":{"$ref":"#/components/schemas/Text"},"Hops":{"properties":{"text":{"$ref":"#/components/schemas/TextAlias"},"legacy":{"$ref":"#/components/schemas/LegacyAlias"},"true":{"$ref":"#/components/schemas/Hops/properties/text","nullable":true},"again":{"$ref":"#/components/schemas/TextBridgeFalse","nullable":true}}},"TextBridgeFalse":{"$ref":"#/components/schemas/Hops/properties/true","nullable":false},"LegacyBridge":{"$ref":"#/components/schemas/Hops/properties/legacy"},"AgainBridge":{"$ref":"#/components/schemas/Hops/properties/again"}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.builtin", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, alias, canonical string
		steps                  int
		nullable               bool
	}{
		{"Hops/properties/true", "TextBridgeFalse", "Text", 4, true},
		{"Hops/properties/legacy", "LegacyBridge", "Legacy", 3, false},
		{"Hops/properties/again", "AgainBridge", "Text", 6, true},
	} {
		target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
		if err != nil {
			t.Fatal(tc, err)
		}
		canonical := a.components[tc.canonical]
		if target.CanonicalBuiltin != canonical || target.Schema != canonical.Schema || target.Schema != get(catalog, tc.canonical) || target.Pointer != canonical.Pointer || target.Steps != tc.steps || target.ModelName != "" || target.Deferred != nil {
			t.Fatal("lost original transparent source", tc, target)
		}
		if value, err := target.effectiveNullable(); err != nil || value != tc.nullable {
			t.Fatal("nullable path drift", tc, value, err)
		}
		if a.deferredNominals[target.Schema] != nil || len(a.components) != len(catalog.Members()) {
			t.Fatal("builtin acquired fabricated catalog or nominal barrier")
		}
		a.aliasPlanning = true
		no := false
		ty, err := a.referenceType(target, &no, 48-tc.steps, "/depth48")
		if err != nil || ty.Kind != Scalar || ty.Name != canonical.Scalar || ty.Validation != canonical.Validation || ty.SchemaSteps != tc.steps || a.validationSources[ty.Validation] != canonical.Pointer {
			t.Fatal("scalar/check pointer/source work lost", ty, err)
		}
		if _, err := a.referenceType(target, &no, 49-tc.steps, "/depth49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("false narrowed source budget", err)
		}
		alias := a.components[tc.alias]
		cached, err := a.lookupSchemaReference("#/components/schemas/"+tc.alias, "/cache")
		if err != nil || cached.Catalog != alias || cached.CanonicalBuiltin != canonical || cached.Steps != tc.steps+1 || alias.Schema != get(catalog, tc.alias) {
			t.Fatal("root catalog/height changed", cached, err)
		}
		ty, err = a.referenceType(cached, &no, 47-tc.steps, "/cache48")
		if err != nil || ty.Kind != Scalar || ty.SchemaSteps != tc.steps+1 || ty.Validation != canonical.Validation {
			t.Fatal(ty, err)
		}
		if _, err := a.referenceType(cached, &no, 48-tc.steps, "/cache49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
		a.aliasPlanning = false
	}
	if a.componentType(a.components["TextBridgeFalse"]).Kind != Scalar || a.componentType(a.components["AgainBridge"]).Kind != Nullable {
		t.Fatal("root false/entering internal true priority changed")
	}
	legacy, err := a.lookupSchemaReference("#/components/schemas/LegacyAlias", "/legacy")
	if err != nil || legacy.Schema != a.components["LegacyAlias"].Schema || legacy.CanonicalBuiltin != nil {
		t.Fatal("historical direct alias lookup flattened", legacy, err)
	}
	changed := *legacy
	changed.Schema = a.components["Text"].Schema
	if a.bindNamedChainBuiltin(&changed, "/changed") {
		t.Fatal("changed/copy carrier authorized")
	}
	if !a.bindNamedChainBuiltin(legacy, "/valid") || legacy.Catalog != a.components["LegacyAlias"] || legacy.Schema != a.components["Legacy"].Schema {
		t.Fatal("new carrier did not preserve declared legacy catalog")
	}
	canonical := a.components["Text"]
	missing := []Declaration{}
	var original Declaration
	for _, d := range ds {
		if d.Name() == "TextDto" {
			original = d
		} else {
			missing = append(missing, d)
		}
	}
	if err := a.validateBuiltinBindings(missing); err == nil {
		t.Fatal("missing retained canonical authorized")
	}
	copy := &ValueAlias{ClassName: "TextDto", Type: a.componentType(canonical)}
	if err := a.validateBuiltinBindings(append(append([]Declaration{}, missing...), copy)); err == nil {
		t.Fatal("same-name replacement authorized")
	}
	a.builtinOrigins[copy] = a.builtinOrigins[original]
	if err := a.validateBuiltinBindings(append(append([]Declaration{}, ds...), copy)); err == nil {
		t.Fatal("multiple retained declarations authorized")
	}
	delete(a.builtinOrigins, copy)
	checks := canonical.Validation
	copyChecks := *checks
	canonical.Validation = &copyChecks
	if err := a.validateBuiltinBindings(ds); err == nil {
		t.Fatal("equivalent but replaced checks authorized")
	}
	canonical.Validation = checks
	originalSchema := canonical.Schema
	canonical.Schema = a.components["Legacy"].Schema
	if err := a.validateBuiltinBindings(ds); err == nil {
		t.Fatal("replaced original schema authorized")
	}
	canonical.Schema = originalSchema
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original document mutated")
	}
}
