package core_test

import (
	"fmt"
	"math"
)

func exportObjectResources(generate func([]byte, string, string, string), marshal func(map[string]any) []byte) {
	fields := map[string]any{"kind": map[string]any{"type": "string", "const": "wide"}, "flag": map[string]any{"type": "boolean"}}
	names := []string{"kind", "flag"}
	for i := range 119 {
		name := fmt.Sprintf("field%d", i)
		fields[name] = map[string]any{"type": "integer", "format": "int64"}
		names = append(names, name)
	}
	generate(marshal(map[string]any{"WideEnvelope": map[string]any{"type": "object", "properties": map[string]any{"event": map[string]any{"oneOf": []any{map[string]any{"type": "object", "properties": fields, "required": names}, map[string]any{"type": "object", "properties": map[string]any{}}}}}}}), "org.example.object_union_limits", "", "object-union-limits")
	wide := map[string]any{}
	required := []string{}
	carrier := map[string]any{"root": map[string]any{"$ref": "#/components/schemas/WideRoot"}, "flag": map[string]any{"type": "boolean"}}
	carrierNames := []string{"root", "flag"}
	for i := range 120 {
		name := fmt.Sprintf("field%d", i)
		wide[name] = map[string]any{"type": "integer", "format": "int64"}
		required = append(required, name)
		if i < 119 {
			carrier[name] = wide[name]
			carrierNames = append(carrierNames, name)
		}
	}
	generate(marshal(map[string]any{
		"WideRoot":    map[string]any{"oneOf": []any{map[string]any{"type": "object", "properties": wide, "required": required}, map[string]any{"type": "boolean"}}},
		"WideCarrier": map[string]any{"type": "object", "properties": carrier, "required": carrierNames},
	}), "org.example.root_kind_limits", "", "root-kind-limits")
	enums, constants := map[string]any{}, map[string]any{}
	for i := range 120 {
		name := fmt.Sprintf("field%d", i)
		enums[name] = map[string]any{"$ref": "#/components/schemas/NativeLong"}
		constants[name] = map[string]any{"$ref": "#/components/schemas/PinnedLong"}
	}
	generate(marshal(map[string]any{
		"NativeLong":   map[string]any{"type": "integer", "format": "int64", "enum": []int64{math.MinInt64, math.MaxInt64}},
		"PinnedLong":   map[string]any{"type": "integer", "format": "int64", "const": int64(math.MinInt64)},
		"EnumWide":     map[string]any{"type": "object", "properties": enums, "required": required},
		"ConstRefWide": map[string]any{"type": "object", "properties": constants},
	}), "org.example.scalar_projection_limits", "", "scalar-projection-limits")
}
