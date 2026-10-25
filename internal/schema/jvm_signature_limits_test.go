package schema

import "testing"

func signatureListTree(depth int) TypeRef {
	ty := TypeRef{Kind: Scalar, Name: "kotlin.Int"}
	for range depth {
		child := ty
		ty = TypeRef{Kind: List, Element: &child}
	}
	return ty
}

func TestJVMSignatureMinimumIsOneSidedAgainstCompiledClassfile(t *testing.T) {
	// The actual 200x16 constructor has 64,403 UTF8 bytes. Dropping wildcard
	// variance gives 61,403: below the observed value, never an estimate used
	// to reject a feasible class. 240x16 is already over the hard limit even
	// after dropping all those bytes.
	ty := signatureListTree(16)
	minimum := minimumJVMSignatureBytes(ty, false)
	if minimum != 307 || 3+200*minimum != 61403 || 3+200*minimum >= 64403 || 3+240*minimum <= maxJVMUTF8ConstantBytes {
		t.Fatal("signature minimum stopped being a proof", minimum)
	}
	if minimumJVMSignatureBytes(TypeRef{Kind: Scalar, Name: "kotlin.Int"}, false) != 1 || minimumJVMSignatureBytes(TypeRef{Kind: Scalar, Name: "kotlin.Int"}.Null(), false) != 19 {
		t.Fatal("boxed primitive signature changed")
	}
	if minimumJVMSignatureBytes(TypeRef{Kind: Named, Name: "unknown.Provider", Element: &ty}, false) != 0 {
		t.Fatal("unknown alias was guessed or expanded")
	}
	mapType := TypeRef{Kind: Map, Element: &ty}
	if minimumJVMSignatureBytes(mapType, false) != len("Ljava/util/Map<Ljava/lang/String;")+minimum+2 {
		t.Fatal("known Map signature lost fixed key bytes")
	}
}
