@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.long
import kotlinx.serialization.json.put
import org.example.composition.models.Base
import org.example.composition.models.Extended
import org.example.composition.models.Final
import org.example.composition.models.Holder
import org.example.composition_limits.models.MergedWide
import org.example.composition_suffix.models.ExtendedDto
import org.example.composition_suffix.models.HolderDto
import org.example.composition_suffix.models.BaseDto

fun main() {
    val descriptor = Extended.serializer().descriptor
    check((0 until descriptor.elementsCount).map(descriptor::getElementName) == listOf("id", "score", "enabled", "label", "kind"))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = Extended(id = "entry", score = Long.MAX_VALUE, enabled = true, label = "primary")
        val wire = json.encodeToJsonElement(Extended.serializer(), value).jsonObject
        check(wire.getValue("kind").jsonPrimitive.content == "extended")
        check(wire.getValue("score").jsonPrimitive.long == Long.MAX_VALUE)
        check(json.decodeFromJsonElement(Extended.serializer(), wire) == value)
        check(runCatching { json.decodeFromJsonElement(Extended.serializer(), JsonObject(wire - "enabled")) }.exceptionOrNull() is kotlinx.serialization.SerializationException)
        check(runCatching { json.decodeFromJsonElement(Extended.serializer(), JsonObject(wire + ("score" to JsonPrimitive("wrong-shape")))) }.exceptionOrNull() is kotlinx.serialization.SerializationException)
        check(value.copy(id = "copy").kind == "extended")
        val final = Final(id = "final", enabled = false, label = "inherited", note = "last")
        val decoded = json.decodeFromString(Final.serializer(), json.encodeToString(Final.serializer(), final))
        check(decoded == final && decoded.kind == "extended")
        val holder = Holder(
            wrapped = Base(id = "reference"), native = Long.MIN_VALUE,
            merged = Holder.Merged(id = "inline", extra = "tail"),
            entries = listOf(Holder.EntriesInner(id = "array")),
        )
        val tree = json.encodeToJsonElement(Holder.serializer(), holder).jsonObject
        check(tree.getValue("native").jsonPrimitive.long == Long.MIN_VALUE)
        check(tree.getValue("merged").jsonObject.getValue("kind").jsonPrimitive.content == "base")
        check(tree.getValue("entries").jsonArray.single().jsonObject.getValue("tag").jsonPrimitive.content == "entry")
        check(json.decodeFromJsonElement(Holder.serializer(), tree) == holder)
        check(runCatching { json.decodeFromJsonElement(Holder.serializer(), JsonObject(tree + ("native" to JsonNull))) }.exceptionOrNull() is kotlinx.serialization.SerializationException)
        // The original merged allOf combines non-null Base with a nullable
        // extra operand. The intersection is non-null; wrapper nullable stays.
        check(runCatching { json.decodeFromJsonElement(Holder.serializer(), JsonObject(tree + ("merged" to JsonNull))) }.exceptionOrNull() is kotlinx.serialization.SerializationException)
        val nullable = holder.copy(wrapped = null)
        val nullableWire = json.encodeToJsonElement(Holder.serializer(), nullable).jsonObject
        if (defaults && nulls) check(nullableWire["wrapped"] == JsonNull)
        check(nullableWire.getValue("merged").jsonObject.getValue("kind").jsonPrimitive.content == "base")
        check(json.decodeFromJsonElement(Holder.serializer(), nullableWire) == nullable)
        val suffixed = ExtendedDto(id = "suffix", enabled = true, label = "dto")
        check(json.decodeFromString(ExtendedDto.serializer(), json.encodeToString(ExtendedDto.serializer(), suffixed)) == suffixed)
        val suffixHolder = HolderDto(wrapped = BaseDto(id = "named"), native = 2L, merged = HolderDto.Merged(id = "nested"), entries = emptyList())
        check(json.decodeFromString(HolderDto.serializer(), json.encodeToString(HolderDto.serializer(), suffixHolder)) == suffixHolder)
    }
    val wide = Json.decodeFromJsonElement(MergedWide.serializer(), buildJsonObject {
        repeat(120) { put("field$it", it.toLong()) }
    })
    check(wide.field0 == 0L && wide.field119 == 119L)
    check(MergedWide.serializer().descriptor.elementsCount == 120)
    check(Json.encodeToJsonElement(MergedWide.serializer(), wide).jsonObject.size == 120)
    println("Object allOf generated-code contracts passed")
}
