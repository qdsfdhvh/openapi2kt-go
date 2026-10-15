package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.nullable_composition.models.Node
import org.example.nullable_composition.models.Record
import org.example.nullable_composition.models.Status

private fun rejectsNullableComposition(json: Json, input: String) {
    try {
        json.decodeFromString<Record>(input)
        error("Invalid nullable composition input unexpectedly decoded")
    } catch (_: SerializationException) {
        // A nullable wrapper must retain the generated underlying serializer.
    }
}

fun main() {
    val empty = Record(items = listOf(true, null, false))
    val populated = Record(
        one = "text", any = Long.MAX_VALUE, ref = Node(name = "node"), list = listOf("item"),
        items = listOf(null, false), status = Status.Unrecognized("future"),
        nested = Record.Nested(label = "nested"),
    )
    for (encodeDefaults in listOf(false, true)) {
        for (explicitNulls in listOf(false, true)) {
            val json = Json { this.encodeDefaults = encodeDefaults; this.explicitNulls = explicitNulls }
            check(json.decodeFromString<Record>(json.encodeToString(populated)) == populated)
            val encoded = json.encodeToJsonElement(populated) as JsonObject
            check(encoded["any"] == JsonPrimitive(Long.MAX_VALUE))
            check(encoded["status"] == JsonPrimitive("future"))
            val emptyEncoded = json.encodeToJsonElement(empty) as JsonObject
            check(emptyEncoded.values.count { it == JsonNull } == if (encodeDefaults && explicitNulls) 6 else 0)
            check(json.decodeFromString<Record>(json.encodeToString(empty)) == empty)
            val explicit = """{"items":[true,null,false],"one":null,"any":null,"ref":null,"list":null,"status":null,"nested":null}"""
            check(json.decodeFromString<Record>(explicit) == empty)
            // Component nullable fields retain their pre-existing null defaults; required list does not.
            rejectsNullableComposition(json, "{}")
            rejectsNullableComposition(json, """{"items":null}""")
            for (invalid in listOf(
                """"one":[]""", """"any":1.5""", """"ref":true""",
                """"list":false""", """"status":42""", """"nested":true""",
            )) {
                rejectsNullableComposition(json, "{\"items\":[],$invalid}")
            }
            rejectsNullableComposition(json, """{"items":[7]}""")
        }
    }
    println("Generated nullable composition contracts passed: oneOf/anyOf, refs/enums/nested scopes, list items, missing/null/value and JSON configurations.")
}
