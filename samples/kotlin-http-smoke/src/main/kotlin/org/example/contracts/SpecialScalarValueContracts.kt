package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.datetime.LocalDate
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import org.example.specialvalues.ApiClient
import org.example.specialvalues.models.DayVocabularyDto
import org.example.specialvalues.models.ValueDto
import kotlin.time.Instant

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun specialScalarValueContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // Enum fields remain string vocabularies, not date/byte validators.
        val value = ValueDto(DayVocabularyDto.fromSerialName("not-a-calendar-date"),
            ValueDto.Tokens.fromSerialName("unknown-token"), listOf(ValueDto.StampsInner.fromSerialName("not-an-instant")))
        val tree = codec.encodeToJsonElement(ValueDto.serializer(), value).jsonObject
        check(tree["day"] == JsonPrimitive("not-a-calendar-date"))
        check(tree["tokens"] == JsonPrimitive("unknown-token"))
        check(tree["stamps"] == JsonArray(listOf(JsonPrimitive("not-an-instant"))))
        check(tree["fixedDay"] == JsonPrimitive("2024-02-29"))
        check(tree["fixedStamp"] == JsonPrimitive("2024-02-29T12:34:56Z"))
        check(tree["nativeStamp"] == JsonPrimitive("2024-02-29T12:34:56Z"))
        // Const body generation returns before field-only bigint annotation.
        check(tree["fixedBig"] == JsonPrimitive(Long.MAX_VALUE))
        val back = codec.decodeFromJsonElement(ValueDto.serializer(), tree)
        check(back.day == value.day && back.fixedBig == Long.MAX_VALUE)
        val bare = codec.decodeFromJsonElement(ValueDto.serializer(), JsonObject(tree.filterKeys { !it.startsWith("fixed") && it != "nativeStamp" }))
        check(bare.fixedDay == LocalDate.parse("2024-02-29") && bare.fixedStamp == Instant.parse("2024-02-29T12:34:56Z"))
        val changed = codec.decodeFromJsonElement(ValueDto.serializer(), JsonObject(tree + ("fixedDay" to JsonPrimitive("2025-01-01"))))
        check(changed.fixedDay == LocalDate.parse("2025-01-01")) // not const membership enforcement
        check(codec.encodeToJsonElement(ValueDto.serializer(), changed).jsonObject["fixedDay"] == JsonPrimitive("2025-01-01"))
        val day = LocalDate.parse("2025-01-01")
        val stamp = Instant.parse("2025-01-01T00:00:00Z")
        val bytes = byteArrayOf(-1, 0, 127)
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            val payload = (request.body as OutgoingContent.ByteArrayContent).bytes()
            when (calls) {
                1 -> {
                    check(codec.parseToJsonElement(payload.decodeToString()) == tree)
                    respond(codec.encodeToString(ValueDto.serializer(), value), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                2 -> {
                    check(payload.decodeToString() == "\"2025-01-01\"")
                    respond("\"2025-01-01\"", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                3 -> {
                    check(payload.decodeToString() == "\"2025-01-01T00:00:00Z\"")
                    respond("\"2025-01-01T00:00:00Z\"", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                4 -> {
                    check(payload.contentEquals(bytes))
                    respond(bytes, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                else -> error("Unexpected special value request")
            }
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http).default
            check(api.values(value).fixedDay == LocalDate.parse("2024-02-29"))
            check(api.day(day) == day)
            check(api.stamp(stamp) == stamp)
            check(api.bytes(bytes).contentEquals(bytes))
            check(calls == 4)
            println("Special scalar value contracts passed: defaults=$defaults/nulls=$nulls; string vocabulary vs simple formatted enum scalar, typed date const defaults and numeric bigint const; $calls requests")
        } finally { http.close(); engine.close() }
    }
}
