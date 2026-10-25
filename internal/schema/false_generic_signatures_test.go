package schema

import (
	"fmt"
	"strings"
	"testing"
)

func TestFalseGenericLowerBounds(t *testing.T) {
	leaf := TypeRef{Kind: Named, Name: "org.example.models.Blocked"}
	known := map[string]int{leaf.Name: len(leaf.Name) + 2}
	list := func(t TypeRef) TypeRef { return TypeRef{Kind: List, Element: &t} }
	nullable := func(t TypeRef) TypeRef { return TypeRef{Kind: Nullable, Element: &t} }
	for _, tc := range []struct {
		name     string
		typeRef  TypeRef
		bytes    int
		variance int
	}{
		{"direct", leaf, known[leaf.Name], 0},
		{"list", list(leaf), known[leaf.Name] + 18, 1},
		{"nested-list", list(list(leaf)), known[leaf.Name] + 36, 2},
		{"nullable-outer", nullable(list(leaf)), known[leaf.Name] + 18, 1},
		{"nullable-inner", list(nullable(leaf)), known[leaf.Name] + 18, 0},
		{"map", TypeRef{Kind: Map, Element: &leaf}, known[leaf.Name] + len("Ljava/util/Map<Ljava/lang/String;>;"), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bytes, found := minimumFalseGenericSignature(tc.typeRef, false, known)
			if !found || bytes != tc.bytes || knownFalseListVarianceBytes(tc.typeRef, known) != tc.variance {
				t.Fatal("incorrect known-class lower bound", bytes, found, tc)
			}
			if knownListChainVarianceBytes(tc.typeRef) != 0 {
				t.Fatal("builtin variance counted a false classifier twice")
			}
		})
	}
}

func TestFalseGenericGuardNeedsActualClassifier(t *testing.T) {
	pkg := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
	name := pkg + ".models.BlockedDto"
	model := &Model{ClassName: "HolderDto"}
	for i := 0; i < 240; i++ {
		model.Properties = append(model.Properties, Property{Name: fmt.Sprintf("p%03d", i), Type: TypeRef{Kind: List, Element: &TypeRef{Kind: Named, Name: name}}})
	}
	plan := &Plan{Models: []Declaration{model, &ScalarAlias{ClassName: "BlockedDto", Scalar: "kotlin.Int"}}, modelDiagnosticPointers: map[*Model]string{model: "/components/schemas/Holder"}}
	if err := checkFalseNominalSignatureLimits(plan, pkg); err != nil {
		t.Fatal("alias spelling became false nominal authority", err)
	}
	plan.Models[1] = &FalseValue{ClassName: "BlockedDto"}
	if err := checkFalseNominalSignatureLimits(plan, pkg); err == nil {
		t.Fatal("actual false generic classifier was omitted")
	}
	// A speculative alias backing cannot authorize nominal or variance costs,
	// even when its exported spelling happens to match a retained false class.
	for i := range model.Properties {
		model.Properties[i].Type.Element.Element = &TypeRef{Kind: Scalar, Name: "kotlin.Int"}
	}
	if err := checkFalseNominalSignatureLimits(plan, pkg); err != nil {
		t.Fatal("unproven alias backing expanded", err)
	}
}
