package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionNominalActualOwnersAndTypedNullViews(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/union-nominal-owners.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, checked := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = checked
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		for _, tc := range []struct {
			alias, owner string
			nullable     bool
		}{{"UnionInner", "UnionBoxDto.Choice", true}, {"UnionInnerFalse", "UnionBoxDto.Choice", false}, {"NativeInner", "UnionBoxDto.Native", true}, {"NativeInnerFalse", "UnionBoxDto.Native", false}, {"UnionRootBridge", "ChoiceDto", false}, {"UnionNativeBridge", "MixedChoiceDto", false}, {"UnionInnerBridge", "UnionBoxDto.Choice", true}, {"NullableUnion", "ChoiceDto", true}, {"NullableUnionInherit", "ChoiceDto", true}, {"NullableUnionFalse", "ChoiceDto", false}, {"NullableUnionAgain", "ChoiceDto", true}, {"NullableCarrier", "CarrierUnionDto", true}, {"CarrierFalse", "CarrierUnionDto", false}, {"RootRowUnion", "RootRowsDtoInner", false}, {"RootMapUnion", "RootMapDtoValue", false}, {"RootRowUnionMaybe", "RootRowsDtoInner", true}} {
			want := "public typealias " + tc.alias + "Dto = org.example.models.models." + tc.owner
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.alias+"Dto"), want) {
				t.Fatal(tc, source(t, r, tc.alias+"Dto"))
			}
		}
		if !strings.Contains(source(t, r, "UnionBoxDto"), "next: org.example.models.models.UnionBoxDto.Choice? = null") {
			t.Fatal("canonical nested union recursion")
		}
		patch := source(t, r, "PatchUnionInputDto")
		if !strings.Contains(patch, "Patch<org.example.models.models.UnionBoxDto.Choice>") || !strings.Contains(patch, "Patch<org.example.models.models.UnionBoxDto.Native>") {
			t.Fatal("internal union intent did not create request Patch", patch)
		}
	}
}
