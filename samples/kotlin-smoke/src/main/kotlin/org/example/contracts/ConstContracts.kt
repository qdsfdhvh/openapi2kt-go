package org.example.contracts

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*
import org.example.const_fields.models.Flag
import org.example.const_fields.models.Record
import org.example.const_fields_suffix.models.FlagDto
import org.example.const_fields_suffix.models.RecordDto
import org.example.const_limits.models.ConstWide

private fun rejectsConstWire(block: () -> Unit) {
    try {
        block()
        error("Invalid const field wire shape was accepted")
    } catch (_: SerializationException) {
        // Wrong mapped scalar shape, not const membership validation.
    }
}

@OptIn(ExperimentalSerializationApi::class)
fun main() {
    // Constants are not named constructor arguments, even when optional/nullable.
    val record = Record(id = "identifier", nested = Record.Nested(), flags = listOf(Record.FlagsInner()), ref = Flag())
    check(record.kind == "note" && record.numericText == "1")
    check(record.double == 1.0 && record.float == 1.0f && record.flag)
    check(record.low == Long.MIN_VALUE && record.high == Long.MAX_VALUE)
    check(record.minInt == Int.MIN_VALUE && record.maxInt == Int.MAX_VALUE)
    check(record.fraction == 1.25 && !record.booleanLiteral)
    check(record.negativeZero.toBits() == (-0.0).toBits())
    check(record.quote == "line\"\n$\\")
    check(record.copy(id = "other").kind == "note")
    check(Flag() !== Flag()) // Ordinary const-only model is a class, not a singleton.
    check(Flag() != Flag()) // No invented structural equality for a plain class.

    val minimal = """{"id":"identifier","nested":{},"flags":[{}],"ref":{}}"""
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val encoded = json.encodeToJsonElement(record).jsonObject
        check(encoded["kind"] == JsonPrimitive("note"))
        check(encoded["numericText"] == JsonPrimitive("1"))
        check(encoded["double"] == JsonPrimitive(1.0))
        check(encoded["float"] == JsonPrimitive(1.0f))
        check(encoded["flag"] == JsonPrimitive(true))
        check(encoded["count"] == JsonPrimitive(1))
        check(encoded["minInt"] == JsonPrimitive(Int.MIN_VALUE))
        check(encoded["maxInt"] == JsonPrimitive(Int.MAX_VALUE))
        check(encoded["fraction"] == JsonPrimitive(1.25))
        check(encoded["booleanLiteral"] == JsonPrimitive(false))
        check(encoded["negativeZero"] == JsonPrimitive(-0.0))
        check(encoded["low"] == JsonPrimitive(Long.MIN_VALUE))
        check(encoded["high"] == JsonPrimitive(Long.MAX_VALUE))
        check(encoded["quote"] == JsonPrimitive("line\"\n$\\"))
        check(encoded["nested"]!!.jsonObject["kind"] == JsonPrimitive("inner"))
        check(encoded["flags"]!!.jsonArray.single().jsonObject["kind"] == JsonPrimitive("item"))
        check(encoded["ref"]!!.jsonObject["kind"] == JsonPrimitive("flag"))
        check(("ordinary" in encoded) == (defaults && nulls))
        check(json.encodeToJsonElement(json.decodeFromString<Record>(json.encodeToString(record))) == encoded)
        val missing = json.decodeFromString<Record>(minimal)
        check(missing.kind == "note" && missing.double == 1.0 && missing.flag)
        check(missing.nested.kind == "inner" && missing.ref.kind == "flag")
        check(missing.flags.single().kind == "item")

        // Standard DTO decoding can supply another scalar value. EncodeDefault
        // is an encoding/default contract, not JSON Schema const enforcement.
        val other = json.decodeFromString<Flag>("""{"kind":"different"}""")
        check(other.kind == "different")
        check(json.encodeToJsonElement(other).jsonObject["kind"] == JsonPrimitive("different"))
        check(json.decodeFromString<Flag>("{}").kind == "flag")
        rejectsConstWire { json.decodeFromString<Flag>("""{"kind":null}""") }
        rejectsConstWire { json.decodeFromString<Record>(minimal.dropLast(1) + ",\"double\":\"not-number\"}") }

        val suffix = RecordDto(id = "identifier", nested = RecordDto.Nested(), flags = listOf(RecordDto.FlagsInner()), ref = FlagDto())
        check(json.encodeToJsonElement(suffix) == encoded)
        check(json.decodeFromString<RecordDto>(minimal).ref.kind == "flag")

        // Body fields still participate in the plugin's serialization constructor.
        val wide = ConstWide()
        val wideEncoded = json.encodeToJsonElement(wide).jsonObject
        check(wideEncoded.size == 120)
        for (i in 0 until 120) check(wideEncoded["field$i"] == JsonPrimitive(i.toLong()))
        check(json.encodeToJsonElement(json.decodeFromString<ConstWide>("{}")) == wideEncoded)
        check(json.encodeToJsonElement(json.decodeFromString<ConstWide>(json.encodeToString(wide))) == wideEncoded)
    }
    println("Const body, scope, suffix, defaults, decode limits and serialization constructor contracts passed")
}
