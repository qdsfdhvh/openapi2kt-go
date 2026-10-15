package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import org.example.primitives.models.Int32 as IntAlias
import org.example.primitives.models.Int64 as LongAlias
import org.example.primitives.models.Record
import org.example.primitives_suffix.models.Int64Dto as SuffixedLongAlias
import org.example.primitives_suffix.models.RecordDto
import org.example.primitive_limits.models.Boxed
import org.example.primitive_limits.models.Wide

private inline fun <reified T> rejects(json: Json, input: String) {
    try {
        json.decodeFromString<T>(input)
        error("Invalid primitive input unexpectedly decoded")
    } catch (_: SerializationException) {
        // Primitive aliases retain the standard serializer's range checks.
    }
}

fun main() {
    check(Json.encodeToString<LongAlias>(Long.MAX_VALUE) == "9223372036854775807")
    check(Json.encodeToString<SuffixedLongAlias>(Long.MIN_VALUE) == "-9223372036854775808")
    rejects<IntAlias>(Json, "2147483648")
    rejects<LongAlias>(Json, "9223372036854775808")
    val input = """{"text":"hello","flag":true,"i32":2147483647,"i64":9223372036854775807,"f32":1.25,"f64":-0.5,"values":[1,null,-2],"id":-9223372036854775808}"""
    for (encodeDefaults in listOf(false, true)) {
        for (explicitNulls in listOf(false, true)) {
            val json = Json { this.encodeDefaults = encodeDefaults; this.explicitNulls = explicitNulls }
            val value = json.decodeFromString<Record>(input)
            check(value.i64 == Long.MAX_VALUE && value.id == Long.MIN_VALUE)
            check(value.i32 == Int.MAX_VALUE && value.flag && value.text == "hello")
            check(value.f32 == 1.25f && value.f64 == -0.5)
            check(value.values == listOf(1L, null, -2L))
            check(value.optionalFlag == null)
            check(json.decodeFromString<Record>(json.encodeToString(value)) == value)
            val suffixed = json.decodeFromString<RecordDto>(input)
            check(suffixed.i64 == value.i64)
            check(json.decodeFromString<RecordDto>(json.encodeToString(suffixed)) == suffixed)
            val wideInput = JsonObject((0 until 120).associate { "field$it" to JsonPrimitive(Long.MAX_VALUE) })
            val wide = json.decodeFromJsonElement<Wide>(wideInput)
            check(json.encodeToJsonElement(wide) == wideInput)
            val boxedInput = JsonObject((0 until 240).associate {
                "field$it" to if (it % 2 == 0) JsonNull else JsonPrimitive(Long.MIN_VALUE)
            })
            val boxed = json.decodeFromJsonElement<Boxed>(boxedInput)
            val boxedEncoded = json.encodeToJsonElement(boxed) as JsonObject
            check(boxedEncoded.values.count { it == JsonNull } == if (encodeDefaults && explicitNulls) 120 else 0)
            check(json.decodeFromJsonElement<Boxed>(boxedEncoded) == boxed)
            rejects<Wide>(json, "{}")
        }
    }
    println("Generated primitive contracts passed: aliases/ref/suffix, numeric boundaries, nullable list items, and 120-wide/240-boxed constructor boundaries.")
}
