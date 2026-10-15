package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.*
import org.example.union_normalization.models.*
import org.example.union_normalization.models.Record as NormalizedRecord

private inline fun rejectsNormalization(block: () -> Unit) {
    try { block(); error("Expected serialization failure") } catch (_: SerializationException) { }
}

fun main() {
    check(NormalizedRecord.Status.entries.size == 3)
    check(NormalizedRecord.Status.entries.map { it.serialName } == listOf("queued", "ready", "complete"))
    check(NormalizedRecord.Status.Serializer.fromSerialName("ready") === NormalizedRecord.Status.READY)
    check(NormalizedRecord.Status.Unrecognized("future") == NormalizedRecord.Status.Unrecognized("future"))
    check(NormalizedRecord.StatesInner.fromSerialName("scheduled") === NormalizedRecord.StatesInner.SCHEDULED)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val record = NormalizedRecord(
            status = NormalizedRecord.Status.Unrecognized("future"),
            states = listOf(NormalizedRecord.StatesInner.COMPLETE, null, NormalizedRecord.StatesInner.Unrecognized("other")),
            details = NormalizedRecord.Details(NormalizedRecord.Details.Phase.READY),
            rawText = "", // Folded branch constraints deliberately aren't validators.
            range = Long.MAX_VALUE,
            explicitInt = -12,
            floatChoice = NormalizedRecord.IntFloatChoice(Long.MIN_VALUE),
            inlinePinned = NormalizedRecord.StringInlinePinned("outside-pin"),
            choice = IntPinnedChoice(Long.MAX_VALUE), // Native Long wrapper, not int32 pin membership.
            choices = listOf(NullPinnedChoice, BooleanPinnedChoice(true), NumberPinnedChoice(7.25))
        )
        val wire = json.encodeToJsonElement(NormalizedRecord.serializer(), record).jsonObject
        check(wire["status"] == JsonPrimitive("future"))
        check(wire["states"] == buildJsonArray { add("complete"); add(JsonNull); add("other") })
        check(wire["rawText"] == JsonPrimitive(""))
        check(wire["range"] == JsonPrimitive(Long.MAX_VALUE))
        check(wire["explicitInt"] == JsonPrimitive(-12))
        check(wire["floatChoice"] == JsonPrimitive(Long.MIN_VALUE))
        check(wire["inlinePinned"] == JsonPrimitive("outside-pin"))
        check(wire["choice"] == JsonPrimitive(Long.MAX_VALUE))
        check(json.decodeFromJsonElement(NormalizedRecord.serializer(), wire) == record)
        val cleared = record.copy(status = null, choices = listOf(NullPinnedChoice))
        val clearedWire = json.encodeToJsonElement(NormalizedRecord.serializer(), cleared).jsonObject
        check(json.decodeFromJsonElement(NormalizedRecord.serializer(), clearedWire) == cleared)
        check(clearedWire["choices"] == buildJsonArray { add(JsonNull) })
        for (bad in listOf("7", "true", "{}", "[]")) rejectsNormalization { json.decodeFromString(NormalizedRecord.Status.serializer(), bad) }
        rejectsNormalization { json.decodeFromJsonElement(NormalizedRecord.serializer(), JsonObject(wire + ("status" to JsonPrimitive(7)))) }
        for (value in listOf(Long.MIN_VALUE, 0L, Long.MAX_VALUE)) {
            check(json.decodeFromString(PinnedChoice.serializer(), value.toString()) == IntPinnedChoice(value))
            check(json.encodeToString(PinnedChoice.serializer(), IntPinnedChoice(value)) == value.toString())
        }
        check(json.decodeFromString(PinnedChoice.serializer(), "true") == BooleanPinnedChoice(true))
        check(json.decodeFromString(PinnedChoice.serializer(), "\"outside\"") == StringPinnedChoice("outside"))
        check(json.decodeFromString(PinnedChoice.serializer(), "1.25") == NumberPinnedChoice(1.25))
        check(json.decodeFromString(PinnedChoice.serializer(), "null") === NullPinnedChoice)
        rejectsNormalization { json.decodeFromString(PinnedChoice.serializer(), "{}") }
        rejectsNormalization { json.decodeFromString(PinnedChoice.serializer(), "[]") }
        check(json.decodeFromString(NormalizedRecord.InlinePinned.serializer(), "false") == NormalizedRecord.BooleanInlinePinned(false))
        val suffix = org.example.union_normalization_suffix.models.StringPinnedChoiceDto("future")
        check(json.encodeToString(org.example.union_normalization_suffix.models.PinnedChoiceDto.serializer(), suffix) == "\"future\"")
        check(org.example.union_normalization_suffix.models.RecordDto.Status.fromSerialName("ready") === org.example.union_normalization_suffix.models.RecordDto.Status.READY)
    }
    check(org.example.union_normalization_limits.models.EnumHolder.Value.entries.size == 240)
    check(org.example.union_normalization_limits.models.EnumHolder.Value.Serializer.fromSerialName("token_239") === org.example.union_normalization_limits.models.EnumHolder.Value.TOKEN_239)
    val payload = buildJsonObject { repeat(120) { put("field$it", JsonPrimitive(Long.MAX_VALUE)) } }
    val wide = Json.decodeFromJsonElement(org.example.union_normalization_limits.models.WideFold.serializer(), payload)
    check(wide.field119 == Long.MAX_VALUE)
    check(Json.encodeToJsonElement(org.example.union_normalization_limits.models.WideFold.serializer(), wide) == payload)
    println("Borrowed union normalization/open enums/native pins/source ranges/all JSON configurations and actual 240-slot contracts passed")
}
