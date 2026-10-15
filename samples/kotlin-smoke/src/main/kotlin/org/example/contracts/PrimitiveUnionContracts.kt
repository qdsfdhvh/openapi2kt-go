@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.*
import org.example.primitive_union_combinations.models.CombinationEnvelope
import org.example.primitive_unions.models.Record
import org.example.primitive_unions_suffix.models.RecordDto

private fun checkAllKindCombinations(json: Json) {
    val accessors: Map<Int, (CombinationEnvelope) -> Any?> = mapOf(
        3 to { it.case3 }, 5 to { it.case5 }, 6 to { it.case6 }, 7 to { it.case7 },
        9 to { it.case9 }, 10 to { it.case10 }, 11 to { it.case11 }, 12 to { it.case12 },
        13 to { it.case13 }, 14 to { it.case14 }, 15 to { it.case15 },
    )
    for ((mask, accessor) in accessors) {
        for ((wire, prefix) in listOf(
            "true" to if (mask and 8 != 0) "Boolean" else null,
            "1" to if (mask and 2 != 0) "Int" else if (mask and 4 != 0) "Number" else null,
            "1.5" to if (mask and 4 != 0) "Number" else null,
            "\"true\"" to if (mask and 1 != 0) "String" else null,
            "\"1\"" to if (mask and 1 != 0) "String" else null,
            "[]" to null, "{}" to null,
        )) {
            val field = "case$mask"
            val tree = buildJsonObject { put(field, json.parseToJsonElement(wire)) }
            val decoded = runCatching { json.decodeFromJsonElement(CombinationEnvelope.serializer(), tree) }
            if (prefix == null) check(decoded.exceptionOrNull() is SerializationException) else {
                val model = decoded.getOrThrow()
                check(accessor(model)!!.javaClass.simpleName == "${prefix}Case$mask")
                val output = json.encodeToJsonElement(CombinationEnvelope.serializer(), model).jsonObject.getValue(field)
                if (wire == "1" && prefix == "Number") check(output.jsonPrimitive.double == 1.0)
                else check(output == json.parseToJsonElement(wire))
            }
        }
    }
}

fun main() {
    check(Record.IntValue.serializer().descriptor.serialName == "org.example.primitive_unions.models.Record.IntValue")
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        checkAllKindCombinations(json)
        check(json.decodeFromString(Record.Value.serializer(), "true") == Record.BooleanValue(true))
        check(json.decodeFromString(Record.Value.serializer(), "1") == Record.IntValue(1L))
        check(json.decodeFromString(Record.Value.serializer(), "1.5") == Record.NumberValue(1.5))
        check(json.decodeFromString(Record.Value.serializer(), "\"true\"") == Record.StringValue("true"))
        check(json.decodeFromString(Record.Value.serializer(), "\"1\"") == Record.StringValue("1"))
        check(json.decodeFromString(Record.Value.serializer(), "9223372036854775808") is Record.NumberValue)
        check(json.encodeToString(Record.Value.serializer(), Record.IntValue(Long.MAX_VALUE)) == Long.MAX_VALUE.toString())
        check(json.encodeToString(Record.Value.serializer(), Record.BooleanValue(false)) == "false")
        check(json.encodeToString(Record.Value.serializer(), Record.StringValue("quoted")) == "\"quoted\"")
        check(json.decodeFromString(Record.Binary.serializer(), "true") == Record.BooleanBinary(true))
        check(json.decodeFromString(Record.Binary.serializer(), "7") == Record.IntBinary(7L))
        for (wire in listOf("[]", "{}", "null")) {
            check(runCatching { json.decodeFromString(Record.Value.serializer(), wire) }.exceptionOrNull() is SerializationException)
        }
        for (wire in listOf("1.5", "\"true\"", "\"1\"", "9223372036854775808")) {
            check(runCatching { json.decodeFromString(Record.Binary.serializer(), wire) }.exceptionOrNull() is SerializationException)
        }
        val mixed = json.decodeFromString(Record.Mixed.serializer(), "{\"id\":\"object\"}") as Record.ObjectMixed
        check(mixed.id == "object" && mixed.tag == "object")
        check(json.decodeFromString(Record.Mixed.serializer(), "\"reference\"") == Record.StringMixed("reference"))
        check(runCatching { json.decodeFromString(Record.Mixed.serializer(), "1") }.exceptionOrNull() is SerializationException)
        val record = Record(
            value = Record.IntValue(Long.MIN_VALUE), mixed = mixed,
            items = listOf(Record.BooleanItemsInner(true), Record.IntItemsInner(Long.MAX_VALUE), null),
            binary = Record.BooleanBinary(false), onlyString = "direct",
            nullable = Record.NumberNullable(2.5), nested = Record.Nested(value = Record.Nested.StringValue("nested")),
        )
        val tree = json.encodeToJsonElement(Record.serializer(), record).jsonObject
        check(tree.getValue("value").jsonPrimitive.long == Long.MIN_VALUE)
        check(tree.getValue("mixed").jsonObject.getValue("tag").jsonPrimitive.content == "object")
        check(tree.getValue("items").jsonArray[2] == JsonNull)
        check(json.decodeFromJsonElement(Record.serializer(), tree) == record)
        val nullable = record.copy(nullable = null)
        check(json.decodeFromString(Record.serializer(), json.encodeToString(Record.serializer(), nullable)) == nullable)
        check(runCatching { json.decodeFromJsonElement(Record.serializer(), JsonObject(tree - "value")) }.exceptionOrNull() is SerializationException)
        val constant = Record.ObjectConstantObject()
        val constantWire = json.encodeToJsonElement(Record.ConstantObject.serializer(), constant).jsonObject
        check(constantWire.getValue("tag").jsonPrimitive.content == "empty")
        check((json.decodeFromJsonElement(Record.ConstantObject.serializer(), constantWire) as Record.ObjectConstantObject).tag == "empty")
        val suffixed = RecordDto(value = RecordDto.BooleanValue(true), mixed = RecordDto.StringMixed("id"), items = emptyList(), binary = RecordDto.IntBinary(3L), onlyString = "plain")
        check(json.decodeFromString(RecordDto.serializer(), json.encodeToString(RecordDto.serializer(), suffixed)) == suffixed)
    }
    println("Inline primitive union generated-code contracts passed")
}
