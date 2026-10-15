@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.double
import kotlinx.serialization.json.float
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.long
import kotlinx.serialization.json.put
import org.example.enum_contexts.models.Mode
import org.example.enum_contexts.models.Record
import org.example.enum_contexts_suffix.models.ModeDto
import org.example.enum_contexts_suffix.models.RecordDto
import org.example.enum_limits.models.TypelessBoundary
import org.example.primitive_limits.models.NativeEnumWide

fun main() {
    check(Mode.Serializer.fromSerialName("alpha") === Mode.ALPHA)
    check(Mode.fromSerialName("true") === Mode.TRUE)
    check(Mode.fromSerialName("1") === Mode._1)
    check(TypelessBoundary.entries.size == 240)
    check(TypelessBoundary.Serializer.fromSerialName("v239") === TypelessBoundary.V239)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // Native enums intentionally do not enforce membership: all these values
        // are representable native scalars outside the declared member lists.
        val record = Record(
            mode = Mode.TRUE, count = 3, wide = Long.MIN_VALUE,
            ratio = 3.125, small = 4.125f, enabled = false,
            local = Record.Local.Unrecognized("later"),
            modes = listOf(Record.ModesInner.PLAIN, Record.ModesInner.Unrecognized("future")),
            counts = listOf(3, null), folded = false, nested = Record.Nested(enabled = true),
        )
        val wire = json.encodeToJsonElement(Record.serializer(), record).jsonObject
        check(wire.getValue("mode").jsonPrimitive.isString && wire.getValue("mode").jsonPrimitive.content == "true")
        check(!wire.getValue("count").jsonPrimitive.isString && wire.getValue("count").jsonPrimitive.int == 3)
        check(!wire.getValue("enabled").jsonPrimitive.isString && !wire.getValue("enabled").jsonPrimitive.boolean)
        check(wire.getValue("wide").jsonPrimitive.long == Long.MIN_VALUE)
        check(wire.getValue("ratio").jsonPrimitive.double == 3.125)
        check(wire.getValue("small").jsonPrimitive.float == 4.125f)
        check(wire.getValue("counts").jsonArray[1] == JsonNull)
        check(json.decodeFromJsonElement(Record.serializer(), wire) == record)
        val nullable = record.copy(local = null, folded = null, nested = null)
        check(json.decodeFromString(Record.serializer(), json.encodeToString(Record.serializer(), nullable)) == nullable)
        check(runCatching { json.decodeFromJsonElement(Record.serializer(), JsonObject(wire + ("count" to JsonPrimitive(2147483648L)))) }.exceptionOrNull() is SerializationException)
        check(runCatching { json.decodeFromJsonElement(Record.serializer(), JsonObject(wire + ("enabled" to JsonPrimitive("invalid-boolean")))) }.exceptionOrNull() is SerializationException)
        check(runCatching { json.decodeFromJsonElement(Record.serializer(), JsonObject(wire - "enabled")) }.exceptionOrNull() is SerializationException)
        check(runCatching { json.decodeFromString(Mode.serializer(), "1") }.exceptionOrNull() is SerializationException)
        val suffix = RecordDto(mode = ModeDto.Unrecognized("other"), count = 9, wide = Long.MAX_VALUE, ratio = 7.25, small = 2.25f, enabled = false, modes = emptyList(), counts = emptyList())
        check(json.decodeFromString(RecordDto.serializer(), json.encodeToString(RecordDto.serializer(), suffix)) == suffix)
        check(json.decodeFromString(TypelessBoundary.serializer(), "\"v239\"") === TypelessBoundary.V239)
    }
    val wide = Json.decodeFromJsonElement(NativeEnumWide.serializer(), buildJsonObject {
        repeat(120) { put("field$it", it.toLong()) }
    })
    check(wide.field0 == 0L && wide.field119 == 119L)
    check(NativeEnumWide.serializer().descriptor.elementsCount == 120)
    nestedNativeEnumOwnDefaultsModelContracts()
    println("Typeless string/native scalar enum generated-code contracts passed")
}
