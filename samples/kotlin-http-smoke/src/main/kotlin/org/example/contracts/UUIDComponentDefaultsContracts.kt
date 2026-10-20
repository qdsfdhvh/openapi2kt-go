@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlin.uuid.Uuid
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.uuiddefaults.models.TextDto
import org.example.uuiddefaults.models.EscapedDto
import org.example.uuiddefaults.models.OuterDto
import org.example.uuiddefaults.models.HolderDto
import org.example.uuiddefaults.models.RequiredDto
import org.example.uuiddefaults.models.`TextDto default`
import org.example.uuiddefaults.models.`EscapedDto default`
import org.example.uuiddefaults.models.`BoundedDto default`
import org.example.uuiddefaults.models.`BoundedAliasDto default`
import org.example.uuiddefaults.models.`MiddleDto default`
import org.example.uuiddefaults.models.`OuterDto default`
import org.example.uuiddefaults.models.`NullableRootDto default`
import org.example.uuiddefaults.models.`NullableAliasDto default`
import org.example.uuiddefaults.models.`NarrowedOwnDto default`
import org.example.uuiddefaults.models.`ReopenedOwnDto default`

fun uuidComponentDefaultsContracts() {
    val id = Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
    val middle = Uuid.parse("550e8400-e29b-41d4-a716-446655440001")
    val last = Uuid.parse("550e8400-e29b-41d4-a716-446655440002")
    val other = Uuid.parse("650e8400-e29b-41d4-a716-446655440000")
    val text: TextDto = `TextDto default`()
    val escaped: EscapedDto = `EscapedDto default`()
    val outer: OuterDto = `OuterDto default`()
    check(text == id && escaped == "quote\"\\\né" && outer == last)
    check(`MiddleDto default`() == middle)
    check(`NullableRootDto default`() == null && `NullableAliasDto default`() == null)
    check(`NarrowedOwnDto default`() == last && `ReopenedOwnDto default`() == null)
    check(`BoundedDto default`() == id && `BoundedAliasDto default`() == middle)
    check(!serializer<OuterDto>().descriptor.isNullable)
    check(runCatching { HolderDto(bounded = other) }.isFailure)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<TextDto>(codec.encodeToString(text)) == text)
        check(codec.decodeFromString<EscapedDto>(codec.encodeToString(escaped)) == escaped)
        check(runCatching { codec.decodeFromString<TextDto>("\"\"") }.isFailure)
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.text == null && missing.escaped == null && missing.bounded == null)
        check(missing.outer == null && missing.plain == null)
        check(codec.decodeFromString<HolderDto>("""{"text":null,"outer":null}""").text == null)
        val adopted = HolderDto(text = text, escaped = escaped, bounded = `BoundedDto default`(), outer = outer)
        val wire = codec.encodeToJsonElement(HolderDto.serializer(), adopted) as JsonObject
        check(wire["text"] == JsonPrimitive(text.toString()) && wire["escaped"] == JsonPrimitive(escaped))
        val decoded = codec.decodeFromString<HolderDto>(codec.encodeToString(adopted))
        check(decoded.escaped == escaped && decoded.bounded == id && decoded.outer == outer)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"bounded":"UPPER"}""") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("""{"text":null,"outer":"given"}""") }.isFailure)
        val required = RequiredDto(text = text, outer = outer)
        check(codec.decodeFromString<RequiredDto>(codec.encodeToString(required)).text == text)
    }
}
