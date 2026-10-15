package core_test

const rootPrimitivePortRuntime = `@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.*
import org.example.root_primitive_port.models.*
import org.example.root_primitive_limits.models.WideRoot
import org.example.root_primitive_limits.models.ObjectWideRoot

fun main() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(json.decodeFromString(Choice.serializer(), "null") === NullChoice)
        check(json.decodeFromString(Choice.serializer(), "9223372036854775807") == IntChoice(Long.MAX_VALUE))
        check(json.decodeFromString(Choice.serializer(), "\"outside-pin\"") == StringChoice("outside-pin"))
        check(json.decodeFromString(Choice.serializer(), "true") == BooleanChoice(true))
        check(json.encodeToString(Choice.serializer(), NullChoice) == "null")
        check(json.decodeFromString(NullChoice.serializer(), "null") === NullChoice)
        for (wire in listOf("0", "true", "{}", "[]", "\"null\"")) {
            check(runCatching { json.decodeFromString(NullChoice.serializer(), wire) }.exceptionOrNull() is SerializationException)
        }
        val carrier = Carrier(value = NullChoice, items = listOf(NullChoice, IntChoice(Long.MIN_VALUE), StringChoice("future")))
        val wire = json.encodeToJsonElement(Carrier.serializer(), carrier).jsonObject
        check(wire.getValue("value") == JsonNull)
        check(wire.getValue("items").jsonArray[1] == JsonPrimitive(Long.MIN_VALUE))
        check(json.decodeFromJsonElement(Carrier.serializer(), wire) == carrier)
        check(runCatching { json.decodeFromJsonElement(Carrier.serializer(), JsonObject(wire - "value")) }.exceptionOrNull() is SerializationException)
        check(json.decodeFromString(Mixed.serializer(), "null") === NullMixed)
        val obj = json.decodeFromString(Mixed.serializer(), """{"id":"entry"}""") as ObjectMixed
        check(obj.id == "entry" && obj.tag == "object")
        val objWire = json.encodeToJsonElement(Mixed.serializer(), obj).jsonObject
        check(objWire.getValue("tag") == JsonPrimitive("object"))
        check(json.decodeFromJsonElement(Mixed.serializer(), objWire) == obj)
        check(json.decodeFromString(Mixed.serializer(), "\"scalar\"") == StringMixed("scalar"))
        check(runCatching { json.decodeFromString(Mixed.serializer(), "false") }.exceptionOrNull() is SerializationException)
    }
    val payload = buildJsonObject { repeat(120) { put("field$it", Long.MAX_VALUE) } }
    val wide = Json.decodeFromJsonElement(WideRoot.serializer(), payload) as ObjectWideRoot
    check(wide.field119 == Long.MAX_VALUE)
    check(Json.encodeToJsonElement(WideRoot.serializer(), wide) == payload)
    println("Go primitive-root strict null/native pins/ref/items/mixed-object and 240-slot contracts passed")
}
`
