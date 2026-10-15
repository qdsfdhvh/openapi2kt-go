package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.put
import org.example.nativeenums.ApiClient
import org.example.nativeenums.api.models.BigVocabularyDto
import org.example.nativeenums.api.models.RecordInputDto
import org.example.nativeenums.models.Patch
import org.example.nativeenums.models.RecordDto
import org.example.nativeenums.models.VocabularyDto
import org.example.scalarvalues.Money
import kotlin.time.Instant

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun nativeEnumContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = RecordDto(
            VocabularyDto.fromSerialName("unknown-not-numeric"),
            RecordDto.Stamp.fromSerialName("unknown-not-temporal"),
            RecordDto.Money.fromSerialName("unknown-not-money"),
            listOf(RecordDto.ItemsInner.fromSerialName("unknown-item")),
            mapOf("key" to RecordDto.LookupValue.fromSerialName("unknown-map-value")))
        val tree = codec.encodeToJsonElement(RecordDto.serializer(), value).jsonObject
        check(tree["big"] == JsonPrimitive("unknown-not-numeric"))
        check(tree["stamp"] == JsonPrimitive("unknown-not-temporal"))
        check(tree["money"] == JsonPrimitive("unknown-not-money"))
        check(tree["items"] == JsonArray(listOf(JsonPrimitive("unknown-item"))))
        check(tree["lookup"] == buildJsonObject { put("key", "unknown-map-value") })
        check(codec.decodeFromJsonElement(RecordDto.serializer(), tree) == value)
        check(VocabularyDto.entries.single().serialName == "known-not-numeric")
        // A native bigint marker on a vocabulary is not a Long serializer.
        // The vocabulary's own serializer also survives the Patch type argument.
        val absent = RecordInputDto()
        val explicitNull = RecordInputDto(Patch.Null)
        val present = RecordInputDto(Patch.Value(RecordInputDto.Choice.fromSerialName("unknown-choice")))
        for ((input, expected) in listOf(
            absent to buildJsonObject {},
            explicitNull to buildJsonObject { put("choice", JsonNull) },
            present to buildJsonObject { put("choice", "unknown-choice") })) {
            val encoded = codec.encodeToJsonElement(RecordInputDto.serializer(), input)
            check(encoded == expected)
            check(codec.decodeFromJsonElement(RecordInputDto.serializer(), encoded) == input)
        }
        val stamp = Instant.parse("2025-01-01T00:00:00Z")
        val money = Money("caller-owned-wire")
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            val payload = (request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString()
            val response = when (calls) {
                1 -> {
                    check(codec.parseToJsonElement(payload) == buildJsonObject { put("choice", "unknown-choice") })
                    codec.encodeToString(RecordDto.serializer(), value)
                }
                2 -> { check(payload == "\"2025-01-01T00:00:00Z\""); payload }
                3 -> {
                    check(payload == Long.MIN_VALUE.toString())
                    check(request.url.parameters["vocabulary"] == "unknown-query")
                    payload
                }
                4 -> { check(payload == "\"caller-owned-wire\""); payload }
                else -> error("Unexpected native enum request")
            }
            respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http).default
            check(api.record(present) == value)
            check(api.stamp(stamp) == stamp)
            check(api.big(BigVocabularyDto.fromSerialName("unknown-query"), Long.MIN_VALUE) == Long.MIN_VALUE)
            check(api.money(money) == money)
            check(calls == 4)
            println("Native enum contracts passed: defaults=$defaults/nulls=$nulls; model/ref/item/map/parameter vocabulary vs native HTTP scalar, caller-owned type and Patch; $calls requests")
        } finally { http.close(); engine.close() }
    }
}
