@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpMethod
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.yaml.YamlClient
import org.example.yaml.api.models.UpdateInputDto
import org.example.yaml.models.Patch
import org.example.yaml.models.RecordDto

private fun yamlBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> yamlBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected body $body")
}

fun main() = runBlocking {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = codec.decodeFromString<RecordDto>("""{"id":"synthetic"}""")
        check(value.enabled == true && value.state == null && value.note == null)
        check(runCatching { codec.decodeFromString<RecordDto>("{}") }.isFailure)
        val unknown = codec.decodeFromString<RecordDto>("""{"id":"future","state":"future","tags":["one"]}""")
        check(unknown.state is RecordDto.State.Unrecognized && unknown.state.serialName == "future")
        check(unknown.tags == listOf("one"))
        check(codec.decodeFromJsonElement(RecordDto.serializer(), codec.encodeToJsonElement(RecordDto.serializer(), unknown)) == unknown)
        val encoded = codec.encodeToJsonElement(RecordDto.serializer(), value) as JsonObject
        check(encoded.containsKey("enabled") == defaults)
        check((codec.encodeToJsonElement(UpdateInputDto.serializer(), UpdateInputDto()) as JsonObject).isEmpty())
        check(codec.decodeFromString<UpdateInputDto>("{}").note === Patch.Absent)
        check(codec.decodeFromString<UpdateInputDto>("""{"note":null}""").note === Patch.Null)
        check((codec.encodeToJsonElement(UpdateInputDto.serializer(), UpdateInputDto(Patch.Null)) as JsonObject)["note"] === JsonNull)
        val update = UpdateInputDto(Patch.Value("updated"))
        check(codec.decodeFromJsonElement(UpdateInputDto.serializer(), codec.encodeToJsonElement(UpdateInputDto.serializer(), update)) == update)
        var calls = 0
        val engine = MockEngine { request ->
            check(request.url.encodedPath == "/records")
            calls++
            when (request.method) {
                HttpMethod.Get -> {
                    check(request.url.parameters["limit"] == "2")
                    respond("""[{"id":"wire","state":"future"}]""", headers = headersOf("Content-Type", "application/json"))
                }
                HttpMethod.Patch -> {
                    check(request.body.contentType?.toString()?.startsWith("application/json") == true)
                    val body = codec.parseToJsonElement(yamlBody(request.body)) as JsonObject
                    val expected = if (calls == 2) "{}" else if (calls == 3) """{"note":null}""" else """{"note":"updated"}"""
                    check(body == codec.parseToJsonElement(expected))
                    respond("""{"id":"wire"}""", headers = headersOf("Content-Type", "application/json"))
                }
                else -> error("Unexpected method")
            }
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val client = YamlClient(http, "https://example.test/")
            check(client.records.records(limit = 2).single().state?.serialName == "future")
            for (input in listOf(UpdateInputDto(), UpdateInputDto(Patch.Null), update)) {
                check(client.records.updateUpdate(input).enabled == true)
            }
            check(calls == 4)
        } finally { http.close() }
        println("YAML fresh source serde/Patch/HTTP defaults=$defaults nulls=$nulls passed")
    }
}
