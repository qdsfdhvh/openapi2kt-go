package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumConsumerVocabularyIsTypedReuseIdentity(t *testing.T) {
	r := request(schemas(`{"CodeA":{"type":"integer","enum":[1,2]},"CodeB":{"type":"integer","enum":[1,3]},"Alpha":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/CodeA"}}}},"Beta":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/CodeA"}}}},"Gamma":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/CodeB"}}}}}`), "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
		t.Fatal("typed reference vocabulary was lost by merge")
	}
	again, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(got, again) {
		t.Fatal(e)
	}
}
