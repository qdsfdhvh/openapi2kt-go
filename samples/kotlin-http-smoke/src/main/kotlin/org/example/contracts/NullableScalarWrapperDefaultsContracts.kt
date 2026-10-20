@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import org.example.refwrapper.models.CheckedWrapperHolderDto
import org.example.refwrapper.models.NullableCheckedTextDto
import org.example.refwrapper.models.NullableCheckedEnumDto
import org.example.refwrapper.models.NullCheckedTextDto
import org.example.refwrapper.models.NullCheckedEnumDto
import org.example.refwrapper.models.`NullableCheckedTextDto default`
import org.example.refwrapper.models.`NullableCheckedEnumDto default`
import org.example.refwrapper.models.`NullCheckedTextDto default`
import org.example.refwrapper.models.`NullCheckedEnumDto default`

import org.example.refwrapper.models.InheritedCheckedTextDto
import org.example.refwrapper.models.`InheritedCheckedTextDto default`
import org.example.refwrapper.models.SelectedCheckedTextDto
import org.example.refwrapper.models.`SelectedCheckedTextDto default`
import org.example.refwrapper.models.MiddleCheckedTextDto
import org.example.refwrapper.models.`MiddleCheckedTextDto default`
import org.example.refwrapper.models.ReopenedCheckedTextDto
import org.example.refwrapper.models.`ReopenedCheckedTextDto default`
import org.example.refwrapper.models.InheritedCheckedEnumDto
import org.example.refwrapper.models.`InheritedCheckedEnumDto default`
import org.example.refwrapper.models.SelectedCheckedEnumDto
import org.example.refwrapper.models.`SelectedCheckedEnumDto default`
import org.example.refwrapper.models.MiddleCheckedEnumDto
import org.example.refwrapper.models.`MiddleCheckedEnumDto default`
import org.example.refwrapper.models.ReopenedCheckedEnumDto
import org.example.refwrapper.models.`ReopenedCheckedEnumDto default`

import org.example.refwrapper.models.OwnedTextDto
import org.example.refwrapper.models.`OwnedTextDto default`
import org.example.refwrapper.models.OwnedTextBridgeDto
import org.example.refwrapper.models.`OwnedTextBridgeDto default`
import org.example.refwrapper.models.OwnedTextOuterDto
import org.example.refwrapper.models.`OwnedTextOuterDto default`
import org.example.refwrapper.models.OwnedTextNullDto
import org.example.refwrapper.models.`OwnedTextNullDto default`
import org.example.refwrapper.models.OwnedNumberDto
import org.example.refwrapper.models.`OwnedNumberDto default`
import org.example.refwrapper.models.OwnedNumberBridgeDto
import org.example.refwrapper.models.`OwnedNumberBridgeDto default`
import org.example.refwrapper.models.OwnedNumberOuterDto
import org.example.refwrapper.models.`OwnedNumberOuterDto default`
import org.example.refwrapper.models.OwnedNumberNullDto
import org.example.refwrapper.models.`OwnedNumberNullDto default`

fun nullableScalarWrapperDefaultsContracts() {
    check(`InheritedCheckedTextDto default`() == null && `ReopenedCheckedTextDto default`() == null)
    check(`SelectedCheckedTextDto default`() == "given" && `MiddleCheckedTextDto default`() == "given")
    check(serializer<InheritedCheckedTextDto>().descriptor.isNullable)
    check(!serializer<SelectedCheckedTextDto>().descriptor.isNullable)
    check(!serializer<MiddleCheckedTextDto>().descriptor.isNullable)
    check(`InheritedCheckedEnumDto default`() == null && `ReopenedCheckedEnumDto default`() == null)
    check(`SelectedCheckedEnumDto default`() == 2 && `MiddleCheckedEnumDto default`() == 2)
    check(serializer<InheritedCheckedEnumDto>().descriptor.isNullable)
    check(!serializer<SelectedCheckedEnumDto>().descriptor.isNullable)
    check(!serializer<MiddleCheckedEnumDto>().descriptor.isNullable)
    check(`OwnedTextDto default`() == "target")
    check(`OwnedTextBridgeDto default`() == "middle")
    check(`OwnedTextOuterDto default`() == "outer")
    check(`OwnedTextNullDto default`() == null)
    check(`OwnedNumberDto default`() == 1)
    check(`OwnedNumberBridgeDto default`() == 2)
    check(`OwnedNumberOuterDto default`() == 3)
    check(`OwnedNumberNullDto default`() == null)
    val text: NullableCheckedTextDto = `NullableCheckedTextDto default`()
    val number: NullableCheckedEnumDto = `NullableCheckedEnumDto default`()
    check(text == "given" && number == 2)
    check(`NullCheckedTextDto default`() == null && `NullCheckedEnumDto default`() == null)
    check(serializer<NullCheckedTextDto>().descriptor.isNullable)
    check(serializer<NullCheckedEnumDto>().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<NullableCheckedTextDto>(codec.encodeToString(text)) == text)
        check(codec.decodeFromString<NullableCheckedEnumDto>(codec.encodeToString(number)) == number)
        check(codec.decodeFromString<NullCheckedTextDto>("null") == null)
        check(codec.decodeFromString<NullCheckedEnumDto>("null") == null)
        check(codec.encodeToString<NullCheckedTextDto>(`NullCheckedTextDto default`()) == "null")
        check(codec.encodeToString<NullCheckedEnumDto>(`NullCheckedEnumDto default`()) == "null")
        val missing = codec.decodeFromString<CheckedWrapperHolderDto>("{}")
        check(missing.text == null && missing.number == null)
        val explicit = codec.decodeFromString<CheckedWrapperHolderDto>("""{"text":null,"number":null}""")
        check(explicit.text == null && explicit.number == null)
        val supplied = codec.decodeFromString<CheckedWrapperHolderDto>("""{"text":"wire","number":1}""")
        check(supplied.text == "wire" && supplied.number == 1)
    }
}
