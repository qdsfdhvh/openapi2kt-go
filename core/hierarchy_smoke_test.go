package core_test

import "fmt"

func exportHierarchyResources(generate func([]byte, string, string, string), marshal func(map[string]any) []byte) {
	fields := map[string]any{"tag": map[string]any{"type": "string", "const": "wide"}, "flag": map[string]any{"type": "boolean"}}
	names := []string{"tag", "flag"}
	for i := range 119 {
		name := fmt.Sprintf("field%d", i)
		fields[name] = map[string]any{"type": "integer", "format": "int64"}
		names = append(names, name)
	}
	generate(marshal(map[string]any{
		"WideLeaf":  map[string]any{"type": "object", "properties": fields, "required": names},
		"SmallLeaf": map[string]any{"type": "object", "properties": map[string]any{"tag": map[string]any{"type": "string", "const": "small"}}},
		"WideRoot":  map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/WideLeaf"}, map[string]any{"$ref": "#/components/schemas/SmallLeaf"}}, "discriminator": map[string]any{"propertyName": "tag", "mapping": map[string]any{"wide": "WideLeaf", "small": "SmallLeaf"}}},
	}), "org.example.hierarchy_limits", "", "hierarchy-limits")
	vocabulary := make([]string, 240)
	for i := range vocabulary {
		vocabulary[i] = fmt.Sprintf("token_%d", i)
	}
	generate(marshal(map[string]any{
		"EnumFirst":    map[string]any{"type": "object", "properties": map[string]any{"tag": map[string]any{"enum": vocabulary[:120]}}},
		"EnumSecond":   map[string]any{"type": "object", "properties": map[string]any{"tag": map[string]any{"enum": vocabulary[120:]}}},
		"EnumRoot":     map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/EnumFirst"}, map[string]any{"$ref": "#/components/schemas/EnumSecond"}}, "discriminator": map[string]any{"propertyName": "tag"}},
		"SingleFirst":  map[string]any{"type": "object", "properties": map[string]any{"tag": map[string]any{"type": "string", "enum": []string{"one"}, "nullable": true}}},
		"SingleSecond": map[string]any{"type": "object", "properties": map[string]any{"tag": map[string]any{"type": "string", "const": "two"}}},
		"SingleRoot":   map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/SingleFirst"}, map[string]any{"$ref": "#/components/schemas/SingleSecond"}}, "discriminator": map[string]any{"propertyName": "tag", "mapping": map[string]any{"one": "SingleFirst", "two": "SingleSecond"}}},
	}), "org.example.hierarchy_port", "", "hierarchy-enum-constructor-boundaries")
}

const hierarchyPortRuntime = `package org.example.contracts

import kotlinx.serialization.json.*
import org.example.hierarchy_port.models.*

fun main() {
    check(EnumRootTag.entries.size == 240)
    check(EnumRootTag.entries.last() === EnumRootTag.TOKEN_239)
    check(EnumRootTag.fromSerialName("token_239") === EnumRootTag.TOKEN_239)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = EnumSecond(tag = EnumRootTag.TOKEN_239)
        val wire = json.encodeToJsonElement(EnumRoot.serializer(), value)
        check(wire == buildJsonObject { put("tag", "token_239") })
        check(json.decodeFromJsonElement(EnumRoot.serializer(), wire) == value)
        val single = SingleFirst(tag = SingleRootTag.ONE)
        check(single.copy(tag = SingleRootTag.ONE) == single)
        val singleWire = json.encodeToJsonElement(SingleRoot.serializer(), single)
        check(singleWire == buildJsonObject { put("tag", "one") })
        check(json.decodeFromJsonElement(SingleRoot.serializer(), singleWire) == single)
        check(runCatching { json.decodeFromString(SingleFirst.serializer(), "{}") }.isFailure)
        check(runCatching { json.decodeFromString(SingleFirst.serializer(), """{"tag":null}""") }.isFailure)
        val raw = buildJsonObject { put("tag", "future"); put("extra", buildJsonArray { add(true); add(JsonNull) }) }
        val unknown = json.decodeFromJsonElement(SingleRoot.serializer(), raw) as UnknownSingleRoot
        check(unknown.tag == SingleRootTag.Unrecognized(""))
        check(json.encodeToJsonElement(SingleRoot.serializer(), unknown) == raw)
        check(json.decodeFromString(SingleRoot.serializer(), """{"tag":"two","ignored":true}""") === SingleSecond)
        check(json.encodeToJsonElement(SingleRoot.serializer(), SingleSecond) == buildJsonObject { put("tag", "two") })
    }
    println("Go hierarchy 240-vocabulary/single nullable enum required constructor/raw Unknown/fixed singleton contracts passed")
}
`
