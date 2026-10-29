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
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import org.example.booleanmembers.api.DefaultApi
import org.example.booleanmembers.api.models.EventsEventDto
import org.example.booleanmembers.api.models.FalseEventsEventDto
import org.example.booleanmembers.api.models.UpdateInputDto
import org.example.booleanmembers.models.HolderDto
import org.example.booleanmembers.models.ZeroDto
import org.example.booleanmembers.models.Patch
import org.example.booleanmembers.models.`ClosedDto default`
import org.example.booleanmembers.models.`AliasDto default`
import org.example.booleanmembers.models.`MaybeDto default`
import org.example.booleanmembers.models.`OpenDto default`
import org.example.booleanmembers.models.`MixedDto default`
import org.example.booleanmembers.models.`TailDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BooleanMemberStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun booleanMemberBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> booleanMemberBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic Boolean member body")
}

suspend fun tupleBooleanMembersContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = HolderDto(required = emptyList())
        check(value.zeros!!.isEmpty() && value.open == listOf(JsonNull, JsonPrimitive(true)))
        check(value.borrowed == null)
        check(`ClosedDto default`().isEmpty() && `AliasDto default`().isEmpty())
        check(`MaybeDto default`() == null)
        check(`OpenDto default`() == codec.parseToJsonElement("""[null,true,{"default":"data"}]"""))
        check(`MixedDto default`() == codec.parseToJsonElement("""[null,{"enum":"data"}]"""))
        check(`TailDto default`() == listOf(1, 2))
        check(codec.decodeFromString<ZeroDto>("[]").isEmpty())
        check(codec.encodeToString<ZeroDto>(emptyList()) == "[]")
        for (literal in listOf("[null]", "[false]", "[true]", "[1]", "[{}]", "[[]]", "[\"false\"]")) {
            check(runCatching { codec.decodeFromString<ZeroDto>(literal) }.isFailure)
            check(runCatching { codec.decodeFromString<HolderDto>("{\"required\":$literal}") }.isFailure)
        }
        check(codec.decodeFromString<HolderDto>("""{"required":[]}""") == value)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<HolderDto>("""{"required":[],"zeros":null}""").zeros == null)
        val nested = HolderDto(required = emptyList(), nested = listOf(mapOf("x" to emptyList())))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(nested)) == nested)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"required":[],"nested":[{"x":[null]}]}""") }.isFailure)
        check(("zeros" in codec.parseToJsonElement(codec.encodeToString(value)).jsonObject) == defaults)
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = BooleanMemberStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(booleanMemberBody(request.body)) == value)
                        respond(if (bad) """{"required":[null]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/zero", "/inline" -> respond(if (bad) "[null]" else "[]", headers = headersOf("Content-Type", "application/json"))
                    "/send-zero" -> {
                        check(booleanMemberBody(request.body) == "[]")
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/patch" -> {
                        val body = codec.parseToJsonElement(booleanMemberBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in body)
                            1 -> check(body["optional"] == JsonNull)
                            2 -> check(body["optional"] == JsonArray(emptyList()))
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events", "/false-events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val raw = request.url.encodedPath == "/false-events"
                        val payload = if (raw) { if (bad) "[null]" else "[]" }
                            else if (bad) """{"required":[null]}""" else codec.encodeToString(value)
                        val event = if (raw) "empty" else "message"
                        val frame = "event: $event\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic Boolean tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            check(api.zero().isEmpty() && api.inlineZero().isEmpty())
            api.sendZero(emptyList())
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = emptyList(), optional = Patch.Value(emptyList())))
            check((api.events().take(1).toList().single() as EventsEventDto.Message).data == value)
            check(channels.last().isClosedForRead)
            check((api.falseEvents().take(1).toList().single() as FalseEventsEventDto.Empty).data.isEmpty())
            check(channels.last().isClosedForRead)
            if (checked) {
                val before = calls
                check(runCatching { api.echo(HolderDto(required = emptyList(), open = listOf(JsonNull, JsonNull, JsonNull))) }.isFailure)
                check(calls == before)
            }
            bad = true
            check(runCatching { api.zero() }.isFailure)
            check(runCatching { api.inlineZero() }.isFailure)
            check(runCatching { api.echo(value) }.isFailure)
            check(runCatching { api.events().take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead)
            check(runCatching { api.falseEvents().take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead)
            bad = false
            check(api.echo(value) == value && api.zero().isEmpty())
        } finally {
            client.close()
            writers.forEach { it.cancel() }
            channels.forEach { it.cancel(null) }
            scope.cancel()
        }
    }
}
