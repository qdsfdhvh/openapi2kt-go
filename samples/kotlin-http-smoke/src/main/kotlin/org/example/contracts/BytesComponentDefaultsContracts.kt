@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement
import org.example.bytescomponentdefaults.models.BlobDto
import org.example.bytescomponentdefaults.models.EmptyBytesDto
import org.example.bytescomponentdefaults.models.HolderDto
import org.example.bytescomponentdefaults.models.SortKeyDto
import org.example.bytescomponentdefaults.models.`SortKeyDto default`
import org.example.bytescomponentdefaults.models.`BlobDto default`
import org.example.bytescomponentdefaults.models.`EmptyBytesDto default`

private fun bytesComponentReject(block: () -> Unit) {
    try {
        block()
        error("Expected a component bytes failure")
    } catch (_: SerializationException) {
    }
}

// A component root owns an opt-in factory; the consuming field does not inherit
// it. Both halves are asserted here.
suspend fun bytesComponentDefaultsContracts() {
    check(`SortKeyDto default`().contentEquals(byteArrayOf(-128, -1, 0, 127))) {
        "Signed byte bounds lost: ${`SortKeyDto default`().toList()}"
    }
    check(`BlobDto default`().contentEquals(byteArrayOf(4)))
    check(`EmptyBytesDto default`().isEmpty())

    // Each call evaluates the factory again, so no caller shares one array.
    check(`SortKeyDto default`() !== `SortKeyDto default`()) { "Component factory shared one array instance" }

    val holder = HolderDto()
    // A field that references a component keeps its neutral presence: the
    // component default never leaks into an unrelated call site.
    check(holder.key == null) { "Component default leaked into a referencing field" }
    check(holder.blob == null) { "Component default leaked into a referencing field" }
    // An inline field with its own default keeps the field role.
    check(holder.inline!!.contentEquals(byteArrayOf(1, 2)))

    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }

        // The factories decode through the component's own serializer, so a
        // value survives a Json round-trip and matches the field wire.
        val key = `SortKeyDto default`()
        check(codec.decodeFromString<SortKeyDto>(codec.encodeToString(key)).contentEquals(key))

        // The factory output is written as the signed integer array wire.
        val tree = codec.encodeToJsonElement(key)
        check(tree == codec.parseToJsonElement("[-128,-1,0,127]")) { "Component bytes wire changed: $tree" }
        check(codec.encodeToJsonElement(`BlobDto default`()) == codec.parseToJsonElement("[4]"))
        check(codec.encodeToJsonElement(`EmptyBytesDto default`()) == codec.parseToJsonElement("[]"))

        // Supplied values still win over the factory.
        val supplied = codec.decodeFromString<SortKeyDto>("[7]")
        check(supplied.contentEquals(byteArrayOf(7)))

        // The wire stays strict: no base64 text and no out-of-range byte.
        for (bad in listOf("[128]", "[-129]", "\"AAEC\"", "[1.5]", "[null]")) {
            bytesComponentReject { codec.decodeFromString<SortKeyDto>(bad) }
        }

        // The referencing field keeps ordinary optional-null behavior.
        val decoded = codec.decodeFromString<HolderDto>("{}")
        check(decoded.key == null && decoded.blob == null)
        check(decoded.inline!!.contentEquals(byteArrayOf(1, 2)))
        val explicit = codec.decodeFromString<HolderDto>("""{"key":[9],"blob":[3],"inline":[5]}""")
        check(explicit.key!!.contentEquals(byteArrayOf(9)))
        check(explicit.blob!!.contentEquals(byteArrayOf(3)))
        check(explicit.inline!!.contentEquals(byteArrayOf(5)))

        println("BytesComponentDefaults Json defaults=$defaults nulls=$nulls factory-signed-array-no-field-inheritance passed")
    }
}
