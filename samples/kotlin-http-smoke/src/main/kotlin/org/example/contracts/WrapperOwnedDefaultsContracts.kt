@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.owneddefaults.models.TargetObjectDto
import org.example.owneddefaults.models.MiddleObjectDto
import org.example.owneddefaults.models.`MiddleObjectDto default`
import org.example.owneddefaults.models.OuterObjectDto
import org.example.owneddefaults.models.`OuterObjectDto default`
import org.example.owneddefaults.models.NullObjectDto
import org.example.owneddefaults.models.`NullObjectDto default`
import org.example.owneddefaults.models.TargetCombinedDto
import org.example.owneddefaults.models.MiddleCombinedDto
import org.example.owneddefaults.models.`MiddleCombinedDto default`
import org.example.owneddefaults.models.OuterCombinedDto
import org.example.owneddefaults.models.`OuterCombinedDto default`
import org.example.owneddefaults.models.NullCombinedDto
import org.example.owneddefaults.models.`NullCombinedDto default`
import org.example.owneddefaults.models.TargetRowsDto
import org.example.owneddefaults.models.`TargetRowsDto default`
import org.example.owneddefaults.models.MiddleRowsDto
import org.example.owneddefaults.models.`MiddleRowsDto default`
import org.example.owneddefaults.models.OuterRowsDto
import org.example.owneddefaults.models.`OuterRowsDto default`
import org.example.owneddefaults.models.NullRowsDto
import org.example.owneddefaults.models.`NullRowsDto default`
import org.example.owneddefaults.models.TargetMapDto
import org.example.owneddefaults.models.`TargetMapDto default`
import org.example.owneddefaults.models.MiddleMapDto
import org.example.owneddefaults.models.`MiddleMapDto default`
import org.example.owneddefaults.models.OuterMapDto
import org.example.owneddefaults.models.`OuterMapDto default`
import org.example.owneddefaults.models.NullMapDto
import org.example.owneddefaults.models.`NullMapDto default`
import org.example.owneddefaults.models.TargetModeDto
import org.example.owneddefaults.models.`TargetModeDto default`
import org.example.owneddefaults.models.MiddleModeDto
import org.example.owneddefaults.models.`MiddleModeDto default`
import org.example.owneddefaults.models.OuterModeDto
import org.example.owneddefaults.models.`OuterModeDto default`
import org.example.owneddefaults.models.NullModeDto
import org.example.owneddefaults.models.`NullModeDto default`
import org.example.owneddefaults.models.TargetDayDto
import org.example.owneddefaults.models.`TargetDayDto default`
import org.example.owneddefaults.models.MiddleDayDto
import org.example.owneddefaults.models.`MiddleDayDto default`
import org.example.owneddefaults.models.OuterDayDto
import org.example.owneddefaults.models.`OuterDayDto default`
import org.example.owneddefaults.models.NullDayDto
import org.example.owneddefaults.models.`NullDayDto default`
import org.example.owneddefaults.models.TargetBlobDto
import org.example.owneddefaults.models.`TargetBlobDto default`
import org.example.owneddefaults.models.MiddleBlobDto
import org.example.owneddefaults.models.`MiddleBlobDto default`
import org.example.owneddefaults.models.OuterBlobDto
import org.example.owneddefaults.models.`OuterBlobDto default`
import org.example.owneddefaults.models.NullBlobDto
import org.example.owneddefaults.models.`NullBlobDto default`
import org.example.owneddefaults.models.TargetRawDto
import org.example.owneddefaults.models.`TargetRawDto default`
import org.example.owneddefaults.models.MiddleRawDto
import org.example.owneddefaults.models.`MiddleRawDto default`
import org.example.owneddefaults.models.OuterRawDto
import org.example.owneddefaults.models.`OuterRawDto default`
import org.example.owneddefaults.models.NullRawDto
import org.example.owneddefaults.models.`NullRawDto default`
import org.example.owneddefaults.models.TargetOpenDto
import org.example.owneddefaults.models.`TargetOpenDto default`
import org.example.owneddefaults.models.MiddleOpenDto
import org.example.owneddefaults.models.`MiddleOpenDto default`
import org.example.owneddefaults.models.OuterOpenDto
import org.example.owneddefaults.models.`OuterOpenDto default`
import org.example.owneddefaults.models.NullOpenDto
import org.example.owneddefaults.models.`NullOpenDto default`
import org.example.owneddefaults.models.HolderDto
import org.example.owneddefaults.models.RequiredDto
import org.example.owneddefaults.models.NullTargetDto
import org.example.owneddefaults.models.`NullTargetDto default`
import org.example.owneddefaults.models.NonNullOverNullDto
import org.example.owneddefaults.models.`NonNullOverNullDto default`

fun wrapperOwnedDefaultsContracts() {
    check(TargetObjectDto.default().z == "target")
    check(`MiddleObjectDto default`()!!.z == "middle")
    check(`OuterObjectDto default`().z == "outer")
    check(TargetCombinedDto.default().z == "target")
    check(`MiddleCombinedDto default`()!!.z == "middle")
    check(`OuterCombinedDto default`().z == "outer")
    check(`TargetRowsDto default`() == listOf("target"))
    check(`MiddleRowsDto default`() == listOf("middle", null))
    check(`OuterRowsDto default`() == listOf("outer"))
    check(`TargetMapDto default`() == mapOf("z" to 1))
    check(`MiddleMapDto default`() == mapOf("z" to 2, "nil" to null))
    check(`OuterMapDto default`() == mapOf("z" to 3))
    check(`TargetModeDto default`() == TargetModeDto.A)
    check(`MiddleModeDto default`() == TargetModeDto.B)
    check(`OuterModeDto default`() == TargetModeDto.C)
    check(`TargetDayDto default`() == kotlinx.datetime.LocalDate.parse("2024-01-01"))
    check(`MiddleDayDto default`() == kotlinx.datetime.LocalDate.parse("2024-02-29"))
    check(`OuterDayDto default`() == kotlinx.datetime.LocalDate.parse("2024-12-31"))
    check(`TargetBlobDto default`().contentEquals(byteArrayOf(-128)))
    check(`MiddleBlobDto default`()!!.contentEquals(byteArrayOf(0)))
    check(`OuterBlobDto default`().contentEquals(byteArrayOf(127)))
    check(`TargetRawDto default`() is JsonObject)
    check(`MiddleRawDto default`() == Json.parseToJsonElement("[1,null]"))
    check(`OuterRawDto default`() == JsonPrimitive("outer"))
    check(`TargetOpenDto default`()["z"] == JsonPrimitive(1))
    check(`MiddleOpenDto default`()!!["nil"] === JsonNull)
    check(`OuterOpenDto default`()["z"] == JsonPrimitive(3))
    check(`NullTargetDto default`() == null)
    check(`NonNullOverNullDto default`() == listOf("outer"))
    val blob = `OuterBlobDto default`()
    blob[0] = 0
    check(`OuterBlobDto default`()[0] == 127.toByte())
    check(TargetObjectDto.default() !== TargetObjectDto.default())
    check(`MiddleRowsDto default`() !== `MiddleRowsDto default`())
    check(`OuterMapDto default`() !== `OuterMapDto default`())
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(`NullObjectDto default`() == null)
        check(serializer<NullObjectDto>().descriptor.isNullable)
        check(!serializer<OuterObjectDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullObjectDto>("null") == null)
        check(codec.decodeFromString<OuterObjectDto>(codec.encodeToString(`OuterObjectDto default`())) == `OuterObjectDto default`())
        check(`NullCombinedDto default`() == null)
        check(serializer<NullCombinedDto>().descriptor.isNullable)
        check(!serializer<OuterCombinedDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullCombinedDto>("null") == null)
        check(codec.decodeFromString<OuterCombinedDto>(codec.encodeToString(`OuterCombinedDto default`())) == `OuterCombinedDto default`())
        check(`NullRowsDto default`() == null)
        check(serializer<NullRowsDto>().descriptor.isNullable)
        check(!serializer<OuterRowsDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullRowsDto>("null") == null)
        check(codec.decodeFromString<OuterRowsDto>(codec.encodeToString(`OuterRowsDto default`())) == `OuterRowsDto default`())
        check(`NullMapDto default`() == null)
        check(serializer<NullMapDto>().descriptor.isNullable)
        check(!serializer<OuterMapDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullMapDto>("null") == null)
        check(codec.decodeFromString<OuterMapDto>(codec.encodeToString(`OuterMapDto default`())) == `OuterMapDto default`())
        check(`NullModeDto default`() == null)
        check(serializer<NullModeDto>().descriptor.isNullable)
        check(!serializer<OuterModeDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullModeDto>("null") == null)
        check(codec.decodeFromString<OuterModeDto>(codec.encodeToString(`OuterModeDto default`())) == `OuterModeDto default`())
        check(`NullDayDto default`() == null)
        check(serializer<NullDayDto>().descriptor.isNullable)
        check(!serializer<OuterDayDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullDayDto>("null") == null)
        check(codec.decodeFromString<OuterDayDto>(codec.encodeToString(`OuterDayDto default`())) == `OuterDayDto default`())
        check(`NullBlobDto default`() == null)
        check(serializer<NullBlobDto>().descriptor.isNullable)
        check(!serializer<OuterBlobDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullBlobDto>("null") == null)
        check(`NullRawDto default`() == null)
        check(serializer<NullRawDto>().descriptor.isNullable)
        check(!serializer<OuterRawDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullRawDto>("null") == null)
        check(codec.decodeFromString<OuterRawDto>(codec.encodeToString(`OuterRawDto default`())) == `OuterRawDto default`())
        check(`NullOpenDto default`() == null)
        check(serializer<NullOpenDto>().descriptor.isNullable)
        check(!serializer<OuterOpenDto>().descriptor.isNullable)
        check(codec.decodeFromString<NullOpenDto>("null") == null)
        check(codec.decodeFromString<OuterOpenDto>(codec.encodeToString(`OuterOpenDto default`())) == `OuterOpenDto default`())
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.targetObject == null && missing.middleObject == null && missing.outerObject == null && missing.plainObject == null)
        check(missing.targetRows == null && missing.middleRows == null && missing.outerRows == null)
        val explicit = codec.decodeFromString<HolderDto>("""{"targetObject":null,"middleObject":null,"outerObject":null,"outerRows":null}""")
        check(explicit.targetObject == null && explicit.middleObject == null && explicit.outerObject == null && explicit.outerRows == null)
        val supplied = codec.decodeFromString<HolderDto>("""{"targetObject":{"z":"wire"},"middleObject":{"z":"wire"},"outerObject":{"z":"wire"},"outerRows":["wire"]}""")
        check(supplied.targetObject?.z == "wire" && supplied.middleObject?.z == "wire" && supplied.outerObject?.z == "wire")
        check(supplied.outerRows == listOf("wire"))
        val adopted = HolderDto(targetObject = TargetObjectDto.default(), middleObject = `MiddleObjectDto default`(), outerObject = `OuterObjectDto default`())
        val tree = codec.encodeToJsonElement(HolderDto.serializer(), adopted) as JsonObject
        check(tree["targetObject"] == codec.parseToJsonElement("""{"z":"target"}"""))
        check(tree["middleObject"] == codec.parseToJsonElement("""{"z":"middle"}"""))
        check(tree["outerObject"] == codec.parseToJsonElement("""{"z":"outer"}"""))
        check(runCatching { codec.decodeFromString<RequiredDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("""{"value":null}""") }.isFailure)
        check(codec.decodeFromString<RequiredDto>(codec.encodeToString(RequiredDto(`OuterObjectDto default`()))).value.z == "outer")
    }
}
