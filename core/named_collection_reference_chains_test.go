package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestNamedCollectionReferenceChainsOriginalBackingAndPeers(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/named-collection-reference-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("namedcollection chain", checked, err)
		}
		for _, tc := range []struct{ name, want string }{
			{"ListBridgeDto", "kotlin.collections.List<kotlin.String?>?"}, {"AgainBridgeDto", "kotlin.collections.List<kotlin.String?>?"}, {"ListFalseBridgeDto", "kotlin.collections.List<kotlin.String?>"},
			{"MapBridgeDto", "kotlin.collections.Map<kotlin.String, kotlin.String?>?"}, {"MapFalseBridgeDto", "kotlin.collections.Map<kotlin.String, kotlin.String?>"}, {"OpenBridgeDto", "kotlinx.serialization.json.JsonObject?"}, {"RowsBridgeDto", "kotlin.collections.List<org.example.models.models.ZRowsDtoInner>"},
		} {
			if !strings.Contains(source(t, got, tc.name), "= "+tc.want+"\n") {
				t.Fatal(tc, source(t, got, tc.name))
			}
		}
		if !strings.Contains(source(t, got, "ZRowsDtoInner"), "val id: kotlin.String") {
			t.Fatal("original canonical nominal item peer lost")
		}
	}
}
