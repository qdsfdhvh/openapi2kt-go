@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.rawjsondefaults.models.FreeArrayDto
import org.example.rawjsondefaults.models.FreeDto
import org.example.rawjsondefaults.models.FreeScalarDto
import org.example.rawjsondefaults.models.HolderDto
import org.example.rawjsondefaults.models.MaybeFreeDto
import org.example.rawjsondefaults.models.MaybeOpenDto
import org.example.rawjsondefaults.models.NullEmptyDto
import org.example.rawjsondefaults.models.NullFreeDto
import org.example.rawjsondefaults.models.NullOpenDto
import org.example.rawjsondefaults.models.RequiredWrappedDto
import org.example.rawjsondefaults.models.WrappedFreeDto
import org.example.rawjsondefaults.models.WrappedNilFreeDto
import org.example.rawjsondefaults.models.WrappedNilOpenDto
import org.example.rawjsondefaults.models.WrappedOpenDto
import org.example.rawjsondefaults.models.WrappedTextDto
import org.example.rawjsondefaults.models.`WrappedFreeDto default`
import org.example.rawjsondefaults.models.`WrappedMaybeFreeDto default`
import org.example.rawjsondefaults.models.`WrappedMaybeOpenDto default`
import org.example.rawjsondefaults.models.`WrappedNilFreeDto default`
import org.example.rawjsondefaults.models.`WrappedNilOpenDto default`
import org.example.rawjsondefaults.models.`WrappedOpenDto default`
import org.example.rawjsondefaults.models.`WrappedTextDto default`
import org.example.rawjsondefaults.models.OpenDto
import org.example.rawjsondefaults.models.`FreeArrayDto default`
import org.example.rawjsondefaults.models.`FreeDto default`
import org.example.rawjsondefaults.models.`FreeScalarDto default`
import org.example.rawjsondefaults.models.`MaybeFreeDto default`
import org.example.rawjsondefaults.models.`MaybeOpenDto default`
import org.example.rawjsondefaults.models.`NullEmptyDto default`
import org.example.rawjsondefaults.models.`NullFreeDto default`
import org.example.rawjsondefaults.models.`NullOpenDto default`
import org.example.rawjsondefaults.models.`OpenDto default`

// A raw JSON carrier component root now owns an opt-in factory, using the same
// grammar as the field role: an open dictionary yields JsonObject, and an
// unconstrained/empty object yields JsonElement.
suspend fun rawJSONComponentDefaultsContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        rawJSONWrapperDefaults(codec)

        // Nullable carrier factories return Kotlin null through their actual
        // alias serializers. A narrowed JsonElement keeps its JsonNull value.
        val nullFree: NullFreeDto = `NullFreeDto default`()
        val nullEmpty: NullEmptyDto = `NullEmptyDto default`()
        val nullOpen: NullOpenDto = `NullOpenDto default`()
        check(nullFree == null && nullEmpty == null && nullOpen == null)
        check(serializer<NullFreeDto>().descriptor.isNullable)
        check(serializer<NullEmptyDto>().descriptor.isNullable)
        check(serializer<NullOpenDto>().descriptor.isNullable)
        check(codec.encodeToJsonElement(serializer<NullFreeDto>(), nullFree) === JsonNull)
        check(codec.encodeToJsonElement(serializer<NullEmptyDto>(), nullEmpty) === JsonNull)
        check(codec.encodeToJsonElement(serializer<NullOpenDto>(), nullOpen) === JsonNull)
        check(codec.decodeFromJsonElement(serializer<NullFreeDto>(), JsonNull) == null)
        check(codec.decodeFromJsonElement(serializer<NullEmptyDto>(), JsonNull) == null)
        check(codec.decodeFromJsonElement(serializer<NullOpenDto>(), JsonNull) == null)
        check(codec.decodeFromJsonElement(serializer<JsonElement>(), JsonNull) === JsonNull)
        check(codec.decodeFromJsonElement(serializer<NullFreeDto>(), JsonPrimitive("null")) == JsonPrimitive("null"))
        check(runCatching { codec.decodeFromJsonElement(serializer<NullOpenDto>(), JsonPrimitive("null")) }.isFailure)

        // Existing non-null nullable factories preserve raw object null members.
        val maybeFree: MaybeFreeDto = `MaybeFreeDto default`()
        val maybeOpen: MaybeOpenDto = `MaybeOpenDto default`()
        check((maybeFree as JsonObject)["kept"] === JsonNull && maybeOpen?.get("kept") === JsonNull)
        check((codec.encodeToJsonElement(serializer<MaybeFreeDto>(), maybeFree) as JsonObject)["kept"] === JsonNull)
        check((codec.encodeToJsonElement(serializer<MaybeOpenDto>(), maybeOpen) as JsonObject)["kept"] === JsonNull)

        // The open dictionary factory carries its own object value.
        val open: JsonObject = `OpenDto default`()
        check(open["a"] == JsonPrimitive(1) && open["b"] == JsonPrimitive("x")) { "open object lost: $open" }

        // An unconstrained root accepts any JSON value.
        val free: JsonElement = `FreeDto default`()
        check((free as JsonObject)["nested"] is JsonObject)
        val array: JsonElement = `FreeArrayDto default`()
        check(array is JsonArray && array.size == 3 && array[0] == JsonPrimitive(1))
        val scalar: JsonElement = `FreeScalarDto default`()
        check(scalar == JsonPrimitive("raw"))

        // A field referencing a defaulted carrier keeps its neutral presence.
        val holder = HolderDto()
        check(holder.open == null) { "Component default leaked into a referencing field" }
        check(holder.free == null) { "Component default leaked into a referencing field" }
        check(holder.nullFree == null && holder.nullOpen == null)
        check(holder.inline?.get("k") == JsonPrimitive(2))
        val adopted = HolderDto(nullFree = nullFree, nullOpen = nullOpen)
        val adoptedWire = codec.encodeToJsonElement(HolderDto.serializer(), adopted) as JsonObject
        for (key in listOf("nullFree", "nullOpen")) {
            check(adoptedWire.containsKey(key) == (defaults && nulls))
            if (adoptedWire.containsKey(key)) check(adoptedWire[key] === JsonNull)
        }
        val explicitNull = codec.decodeFromString<HolderDto>("""{"nullFree":null,"nullOpen":null}""")
        check(explicitNull.nullFree == null && explicitNull.nullOpen == null)
        val rawValues = codec.decodeFromString<HolderDto>("""{"nullFree":"given","nullOpen":{"kept":null}}""")
        check(rawValues.nullFree == JsonPrimitive("given") && rawValues.nullOpen?.get("kept") === JsonNull)
        val rawWire = codec.encodeToJsonElement(HolderDto.serializer(), rawValues) as JsonObject
        check((rawWire["nullOpen"] as JsonObject)["kept"] === JsonNull)

        // The factories round-trip through the same wire the fields use.
        check(codec.encodeToJsonElement(open) == codec.parseToJsonElement("""{"a":1,"b":"x"}"""))
        check(codec.encodeToJsonElement(`FreeArrayDto default`()) == codec.parseToJsonElement("""[1,"two",false]"""))
        check(codec.encodeToJsonElement(`FreeScalarDto default`()) == codec.parseToJsonElement("\"raw\""))
        val decoded = codec.decodeFromString<FreeDto>(codec.encodeToString(`FreeDto default`()))
        check(decoded == free)

        // A supplied value still wins over the factory.
        val supplied = codec.decodeFromString<OpenDto>("""{"z":true}""")
        check(supplied["z"] == JsonPrimitive(true))

        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.open == null && missing.free == null)
        check(missing.nullFree == null && missing.nullOpen == null)

        println("RawJSONComponentDefaults Json defaults=$defaults nulls=$nulls passed")
    }
}

private fun rawJSONWrapperDefaults(codec: Json) {
    val free: WrappedFreeDto = `WrappedFreeDto default`()
    val open: WrappedOpenDto = `WrappedOpenDto default`()
    val text: WrappedTextDto = `WrappedTextDto default`()
    val nilFree: WrappedNilFreeDto = `WrappedNilFreeDto default`()
    val nilOpen: WrappedNilOpenDto = `WrappedNilOpenDto default`()
    check(free == codec.parseToJsonElement("""[1,"two",null]"""))
    check(text == JsonPrimitive("null"))
    check(open["kept"] === JsonNull && (open["z"] as JsonObject)["${'$'}ref"] == JsonPrimitive("data"))
    check(nilFree == null && nilOpen == null)
    check(`WrappedMaybeFreeDto default`() == codec.parseToJsonElement("""{"kept":null}"""))
    check(`WrappedMaybeOpenDto default`() == codec.parseToJsonElement("""{"kept":null}"""))
    check(free !== `WrappedFreeDto default`() && open !== `WrappedOpenDto default`())
    check(open["z"] !== `WrappedOpenDto default`()["z"])

    // Wrappers use the target's actual alias serializer and intrinsic null intent.
    check(!serializer<WrappedFreeDto>().descriptor.isNullable)
    check(!serializer<WrappedOpenDto>().descriptor.isNullable)
    check(serializer<WrappedNilFreeDto>().descriptor.isNullable)
    check(serializer<WrappedNilOpenDto>().descriptor.isNullable)
    check(codec.decodeFromString<WrappedFreeDto>("null") === JsonNull)
    check(codec.decodeFromString<WrappedNilFreeDto>("null") == null)
    check(codec.decodeFromString<WrappedNilOpenDto>("null") == null)
    check(runCatching { codec.decodeFromString<WrappedOpenDto>("null") }.isFailure)
    check(runCatching { codec.decodeFromString<WrappedOpenDto>("[]") }.isFailure)
    check(codec.decodeFromString<WrappedFreeDto>(codec.encodeToString(free)) == free)
    check(codec.decodeFromString<WrappedOpenDto>(codec.encodeToString(open)) == open)
    check(codec.decodeFromString<WrappedTextDto>(codec.encodeToString(text)) == text)
    check(codec.encodeToJsonElement(nilFree) === JsonNull && codec.encodeToJsonElement(nilOpen) === JsonNull)
    check(codec.encodeToJsonElement(open) == codec.parseToJsonElement("""{"kept":null,"z":{"${'$'}ref":"data"}}"""))

    val missing = codec.decodeFromString<HolderDto>("{}")
    check(missing.wrappedFree == null && missing.wrappedOpen == null && missing.wrappedNilFree == null && missing.wrappedNilOpen == null)
    val explicit = codec.decodeFromString<HolderDto>("""{"wrappedFree":null,"wrappedOpen":null,"wrappedNilFree":null,"wrappedNilOpen":null}""")
    check(explicit.wrappedFree == null && explicit.wrappedOpen == null && explicit.wrappedNilFree == null && explicit.wrappedNilOpen == null)
    val supplied = codec.decodeFromString<HolderDto>("""{"wrappedFree":false,"wrappedOpen":{"other":null}}""")
    check(supplied.wrappedFree == JsonPrimitive(false) && supplied.wrappedOpen?.get("other") === JsonNull)
    val adopted = HolderDto(wrappedFree = free, wrappedOpen = open)
    val wire = codec.encodeToJsonElement(HolderDto.serializer(), adopted) as JsonObject
    check(wire["wrappedFree"] == free && wire["wrappedOpen"] == open)
    check(runCatching { codec.decodeFromString<RequiredWrappedDto>("{}") }.isFailure)
    check(runCatching { codec.decodeFromString<RequiredWrappedDto>("""{"free":[],"open":null}""") }.isFailure)
    check(codec.decodeFromString<RequiredWrappedDto>("""{"free":null,"open":{}}""").free === JsonNull)
    val required = RequiredWrappedDto(free = free, open = open)
    check(codec.decodeFromString<RequiredWrappedDto>(codec.encodeToString(required)).open == open)
}
