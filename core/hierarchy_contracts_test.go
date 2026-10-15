package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTypedHierarchyTopLevelAncestorEnumsUnknownsAndBackendPolicies(t *testing.T) {
	for _, suffix := range []string{"", "Dto"} {
		r := report(t, semanticFixture(t, "hierarchy-unions"), suffix)
		root := source(t, r, "Signal"+suffix)
		for _, part := range []string{"public sealed class Signal" + suffix, "public sealed class Signal" + suffix + "Family", "public abstract val family: org.example.models.models.Signal" + suffix + "Family", "UnknownSignal" + suffix} {
			if !strings.Contains(root, part) {
				t.Fatal("missing", part)
			}
		}
		if strings.Contains(root, "public data class Status(") {
			t.Fatal("hierarchy leaf was duplicated/nested")
		}
		channel := source(t, r, "Channel"+suffix)
		for _, part := range []string{"public sealed class Channel" + suffix + " : org.example.models.models.Signal" + suffix + "()", "public abstract override val family: org.example.models.models.Signal" + suffix + "Family", "public sealed class Channel" + suffix + "Phase", "UnknownChannel" + suffix, `.Unrecognized("")`, `"noon"`} {
			if !strings.Contains(channel, part) {
				t.Fatal("missing", part)
			}
		}
		if strings.Contains(channel, `"alias-only"`) {
			t.Fatal("mapping alias became an extra route")
		}
		flux := source(t, r, "Flux"+suffix)
		if !strings.Contains(flux, "public override val phase: org.example.models.models.Channel"+suffix+"Phase,") || strings.Contains(flux, "Channel"+suffix+"Phase?") || !strings.Contains(flux, "kotlin.collections.List<kotlin.Long>") {
			t.Fatal(flux)
		}
		quiet := source(t, r, "Quiet"+suffix)
		if !strings.Contains(quiet, "public data object Quiet"+suffix+" : org.example.models.models.Channel"+suffix+"()") || !strings.Contains(quiet, "decodeJsonElement()") {
			t.Fatal(quiet)
		}
		inferred := source(t, r, "InferredGroup"+suffix)
		if !strings.Contains(inferred, "public sealed class InferredGroup"+suffix+"Zone") || strings.Index(inferred, "PinLeaf"+suffix+".serializer()") > strings.Index(inferred, "RangeLeaf"+suffix+".serializer()") {
			t.Fatal(inferred)
		}
		plain := source(t, r, "PlainGroup"+suffix)
		if strings.Contains(plain, "UnknownPlainGroup") || strings.Contains(plain, "with = PlainGroup") {
			t.Fatal("default hierarchy became content dispatch")
		}
		if !strings.Contains(source(t, r, "PlainLeft"+suffix), `@kotlinx.serialization.SerialName("plainleft")`) {
			t.Fatal("fallback suffix entered wire name")
		}
		if !reflect.DeepEqual(r, report(t, semanticFixture(t, "hierarchy-unions"), suffix)) {
			t.Fatal("non-deterministic graph report")
		}
	}
}

func TestHierarchyFlatAllOfDirectFieldMappingsEscapedAndUnicodeWireFallback(t *testing.T) {
	input := schemas(`{"Base":{"type":"object","required":["tag"],"properties":{"tag":{"type":"string"}}},"AlphaNode":{"allOf":[{"$ref":"#/components/schemas/Base"},{"type":"object","required":["label"],"properties":{"label":{"type":"string"}}}]},"BetaNode":{"allOf":[{"$ref":"#/components/schemas/Base"},{"type":"object","required":["count"],"properties":{"count":{"type":"integer"}}}]},"Flat":{"oneOf":[{"$ref":"#/components/schemas/AlphaNode"},{"$ref":"#/components/schemas/BetaNode"}],"discriminator":{"propertyName":"tag","mapping":{"a/b~wire":"#/components/schemas/AlphaNode","ignored-alias":"#/components/schemas/AlphaNode"}}}}`)
	r := report(t, input, "Dto")
	s := source(t, r, "FlatDto")
	if !strings.Contains(s, `"a/b~wire"`) || !strings.Contains(s, `"betanode"`) || strings.Contains(s, `"BetaNodeDto"`) || strings.Contains(s, `"ignored-alias"`) {
		t.Fatal(s)
	}
	alpha := source(t, r, "AlphaNodeDto")
	if !strings.Contains(alpha, "public override val tag: org.example.models.models.FlatDtoTag =") || !strings.Contains(alpha, "public val label: kotlin.String,") || strings.Contains(alpha, "val tag: kotlin.String") {
		t.Fatal(alpha)
	}
	_ = report(t, schemas(`{"Leaf/a~b":{"type":"object","properties":{"tag":{"type":"string","const":"one"}}},"Other":{"type":"object","properties":{"tag":{"type":"string","const":"two"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf~1a~0b"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"one":"#/components/schemas/Leaf~1a~0b","two":"Other"}}}}`), "")
	for _, tc := range []struct{ wire, lower string }{{"CODEΣ", "codeς"}, {"ΣCODE", "σcode"}, {"CODEΣTail", "codeσtail"}, {"İNode", "i\u0307node"}, {"CODEΣ\u0301", "codeς\u0301"}, {"CODEΣ'Tail", "codeσ'tail"}} {
		key, _ := json.Marshal(tc.wire)
		ref, _ := json.Marshal("#/components/schemas/" + tc.wire)
		s := source(t, report(t, schemas(`{`+string(key)+`:{"type":"object","properties":{"tag":{"type":"string"}}},"Other":{"type":"object","properties":{"tag":{"type":"string"}}},"Root":{"oneOf":[{"$ref":`+string(ref)+`},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}}}`), "Dto"), "RootDto")
		if !strings.Contains(s, `tag == "`+tc.lower+`"`) {
			t.Fatal(tc, s)
		}
	}
}

func TestHierarchyConstValuesDoNotAddImplicitExplicitDiscriminatorRoutes(t *testing.T) {
	input := schemas(`{"First":{"type":"object","properties":{"tag":{"type":"string","enum":["one"],"nullable":true}}},"Second":{"type":"object","properties":{"tag":{"type":"string","const":"two"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"tag"}}}`)
	s := source(t, report(t, input, "Dto"), "RootDto")
	if !strings.Contains(s, `tag == "second"`) || !strings.Contains(s, `tag == "one"`) || strings.Contains(s, `tag == "two"`) || strings.Contains(s, `tag == "SecondDto"`) {
		t.Fatal("fallback/enum/const route policy", s)
	}
}

func TestHierarchyPreciseGraphMappingOwnershipAndReservedDiagnostics(t *testing.T) {
	for _, tc := range []struct{ body, code, pointer string }{
		{`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Missing"}],"discriminator":{"propertyName":"tag"}}}`, "E_REF_MISSING", "/components/schemas/Root/oneOf/0/$ref"},
		{`{"A":{"oneOf":[{"$ref":"#/components/schemas/B"}],"discriminator":{"propertyName":"tag"}},"B":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`, "E_REF_CYCLE", "/components/schemas/B/oneOf/0/$ref"},
		{`{"Leaf":{"type":"object","properties":{"tag":{"type":"string"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf/1/$ref"},
		{`{"Root":{"oneOf":[{"type":"object"}],"discriminator":{"propertyName":"tag"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf/0/type"},
		{`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf","nullable":true}],"discriminator":{"propertyName":"tag"}},"Leaf":{"type":"object"}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf/0/nullable"},
		{`{"Leaf":{"type":"object","properties":{"tag":{"type":"string","const":"one"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":false}}}`, "E_SCHEMA_INVALID", "/components/schemas/Root/discriminator/propertyName"},
		{`{"Leaf":{"type":"object"},"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag","mapping":{"a":17}}}}`, "E_SCHEMA_INVALID", "/components/schemas/Root/discriminator/mapping/a"},
		{`{"Leaf":{"type":"object"},"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"rawElement"}}}`, "E_NAME_COLLISION", "/components/schemas/Root/discriminator/propertyName"},
		{`{"One":{"type":"object","properties":{"tag":{"type":"string","const":"same"}}},"Two":{"type":"object","properties":{"tag":{"type":"string","const":"same"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/One"},{"$ref":"#/components/schemas/Two"}],"discriminator":{"propertyName":"tag"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root"},
		{`{"One":{"type":"object","properties":{"tag":{"type":"integer"}}},"Two":{"type":"object","properties":{"tag":{"type":"integer"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/One"},{"$ref":"#/components/schemas/Two"}],"discriminator":{"propertyName":"tag"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/One/properties/tag"},
	} {
		if tc.pointer == "/components/schemas/Root/oneOf/0/nullable" {
			// The unchanged historical branch input now carries parent nullable
			// intent while preserving its physical mapped singleton leaf.
			r := report(t, schemas(tc.body), "Dto")
			if !strings.Contains(source(t, r, "RootDto"), "public sealed class RootDto {") || !strings.Contains(source(t, r, "LeafDto"), "public data object LeafDto : org.example.models.models.RootDto() {") {
				t.Fatal("nullable member physical identity")
			}
			continue
		}
		failure(t, schemas(tc.body), tc.code, tc.pointer)
	}
	var fixture map[string]any
	if err := json.Unmarshal(semanticFixture(t, "hierarchy-unions"), &fixture); err != nil {
		t.Fatal(err)
	}
	components := fixture["components"].(map[string]any)["schemas"].(map[string]any)
	for _, peer := range []string{"UnknownSignal", "SignalFamily"} {
		components[peer] = map[string]any{"type": "object"}
		input, _ := json.Marshal(fixture)
		failure(t, input, "E_NAME_COLLISION", "/components/schemas/Signal")
		delete(components, peer)
	}
	components["Signal"].(map[string]any)["example"] = map[string]any{"discriminator": 17, "oneOf": false, "$ref": "missing"}
	input, _ := json.Marshal(fixture)
	_ = report(t, input, "")
	components["PlainLeft"] = map[string]any{"type": "object", "properties": map[string]any{"family": map[string]any{"type": "string", "const": "plain"}}}
	input, _ = json.Marshal(fixture)
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/PlainLeft")
}

func TestHierarchyGraphVisitWorkBudgetBeforeFixedOwnContexts(t *testing.T) {
	var nodes []string
	for i := range 8193 {
		nodes = append(nodes, fmt.Sprintf(`"Root%05d":{"oneOf":[{"$ref":"#/components/schemas/Leaf%05d"}],"discriminator":{"propertyName":"tag"}}`, i, i), fmt.Sprintf(`"Leaf%05d":{"type":"object","properties":{"tag":{"type":"string"}}}`, i))
	}
	failure(t, schemas(`{`+strings.Join(nodes, ",")+`}`), "E_SCHEMA_LIMIT", "/components/schemas/Root08192")
}

func TestHierarchyCachedHeightBranchEnumAndTypedSlotsLimits(t *testing.T) {
	var nodes []string
	for i := range 47 {
		nodes = append(nodes, fmt.Sprintf(`"A%02d":{"oneOf":[{"$ref":"#/components/schemas/A%02d"}],"discriminator":{"propertyName":"tag"}}`, i, i+1))
	}
	nodes = append(nodes, `"A47":{"type":"object","properties":{"tag":{"type":"string","const":"leaf"}}}`, `"ZParent":{"oneOf":[{"$ref":"#/components/schemas/A00"}],"discriminator":{"propertyName":"tag"}}`)
	failure(t, schemas(`{`+strings.Join(nodes, ",")+`}`), "E_SCHEMA_DEPTH", "/components/schemas/ZParent/oneOf/0/$ref")
	branches := make([]string, 241)
	for i := range branches {
		branches[i] = `{"$ref":"#/components/schemas/Leaf"}`
	}
	failure(t, schemas(`{"Root":{"oneOf":[`+strings.Join(branches, ",")+`],"discriminator":{"propertyName":"tag"}},"Leaf":{"type":"object"}}`), "E_UNION_LIMIT", "/components/schemas/Root/oneOf")
	for _, vocabularySize := range []int{240, 241} {
		vocabulary := make([]string, vocabularySize)
		for i := range vocabulary {
			vocabulary[i] = fmt.Sprintf("value_%d", i)
		}
		first, _ := json.Marshal(vocabulary[:120])
		second, _ := json.Marshal(vocabulary[120:])
		input := schemas(`{"First":{"type":"object","properties":{"tag":{"enum":` + string(first) + `}}},"Second":{"type":"object","properties":{"tag":{"enum":` + string(second) + `}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"tag"}}}`)
		_, err := core.Generate(request(input, ""))
		if (err == nil) != (vocabularySize == 240) || err != nil && err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" {
			t.Fatal(vocabularySize, err)
		}
	}
	for _, count := range []int{119, 120} {
		fields, required := []string{`"tag":{"type":"string","const":"wide"}`, `"flag":{"type":"boolean"}`}, []string{`"tag"`, `"flag"`}
		for i := range count {
			name := fmt.Sprintf("field%d", i)
			fields = append(fields, fmt.Sprintf(`%q:{"type":"integer","format":"int64"}`, name))
			required = append(required, fmt.Sprintf("%q", name))
		}
		input := schemas(`{"Wide":{"type":"object","properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]},"Small":{"type":"object","properties":{"tag":{"type":"string","const":"small"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Wide"},{"$ref":"#/components/schemas/Small"}],"discriminator":{"propertyName":"tag","mapping":{"wide":"Wide","small":"Small"}}}}`)
		_, err := core.Generate(request(input, ""))
		if (err == nil) != (count == 119) || err != nil && err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
			t.Fatal(count, err)
		}
	}
}
