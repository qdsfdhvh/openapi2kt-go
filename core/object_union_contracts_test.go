package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestObjectUnionsCommonOwnDefaultsDispatchAndSingletonNames(t *testing.T) {
	for _, suffix := range []string{"", "Dto"} {
		r := report(t, semanticFixture(t, "object-unions"), suffix)
		s := source(t, r, "Envelope"+suffix)
		for _, part := range []string{"public sealed class Tagged {", "public data class Note(", "public abstract val kind: kotlin.String", "public abstract val count: kotlin.Int?", "public override val count: kotlin.Int?,", "public override val count: kotlin.Int? = null,", "public data object Idle", "public data object Ready", "public data class UnknownTagged(", "public val rawElement: kotlinx.serialization.json.JsonElement", "encodeJsonElement(value.rawElement)", `tag == "batch-more"`, `"body" in obj`, "public data object Dicecheck", "public data object V1", "public data class Companion1(", "public data class Event2(", "Same2"} {
			if !strings.Contains(s, part) {
				t.Fatal("missing", part)
			}
		}
		if strings.Index(s, `tag == "fixed"`) > strings.Index(s, `"body" in obj`) {
			t.Fatal("dispatch priority")
		}
		unreachable := 0
		for _, w := range r.Warnings {
			if w.Code == "W_UNION_UNREACHABLE_VARIANT" {
				unreachable++
			}
		}
		if unreachable != 4 {
			t.Fatal(r.Warnings)
		}
	}
}

func TestObjectUnionSourceValidationReachabilityAndDeferredContexts(t *testing.T) {
	for _, tc := range []struct {
		branches    string
		unreachable int
	}{
		{`{"type":"object","properties":{"left":{"type":"string"}},"required":["left"]},{"type":"object","properties":{"right":{"type":"string"}},"required":["right"]}`, 0},
		{`{"type":"object","properties":{"left":{"type":"string"}},"required":["left"]},{"type":"object","properties":{"left":{"type":"string"},"right":{"type":"string"}},"required":["right"]}`, 1},
		{`{"type":"object","properties":{"left":{"type":"string"}}},{"type":"object","properties":{"right":{"type":"string"}}}`, 2},
	} {
		r := report(t, fieldDoc(`{"oneOf":[`+tc.branches+`]}`), "")
		count := 0
		for _, w := range r.Warnings {
			if w.Code == "W_UNION_UNREACHABLE_VARIANT" {
				count++
			}
		}
		if count != tc.unreachable {
			t.Fatal(r.Warnings)
		}
	}
	for _, tc := range []struct{ field, code, tail string }{
		{`{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"a"},"bad":{"type":"string","pattern":"x"}}},{"type":"object"}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/0/properties/bad/pattern"},
		{`{"oneOf":[{"type":"object"},{"type":"object"}],"discriminator":{"propertyName":"kind"}}`, "E_SCHEMA_UNSUPPORTED", "discriminator"},
		{`{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"serializer"}}},{"type":"object","properties":{"kind":{"type":"string","const":"plain"}}}]}`, "E_NAME_COLLISION", "oneOf/0"},
		{`{"oneOf":[{"type":"object","properties":{"rawElement":{"type":"string"}}},{"type":"object","properties":{"rawElement":{"type":"string"}}}]}`, "E_NAME_COLLISION", "oneOf/0/properties/rawElement"},
		{`{"oneOf":[{"allOf":[{"type":"object","nullable":true,"properties":{"kind":{"type":"string","const":"a"}}}]},{"type":"object"}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/0"},
		{`{"oneOf":[{"type":"object","properties":{"kind":{"type":"boolean","enum":[true]}}},{"type":"object","properties":{"kind":{"type":"boolean","enum":[false]}}}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/0/properties/kind"},
	} {
		failure(t, fieldDoc(tc.field), tc.code, "/components/schemas/Record/properties/value/"+tc.tail)
	}
	// The original inline common-object rejection now preserves two actual
	// variant-owned declarations; it never fabricates an abstract classifier.
	original := `{"oneOf":[{"type":"object","properties":{"nested":{"type":"object","properties":{"v":{"type":"string"}}}}},{"type":"object","properties":{"nested":{"type":"object","properties":{"v":{"type":"string"}}}}}]}`
	graduated := report(t, fieldDoc(original), "")
	if text := source(t, graduated, "Record"); strings.Count(text, "public val nested: Nested? = null,") != 2 || strings.Contains(text, "abstract val nested:") {
		t.Fatal("original inline union input lost lexical owners", text)
	}
	failure(t, schemas(`{"Record":{"type":"object","properties":{"value":{"oneOf":[{"type":"object"},{"type":"object"}]},"unknownValue":{"type":"object","properties":{"id":{"type":"string"}}}}}}`), "E_NAME_COLLISION", "/components/schemas/Record/properties/unknownValue")
}

func TestObjectUnionDepthAndSerializedSlots(t *testing.T) {
	for _, count := range []int{240, 241} {
		var branches []string
		for i := range count {
			branches = append(branches, fmt.Sprintf(`{"type":"object","properties":{"kind":{"type":"string","const":"v%d"}}}`, i))
		}
		_, err := core.Generate(request(fieldDoc(`{"oneOf":[`+strings.Join(branches, ",")+`]}`), ""))
		if (err == nil) != (count == 240) || err != nil && err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" {
			t.Fatal(count, err)
		}
	}
	for _, count := range []int{45, 46} {
		field := `{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"a"}}},{"type":"object","properties":{"kind":{"type":"string","const":"b"}}}]}`
		for i := range count {
			field = `{"type":"object","properties":{"level` + fmt.Sprint(i) + `":` + field + `}}`
		}
		_, err := core.Generate(request(fieldDoc(field), ""))
		if (err == nil) != (count == 45) || err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
			t.Fatal(count, err)
		}
	}
	for _, last := range []string{`{"type":"boolean"}`, `{"type":"integer","format":"int64"}`} {
		var props, required []string
		for i := range 119 {
			name := fmt.Sprintf("field%d", i)
			props = append(props, fmt.Sprintf(`%q:{"type":"integer","format":"int64"}`, name))
			required = append(required, fmt.Sprintf("%q", name))
		}
		props = append(props, `"kind":{"type":"string","const":"wide"}`, `"last":`+last)
		required = append(required, `"kind"`, `"last"`)
		field := `{"oneOf":[{"type":"object","properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]},{"type":"object"}]}`
		_, err := core.Generate(request(fieldDoc(field), ""))
		if (err == nil) != strings.Contains(last, "boolean") || err != nil && err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
			t.Fatal(last, err)
		}
	}
}

func TestObjectUnionDiscriminatorSpellingKeepsFloatAndFullUpperIdentity(t *testing.T) {
	for _, tc := range []struct{ literal, name, tag string }{
		{`1e6`, "V10000000", "1000000.0"}, {`1e15`, "V10000000000000000", "1000000000000000.0"},
		{`1e16`, "V1e16", "1e+16"}, {`1e-5`, "V000001", "0.00001"}, {`1e-6`, "V1e6", "1e-6"},
		{`-0`, "V00", "-0.0"},
	} {
		field := `{"oneOf":[{"type":"object","properties":{"kind":{"type":"number","const":` + tc.literal + `}}},{"type":"object","properties":{"kind":{"type":"number","const":3.0}}}]}`
		s := source(t, report(t, fieldDoc(field), ""), "Record")
		if !strings.Contains(s, "public data object "+tc.name+" ") || !strings.Contains(s, `tag == "`+tc.tag+`"`) {
			t.Fatal(tc, s)
		}
	}
	for _, tc := range []struct{ wire, name string }{{"ßfoo", "SSfoo"}, {"ﬃ-name", "FFIName"}, {"ŉever", "Never"}, {"ǰtag", "Jtag"}} {
		s := source(t, report(t, fieldDoc(fmt.Sprintf(`{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":%q}}},{"type":"object","properties":{"kind":{"type":"string","const":"other"}}}]}`, tc.wire)), ""), "Record")
		if !strings.Contains(s, "public data object "+tc.name+" ") {
			t.Fatal(tc, s)
		}
	}
}

func TestObjectRootsAndRootKindsUseContextOwnedBackends(t *testing.T) {
	for _, fixture := range []string{"root-unions", "root-kind-unions", "scalar-projections"} {
		for _, suffix := range []string{"", "Dto"} {
			_ = report(t, semanticFixture(t, fixture), suffix)
		}
	}
	r := report(t, semanticFixture(t, "root-unions"), "")
	if strings.Contains(source(t, r, "Untyped"), "UnknownUntyped") {
		t.Fatal("closed root acquired Unknown")
	}
	failure(t, schemas(`{"Root":{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"fixed"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf")
	// Closed roots do not reserve nonexistent serializer/Unknown peers.
	_ = report(t, schemas(`{"Root":{"oneOf":[{"type":"object","properties":{"name":{"type":"string"}}}]},"UnknownRoot":{"type":"object"},"RootSerializer":{"type":"object"}}`), "")
}
