@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.boundeddefaults.models.TextDto
import org.example.boundeddefaults.models.CountDto
import org.example.boundeddefaults.models.RatioDto
import org.example.boundeddefaults.models.NullTextDto
import org.example.boundeddefaults.models.NullCountDto
import org.example.boundeddefaults.models.NullRatioDto
import org.example.boundeddefaults.models.MiddleDto
import org.example.boundeddefaults.models.OuterDto
import org.example.boundeddefaults.models.CountAliasDto
import org.example.boundeddefaults.models.HolderDto
import org.example.boundeddefaults.models.RequiredDto
import org.example.boundeddefaults.models.`TextDto default`
import org.example.boundeddefaults.models.`CountDto default`
import org.example.boundeddefaults.models.`RatioDto default`
import org.example.boundeddefaults.models.`NullTextDto default`
import org.example.boundeddefaults.models.`NullCountDto default`
import org.example.boundeddefaults.models.`NullRatioDto default`
import org.example.boundeddefaults.models.`MiddleDto default`
import org.example.boundeddefaults.models.`OuterDto default`
import org.example.boundeddefaults.models.`CountAliasDto default`

fun constrainedComponentDefaultsContracts() {
    val text: TextDto = `TextDto default`()
    val count: CountDto = `CountDto default`()
    val ratio: RatioDto = `RatioDto default`()
    check(text == "given" && count == 4 && ratio == 1.5)
    check(`NullTextDto default`() == null && `NullCountDto default`() == null && `NullRatioDto default`() == null)
    check(`MiddleDto default`() == null && `OuterDto default`() == "outer" && `CountAliasDto default`() == 6)
    check(serializer<NullTextDto>().descriptor.isNullable)
    check(serializer<NullCountDto>().descriptor.isNullable)
    check(serializer<NullRatioDto>().descriptor.isNullable)
    check(!serializer<OuterDto>().descriptor.isNullable)
    check(runCatching { HolderDto(text = "X") }.isFailure)
    check(runCatching { HolderDto(count = 3) }.isFailure)
    check(runCatching { HolderDto(ratio = 2.0) }.isFailure)
    check(runCatching { HolderDto(outer = "X") }.isFailure)
    check(runCatching { HolderDto(nullText = "x") }.isFailure)
    check(runCatching { RequiredDto(text = "X", count = 4) }.isFailure)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<TextDto>(codec.encodeToString(text)) == text)
        check(codec.decodeFromString<CountDto>(codec.encodeToString(count)) == count)
        check(codec.decodeFromString<RatioDto>(codec.encodeToString(ratio)) == ratio)
        check(codec.decodeFromString<NullTextDto>("null") == null)
        check(codec.decodeFromString<NullCountDto>("null") == null)
        check(codec.decodeFromString<NullRatioDto>("null") == null)
        check(codec.encodeToString<MiddleDto>(`MiddleDto default`()) == "null")
        // Bare scalar aliases retain their unchecked standalone serializer.
        check(codec.decodeFromString<TextDto>("\"X\"") == "X")
        check(codec.decodeFromString<CountDto>("3") == 3)
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.text == null && missing.count == null && missing.ratio == null && missing.middle == null && missing.outer == null)
        val explicit = codec.decodeFromString<HolderDto>("""{"text":null,"count":null,"ratio":null,"outer":null}""")
        check(explicit.text == null && explicit.count == null && explicit.ratio == null && explicit.outer == null)
        val given = HolderDto(text = text, count = count, ratio = ratio, outer = `OuterDto default`())
        val wire = codec.encodeToJsonElement(HolderDto.serializer(), given) as JsonObject
        check(wire["text"] == JsonPrimitive("given") && wire["count"] == JsonPrimitive(4) && wire["ratio"] == JsonPrimitive(1.5))
        val decoded = codec.decodeFromString<HolderDto>(codec.encodeToString(given))
        check(decoded.text == text && decoded.count == count && decoded.ratio == ratio && decoded.outer == "outer")
        check(runCatching { codec.decodeFromString<HolderDto>("""{"text":"X"}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"count":3}""") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("""{"text":null,"count":4}""") }.isFailure)
        val required = RequiredDto(text = `TextDto default`(), count = `CountDto default`())
        check(codec.decodeFromString<RequiredDto>(codec.encodeToString(required)).text == "given")
    }
}
