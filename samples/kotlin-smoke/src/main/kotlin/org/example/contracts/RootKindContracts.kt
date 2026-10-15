package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.*
import org.example.root_kind_unions.models.*

private inline fun rejectsRoot(block: () -> Unit) {
    try { block(); error("Expected serialization failure") } catch (_: SerializationException) { }
}

private fun <T> verifyRootNull(json: Json, serializer: kotlinx.serialization.KSerializer<T>) {
    val decoded = json.decodeFromString(serializer, "null")
    check(json.encodeToString(serializer, decoded) == "null")
}

fun main() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(json.decodeFromString(PrimitiveChoice.serializer(), "true") == BooleanPrimitiveChoice(true))
        check(json.decodeFromString(PrimitiveChoice.serializer(), "\"true\"") == StringPrimitiveChoice("true"))
        for (value in listOf(Long.MIN_VALUE, 0L, Long.MAX_VALUE)) {
            val wrapped = IntPrimitiveChoice(value)
            check(json.encodeToString(PrimitiveChoice.serializer(), wrapped) == value.toString())
            check(json.decodeFromString(PrimitiveChoice.serializer(), value.toString()) == wrapped)
        }
        check(json.decodeFromString(PrimitiveChoice.serializer(), "1.25") == NumberPrimitiveChoice(1.25))
        check(json.encodeToString(PrimitiveChoice.serializer(), StringPrimitiveChoice("hello")) == "\"hello\"")
        for (wrong in listOf("null", "[]", "{}")) rejectsRoot { json.decodeFromString(PrimitiveChoice.serializer(), wrong) }
        check(json.decodeFromString(MixedChoice.serializer(), "{\"message\":\"hello\"}") == ObjectMixedChoice("hello"))
        check(json.encodeToString(MixedChoice.serializer(), ObjectMixedChoice("hello")) == "{\"message\":\"hello\"}")
        rejectsRoot { json.decodeFromString(MixedChoice.serializer(), "{}") }
        rejectsRoot { json.decodeFromString(MixedChoice.serializer(), "{\"message\":7}") }
        verifyRootNull(json, NullableChoice.serializer())
        verifyRootNull(json, NullableLong.serializer())
        verifyRootNull(json, AnnotatedChoice.serializer())
        verifyRootNull(json, NullableTagged.serializer())
        verifyRootNull(json, NullablePoly.serializer())
        check(json.decodeFromString(NullableLong.serializer(), "null") === NullNullableLong)
        check(json.decodeFromString(NullableLong.serializer(), Long.MAX_VALUE.toString()) == IntNullableLong(Long.MAX_VALUE))
        rejectsRoot { json.decodeFromString(NullableLong.serializer(), "\"7\"") }
        rejectsRoot { json.decodeFromString(NullNullableLong.serializer(), "{}") }
        rejectsRoot { json.decodeFromString(NullNullableLong.serializer(), "7") }
        check(json.encodeToString(NullNullableLong.serializer(), NullNullableLong) == "null")
        check(json.decodeFromString(NullableTagged.serializer(), "null") === NullNullableTagged)
        check(NullNullableTagged.type == "")
        val tagged = json.decodeFromString(NullableTagged.serializer(), "{\"type\":\"left\",\"amount\":3}")
        check(tagged is NullableTagged.Left && tagged.amount == 3 && tagged.type == "left")
        check(json.parseToJsonElement(json.encodeToString(NullableTagged.serializer(), tagged)).jsonObject["type"] == JsonPrimitive("left"))
        rejectsRoot { json.decodeFromString(NullableTagged.serializer(), "{\"type\":\"left\",\"amount\":\"bad\"}") }
        val raw = json.parseToJsonElement("{\"type\":\"future\",\"other\":false}")
        val unknown = json.decodeFromJsonElement(NullableTagged.serializer(), raw)
        check(unknown is UnknownNullableTagged && unknown.rawElement == raw)
        check(json.encodeToJsonElement(NullableTagged.serializer(), unknown) == raw)
        val poly = NullablePoly.Variant1(true)
        val payload = json.encodeToJsonElement(NullablePoly.serializer(), poly)
        check(payload.jsonObject["type"] == JsonPrimitive("Variant1"))
        check(json.decodeFromJsonElement(NullablePoly.serializer(), payload) == poly)
        check(NullNullablePoly.flag == false)
        rejectsRoot { json.decodeFromString(NullablePoly.serializer(), "{\"flag\":true}") }
        rejectsRoot { json.decodeFromString(NullablePoly.serializer(), "{\"type\":\"future\",\"flag\":true}") }
        rejectsRoot { json.decodeFromString(NullablePoly.serializer(), "7") }
        val carrier = Carrier(IntPrimitiveChoice(7), ObjectMixedChoice("x"), NullNullableChoice, listOf(NullNullableLong, IntNullableLong(9)), NullNullableTagged, NullNullablePoly)
        val encoded = json.encodeToJsonElement(Carrier.serializer(), carrier).jsonObject
        check(encoded["nullable"] == JsonNull && encoded["tagged"] == JsonNull && encoded["poly"] == JsonNull)
        check(encoded["longs"] == JsonArray(listOf(JsonNull, JsonPrimitive(9))))
        check(json.decodeFromJsonElement(Carrier.serializer(), encoded) == carrier)
        val suffix = org.example.root_kind_union_suffix.models.IntPrimitiveChoiceDto(9)
        check(json.encodeToString(org.example.root_kind_union_suffix.models.PrimitiveChoiceDto.serializer(), suffix) == "9")
        check(json.decodeFromString(org.example.root_kind_union_suffix.models.NullableLongDto.serializer(), "null") === org.example.root_kind_union_suffix.models.NullNullableLongDto)
    }
    val custom = Json { classDiscriminator = "_variant" }
    val poly = NullablePoly.Variant2(false)
    val encoded = custom.encodeToJsonElement(NullablePoly.serializer(), poly).jsonObject
    check(encoded["_variant"] == JsonPrimitive("Variant2") && "type" !in encoded)
    check(custom.decodeFromJsonElement(NullablePoly.serializer(), encoded) == poly)
    check(custom.encodeToString(NullablePoly.serializer(), NullNullablePoly) == "null")
    val arrays = Json { useArrayPolymorphism = true }
    check(arrays.decodeFromString(NullablePoly.serializer(), arrays.encodeToString(NullablePoly.serializer(), poly)) == poly)
    check(arrays.decodeFromString(NullablePoly.serializer(), "null") === NullNullablePoly)
    val widePayload = buildJsonObject { repeat(120) { put("field$it", JsonPrimitive(it)) } }
    val wide = Json.decodeFromJsonElement(org.example.root_kind_limits.models.WideRoot.serializer(), widePayload)
    check(wide is org.example.root_kind_limits.models.ObjectWideRoot && wide.field119 == 119L)
    check(Json.encodeToJsonElement(org.example.root_kind_limits.models.WideRoot.serializer(), wide) == widePayload)
    val carrierPayload = buildJsonObject {
        repeat(119) { put("field$it", JsonPrimitive(it)) }
        put("root", JsonPrimitive(true)); put("flag", JsonPrimitive(false))
    }
    val wideCarrier = Json.decodeFromJsonElement(org.example.root_kind_limits.models.WideCarrier.serializer(), carrierPayload)
    check(wideCarrier.root == org.example.root_kind_limits.models.BooleanWideRoot(true) && wideCarrier.field118 == 118L)
    check(Json.encodeToJsonElement(org.example.root_kind_limits.models.WideCarrier.serializer(), wideCarrier) == carrierPayload)
    println("Root native-kind payloads, named null/ref/list/content, closed nullable polymorphic and actual 240-slot contracts passed")
}
