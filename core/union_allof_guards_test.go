package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionAllOfOriginalSourcesAndCycles(t *testing.T) {
	choice := `{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"a"},"bad":{"type":"string","format":"unsupported"}}},{"type":"object","properties":{"kind":{"type":"string","const":"b"}}}]}`
	failure(t, schemas(`{"Root":{"allOf":[`+choice+`,{"type":"object","properties":{"bad":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/allOf/0/oneOf/0/properties/bad/format")
	failure(t, schemas(`{"Root":{"allOf":[{"$ref":"#/components/schemas/Root"},{"oneOf":[{"type":"object"}]}]}}`), "E_REF_CYCLE", "/components/schemas/Root/allOf/0/$ref")
	failure(t, schemas(`{"Root":{"allOf":[{"oneOf":[{"type":"object"},{"type":"string"}]},{"type":"object"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/allOf/0/oneOf/1/type")
	failure(t, schemas(`{"Root":{"allOf":[{"oneOf":[{"type":"object"}]},{"type":"object","minProperties":1}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/allOf/1/minProperties")
}

func TestUnionAllOfCartesianBranchLimits(t *testing.T) {
	for _, n := range []int{15, 16} {
		left, right := []string{}, []string{}
		for i := 0; i < n; i++ {
			left = append(left, fmt.Sprintf(`{"type":"object","properties":{"kind":{"type":"string","const":"k%d"}}}`, i))
		}
		for i := 0; i < 16; i++ {
			right = append(right, fmt.Sprintf(`{"type":"object","properties":{"mode":{"type":"string","const":"m%d"}}}`, i))
		}
		input := schemas(`{"Root":{"allOf":[{"oneOf":[` + strings.Join(left, ",") + `]},{"anyOf":[` + strings.Join(right, ",") + `]}]}}`)
		r := request(input, "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if n == 15 {
			if err != nil {
				t.Fatal(err)
			}
			src := source(t, got, "models/RootDto")
			if strings.Count(src, " && ") != 240 {
				t.Fatal("missing combinations")
			}
		} else if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || !strings.Contains(err.Error(), "E_UNION_LIMIT") {
			t.Fatal(got, err)
		}
	}
}

func TestUnionAllOfDepthAndModelSlots(t *testing.T) {
	for _, levels := range []int{45, 46} {
		value := `{"allOf":[{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"a"}}},{"type":"object","properties":{"kind":{"type":"string","const":"b"}}}]}]}`
		for i := 0; i < levels; i++ {
			value = fmt.Sprintf(`{"type":"object","properties":{"level%d":%s}}`, i, value)
		}
		_, err := core.Generate(request(schemas(`{"Root":`+value+`}`), "Dto"))
		if (err == nil) != (levels == 45) {
			t.Fatal(levels, err)
		}
	}
	for _, fields := range []int{239, 240} {
		var props, required []string
		for i := 0; i < fields; i++ {
			props = append(props, fmt.Sprintf(`"p%d":{"type":"boolean"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		value := `{"allOf":[{"type":"object","properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]},{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"a"}}},{"type":"object","properties":{"kind":{"type":"string","const":"b"}}}]}]}`
		_, err := core.Generate(request(schemas(`{"Root":`+value+`}`), "Dto"))
		if (err == nil) != (fields == 239) {
			t.Fatal(fields, err)
		}
	}
}

func TestUnionAllOfOwnOverridesAndRequiredOrder(t *testing.T) {
	input := schemas(`{"Root":{"allOf":[{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}},{"oneOf":[{"type":"object","properties":{"kind":{"type":"string","const":"a"},"id":{"type":"integer"}}},{"type":"object","properties":{"kind":{"type":"string","const":"b"},"id":{"type":"integer"}}}]}],"properties":{"id":{"type":"boolean"}}}}`)
	s := source(t, report(t, input, "Dto"), "models/RootDto")
	if !strings.Contains(s, "public override val id: kotlin.Boolean,") || strings.Contains(s, "id: kotlin.Boolean? = null") {
		t.Fatal(s)
	}
	if !strings.Contains(s, `obj["kind"]?.jsonPrimitive?.content == "a"`) {
		t.Fatal(s)
	}
}
