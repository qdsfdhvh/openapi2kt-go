package schema

import "testing"

func TestKnownListChainVarianceCountsOnlyProvenInlineParameters(t *testing.T) {
	ty := signatureListTree(16)
	if knownListChainVarianceBytes(ty) != 15 || knownListChainVarianceBytes(ty.Null()) != 15 {
		t.Fatal("known nested List variance lost")
	}
	perParameter := minimumJVMSignatureBytes(ty, false) + knownListChainVarianceBytes(ty)
	if 3+203*perParameter != 65369 || 3+204*perParameter != 65691 {
		t.Fatal("compiled border signature changed", perParameter)
	}
	for _, uncertain := range []TypeRef{
		{Kind: Named, Name: "Alias", Element: &ty},
		{Kind: Map, Element: &ty},
		{Kind: List, Element: &TypeRef{Kind: Nullable, Element: &ty}},
		{Kind: List, Element: &TypeRef{Kind: List, Element: &TypeRef{Kind: Scalar, Name: "unknown.Type"}}},
	} {
		if knownListChainVarianceBytes(uncertain) != 0 {
			t.Fatal("uncertain variance guessed", uncertain)
		}
	}
}
