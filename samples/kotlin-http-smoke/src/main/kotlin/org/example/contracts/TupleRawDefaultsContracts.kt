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
import org.example.rawtuples.api.DefaultApi
import org.example.rawtuples.api.models.EventsEventDto
import org.example.rawtuples.api.models.UpdateInputDto
import org.example.rawtuples.models.HolderDto
import org.example.rawtuples.models.MixedDto
import org.example.rawtuples.models.OpenDto
import org.example.rawtuples.models.Patch
import org.example.rawtuples.models.`MixedDto default`
import org.example.rawtuples.models.`ClosedDto default`
import org.example.rawtuples.models.`OpenDto default`
import org.example.rawtuples.models.`MaybeDto default`
import org.example.rawtuples.models.`AliasDto default`
import org.example.rawtuples.models.`ReferencedDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RawTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun rawTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> rawTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic raw tuple body")
}

suspend fun tupleRawDefaultsContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = HolderDto(required = listOf(JsonNull))
        val expected = codec.parseToJsonElement("""[null,true,{"default":{"description":"data","${'$'}ref":"literal"}},["text",1.0]]""") as JsonArray
        check(value.mixed == expected && value.open == expected)
        check(value.closed == listOf(JsonPrimitive(false), JsonNull))
        check(value.maybe == null && value.borrowed == null)
        check(HolderDto(required = emptyList()).mixed !== value.mixed)
        check(`MixedDto default`() == expected && `OpenDto default`() == expected)
        check(`MixedDto default`() !== `MixedDto default`())
        check(`ClosedDto default`() == value.closed && `MaybeDto default`() == null)
        check(`AliasDto default`() == listOf(JsonPrimitive(2)))
        check(`ReferencedDto default`() == listOf(JsonPrimitive(true), JsonNull))
        check(codec.decodeFromString<HolderDto>("""{"required":[null]}""") == value)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<MixedDto>("[]").isEmpty())
        check(codec.decodeFromString<OpenDto>("[{},[],null,3,true]").size == 5)
        val explicitNull = codec.decodeFromString<HolderDto>("""{"required":[],"mixed":null}""")
        check(explicitNull.mixed == null)
        val raw = codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(("mixed" in raw) == defaults)
        if (defaults) check(raw["mixed"] == expected)
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = RawTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(rawTupleBody(request.body)) == value)
                        respond(codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        val body = codec.parseToJsonElement(rawTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in body)
                            1 -> check(body["optional"] == JsonNull)
                            2 -> check(body["optional"] == JsonArray(listOf(JsonNull, JsonPrimitive(false))))
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (bad) "{\"required\":[null],\"mixed\":{}}" else codec.encodeToString(value)
                        val frame = "event: message\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic raw tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = listOf(JsonNull), optional = Patch.Value(listOf(JsonNull, JsonPrimitive(false)))))
            if (checked) {
                val before = calls
                var callbacks = 0
                check(runCatching { HolderDto(required = emptyList(), open = emptyList()) }.isFailure)
                check(runCatching { api.echo(HolderDto(required = emptyList(), open = listOf(JsonNull, JsonNull))) { callbacks++ } }.isFailure)
                check(callbacks == 0 && calls == before)
                // An alias factory validates its annotation, but ordinary alias
                // assignment/standalone codecs do not become constructor barriers.
                check(codec.decodeFromString<OpenDto>("[]").isEmpty())
            }
            val flow = api.events()
            check(channels.isEmpty())
            check((flow.take(1).toList().single() as EventsEventDto.Message).data == value)
            check(channels.last().isClosedForRead)
            bad = true
            check(runCatching { flow.take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead)
            bad = false
            check(flow.take(1).toList().size == 1 && api.echo(value) == value)
        } finally {
            client.close()
            scope.cancel()
            writers.forEach { it.cancel() }
            channels.forEach { it.cancel(null) }
        }
    }
}
