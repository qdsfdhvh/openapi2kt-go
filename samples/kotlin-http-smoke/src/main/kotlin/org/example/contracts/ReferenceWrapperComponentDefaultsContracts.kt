@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.refwrapper.models.HolderDto
import org.example.refwrapper.models.WrappedBlobDto
import org.example.refwrapper.models.WrappedDayDto
import org.example.refwrapper.models.ModeDto
import org.example.refwrapper.models.WrappedModeDto
import org.example.refwrapper.models.WrappedObjectDto
import org.example.refwrapper.models.WrappedRowsDto
import org.example.refwrapper.models.`WrappedObjectDto default`
import org.example.refwrapper.models.`WrappedRowsDto default`
import org.example.refwrapper.models.`WrappedModeDto default`
import org.example.refwrapper.models.`WrappedDayDto default`
import org.example.refwrapper.models.`WrappedBlobDto default`

import org.example.refwrapper.models.RequiredHolderDto
import org.example.refwrapper.models.WrappedNestedDto
import org.example.refwrapper.models.WrappedRawDto
import org.example.refwrapper.models.WrappedNilRawDto
import org.example.refwrapper.models.WrappedNilObjectDto
import org.example.refwrapper.models.`WrappedNestedDto default`
import org.example.refwrapper.models.`WrappedRawDto default`
import org.example.refwrapper.models.`WrappedNilRawDto default`
import org.example.refwrapper.models.`WrappedNilObjectDto default`

import org.example.refwrapper.models.NullableObjectDto
import org.example.refwrapper.models.`NullableObjectDto default`
import org.example.refwrapper.models.NullObjectDto
import org.example.refwrapper.models.`NullObjectDto default`
import org.example.refwrapper.models.NullableCombinedDto
import org.example.refwrapper.models.`NullableCombinedDto default`
import org.example.refwrapper.models.NullCombinedDto
import org.example.refwrapper.models.`NullCombinedDto default`
import org.example.refwrapper.models.NullableNestedDto
import org.example.refwrapper.models.`NullableNestedDto default`
import org.example.refwrapper.models.NullNestedDto
import org.example.refwrapper.models.`NullNestedDto default`
import org.example.refwrapper.models.NullableModeDto
import org.example.refwrapper.models.`NullableModeDto default`
import org.example.refwrapper.models.NullModeDto
import org.example.refwrapper.models.`NullModeDto default`
import org.example.refwrapper.models.NullableDayDto
import org.example.refwrapper.models.`NullableDayDto default`
import org.example.refwrapper.models.NullDayDto
import org.example.refwrapper.models.`NullDayDto default`
import org.example.refwrapper.models.NullableBlobDto
import org.example.refwrapper.models.`NullableBlobDto default`
import org.example.refwrapper.models.NullBlobDto
import org.example.refwrapper.models.`NullBlobDto default`
import org.example.refwrapper.models.NullableRawDto
import org.example.refwrapper.models.`NullableRawDto default`
import org.example.refwrapper.models.NullRawDto
import org.example.refwrapper.models.`NullRawDto default`
import org.example.refwrapper.models.NullableOpenDto
import org.example.refwrapper.models.`NullableOpenDto default`
import org.example.refwrapper.models.NullOpenDto
import org.example.refwrapper.models.`NullOpenDto default`
import org.example.refwrapper.models.RequiredNullableDto

// A `$ref` wrapper is a canonical alias, so the alias owns the factory while a
// referencing field keeps its neutral presence. Both halves are asserted.
suspend fun referenceWrapperComponentDefaultsContracts() {
    val obj: WrappedObjectDto = `WrappedObjectDto default`()
    check(obj.z == "1") { "Wrapper object default lost: ${obj.z}" }
    val rows: WrappedRowsDto = `WrappedRowsDto default`()
    check(rows == listOf("x")) { "Wrapper collection default lost: $rows" }
    val mode: WrappedModeDto = `WrappedModeDto default`()
    // A typealias cannot reach nested classifiers, so the canonical owner names
    // the member; this is the documented access rule, not a wrapper restriction.
    check(mode == ModeDto.A) { "Wrapper enum default lost: $mode" }
    val day: WrappedDayDto = `WrappedDayDto default`()
    check(day == kotlinx.datetime.LocalDate.parse("2024-02-29")) { "Wrapper temporal default lost: $day" }
    val blob: WrappedBlobDto = `WrappedBlobDto default`()
    check(blob.contentEquals(byteArrayOf(-128, 127))) { "Wrapper bytes default lost: ${blob.toList()}" }

    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }

        chainedWrapperDefaults(codec)
        explicitNullableWrapperDefaults(codec)

        // A field that references a defaulted wrapper must NOT inherit the value.
        val holder = codec.decodeFromString<HolderDto>("{}")
        check(holder.`object` == null) { "Wrapper default leaked into a referencing field" }
        check(holder.rows == null) { "Wrapper default leaked into a referencing field" }
        check(holder.mode == null) { "Wrapper default leaked into a referencing field" }
        check(holder.day == null) { "Wrapper default leaked into a referencing field" }
        check(holder.blob == null) { "Wrapper default leaked into a referencing field" }
        check(holder.plain == null) { "Annotation-less alias leaked into a referencing field" }

        // A supplied value still overrides.
        val supplied = codec.decodeFromString<HolderDto>("""{"object":{"z":"9"},"mode":"b","rows":["y"]}""")
        check(supplied.`object`!!.z == "9")
        check(supplied.mode == ModeDto.B)
        check(supplied.rows == listOf("y"))

        // The factory output is canonical and survives a Json round-trip.
        val tree = codec.encodeToJsonElement(`WrappedObjectDto default`())
        check(tree.toString() == """{"z":"1"}""") { "Wrapper object wire changed: $tree" }
        check(codec.decodeFromString<WrappedObjectDto>(codec.encodeToString(`WrappedObjectDto default`())) == obj)

        // The runtime keeps each role's own documented policy. A String enum stays
        // open and surfaces an unknown wire member as Unrecognized; a DTO keeps its
        // required-member check. Bytes range is a generator-time check because
        // ByteArraySerializer narrows with toByte() instead of rejecting.
        check(codec.decodeFromString<WrappedModeDto>("\"zz\"") == ModeDto.Unrecognized("zz")) {
            "Wrapper enum stopped following the open runtime policy"
        }
        check(runCatching { codec.decodeFromString<WrappedObjectDto>("{}") }.isFailure) {
            "Wrapper object stopped enforcing a required member"
        }

        println("ReferenceWrapperComponentDefaults Json defaults=$defaults nulls=$nulls alias-factory-no-inheritance passed")
    }
}

private fun chainedWrapperDefaults(codec: Json) {
    val nested: WrappedNestedDto = `WrappedNestedDto default`()
    check(nested == listOf(mapOf("a" to 2, "b" to null), emptyMap()))
    val again = `WrappedNestedDto default`()
    check(nested !== again && nested[0] !== again[0])
    val blob = `WrappedBlobDto default`()
    blob[0] = 0
    check(`WrappedBlobDto default`()[0] == (-128).toByte())
    check(`WrappedObjectDto default`() !== `WrappedObjectDto default`())

    val raw: WrappedRawDto = `WrappedRawDto default`()
    check(raw == codec.parseToJsonElement("""{"${'$'}ref":"data","nil":null}"""))
    check(raw !== `WrappedRawDto default`())
    val nilRaw: WrappedNilRawDto = `WrappedNilRawDto default`()
    val nilObject: WrappedNilObjectDto = `WrappedNilObjectDto default`()
    check(nilRaw == null && nilObject == null)
    check(serializer<WrappedNilRawDto>().descriptor.isNullable)
    check(serializer<WrappedNilObjectDto>().descriptor.isNullable)
    check(codec.decodeFromString<WrappedNilRawDto>("null") == null)
    check(codec.decodeFromString<WrappedNilObjectDto>("null") == null)
    check(codec.decodeFromString<WrappedRawDto>("null") === JsonNull)
    check(codec.encodeToJsonElement(nilRaw) === JsonNull)
    check(codec.decodeFromString<WrappedNestedDto>(codec.encodeToString(nested)) == nested)
    check(codec.decodeFromString<WrappedRawDto>(codec.encodeToString(raw)) == raw)

    val missing = codec.decodeFromString<HolderDto>("{}")
    check(missing.nested == null && missing.raw == null && missing.nilRaw == null && missing.nilObject == null)
    val explicit = codec.decodeFromString<HolderDto>("""{"nested":null,"raw":null,"nilRaw":null,"nilObject":null}""")
    check(explicit.nested == null && explicit.raw == null && explicit.nilRaw == null && explicit.nilObject == null)
    val supplied = codec.decodeFromString<HolderDto>("""{"nested":[{"x":3}],"raw":"given","nilRaw":{"kept":null},"nilObject":{"z":"given"}}""")
    check(supplied.nested == listOf(mapOf("x" to 3)))
    check(supplied.raw == JsonPrimitive("given"))
    check(supplied.nilRaw?.get("kept") === JsonNull && supplied.nilObject?.z == "given")
    val adopted = HolderDto(nested = nested, raw = raw)
    val wire = codec.encodeToJsonElement(HolderDto.serializer(), adopted) as JsonObject
    check(wire["raw"] == raw && wire["nested"] == codec.parseToJsonElement("""[{"a":2,"b":null},{}]"""))
    check(runCatching { codec.decodeFromString<RequiredHolderDto>("{}") }.isFailure)
    check(runCatching { codec.decodeFromString<RequiredHolderDto>("""{"object":null,"raw":1,"nilRaw":null}""") }.isFailure)
    val required = RequiredHolderDto(`object` = `WrappedObjectDto default`(), raw = raw, nilRaw = nilRaw)
    val decoded = codec.decodeFromString<RequiredHolderDto>(codec.encodeToString(required))
    check(decoded.`object`.z == "1" && decoded.raw == raw && decoded.nilRaw == null)
}

private fun explicitNullableWrapperDefaults(codec: Json) {
    val obj: NullableObjectDto = `NullableObjectDto default`()
    check(obj?.z == "given")
    check(obj !== `NullableObjectDto default`())
    check(`NullableCombinedDto default`()?.z == "combined")
    check(`NullCombinedDto default`() == null)
    check(`NullObjectDto default`() == null)
    val nested: NullableNestedDto = `NullableNestedDto default`()
    check(nested == listOf(mapOf("a" to 2, "b" to null)))
    check(nested !== `NullableNestedDto default`())
    check(nested?.get(0) !== `NullableNestedDto default`()?.get(0))
    check(`NullNestedDto default`() == null)
    check(`NullableModeDto default`() == ModeDto.A)
    check(`NullModeDto default`() == null)
    check(codec.decodeFromString<NullableModeDto>("\"future\"") == ModeDto.Unrecognized("future"))
    check(`NullableDayDto default`() == kotlinx.datetime.LocalDate.parse("2024-02-29"))
    check(`NullDayDto default`() == null)
    val blob: NullableBlobDto = `NullableBlobDto default`()
    check(blob!!.contentEquals(byteArrayOf(-128, 127)))
    blob[0] = 0
    check(`NullableBlobDto default`()!![0] == (-128).toByte())
    check(`NullBlobDto default`() == null)
    check(`NullableRawDto default`() == codec.parseToJsonElement("""{"nil":null}"""))
    check(`NullableOpenDto default`()?.get("nil") === JsonNull)
    check(`NullRawDto default`() == null && `NullOpenDto default`() == null)
    check(serializer<NullObjectDto>().descriptor.isNullable)
    check(serializer<NullNestedDto>().descriptor.isNullable)
    check(serializer<NullModeDto>().descriptor.isNullable)
    check(serializer<NullDayDto>().descriptor.isNullable)
    check(serializer<NullBlobDto>().descriptor.isNullable)
    check(serializer<NullRawDto>().descriptor.isNullable)
    check(serializer<NullOpenDto>().descriptor.isNullable)
    check(codec.decodeFromString<NullableObjectDto>(codec.encodeToString(obj)) == obj)
    check(codec.decodeFromString<NullableNestedDto>(codec.encodeToString(nested)) == nested)
    check(codec.encodeToString<NullRawDto>(`NullRawDto default`()) == "null")
    check(codec.decodeFromString<NullObjectDto>("null") == null)
    check(codec.decodeFromString<NullRawDto>("null") == null)
    check(codec.decodeFromString<NullOpenDto>("null") == null)
    check(runCatching { codec.decodeFromString<NullableObjectDto>("{}") }.isFailure)
    val missing = codec.decodeFromString<HolderDto>("{}")
    check(missing.nullableObject == null && missing.nullableCombined == null && missing.nullableNested == null)
    check(missing.nullableMode == null && missing.nullableDay == null && missing.nullableBlob == null)
    check(missing.nullableRaw == null && missing.nullableOpen == null)
    val explicit = codec.decodeFromString<HolderDto>("""{"nullableObject":null,"nullableNested":null,"nullableMode":null,"nullableRaw":null}""")
    check(explicit.nullableObject == null && explicit.nullableNested == null && explicit.nullableMode == null && explicit.nullableRaw == null)
    val supplied = codec.decodeFromString<HolderDto>("""{"nullableObject":{"z":"wire"},"nullableNested":[{"a":3}],"nullableMode":"future","nullableRaw":null}""")
    check(supplied.nullableObject?.z == "wire" && supplied.nullableNested == listOf(mapOf("a" to 3)))
    check(supplied.nullableMode == ModeDto.Unrecognized("future") && supplied.nullableRaw == null)
    val required = RequiredNullableDto(value = `NullObjectDto default`())
    check(required.value == null)
    check(codec.decodeFromString<RequiredNullableDto>("""{"value":null}""").value == null)
    // Shared nullable component fields retain their existing nullable default.
    check(codec.decodeFromString<RequiredNullableDto>("{}").value == null)
    val wire = codec.encodeToJsonElement(HolderDto.serializer(), HolderDto(nullableObject = obj, nullableNested = nested)) as JsonObject
    check(wire["nullableObject"] == codec.parseToJsonElement("""{"z":"given"}"""))
    check(wire["nullableNested"] == codec.parseToJsonElement("""[{"a":2,"b":null}]"""))
}
