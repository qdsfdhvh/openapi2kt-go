@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.serializer
import org.example.nulldefaults.models.HolderDto
import org.example.nulldefaults.models.NilBagsDto
import org.example.nulldefaults.models.NilBinaryDto
import org.example.nulldefaults.models.NilByteRowsDto
import org.example.nulldefaults.models.NilBytesDto
import org.example.nulldefaults.models.NilCountDto
import org.example.nulldefaults.models.NilDayDto
import org.example.nulldefaults.models.NilFlagDto
import org.example.nulldefaults.models.NilModeDto
import org.example.nulldefaults.models.NilNameDto
import org.example.nulldefaults.models.NilNamesDto
import org.example.nulldefaults.models.NilObjectDto
import org.example.nulldefaults.models.NilRatioDto
import org.example.nulldefaults.models.NilStampDto
import org.example.nulldefaults.models.`NilBagsDto default`
import org.example.nulldefaults.models.`NilBinaryDto default`
import org.example.nulldefaults.models.`NilByteRowsDto default`
import org.example.nulldefaults.models.`NilBytesDto default`
import org.example.nulldefaults.models.`NilCountDto default`
import org.example.nulldefaults.models.`NilDayDto default`
import org.example.nulldefaults.models.`NilFlagDto default`
import org.example.nulldefaults.models.`NilModeDto default`
import org.example.nulldefaults.models.`NilNameDto default`
import org.example.nulldefaults.models.`NilNamesDto default`
import org.example.nulldefaults.models.`NilRatioDto default`
import org.example.nulldefaults.models.`NilStampDto default`

private inline fun <reified T> componentNullRoundTrip(codec: Json, value: T) {
    check(value == null)
    check(serializer<T>().descriptor.isNullable)
    check(codec.encodeToJsonElement(serializer<T>(), value) === JsonNull)
    check(codec.decodeFromJsonElement(serializer<T>(), JsonNull) == null)
}

internal fun componentNullDefaultsContracts(codec: Json) {
    componentNullRoundTrip<NilNameDto>(codec, `NilNameDto default`())
    componentNullRoundTrip<NilFlagDto>(codec, `NilFlagDto default`())
    componentNullRoundTrip<NilCountDto>(codec, `NilCountDto default`())
    componentNullRoundTrip<NilRatioDto>(codec, `NilRatioDto default`())
    componentNullRoundTrip<NilDayDto>(codec, `NilDayDto default`())
    componentNullRoundTrip<NilStampDto>(codec, `NilStampDto default`())
    componentNullRoundTrip<NilBytesDto>(codec, `NilBytesDto default`())
    componentNullRoundTrip<NilBinaryDto>(codec, `NilBinaryDto default`())
    componentNullRoundTrip<NilNamesDto>(codec, `NilNamesDto default`())
    componentNullRoundTrip<NilBagsDto>(codec, `NilBagsDto default`())
    componentNullRoundTrip<NilByteRowsDto>(codec, `NilByteRowsDto default`())
    componentNullRoundTrip<NilModeDto?>(codec, `NilModeDto default`())
    componentNullRoundTrip<NilObjectDto?>(codec, NilObjectDto.default())

    // The physical enum classifier, vocabulary and open unknown codec remain.
    check(!serializer<NilModeDto>().descriptor.isNullable)
    check(codec.encodeToJsonElement(serializer<NilModeDto>(), NilModeDto.A) == JsonPrimitive("a"))
    val unknown = codec.decodeFromJsonElement(serializer<NilModeDto>(), JsonPrimitive("future"))
    check(unknown is NilModeDto.Unrecognized && unknown.serialName == "future")
    val textNull = codec.decodeFromJsonElement(serializer<NilModeDto?>(), JsonPrimitive("null"))
    check(textNull != null && textNull.serialName == "null")
    check(codec.encodeToJsonElement(serializer<NilModeDto?>(), textNull) == JsonPrimitive("null"))

    val holder = HolderDto(nilName = `NilNameDto default`(), nilMode = `NilModeDto default`(), nilBytes = `NilBytesDto default`(), nilNames = `NilNamesDto default`())
    val wire = codec.encodeToJsonElement(HolderDto.serializer(), holder) as JsonObject
    for (key in listOf("nilName", "nilMode", "nilBytes", "nilNames")) {
        check(wire.containsKey(key) == (codec.configuration.encodeDefaults && codec.configuration.explicitNulls))
        if (wire.containsKey(key)) check(wire[key] === JsonNull)
    }
    val missing = codec.decodeFromString(HolderDto.serializer(), "{}")
    check(missing.nilName == null && missing.nilMode == null && missing.nilBytes == null && missing.nilNames == null)
    val supplied = codec.decodeFromString(HolderDto.serializer(), """{"nilName":"given","nilMode":"future","nilBytes":[-128,127],"nilNames":["one"]}""")
    check(supplied.nilName == "given" && supplied.nilMode is NilModeDto.Unrecognized)
    check(supplied.nilBytes.contentEquals(byteArrayOf(-128,127)) && supplied.nilNames == listOf("one"))
    println("Component null factories aliases/enum/bytes/collections Json defaults=${codec.configuration.encodeDefaults} nulls=${codec.configuration.explicitNulls} passed")
}
