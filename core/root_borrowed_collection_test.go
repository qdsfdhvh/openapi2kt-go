package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func rootBorrowedCollectionInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/root-borrowed-collection.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRootBorrowedCollectionCanonicalBacking(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(rootBorrowedCollectionInput(t), "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct{ name, backing string }{{"ListBridge", "kotlin.collections.List<kotlin.String?>?"}, {"ListFalseBridge", "kotlin.collections.List<kotlin.String?>"}, {"MapBridge", "kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.String?>>"}, {"AgainBridge", "kotlin.collections.List<kotlin.String?>"}} {
				if !strings.Contains(source(t, got, tc.name+"Dto"), "public typealias "+tc.name+"Dto = "+tc.backing+"\n") {
					t.Fatal(tc, source(t, got, tc.name+"Dto"))
				}
			}
			if !strings.Contains(source(t, got, "SubmitInputDto"), "Patch<kotlin.collections.List<kotlin.String?>>") {
				t.Fatal("borrowed collection request presence lost")
			}
		}
	}
}

func TestRootBorrowedCollectionCrossOwnerLeavesAndUnusedRoots(t *testing.T) {
	for _, checked := range []bool{false, true} {
		doc := schemas(`{"Entry":{"properties":{"hop":{"$ref":"#/components/schemas/Terminal/properties/list"}}},"Terminal":{"properties":{"list":{"type":"array","items":{"$ref":"#/components/schemas/Leaf/properties/text"}}}},"Leaf":{"properties":{"text":{"type":["string","null"]}}},"UnusedRoot":{"$ref":"#/components/schemas/Entry/properties/hop"}}`)
		r := request(doc, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "UnusedRootDto"), "= kotlin.collections.List<kotlin.String?>\n") {
			t.Fatal("entryowner confused with terminal/leafsource", checked, err)
		}
	}
}
