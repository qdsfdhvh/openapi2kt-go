@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.wrapperchain.models.InheritedObjectDto
import org.example.wrapperchain.models.`InheritedObjectDto default`
import org.example.wrapperchain.models.SelectedObjectDto
import org.example.wrapperchain.models.`SelectedObjectDto default`
import org.example.wrapperchain.models.MiddleObjectDto
import org.example.wrapperchain.models.`MiddleObjectDto default`
import org.example.wrapperchain.models.ReopenedObjectDto
import org.example.wrapperchain.models.`ReopenedObjectDto default`
import org.example.wrapperchain.models.FreshObjectDto
import org.example.wrapperchain.models.`FreshObjectDto default`
import org.example.wrapperchain.models.InheritedCombinedDto
import org.example.wrapperchain.models.`InheritedCombinedDto default`
import org.example.wrapperchain.models.SelectedCombinedDto
import org.example.wrapperchain.models.`SelectedCombinedDto default`
import org.example.wrapperchain.models.MiddleCombinedDto
import org.example.wrapperchain.models.`MiddleCombinedDto default`
import org.example.wrapperchain.models.ReopenedCombinedDto
import org.example.wrapperchain.models.`ReopenedCombinedDto default`
import org.example.wrapperchain.models.FreshCombinedDto
import org.example.wrapperchain.models.`FreshCombinedDto default`
import org.example.wrapperchain.models.InheritedRowsDto
import org.example.wrapperchain.models.`InheritedRowsDto default`
import org.example.wrapperchain.models.SelectedRowsDto
import org.example.wrapperchain.models.`SelectedRowsDto default`
import org.example.wrapperchain.models.MiddleRowsDto
import org.example.wrapperchain.models.`MiddleRowsDto default`
import org.example.wrapperchain.models.ReopenedRowsDto
import org.example.wrapperchain.models.`ReopenedRowsDto default`
import org.example.wrapperchain.models.FreshRowsDto
import org.example.wrapperchain.models.`FreshRowsDto default`
import org.example.wrapperchain.models.InheritedMapDto
import org.example.wrapperchain.models.`InheritedMapDto default`
import org.example.wrapperchain.models.SelectedMapDto
import org.example.wrapperchain.models.`SelectedMapDto default`
import org.example.wrapperchain.models.MiddleMapDto
import org.example.wrapperchain.models.`MiddleMapDto default`
import org.example.wrapperchain.models.ReopenedMapDto
import org.example.wrapperchain.models.`ReopenedMapDto default`
import org.example.wrapperchain.models.FreshMapDto
import org.example.wrapperchain.models.`FreshMapDto default`
import org.example.wrapperchain.models.InheritedModeDto
import org.example.wrapperchain.models.`InheritedModeDto default`
import org.example.wrapperchain.models.SelectedModeDto
import org.example.wrapperchain.models.`SelectedModeDto default`
import org.example.wrapperchain.models.MiddleModeDto
import org.example.wrapperchain.models.`MiddleModeDto default`
import org.example.wrapperchain.models.ReopenedModeDto
import org.example.wrapperchain.models.`ReopenedModeDto default`
import org.example.wrapperchain.models.FreshModeDto
import org.example.wrapperchain.models.`FreshModeDto default`
import org.example.wrapperchain.models.InheritedDayDto
import org.example.wrapperchain.models.`InheritedDayDto default`
import org.example.wrapperchain.models.SelectedDayDto
import org.example.wrapperchain.models.`SelectedDayDto default`
import org.example.wrapperchain.models.MiddleDayDto
import org.example.wrapperchain.models.`MiddleDayDto default`
import org.example.wrapperchain.models.ReopenedDayDto
import org.example.wrapperchain.models.`ReopenedDayDto default`
import org.example.wrapperchain.models.FreshDayDto
import org.example.wrapperchain.models.`FreshDayDto default`
import org.example.wrapperchain.models.InheritedBlobDto
import org.example.wrapperchain.models.`InheritedBlobDto default`
import org.example.wrapperchain.models.SelectedBlobDto
import org.example.wrapperchain.models.`SelectedBlobDto default`
import org.example.wrapperchain.models.MiddleBlobDto
import org.example.wrapperchain.models.`MiddleBlobDto default`
import org.example.wrapperchain.models.ReopenedBlobDto
import org.example.wrapperchain.models.`ReopenedBlobDto default`
import org.example.wrapperchain.models.FreshBlobDto
import org.example.wrapperchain.models.`FreshBlobDto default`
import org.example.wrapperchain.models.InheritedRawDto
import org.example.wrapperchain.models.`InheritedRawDto default`
import org.example.wrapperchain.models.SelectedRawDto
import org.example.wrapperchain.models.`SelectedRawDto default`
import org.example.wrapperchain.models.MiddleRawDto
import org.example.wrapperchain.models.`MiddleRawDto default`
import org.example.wrapperchain.models.ReopenedRawDto
import org.example.wrapperchain.models.`ReopenedRawDto default`
import org.example.wrapperchain.models.FreshRawDto
import org.example.wrapperchain.models.`FreshRawDto default`
import org.example.wrapperchain.models.InheritedOpenDto
import org.example.wrapperchain.models.`InheritedOpenDto default`
import org.example.wrapperchain.models.SelectedOpenDto
import org.example.wrapperchain.models.`SelectedOpenDto default`
import org.example.wrapperchain.models.MiddleOpenDto
import org.example.wrapperchain.models.`MiddleOpenDto default`
import org.example.wrapperchain.models.ReopenedOpenDto
import org.example.wrapperchain.models.`ReopenedOpenDto default`
import org.example.wrapperchain.models.FreshOpenDto
import org.example.wrapperchain.models.`FreshOpenDto default`
import org.example.wrapperchain.models.HolderDto
import org.example.wrapperchain.models.RequiredDto

fun wrapperChainNullabilityDefaultsContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(`InheritedObjectDto default`() == null && `ReopenedObjectDto default`() == null)
        check(serializer<InheritedObjectDto>().descriptor.isNullable)
        check(serializer<ReopenedObjectDto>().descriptor.isNullable)
        check(!serializer<SelectedObjectDto>().descriptor.isNullable)
        check(!serializer<MiddleObjectDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedObjectDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedObjectDto>("null") }.isFailure)
        check(codec.decodeFromString<SelectedObjectDto>(codec.encodeToString(`SelectedObjectDto default`())) == `MiddleObjectDto default`())
        check(`FreshObjectDto default`() == `SelectedObjectDto default`())
        check(`InheritedCombinedDto default`() == null && `ReopenedCombinedDto default`() == null)
        check(serializer<InheritedCombinedDto>().descriptor.isNullable)
        check(serializer<ReopenedCombinedDto>().descriptor.isNullable)
        check(!serializer<SelectedCombinedDto>().descriptor.isNullable)
        check(!serializer<MiddleCombinedDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedCombinedDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedCombinedDto>("null") }.isFailure)
        check(codec.decodeFromString<SelectedCombinedDto>(codec.encodeToString(`SelectedCombinedDto default`())) == `MiddleCombinedDto default`())
        check(`FreshCombinedDto default`() == `SelectedCombinedDto default`())
        check(`InheritedRowsDto default`() == null && `ReopenedRowsDto default`() == null)
        check(serializer<InheritedRowsDto>().descriptor.isNullable)
        check(serializer<ReopenedRowsDto>().descriptor.isNullable)
        check(!serializer<SelectedRowsDto>().descriptor.isNullable)
        check(!serializer<MiddleRowsDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedRowsDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedRowsDto>("null") }.isFailure)
        check(codec.decodeFromString<SelectedRowsDto>(codec.encodeToString(`SelectedRowsDto default`())) == `MiddleRowsDto default`())
        check(`FreshRowsDto default`() == `SelectedRowsDto default`())
        check(`InheritedMapDto default`() == null && `ReopenedMapDto default`() == null)
        check(serializer<InheritedMapDto>().descriptor.isNullable)
        check(serializer<ReopenedMapDto>().descriptor.isNullable)
        check(!serializer<SelectedMapDto>().descriptor.isNullable)
        check(!serializer<MiddleMapDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedMapDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedMapDto>("null") }.isFailure)
        check(codec.decodeFromString<SelectedMapDto>(codec.encodeToString(`SelectedMapDto default`())) == `MiddleMapDto default`())
        check(`FreshMapDto default`() == `SelectedMapDto default`())
        check(`InheritedModeDto default`() == null && `ReopenedModeDto default`() == null)
        check(serializer<InheritedModeDto>().descriptor.isNullable)
        check(serializer<ReopenedModeDto>().descriptor.isNullable)
        check(!serializer<SelectedModeDto>().descriptor.isNullable)
        check(!serializer<MiddleModeDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedModeDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedModeDto>("null") }.isFailure)
        check(codec.decodeFromString<SelectedModeDto>(codec.encodeToString(`SelectedModeDto default`())) == `MiddleModeDto default`())
        check(`FreshModeDto default`() == `SelectedModeDto default`())
        check(`InheritedDayDto default`() == null && `ReopenedDayDto default`() == null)
        check(serializer<InheritedDayDto>().descriptor.isNullable)
        check(serializer<ReopenedDayDto>().descriptor.isNullable)
        check(!serializer<SelectedDayDto>().descriptor.isNullable)
        check(!serializer<MiddleDayDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedDayDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedDayDto>("null") }.isFailure)
        check(codec.decodeFromString<SelectedDayDto>(codec.encodeToString(`SelectedDayDto default`())) == `MiddleDayDto default`())
        check(`FreshDayDto default`() == `SelectedDayDto default`())
        check(`InheritedBlobDto default`() == null && `ReopenedBlobDto default`() == null)
        check(serializer<InheritedBlobDto>().descriptor.isNullable)
        check(serializer<ReopenedBlobDto>().descriptor.isNullable)
        check(!serializer<SelectedBlobDto>().descriptor.isNullable)
        check(!serializer<MiddleBlobDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedBlobDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedBlobDto>("null") }.isFailure)
        check(`InheritedRawDto default`() == null && `ReopenedRawDto default`() == null)
        check(serializer<InheritedRawDto>().descriptor.isNullable)
        check(serializer<ReopenedRawDto>().descriptor.isNullable)
        check(!serializer<SelectedRawDto>().descriptor.isNullable)
        check(!serializer<MiddleRawDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedRawDto>("null") == null)
        check(codec.decodeFromString<SelectedRawDto>(codec.encodeToString(`SelectedRawDto default`())) == `MiddleRawDto default`())
        check(`FreshRawDto default`() == `SelectedRawDto default`())
        check(`InheritedOpenDto default`() == null && `ReopenedOpenDto default`() == null)
        check(serializer<InheritedOpenDto>().descriptor.isNullable)
        check(serializer<ReopenedOpenDto>().descriptor.isNullable)
        check(!serializer<SelectedOpenDto>().descriptor.isNullable)
        check(!serializer<MiddleOpenDto>().descriptor.isNullable)
        check(codec.decodeFromString<InheritedOpenDto>("null") == null)
        check(runCatching { codec.decodeFromString<SelectedOpenDto>("null") }.isFailure)
        check(codec.decodeFromString<SelectedOpenDto>(codec.encodeToString(`SelectedOpenDto default`())) == `MiddleOpenDto default`())
        check(`FreshOpenDto default`() == `SelectedOpenDto default`())
        check(`SelectedObjectDto default`().z == "given")
        check(`SelectedCombinedDto default`().z == "combined")
        check(`SelectedRowsDto default`() == listOf(null, "given"))
        check(`SelectedMapDto default`() == mapOf("nil" to null, "a" to 1))
        check(`SelectedObjectDto default`() !== `SelectedObjectDto default`())
        check(`SelectedRowsDto default`() !== `SelectedRowsDto default`())
        check(`SelectedMapDto default`() !== `SelectedMapDto default`())
        val blob = `SelectedBlobDto default`()
        check(blob.contentEquals(byteArrayOf(-128, 127)))
        blob[0] = 0
        check(`SelectedBlobDto default`()[0] == (-128).toByte())
        check(`MiddleBlobDto default`().contentEquals(byteArrayOf(-128, 127)))
        check(`FreshBlobDto default`()!!.contentEquals(byteArrayOf(-128, 127)))
        check(codec.decodeFromString<SelectedBlobDto>(codec.encodeToString(`SelectedBlobDto default`())).contentEquals(byteArrayOf(-128, 127)))
        check(`SelectedRawDto default`() == codec.parseToJsonElement("""{"nil":null}"""))
        check(`SelectedOpenDto default`()["nil"] === JsonNull)
        // A non-null raw JsonElement retains JsonNull as a JSON value.
        check(codec.decodeFromString<SelectedRawDto>("null") === JsonNull)
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.selectedObject == null && missing.selectedRows == null && missing.selectedMap == null)
        check(missing.freshObject == null && missing.inheritedObject == null && missing.reopenedObject == null)
        val explicit = codec.decodeFromString<HolderDto>("""{"selectedObject":null,"selectedRows":null,"selectedMap":null}""")
        check(explicit.selectedObject == null && explicit.selectedRows == null && explicit.selectedMap == null)
        val supplied = codec.decodeFromString<HolderDto>("""{"selectedObject":{"z":"wire"},"selectedRows":["wire",null],"selectedMap":{"x":3}}""")
        check(supplied.selectedObject?.z == "wire" && supplied.selectedRows == listOf("wire", null))
        check(supplied.selectedMap == mapOf("x" to 3))
        check(runCatching { codec.decodeFromString<RequiredDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("""{"value":null}""") }.isFailure)
        val required = RequiredDto(value = `SelectedObjectDto default`())
        check(codec.decodeFromString<RequiredDto>(codec.encodeToString(required)).value.z == "given")
        val tree = codec.encodeToJsonElement(HolderDto.serializer(), HolderDto(selectedObject = `SelectedObjectDto default`())) as JsonObject
        check(tree["selectedObject"] == codec.parseToJsonElement("""{"z":"given"}"""))
    }
}
