@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.nullliterals.NullLiterals
import org.example.nullliterals.api.models.DefaultWatchEventDto
import org.example.nullliterals.models.DefaultsDto

suspend fun nullOnlyDefaultContracts() {
    val value = DefaultsDto(required = JsonNull, requiredNullable = null, kotlinx = "shadow", kotlin = "shadow")
    check(value.required === JsonNull && value.requiredNullable == null)
    check(value.optional == null && value.inferred == null)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val decoded = codec.decodeFromString<DefaultsDto>("{\"required\":null,\"requiredNullable\":null}")
        check(decoded.required === JsonNull && decoded.requiredNullable == null)
        check(decoded.optional == null && decoded.inferred == null)
        val wire = codec.parseToJsonElement(codec.encodeToString(value)) as JsonObject
        check(wire["required"] === JsonNull)
        check(wire.containsKey("requiredNullable") == nulls)
        check(wire.containsKey("optional") == (defaults && nulls))
        check(wire.containsKey("inferred") == (defaults && nulls))
        val back = codec.decodeFromJsonElement(DefaultsDto.serializer(), wire)
        check(back == value)
        if (nulls) {
            try {
                codec.decodeFromString<DefaultsDto>("{\"required\":null}")
                error("A required nullable default became a missing-key initializer")
            } catch (_: SerializationException) { }
        } else {
            check(codec.decodeFromString<DefaultsDto>("{\"required\":null}").requiredNullable == null)
        }
        for (wrong in listOf("\"null\"", "false", "0", "{}", "[]")) {
            for (field in listOf("required", "requiredNullable", "optional", "inferred")) {
                val json = if (field == "required") "{\"required\":$wrong,\"requiredNullable\":null}"
                    else if (field == "requiredNullable") "{\"required\":null,\"requiredNullable\":$wrong}"
                    else "{\"required\":null,\"requiredNullable\":null,\"$field\":$wrong}"
                try { codec.decodeFromString<DefaultsDto>(json); error("A null default widened the codec") }
                catch (_: SerializationException) { }
            }
        }
        nullOnlyDefaultWire(codec, value)
        println("NullOnlyDefaults Json defaults=$defaults nulls=$nulls required-no-initializer-optional-null-typed-codec-HTTP-liveSSE passed")
    }
}


@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullDefaultsEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun nullDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected null default body $body")
}

private suspend fun nullDefaultWireReject(block: suspend () -> Unit) {
    try { block(); error("Expected null default codec rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong failure: $cause" }
    }
}

private suspend fun nullOnlyDefaultWire(codec: Json, value: DefaultsDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var calls = 0
    val payload = codec.encodeToString(value)
    val engine = NullDefaultsEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/default-holder" -> {
                    check(nullDefaultBody(request.body) == payload)
                    respond(if (bad) "{\"required\":false,\"requiredNullable\":null}" else payload,
                        HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/default-events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val invalid = bad
                    producers.launch {
                        try {
                            val data = if (invalid) "{\"required\":\"null\",\"requiredNullable\":null}" else payload
                            val frame = "event: defaults\r\nid: default-id\r\ndata: $data\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (invalid) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected null default request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = NullLiterals(http, "http://example.test/", codec).nullLiterals
        check(api.defaultHolder(value) == value)
        val before = calls
        val flow = api.defaultWatch()
        check(calls == before) // Creating the cold Flow sends no request.
        repeat(2) {
            val event = withTimeout(10000) { flow.toList() }.single() as DefaultWatchEventDto.Defaults
            check(event.data == value && event.id == "default-id")
        }
        bad = true
        nullDefaultWireReject { api.defaultHolder(value) }
        val beforeBad = channels.size
        nullDefaultWireReject { withTimeout(10000) { api.defaultWatch().toList() } }
        check(channels.size == beforeBad + 1 && channels.last().isClosedForRead)
        bad = false
        check(api.defaultHolder(value) == value)
    } finally {
        http.close()
        producers.cancel()
        channels.forEach { it.close() }
    }
}
