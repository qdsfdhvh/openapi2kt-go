package kotlin

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"openapi-kt/internal/naming"
)

func TestJVMFunctionNamesKeepKotlinCallsAndLiteralBytes(t *testing.T) {
	source := "// fun `comment fake`(\n/* outer /* fun `nested fake`( */ end */\n" +
		"private val data = \"fun `string fake`( and \\\"quote\\\"\"\n" +
		"private val raw = \"\"\"fun `raw fake`(\"\"\"\n" +
		"private fun `openapi-kt default tree List Scalar ByteArray`(source: kotlin.String) = source\n" +
		"private fun <T : kotlin.Comparable<T>> `openapi-kt uniqueItems`(values: kotlin.collections.List<T>) = values\n" +
		"public fun `Factory default`() = `openapi-kt default tree List Scalar ByteArray`(data)\n" +
		"public fun ordinary() = raw\n"
	got := jvmFunctionNames(source)
	if strings.Count(got, "@kotlin.jvm.JvmName(") != 3 {
		t.Fatal("missing actual helper/factory annotations", got)
	}
	for _, line := range strings.Split(source, "\n") {
		if !strings.Contains(got, line) {
			t.Fatal("changed Kotlin name/call/value/comment", line, got)
		}
	}
	if !strings.Contains(got, "private fun <T : kotlin.Comparable<T>> `openapi-kt uniqueItems`") {
		t.Fatal("generic helper lost Kotlin name", got)
	}
	if jvmFunctionNames(source) != got {
		t.Fatal("nondeterministic JVM names")
	}
	if jvmFunctionNames(got) != got {
		t.Fatal("JVM name annotation pass is not idempotent")
	}
}

func FuzzJVMNamesPreserveUserLiteral(f *testing.F) {
	f.Add("fun `fake name`( /* nested */ \"\"\" and $value")
	f.Add("雪 🍃 \n quote \\\" /* fun `again fake`( */")
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 {
			t.Skip()
		}
		literal := naming.Literal(value)
		source := "private val payload = " + literal + "\nprivate fun `openapi-kt synthetic helper`(value: kotlin.String) = value\n"
		got := jvmFunctionNames(source)
		if !strings.Contains(got, "private val payload = "+literal+"\n") || strings.Count(got, "@kotlin.jvm.JvmName(") != 1 || jvmFunctionNames(got) != got || jvmFunctionNames(source) != got {
			t.Fatal("JVM compatibility pass changed data or drifted", got)
		}
	})
}

func TestJVMFunctionNamesLeaveOrdinarySourcesIdentical(t *testing.T) {
	for _, source := range []string{"public data class Item(val value: kotlin.String)\n", "public fun `class`() = \"valid\"\n", "// fun `fake name`(\nprivate val source = \"data\"\n"} {
		if got := jvmFunctionNames(source); got != source {
			t.Fatal("unaffected source changed", got)
		}
	}
}

func TestJVMFunctionNamesAvoidExistingIdentifiers(t *testing.T) {
	name := "openapi-kt uniqueItems"
	sum := sha256.Sum256([]byte(name))
	base := "openapiKtJvm_" + hex.EncodeToString(sum[:])
	source := "public fun " + base + "(value: kotlin.String) = value\nprivate fun `" + name + "`(value: kotlin.String) = value\n"
	got := jvmFunctionNames(source)
	if !strings.Contains(got, `@kotlin.jvm.JvmName("`+base+`_2")`) || !strings.Contains(got, "public fun "+base+"(") {
		t.Fatal("ASCII JVM helper stole an existing API identifier", got)
	}
}
