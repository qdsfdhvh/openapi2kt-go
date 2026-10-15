package org.example.contracts

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.SerializationException
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.*
import org.example.scalar_projections.models.*
import org.example.scalar_projections_suffix.models.EnvelopeDto
import org.example.scalar_projections_suffix.models.RefOnlyDto
import org.example.scalar_projection_limits.models.ConstRefWide
import org.example.scalar_projection_limits.models.EnumWide

private fun rejectsScalarProjection(block: () -> Unit) {
    try { block(); error("Invalid native scalar wire accepted") } catch (_: SerializationException) { }
}

@OptIn(ExperimentalSerializationApi::class)
fun main() {
    val expectedText = "line\n\"$\\"
    val value = Envelope(choice = 17, wide = Long.MAX_VALUE, flag = false, floats = listOf(-0.0f, null, 7.5f), doubles = listOf(-0.0, 11.0), pinnedItems = listOf(null, Long.MAX_VALUE), literalItems = listOf("different", null))
    check(value.text == expectedText && value.count == 17 && value.long == Long.MIN_VALUE && value.boolean)
    check(value.negativeZero.toBits() == (-0.0f).toBits() && value.escaped == "fixed")
    // Type aliases are transparent native values, not string-token singleton APIs.
    val native: LongChoices = Long.MAX_VALUE
    check(native == Long.MAX_VALUE && value.copy(choice = 123).choice == 123)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(json.encodeToJsonElement(IntegerChoices.serializer(), 99) == JsonPrimitive(99))
        check(json.decodeFromJsonElement(LongChoices.serializer(), JsonPrimitive(Long.MIN_VALUE)) == Long.MIN_VALUE)
        check(json.encodeToJsonElement(FloatChoices.serializer(), 7.5f) == JsonPrimitive(7.5f))
        check(json.encodeToJsonElement(DoubleChoices.serializer(), 11.0) == JsonPrimitive(11.0))
        check(json.encodeToJsonElement(BooleanChoices.serializer(), false) == JsonPrimitive(false))
        // Scalar literal value positions do not enforce const membership.
        check(json.decodeFromJsonElement(PinnedLong.serializer(), JsonPrimitive(Long.MAX_VALUE)) == Long.MAX_VALUE)
        val encoded = json.encodeToJsonElement(Envelope.serializer(), value).jsonObject
        check(encoded["choice"] == JsonPrimitive(17) && !encoded.getValue("choice").jsonPrimitive.isString)
        check(encoded["text"] == JsonPrimitive(expectedText) && encoded["long"] == JsonPrimitive(Long.MIN_VALUE))
        check(encoded["negativeZero"]!!.jsonPrimitive.float.toBits() == (-0.0f).toBits())
        check(encoded["literalItems"] == buildJsonArray { add("different"); add(JsonNull) })
        check(encoded["pinnedItems"] == buildJsonArray { add(JsonNull); add(Long.MAX_VALUE) })
        check(json.encodeToJsonElement(Envelope.serializer(), json.decodeFromJsonElement(Envelope.serializer(), encoded)) == encoded)
        val minimal = JsonObject(encoded.filterKeys { it !in setOf("text", "count", "long", "negativeZero", "boolean", "escaped") })
        val decoded = json.decodeFromJsonElement(Envelope.serializer(), minimal)
        check(decoded.text == expectedText && decoded.long == Long.MIN_VALUE && decoded.negativeZero.toBits() == (-0.0f).toBits())
        check(json.encodeToJsonElement(Envelope.serializer(), decoded) == encoded)
        val changed = json.decodeFromJsonElement(Envelope.serializer(), JsonObject(minimal + ("text" to JsonPrimitive("override"))))
        check(changed.text == "override")
        check(json.encodeToJsonElement(Envelope.serializer(), changed).jsonObject["text"] == JsonPrimitive("override"))
        rejectsScalarProjection { json.decodeFromJsonElement(Envelope.serializer(), JsonObject(minimal + ("long" to JsonNull))) }
        rejectsScalarProjection { json.decodeFromJsonElement(Envelope.serializer(), JsonObject(minimal + ("choice" to JsonPrimitive("bad-integer")))) }
        check(json.encodeToJsonElement(RefOnly.serializer(), RefOnly()) == buildJsonObject { put("pin", expectedText) })
        check(json.decodeFromJsonElement(RefOnly.serializer(), buildJsonObject { put("pin", "other") }).pin == "other")
        val suffixed = EnvelopeDto(choice = value.choice, wide = value.wide, flag = value.flag, floats = value.floats, doubles = value.doubles, pinnedItems = value.pinnedItems, literalItems = value.literalItems)
        check(json.encodeToJsonElement(EnvelopeDto.serializer(), suffixed) == encoded)
        check(json.encodeToJsonElement(RefOnlyDto.serializer(), RefOnlyDto()) == buildJsonObject { put("pin", expectedText) })
        val taggedRaw = buildJsonObject { put("events", buildJsonArray { add(buildJsonObject { put("kind", "left"); put("id", Long.MAX_VALUE) }); add(buildJsonObject { put("kind", "right"); put("note", "text") }) }) }
        val tagged = json.decodeFromJsonElement(TaggedHolder.serializer(), taggedRaw)
        check((tagged.events[0] as TaggedHolder.EventsInner.Left).id == Long.MAX_VALUE)
        check((tagged.events[1] as TaggedHolder.EventsInner.Right).note == "text")
        check(tagged.events[0].kind == "left" && tagged.events[1].kind == "right")
        check(json.encodeToJsonElement(TaggedHolder.serializer(), tagged) == taggedRaw)
        val wideRaw = buildJsonObject { for (i in 0 until 120) put("field$i", if (i % 2 == 0) Long.MIN_VALUE else Long.MAX_VALUE) }
        val wide = json.decodeFromJsonElement(EnumWide.serializer(), wideRaw)
        check(wide.field0 == Long.MIN_VALUE && wide.field119 == Long.MAX_VALUE)
        check(json.encodeToJsonElement(EnumWide.serializer(), wide) == wideRaw)
        val constWide = json.encodeToJsonElement(ConstRefWide.serializer(), ConstRefWide()).jsonObject
        check(constWide.size == 120 && constWide.values.all { it == JsonPrimitive(Long.MIN_VALUE) })
        check(json.encodeToJsonElement(ConstRefWide.serializer(), json.decodeFromJsonElement(ConstRefWide.serializer(), buildJsonObject {})) == constWide)
    }
    println("Native enum component, scalar const projection/ref/body/item and native slot contracts passed")
}
