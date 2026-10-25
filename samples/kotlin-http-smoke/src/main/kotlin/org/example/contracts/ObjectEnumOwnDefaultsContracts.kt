package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.refwrapper.models.AllEnumNarrowDto
import org.example.refwrapper.models.AllEnumTargetDto
import org.example.refwrapper.models.EnumContainerDto
import org.example.refwrapper.models.EnumHolderDto
import org.example.refwrapper.models.EnumNarrowDto
import org.example.refwrapper.models.EnumTargetDto
import org.example.refwrapper.models.`AllEnumAliasDto default`
import org.example.refwrapper.models.`AllEnumNarrowDto default`
import org.example.refwrapper.models.`EnumAliasDto default`
import org.example.refwrapper.models.`EnumNarrowDto default`

fun objectEnumOwnDefaultsContracts(codec: Json) {
    val original = EnumTargetDto.default()
    val narrowed: EnumNarrowDto = `EnumNarrowDto default`()
    check(codec.encodeToJsonElement(original.mode) == JsonPrimitive("off"))
    check(original.count == 1)
    check(codec.encodeToJsonElement(narrowed.mode) == JsonPrimitive("on"))
    check(narrowed.count == 7)
    check(`EnumAliasDto default`() == null)
    check(narrowed !== `EnumNarrowDto default`())
    check(codec.encodeToJsonElement(codec.decodeFromString<EnumTargetDto>("{}").mode) == JsonPrimitive("on"))
    val unknown = codec.decodeFromString<EnumTargetDto>("""{"mode":"future"}""")
    check(codec.encodeToJsonElement(unknown.mode) == JsonPrimitive("future"))
    check(codec.decodeFromString<EnumTargetDto>(codec.encodeToString(original)) == original)

    val combined = AllEnumTargetDto.default()
    val combinedNarrowed: AllEnumNarrowDto = `AllEnumNarrowDto default`()
    check(codec.encodeToJsonElement(combined.mode) == JsonPrimitive("off"))
    check(codec.encodeToJsonElement(combinedNarrowed.mode) == JsonPrimitive("on"))
    check(combined.count == 7 && combinedNarrowed.count == 7)
    check(`AllEnumAliasDto default`() == null)
    check(combinedNarrowed !== `AllEnumNarrowDto default`())
    check(codec.decodeFromString<AllEnumTargetDto>(codec.encodeToString(combined)) == combined)

    val missing = codec.decodeFromString<EnumHolderDto>("{}")
    check(missing.plain == null && missing.combined == null)
    val supplied = EnumHolderDto(plain = narrowed, combined = combinedNarrowed)
    check(codec.decodeFromString<EnumHolderDto>(codec.encodeToString(supplied)) == supplied)
    val nested = EnumContainerDto.default()
    check(codec.encodeToJsonElement(nested.box?.mode) == JsonPrimitive("on"))
    check(nested.box !== EnumContainerDto.default().box)
}
