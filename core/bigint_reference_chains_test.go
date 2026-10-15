package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestBigintReferenceChainsCanonicalBackingAndFieldOnlyCodec(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/bigint-reference-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("new named bigint chains", checked, err)
		}
		for _, name := range []string{"AmountBridgeDto", "AgainBridgeDto"} {
			if !strings.Contains(source(t, got, name), "= kotlin.Long?\n") {
				t.Fatal(name)
			}
		}
		if !strings.Contains(source(t, got, "AmountFalseBridgeDto"), "= kotlin.Long\n") {
			t.Fatal("false must preserve primitive backing")
		}
		holder := source(t, got, "HolderDto")
		if strings.Count(holder, "LongAsStringSerializer::class") != 15 || !strings.Contains(holder, "List<kotlin.Long?>") || !strings.Contains(holder, "Map<kotlin.String, kotlin.Long?>") {
			t.Fatal("original terminal field codec lost", holder)
		}
		patch := source(t, got, "SubmitInputDto")
		if !strings.Contains(patch, "Patch<@kotlinx.serialization.Serializable(with = org.example.models.models.LongAsStringSerializer::class) kotlin.Long>") {
			t.Fatal("Patch argument codec lost", patch)
		}
		api := source(t, got, "DefaultApi")
		if strings.Contains(api, "LongAsStringSerializer") || !strings.Contains(api, "body<kotlin.Long?>") {
			t.Fatal("field codec leaked into roots/SSE", api)
		}
	}
}
