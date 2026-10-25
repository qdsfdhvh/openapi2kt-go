package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func dtoStringCollectionReferenceAnalysis(t *testing.T, validate bool) (*analyzer, *Plan, *document.Value, *dtoUniqueSource) {
	t.Helper()
	data, e := os.ReadFile("../../tests/fixtures/http/dto-string-collection-reference-unique.json")
	if e != nil {
		t.Fatal(e)
	}
	root, err := document.Load(data, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.dtostringcollectionreferenceunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if validate {
		if err := a.validateDTOUniqueUses(plan); err != nil {
			t.Fatal(err)
		}
	}
	src := a.dtoUniqueSources[a.components["Item"].Schema]
	if src == nil || len(src.ReferenceFields) != 39 {
		t.Fatal("missing original reference fields")
	}
	return a, plan, root, src
}

func TestDTOStringCollectionReferenceActualEnumAuthority(t *testing.T) {
	a, plan, root, src := dtoStringCollectionReferenceAnalysis(t, true)
	before, _ := root.MarshalJSON()
	reject := func(label string) {
		t.Helper()
		if err := a.validateDTOUniqueUses(plan); err == nil {
			t.Fatal(label, "authorized foreign enum collection")
		}
	}
	for _, proof := range src.ReferenceFields {
		if !proof.Collection || proof.EnumSchema == nil {
			continue
		}
		actual := a.dtoReferenceEnums[proof.EnumSchema]
		if actual == nil || actual != proof.EnumActual {
			t.Fatal("missing original actual enum")
		}
		name := actual.ClassName
		actual.ClassName = "Foreign"
		reject("changed physical name")
		actual.ClassName = name
		member := actual.Members[0]
		actual.Members[0].Wire = "foreign"
		reject("changed wire vocabulary")
		actual.Members[0] = member
		actual.Members[0].Name = "Foreign"
		reject("changed API member")
		actual.Members[0] = member
		vocabulary := get(proof.EnumSchema, "enum")
		original := vocabulary.Items()[0]
		raw, _ := original.MarshalJSON()
		wrapper, e := document.ParseObject(append(append([]byte(`{"v":`), raw...), '}'), document.MaxDocumentBytes)
		if e != nil {
			t.Fatal(e)
		}
		vocabulary.Items()[0] = get(wrapper, "v")
		reject("same-byte original vocabulary node replacement")
		vocabulary.Items()[0] = original
		origin := a.enumOrigins[actual]
		changed := origin
		changed.Pointer += "/foreign"
		a.enumOrigins[actual] = changed
		reject("foreign original Pointer")
		a.enumOrigins[actual] = origin
	}
	for i, p := range src.Actual.Properties {
		if p.WireName == "stringRefBridge" {
			src.Actual.Properties[i].Default = "kotlin.collections.emptyList()"
			reject("foreign initializer")
			src.Actual.Properties[i] = p
		}
	}
	if err := a.validateDTOUniqueUses(plan); err != nil {
		t.Fatal("restored authority", err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original AST changed")
	}
}

func TestDTOStringCollectionReferenceFirstBarrierIdentity(t *testing.T) {
	for _, nested := range []bool{false, true} {
		a, plan, _, _ := dtoStringCollectionReferenceAnalysis(t, false)
		var restore func()
		if nested {
			owner := componentDeclaration(plan.Models, "StringBorrowOwnerDto").(*Model)
			old := owner.Nested[0]
			copy := *old.(*StringEnum)
			owner.Nested[0] = &copy
			restore = func() { owner.Nested[0] = old }
		} else {
			for i, d := range plan.Models {
				if d.Name() == "StringRefListDtoInner" {
					copy := *d.(*StringEnum)
					plan.Models[i] = &copy
					restore = func() { plan.Models[i] = d }
					break
				}
			}
		}
		if restore == nil {
			t.Fatal("missing original enum")
		}
		if err := a.validateDTOUniqueUses(plan); err == nil {
			t.Fatal("copied original enum authorized before first barrier", nested)
		}
		restore()
		if err := a.validateDTOUniqueUses(plan); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBorrowedInlineEnumCollectionRetainsActualOwner(t *testing.T) {
	a, plan, _, _ := dtoStringCollectionReferenceAnalysis(t, true)
	if len(a.borrowedEnumLeaves) == 0 {
		t.Fatal("inline original enum admission not recorded")
	}
	for _, use := range a.borrowedEnumLeaves {
		if use.Pointer != "/components/schemas/StringBorrowOwner/properties/e~1nums/items" {
			continue
		}
		old := use.OwnerSchema
		use.OwnerSchema = use.Schema
		if err := a.validateBorrowedEnumLeaves(plan.Models); err == nil {
			t.Fatal("foreign lexical owner granted inline enum")
		}
		use.OwnerSchema = old
		actual := use.Actual
		copy := *actual
		use.Actual = &copy
		if err := a.validateBorrowedEnumLeaves(plan.Models); err == nil {
			t.Fatal("copied actual enum authorized")
		}
		use.Actual = actual
		if err := a.validateBorrowedEnumLeaves(plan.Models); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatal("missing original inline collection enum use")
}
