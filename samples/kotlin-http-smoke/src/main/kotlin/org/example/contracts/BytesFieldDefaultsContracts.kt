@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.bytesfielddefaults.models.HolderDto

private fun bytesDefaultsReject(block: () -> Unit) {
    try {
        block()
        error("Expected bytes default typed failure")
    } catch (_: SerializationException) {
    }
}

// The bytes default wire is the existing signed-integer array form, so a
// defaulted value must survive a Json round-trip by content.
suspend fun bytesFieldDefaultsContracts(checked: Boolean = false) {
    val provider = HolderDto.Provider(required = byteArrayOf(9))
    val value = HolderDto(requiredBytes = byteArrayOf(-128, -1, 0, 127), provider = provider)

    check(value.optionalBytes!!.contentEquals(byteArrayOf(4))) { "binary default lost: ${value.optionalBytes?.toList()}" }
    check(value.emptyBytes!!.isEmpty()) { "empty bytes default lost" }
    check(value.nullableBytes == null) { "null bytes default changed" }
    check(value.nullableDefault!!.contentEquals(byteArrayOf(1, 2))) { "nullable bytes default lost" }

    // A required bytes field keeps no initializer, so its default is validated
    // but never silently substituted for a caller value.
    check(value.requiredBytes.contentEquals(byteArrayOf(-128, -1, 0, 127)))
    check(provider.required.contentEquals(byteArrayOf(9)))

    // A default is a constructor initializer, so each construction owns a fresh
    // array rather than a shared mutable instance.
    val again = HolderDto(requiredBytes = byteArrayOf(1))
    check(value.optionalBytes !== again.optionalBytes) { "Bytes default shared one array instance" }
    check(value.emptyBytes !== again.emptyBytes) { "Empty bytes default shared one array instance" }

    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("""{"requiredBytes":[-128,-1,0,127]}""")
        check(missing.optionalBytes!!.contentEquals(byteArrayOf(4)))
        check(missing.emptyBytes!!.isEmpty())
        check(missing.nullableBytes == null)
        check(missing.nullableDefault!!.contentEquals(byteArrayOf(1, 2)))

        // A supplied value still wins over the default.
        val supplied = codec.decodeFromString<HolderDto>("""{"requiredBytes":[7],"optionalBytes":[5],"nullableDefault":null}""")
        check(supplied.requiredBytes.contentEquals(byteArrayOf(7)))
        check(supplied.optionalBytes!!.contentEquals(byteArrayOf(5)))
        check(supplied.nullableDefault == null)

        val tree = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        val fresh = codec.encodeToJsonElement(HolderDto.serializer(), value) as JsonObject
        // A non-null bytes default is always written. The plugin compares against a
        // freshly evaluated byteArrayOf(...), and ByteArray equality is referential,
        // so the comparison never matches and encodeDefaults cannot suppress it.
        for ((label, encoded) in listOf("decoded" to tree, "constructed" to fresh)) {
            check(encoded["optionalBytes"] == codec.parseToJsonElement("[4]")) {
                "$label bytes default missing from the signed integer array wire: ${encoded["optionalBytes"]}"
            }
            check(encoded["emptyBytes"] == JsonArray(emptyList())) { "$label empty bytes default wire changed" }
            check(encoded["requiredBytes"] == codec.parseToJsonElement("[-128,-1,0,127]"))
        }
        // A null default is comparable, so it follows the ordinary optional-null
        // policy instead: present only when defaults and explicit nulls are both on.
        check(tree["nullableBytes"] == (if (defaults && nulls) JsonNull else null)) {
            "Null bytes default did not follow the optional-null policy"
        }

        val roundTrip = codec.decodeFromString<HolderDto>(codec.encodeToString(missing))
        check(roundTrip.optionalBytes!!.contentEquals(missing.optionalBytes!!))
        check(roundTrip.emptyBytes!!.isEmpty() && roundTrip.nullableBytes == null)

        // The generated validator keeps the wire strict: no base64 text and no
        // out-of-range signed byte.
        for (bad in listOf(
            """{"requiredBytes":[128]}""",
            """{"requiredBytes":[-129]}""",
            """{"requiredBytes":"AAEC"}""",
            """{"requiredBytes":[1.5]}""",
            """{"requiredBytes":[null]}""",
        )) bytesDefaultsReject { codec.decodeFromString<HolderDto>(bad) }

        // A nested object default is separate: its own required bytes member is
        // still mandatory, while the field default above never leaks into it.
        bytesDefaultsReject { codec.decodeFromString<HolderDto>("""{"requiredBytes":[1],"provider":{}}""") }
        val nested = codec.decodeFromString<HolderDto>("""{"requiredBytes":[1],"provider":{"required":[3]}}""")
        check(nested.provider!!.required.contentEquals(byteArrayOf(3)))

        bytesDefaultTreeContracts(codec, checked)
        println("BytesFieldDefaults Json defaults=$defaults nulls=$nulls signed-array-wire-fresh-instance-required-no-initializer passed")
    }
}
