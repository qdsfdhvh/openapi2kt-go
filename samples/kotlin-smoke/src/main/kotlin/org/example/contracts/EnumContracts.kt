package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.enum_limits.models.BoundaryStatus
import org.example.enums.models.Record
import org.example.enums.models.Status
import org.example.enums_suffix.models.StatusDto
import org.example.enum_naming.models.Vocabulary
import org.example.enum_naming.models.TOKEN as OwnerToken
import org.example.enum_naming.models.Record as NamingRecord

private inline fun <reified T> rejects(json: Json, input: String) {
    try {
        json.decodeFromString<T>(input)
        error("Invalid enum input unexpectedly decoded")
    } catch (_: SerializationException) {
        // Expected: do not accept numbers/booleans/objects as enum strings.
    }
}

private fun verifyEnumNaming() {
    val known = listOf(
        "active" to Vocabulary.ACTIVE,
        "camelCase" to Vocabulary.CAMELCASE,
        "status" to Vocabulary.STATUS,
        "unknown" to Vocabulary.UNKNOWN,
        "Unrecognized" to Vocabulary.UNRECOGNIZED_VALUE,
        "Serializer" to Vocabulary.SERIALIZER_VALUE,
        "companion" to Vocabulary.COMPANION_VALUE,
        "high-priority" to Vocabulary.HIGH_PRIORITY,
        "high priority" to Vocabulary.HIGH_PRIORITY_VALUE,
        "high_priority" to Vocabulary.HIGH_PRIORITY_VALUE2,
        "1" to Vocabulary._1,
        ">=" to Vocabulary.GREATER_EQUAL,
        "" to Vocabulary.VALUE,
        " " to Vocabulary.VALUE_VALUE,
        "_" to Vocabulary.U005F,
        "☃" to Vocabulary.U2603,
        "🚀" to Vocabulary.UD83D_UDE80,
        "\u0085" to Vocabulary.U0085,
        "serialName" to Vocabulary.SERIALNAME,
        "fromSerialName" to Vocabulary.FROMSERIALNAME,
        "entries" to Vocabulary.ENTRIES,
    )
    // The singleton references above intentionally precede any companion/serializer access.
    check(Vocabulary.entries == known.map { it.second })
    check(Vocabulary.entries === Vocabulary.entries)
    for ((wire, value) in known) {
        check(Vocabulary.Serializer.fromSerialName(wire) === value)
        check(Vocabulary.fromSerialName(wire) === value)
        check(Json.decodeFromString<Vocabulary>(Json.encodeToString<Vocabulary>(value)) === value)
        check(Json.parseToJsonElement(Json.encodeToString<Vocabulary>(value)) == JsonPrimitive(wire))
    }
    check(Vocabulary.Serializer.fromSerialName("future") == Vocabulary.Unrecognized("future"))
    check(OwnerToken.Serializer.fromSerialName("TOKEN") === OwnerToken.TOKEN)
    val record = NamingRecord(
        vocabulary = Vocabulary.GREATER_EQUAL,
        operator = NamingRecord.Operator.GREATER,
        tokens = listOf(NamingRecord.TokensInner.A_B, NamingRecord.TokensInner.A_B_VALUE, NamingRecord.TokensInner.A_B_VALUE2),
    )
    check(NamingRecord.Operator.Serializer.fromSerialName(">") === record.operator)
    check(NamingRecord.TokensInner.Serializer.fromSerialName("a b") === record.tokens[1])
    check(Json.decodeFromString<NamingRecord>(Json.encodeToString(record)) == record)
}

fun main() {
    verifyEnumNaming()
    check(Status.Serializer.fromSerialName("ready") === Status.READY)
    check(Status.fromSerialName("in-progress") === Status.IN_PROGRESS)
    check(Status.fromSerialName("1") === Status._1)
    check(Status.fromSerialName("quote\"$\\line\n") === Status.QUOTE___LINE_)
    check(Status.entries.map { it.serialName } == listOf("ready", "in-progress", "1", "quote\"$\\line\n"))
    check(Status.Unrecognized("future") == Status.Unrecognized("future"))
    check(Status.Unrecognized("future").hashCode() == Status.Unrecognized("future").hashCode())
    check(Status.Unrecognized("future") != Status.Unrecognized("other"))
    check(Status.Serializer.descriptor.serialName == "org.example.enums.models.Status")
    check(Record.Mode.Serializer.descriptor.serialName == "org.example.enums.models.Record.Mode")
    check(Record.Nested.Mode.Serializer.descriptor.serialName == "org.example.enums.models.Record.Nested.Mode")
    check(BoundaryStatus.entries.size == 240)
    for (index in 0 until 240) {
        check(BoundaryStatus.fromSerialName("v$index") === BoundaryStatus.entries[index])
        check(Json.decodeFromString<BoundaryStatus>("\"v$index\"") === BoundaryStatus.entries[index])
    }
    check(StatusDto.Serializer.fromSerialName("ready") === StatusDto.READY)
    check(Json.decodeFromString<StatusDto>("\"future\"") == StatusDto.Unrecognized("future"))

    for (encodeDefaults in listOf(false, true)) {
        for (explicitNulls in listOf(false, true)) {
            val json = Json { this.encodeDefaults = encodeDefaults; this.explicitNulls = explicitNulls }
            check(json.configuration.encodeDefaults == encodeDefaults)
            check(json.configuration.explicitNulls == explicitNulls)
            for (wire in listOf("ready", "in-progress", "1", "quote\"$\\line\n", "future", "", "future🚀")) {
                val value = Status.fromSerialName(wire)
                val encoded = json.encodeToString<Status>(value)
                check(json.parseToJsonElement(encoded) == JsonPrimitive(wire))
                check(json.decodeFromString<Status>(encoded) == value)
            }
            val input = """{"status":"future","states":["on",null,"new-state"],"mode":"new-mode","nested":{"mode":"remote"}}"""
            val record = json.decodeFromString<Record>(input)
            check(record.status == Status.Unrecognized("future"))
            check(record.states == listOf(Record.StatesInner.ON, null, Record.StatesInner.Unrecognized("new-state")))
            check(record.mode == Record.Mode.Unrecognized("new-mode"))
            check(record.nested?.mode === Record.Nested.Mode.REMOTE)
            check(record.optionalStatus == null)
            val encoded = json.parseToJsonElement(json.encodeToString(record)) as JsonObject
            check(encoded["status"] == JsonPrimitive("future"))
            check(encoded["states"] == JsonArray(listOf(JsonPrimitive("on"), JsonNull, JsonPrimitive("new-state"))))
            check(json.decodeFromString<Record>(json.encodeToString(record)) == record)
            rejects<Record>(json, """{"states":[],"mode":"fast"}""")
            for (invalid in listOf("1", "true", "false", "null", "{}", "[]")) {
                rejects<Status>(json, invalid)
            }
        }
    }
    check(Json.decodeFromString<Status?>("null") == null)
    println("Generated enum contracts passed: known/unknown strings, literals, nested/array/ref/suffix scopes, defaults/null configurations and invalid wire shapes.")
}
