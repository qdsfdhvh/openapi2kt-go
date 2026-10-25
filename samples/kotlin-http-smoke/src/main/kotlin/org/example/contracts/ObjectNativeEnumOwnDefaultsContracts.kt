package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.refwrapper.models.AllNativeTargetDto
import org.example.refwrapper.models.NativeContainerDto
import org.example.refwrapper.models.NativeHolderDto
import org.example.refwrapper.models.NativeTargetDto
import org.example.refwrapper.models.NativeWideDto
import org.example.refwrapper.models.NativeVocabularyDto
import org.example.refwrapper.models.`AllNativeAliasDto default`
import org.example.refwrapper.models.`AllNativeNarrowDto default`
import org.example.refwrapper.models.`NativeAliasDto default`
import org.example.refwrapper.models.`NativeNarrowDto default`

fun objectNativeEnumOwnDefaultsContracts(codec: Json) {
    val original = NativeTargetDto.default()
    val narrowed = `NativeNarrowDto default`()
    val bool: Boolean? = narrowed.flag
    val int: Int? = narrowed.i32
    val long: Long? = narrowed.i64
    val float: Float? = narrowed.f32
    val double: Double? = narrowed.f64
    check(bool == false && int == Int.MIN_VALUE && long == Long.MIN_VALUE)
    check(float == 16777216f && double == 2.5)
    check(narrowed.nullableFlag == null)
    check(original.flag == true && original.i32 == Int.MAX_VALUE && original.i64 == Long.MAX_VALUE)
    check(original.f32 == 16777216f && original.f64 == 1.5)
    check(`NativeAliasDto default`() == null)
    check(narrowed !== `NativeNarrowDto default`())
    check(codec.decodeFromString<NativeTargetDto>(codec.encodeToString(original)) == original)
    check(codec.decodeFromString<NativeTargetDto>("{}").i64 == Long.MIN_VALUE)

    val combined = AllNativeTargetDto.default()
    val combinedNarrowed = `AllNativeNarrowDto default`()
    check(combined.flag == true && combined.i32 == Int.MAX_VALUE)
    check(combinedNarrowed.flag == false && combinedNarrowed.i32 == Int.MIN_VALUE)
    check(combined.i64 == Long.MIN_VALUE && combinedNarrowed.i64 == Long.MIN_VALUE)
    check(combinedNarrowed.f32 == 16777216f && combinedNarrowed.f64 == 2.5)
    check(`AllNativeAliasDto default`() == null)
    check(combinedNarrowed !== `AllNativeNarrowDto default`())
    check(codec.decodeFromString<AllNativeTargetDto>(codec.encodeToString(combined)) == combined)

    val missing = codec.decodeFromString<NativeHolderDto>("{}")
    check(missing.plain == null && missing.combined == null)
    val supplied = NativeHolderDto(plain = narrowed, combined = combinedNarrowed)
    check(codec.decodeFromString<NativeHolderDto>(codec.encodeToString(supplied)) == supplied)
    val wide = NativeWideDto.default()
    val finalField: Int? = wide.v239
    check(finalField == 1 && wide.v0 == 1)
    check(NativeVocabularyDto.default().value == 0)
    check(codec.decodeFromString<NativeVocabularyDto>("{}").value == 239)
    val nested = NativeContainerDto.default()
    check(nested.box?.flag == false)
    check(nested.box !== NativeContainerDto.default().box)
}
