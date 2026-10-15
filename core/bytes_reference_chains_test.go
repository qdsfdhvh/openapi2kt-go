package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestBytesReferenceChainsNativeBackingAndOriginalSource(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/bytes-reference-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("new byte/binary source chains", checked, err)
		}
		for _, name := range []string{"BytesBridgeDto", "AgainBridgeDto"} {
			if !strings.Contains(source(t, got, name), "= kotlin.ByteArray?\n") {
				t.Fatal(name)
			}
		}
		if !strings.Contains(source(t, got, "BytesFalseBridgeDto"), "= kotlin.ByteArray\n") {
			t.Fatal("false backing")
		}
		holder := source(t, got, "HolderDto")
		if strings.Contains(holder, "Serializable(with") || !strings.Contains(holder, "List<kotlin.ByteArray?>") || !strings.Contains(holder, "Map<kotlin.String, kotlin.ByteArray?>") {
			t.Fatal("invented bytes codec", holder)
		}
		patch := source(t, got, "SubmitInputDto")
		if !strings.Contains(patch, "Patch<kotlin.ByteArray>") {
			t.Fatal(patch)
		}
		api := source(t, got, "DefaultApi")
		if !strings.Contains(api, "body<kotlin.ByteArray?>") || !strings.Contains(api, "setBody(input)") {
			t.Fatal("raw Ktor projection changed", api)
		}
	}
}
