@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
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
