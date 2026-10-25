package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// DEX before version 040 rejects whitespace in method names. Public Kotlin
// factory spelling and private helper call sites must survive the JVM fix.
func TestJVMNamesPreserveKotlinFactoryAndDefaultLiteral(t *testing.T) {
	input := schemas(`{"Seed":{"type":"string","default":"fun ` + "`fake name`" + `("},"Holder":{"properties":{"values":{"type":"array","items":{"type":"integer"},"uniqueItems":true,"default":[1,2]}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(input, "Dto")
		rq.Options.ModelStyle, rq.Options.Validation = style, true
		first, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		seed := source(t, first, "SeedDto")
		if !strings.Contains(seed, "@kotlin.jvm.JvmName(\"openapiKtJvm_") || !strings.Contains(seed, "public fun `SeedDto default`(): SeedDto =") || !strings.Contains(seed, "fun `fake name`(") {
			t.Fatal("public factory spelling or literal changed", seed)
		}
		holder := source(t, first, "HolderDto")
		if !strings.Contains(holder, "@kotlin.jvm.JvmName(\"openapiKtJvm_") || !strings.Contains(holder, "`openapi-kt uniqueItems") || !strings.Contains(holder, "`openapi-kt default array") {
			t.Fatal("default/unique helper lost compatibility annotations", holder)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatal("JVM annotation changed deterministic result", err)
		}
	}
}
