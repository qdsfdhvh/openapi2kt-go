package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestCallerNativeReferenceChainsOriginalProviderBacking(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/caller-native-reference-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("caller-native chains", checked, err)
		}
		for name, scalar := range map[string]string{"MoneyBridge": "org.example.scalarvalues.Money", "AgainBridge": "org.example.scalarvalues.Money", "LongBridge": "kotlin.Long", "BytesBridge": "kotlin.ByteArray", "ElementBridge": "kotlinx.serialization.json.JsonElement", "ObjectBridge": "kotlinx.serialization.json.JsonObject", "StringBridge": "kotlin.String", "InstantBridge": "kotlin.time.Instant"} {
			if !strings.Contains(source(t, got, name+"Dto"), "= "+scalar+"?\n") {
				t.Fatal(name)
			}
		}
		if !strings.Contains(source(t, got, "MoneyFalseBridgeDto"), "= org.example.scalarvalues.Money\n") {
			t.Fatal("false provider backing")
		}
		if strings.Contains(source(t, got, "HolderDto"), "LongAsStringSerializer") || !strings.Contains(source(t, got, "SubmitInputDto"), "Patch<org.example.scalarvalues.Money>") {
			t.Fatal("provider profile acquired magic codec")
		}
	}
}
