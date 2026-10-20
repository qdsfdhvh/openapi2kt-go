@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.numericdefaults.models.Int32Dto
import org.example.numericdefaults.models.LongMaxDto
import org.example.numericdefaults.models.LongMinDto
import org.example.numericdefaults.models.FloatValueDto
import org.example.numericdefaults.models.DoubleValueDto
import org.example.numericdefaults.models.NullableFloatDto
import org.example.numericdefaults.models.OuterDto
import org.example.numericdefaults.models.HolderDto
import org.example.numericdefaults.models.RequiredDto
import org.example.numericdefaults.models.`Int32Dto default`
import org.example.numericdefaults.models.`LongMaxDto default`
import org.example.numericdefaults.models.`LongMinDto default`
import org.example.numericdefaults.models.`FloatValueDto default`
import org.example.numericdefaults.models.`DoubleValueDto default`
import org.example.numericdefaults.models.`NegativeZeroDto default`
import org.example.numericdefaults.models.`NullableFloatDto default`
import org.example.numericdefaults.models.`CountDto default`
import org.example.numericdefaults.models.`RatioDto default`
import org.example.numericdefaults.models.`MiddleDto default`
import org.example.numericdefaults.models.`OuterDto default`
import org.example.numericdefaults.models.`CountAliasDto default`

fun numericComponentDefaultsContracts() {
    val int: Int32Dto = `Int32Dto default`()
    val max: LongMaxDto = `LongMaxDto default`()
    val min: LongMinDto = `LongMinDto default`()
    val float: FloatValueDto = `FloatValueDto default`()
    val double: DoubleValueDto = `DoubleValueDto default`()
    check(int == Int.MAX_VALUE && max == Long.MAX_VALUE && min == Long.MIN_VALUE)
    check(float == 0.300000004.toFloat() && double == Double.MAX_VALUE)
    check(`NegativeZeroDto default`().toBits() == (-0.0f).toBits())
    check(`NullableFloatDto default`() == null && `MiddleDto default`() == null)
    check(`OuterDto default`() == 0.5f && `CountAliasDto default`() == 6L)
    check(`CountDto default`() == 4L && `RatioDto default`() == float)
    check(serializer<NullableFloatDto>().descriptor.isNullable)
    check(!serializer<OuterDto>().descriptor.isNullable)
    check(runCatching { HolderDto(count = 3L) }.isFailure)
    check(runCatching { HolderDto(ratio = 0.0f) }.isFailure)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<Int32Dto>(codec.encodeToString(int)) == int)
        check(codec.decodeFromString<LongMaxDto>(codec.encodeToString(max)) == max)
        check(codec.decodeFromString<LongMinDto>(codec.encodeToString(min)) == min)
        check(codec.decodeFromString<FloatValueDto>(codec.encodeToString(float)) == float)
        check(codec.decodeFromString<DoubleValueDto>(codec.encodeToString(double)) == double)
        check(codec.decodeFromString<NullableFloatDto>("null") == null)
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.int32 == null && missing.longMax == null && missing.floatValue == null)
        check(missing.count == null && missing.ratio == null && missing.outer == null && missing.plain == null)
        val explicit = codec.decodeFromString<HolderDto>("""{"int32":null,"longMax":null,"outer":null}""")
        check(explicit.int32 == null && explicit.longMax == null && explicit.outer == null)
        val adopted = HolderDto(int32 = int, longMax = max, longMin = min, floatValue = float,
            doubleValue = double, count = `CountDto default`(), ratio = `RatioDto default`(), outer = `OuterDto default`())
        val wire = codec.encodeToJsonElement(HolderDto.serializer(), adopted) as JsonObject
        check(wire["int32"] == JsonPrimitive(Int.MAX_VALUE) && wire["longMax"] == JsonPrimitive(Long.MAX_VALUE))
        check(wire["floatValue"] == JsonPrimitive(float) && wire["outer"] == JsonPrimitive(0.5f))
        val decoded = codec.decodeFromString<HolderDto>(codec.encodeToString(adopted))
        check(decoded.longMin == min && decoded.floatValue == float && decoded.doubleValue == double)
        check(decoded.count == 4L && decoded.ratio == float && decoded.outer == 0.5f)
        check(runCatching { codec.decodeFromString<RequiredDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("""{"int32":null,"longMax":1}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"count":3}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"ratio":0}""") }.isFailure)
        val required = RequiredDto(int32 = int, longMax = max)
        check(codec.decodeFromString<RequiredDto>(codec.encodeToString(required)).longMax == max)
    }
}
