@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.rawjsondefaults.models.FreeArrayDto
import org.example.rawjsondefaults.models.FreeDto
import org.example.rawjsondefaults.models.FreeScalarDto
import org.example.rawjsondefaults.models.HolderDto
import org.example.rawjsondefaults.models.OpenDto
import org.example.rawjsondefaults.models.`FreeArrayDto default`
import org.example.rawjsondefaults.models.`FreeDto default`
import org.example.rawjsondefaults.models.`FreeScalarDto default`
import org.example.rawjsondefaults.models.`OpenDto default`

// A raw JSON carrier component root now owns an opt-in factory, using the same
// grammar as the field role: an open dictionary yields JsonObject, and an
// unconstrained/empty object yields JsonElement.
suspend fun rawJSONComponentDefaultsContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }

        // The open dictionary factory carries its own object value.
        val open: JsonObject = `OpenDto default`()
        check(open["a"] == JsonPrimitive(1) && open["b"] == JsonPrimitive("x")) { "open object lost: $open" }

        // An unconstrained root accepts any JSON value.
        val free: JsonElement = `FreeDto default`()
        check((free as JsonObject)["nested"] is JsonObject)
        val array: JsonElement = `FreeArrayDto default`()
        check(array is JsonArray && array.size == 3 && array[0] == JsonPrimitive(1))
        val scalar: JsonElement = `FreeScalarDto default`()
        check(scalar == JsonPrimitive("raw"))

        // A field referencing a defaulted carrier keeps its neutral presence.
        val holder = HolderDto()
        check(holder.open == null) { "Component default leaked into a referencing field" }
        check(holder.free == null) { "Component default leaked into a referencing field" }
        check(holder.inline?.get("k") == JsonPrimitive(2))

        // The factories round-trip through the same wire the fields use.
        check(codec.encodeToJsonElement(open) == codec.parseToJsonElement("""{"a":1,"b":"x"}"""))
        check(codec.encodeToJsonElement(`FreeArrayDto default`()) == codec.parseToJsonElement("""[1,"two",false]"""))
        check(codec.encodeToJsonElement(`FreeScalarDto default`()) == codec.parseToJsonElement("\"raw\""))
        val decoded = codec.decodeFromString<FreeDto>(codec.encodeToString(`FreeDto default`()))
        check(decoded == free)

        // A supplied value still wins over the factory.
        val supplied = codec.decodeFromString<OpenDto>("""{"z":true}""")
        check(supplied["z"] == JsonPrimitive(true))

        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.open == null && missing.free == null)

        println("RawJSONComponentDefaults Json defaults=$defaults nulls=$nulls passed")
    }
}
