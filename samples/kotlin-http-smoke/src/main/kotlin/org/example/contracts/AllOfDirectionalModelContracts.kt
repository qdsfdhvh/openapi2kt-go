@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.flow.toList
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerializationException
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import org.example.allofdirectional.api.DefaultApi
import org.example.allofdirectional.api.models.GetSecretPublicResponseDto
import org.example.allofdirectional.api.models.SaveAccountPublicResponseDto
import org.example.allofdirectional.api.models.SaveAccountRequestDto
import org.example.allofdirectional.api.models.SaveInlineRequestDto
import org.example.allofdirectional.api.models.WatchEventDto
import org.example.allofdirectional.api.models.WatchPublicResponseDto
import org.example.allofdirectional.models.AccountDto
import org.example.allofdirectional.models.Patch
import org.example.allofdirectionaloptions.api.models.SaveAccountPublicResponseDto as OptimizedResponse

public suspend fun allOfDirectionalModelContracts(checked: Boolean = false) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        if (checked) {
            check(runCatching { SaveAccountRequestDto(name = "", password = "synthetic-value", mode = AccountDto.Mode.ACTIVE) }.isFailure)
            check(runCatching { codec.decodeFromString<SaveAccountPublicResponseDto>("""{"id":"server","name":"","mode":"active"}""") }.isFailure)
        }
        val child = SaveAccountRequestDto(name = "child", password = "synthetic-child", mode = AccountDto.Mode.ACTIVE)
        val input = SaveAccountRequestDto(
            name = "example", password = "synthetic-value", mode = AccountDto.Mode.ACTIVE,
            child = Patch.Value(child), peers = listOf(child), byName = mapOf("example" to child),
        )
        val encoded = codec.encodeToJsonElement(SaveAccountRequestDto.serializer(), input) as JsonObject
        check("id" !in encoded && "note" !in encoded)
        check("id" !in (encoded.getValue("child") as JsonObject))
        check(codec.decodeFromString<SaveAccountRequestDto>(codec.encodeToString(input)).id === Patch.Absent)
        val cleared = input.copyForContract(note = Patch.Null)
        check((codec.encodeToJsonElement(SaveAccountRequestDto.serializer(), cleared) as JsonObject).containsKey("note"))
        val explicit = SaveAccountRequestDto(name = "example", password = "synthetic-value", mode = AccountDto.Mode.ACTIVE, id = Patch.Value("override"))
        check((codec.encodeToJsonElement(SaveAccountRequestDto.serializer(), explicit) as JsonObject).containsKey("id"))
        val responseText = """{"id":"server","name":"example","mode":"active","child":{"id":"child","name":"child","mode":"paused"},"peers":[],"byName":{}}"""
        val response = codec.decodeFromString<SaveAccountPublicResponseDto>(responseText)
        check(response.id == "server" && response.child?.id == "child")
        check("password" !in (codec.encodeToJsonElement(SaveAccountPublicResponseDto.serializer(), response) as JsonObject))
        val unknownPassword = """{"id":"server","name":"example","mode":"active","password":"synthetic-response"}"""
        check(runCatching { codec.decodeFromString<SaveAccountPublicResponseDto>(unknownPassword) }.exceptionOrNull() is SerializationException)
        val permissive = Json(codec) { ignoreUnknownKeys = true }
        check("password" !in (permissive.encodeToJsonElement(SaveAccountPublicResponseDto.serializer(), permissive.decodeFromString(unknownPassword)) as JsonObject))
        check(codec.encodeToString(codec.decodeFromString<GetSecretPublicResponseDto>("{}")) == "{}")
        val optimized = codec.decodeFromString<OptimizedResponse>(responseText)
        check("password" !in (codec.encodeToJsonElement(OptimizedResponse.serializer(), optimized) as JsonObject))
        // Shared components retain their original constructor/codec contract.
        check(codec.decodeFromString<AccountDto>(unknownPassword).password == "synthetic-response")
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            when (request.url.encodedPath) {
                "/account" -> {
                    val body = request.body as OutgoingContent.ByteArrayContent
                    val actual = codec.parseToJsonElement(body.bytes().decodeToString()) as JsonObject
                    check("id" !in actual && "password" in actual)
                    respond(responseText, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/aliases" -> respond("[$responseText]", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                "/inline" -> respond("""{"name":"inline"}""", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                "/secret" -> respond("{}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                else -> error("Unexpected synthetic path")
            }
        }
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = DefaultApi(client, "https://example.test/")
            check(api.saveAccount(input).id == "server")
            check(api.saveInline(SaveInlineRequestDto(password = "synthetic-value", note = Patch.Null)).name == "inline")
            check(codec.encodeToString(api.getSecret()) == "{}")
            check(api.getAliases()!!.single().id == "server")
            check(calls == 4)
        } finally { client.close(); engine.close() }
        val streamEngine = AllOfDirectionalStreamingEngine(MockEngineConfig().apply {
            addHandler { respond("event: account\ndata: $responseText\n\n", HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream")) }
        })
        val streamClient = HttpClient(streamEngine) { install(SSE) }
        try {
            val frames = DefaultApi(streamClient, "https://example.test/", codec).watch().toList()
            check(frames.size == 1)
            val data = (frames.single() as WatchEventDto.Account).data
            check(data.id == "server")
            check("password" !in (codec.encodeToJsonElement(WatchPublicResponseDto.serializer(), data) as JsonObject))
        } finally { streamClient.close(); streamEngine.close() }
        println("Directional DTOs defaults=$defaults nulls=$nulls shared/request/response/recursive/empty/optimized/wire/SSE PASS")
    }
}

// Feed a real byte channel through Ktor's SSE response adapter and parser.
@OptIn(io.ktor.utils.io.InternalAPI::class)
private class AllOfDirectionalStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers,
            response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers,
            response.version, body, response.callContext)
    }
}

// Explicit construction works in both data_class and plain_class output.
private fun SaveAccountRequestDto.copyForContract(note: Patch<String>): SaveAccountRequestDto =
    SaveAccountRequestDto(name = name, password = password, mode = mode, note = note)
