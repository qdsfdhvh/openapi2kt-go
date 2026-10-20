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
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.yield
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.uuiddefaults.UuidDefaultsApiClient
import org.example.uuiddefaults.api.models.SubmitInputDto
import org.example.uuiddefaults.api.models.WatchEventDto
import org.example.uuiddefaults.models.IdDto
import org.example.uuiddefaults.models.NullIdDto
import org.example.uuiddefaults.models.InheritedDto
import org.example.uuiddefaults.models.NarrowDto
import org.example.uuiddefaults.models.WidenDto
import org.example.uuiddefaults.models.OwnNullableDto
import org.example.uuiddefaults.models.Patch
import org.example.uuiddefaults.models.UuidBridgeDto
import org.example.uuiddefaults.models.UuidFalseDto
import org.example.uuiddefaults.models.HolderDto
import org.example.uuiddefaults.models.RequiredDto
import org.example.uuiddefaults.models.`IdDto default`
import org.example.uuiddefaults.models.`TextDto default`
import org.example.uuiddefaults.models.`BoundedDto default`
import org.example.uuiddefaults.models.`MiddleDto default`
import org.example.uuiddefaults.models.`OuterDto default`
import org.example.uuiddefaults.models.`NullIdDto default`
import org.example.uuiddefaults.models.`MaybeIdDto default`
import org.example.uuiddefaults.models.`NarrowDto default`
import org.example.uuiddefaults.models.`WidenDto default`
import org.example.uuiddefaults.models.`OwnNullableDto default`

// Match the repository's streaming test adapters: Ktor parses actual fragmented
// channels, while MockEngine needs to advertise and apply the SSE capability.
@OptIn(io.ktor.utils.io.InternalAPI::class)
private class UUIDStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun uuidBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> uuidBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected UUID body ${content::class}")
}

suspend fun uuidComponentDefaultsContracts() {
    val id: IdDto = `IdDto default`()
    val text: String = `TextDto default`()
    check(id == "123e4567-e89b-12d3-a456-426614174000")
    check(text == "ordinary text\n雪")
    check(`MiddleDto default`() == "middle")
    check(`BoundedDto default`() == "given" && `OuterDto default`() == "outer")
    check(`NullIdDto default`() == null && `WidenDto default`() == null && `OwnNullableDto default`() == null)
    check(`MaybeIdDto default`() == "present" && `NarrowDto default`() == "narrow")
    check(serializer<NullIdDto>().descriptor.isNullable && serializer<InheritedDto>().descriptor.isNullable)
    check(serializer<WidenDto>().descriptor.isNullable && serializer<OwnNullableDto>().descriptor.isNullable)
    check(!serializer<NarrowDto>().descriptor.isNullable)
    check(serializer<UuidBridgeDto>().descriptor.isNullable && !serializer<UuidFalseDto>().descriptor.isNullable)
    check(!SubmitInputDto.serializer().descriptor.isElementOptional(0))
    check(runCatching { HolderDto(bounded = "x") }.isFailure)
    check(runCatching { HolderDto(outer = "UPPER") }.isFailure)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<IdDto>(codec.encodeToString(id)) == id)
        check(codec.decodeFromString<IdDto>("\"arbitrary string\"") == "arbitrary string")
        check(codec.decodeFromString<NullIdDto>("null") == null)
        check(codec.decodeFromString<OwnNullableDto>("null") == null)
        check(codec.decodeFromString<UuidBridgeDto>("null") == null)
        check(runCatching { codec.decodeFromString<UuidFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<NarrowDto>("null") }.isFailure)
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.id == null && missing.text == null)
        check(missing.bounded == null && missing.middle == null && missing.outer == null && missing.plain == null)
        check(missing.nullId == null && missing.maybeId == null && missing.narrow == null && missing.widen == null)
        check(missing.borrowed == "own")
        check(codec.decodeFromString<HolderDto>("""{"borrowed":null}""").borrowed == null)
        val adopted = HolderDto(id = id, text = text, bounded = `BoundedDto default`(), outer = `OuterDto default`(),
            ids = listOf(null, "future"), values = mapOf("nil" to null, "text" to "future"))
        val wire = codec.encodeToJsonElement(HolderDto.serializer(), adopted) as JsonObject
        check(wire["id"] == JsonPrimitive(id) && wire["text"] == JsonPrimitive(text))
        val decoded = codec.decodeFromString<HolderDto>(codec.encodeToString(adopted))
        check(decoded.id == id && decoded.text == text && decoded.bounded == "given" && decoded.outer == "outer")
        check(decoded.ids == listOf(null, "future") && decoded.values == mapOf("nil" to null, "text" to "future"))
        check(decoded.borrowed == "own")
        check(runCatching { codec.decodeFromString<RequiredDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<RequiredDto>("""{"id":null}""") }.isFailure)
        val required = RequiredDto(id = id, nullableId = null, narrow = "future")
        check(codec.decodeFromString<RequiredDto>(codec.encodeToString(required)).id == id)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"bounded":"x"}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"outer":"UPPER"}""") }.isFailure)
        val request = SubmitInputDto(requiredId = null, narrow = "future")
        val absent = codec.encodeToJsonElement(SubmitInputDto.serializer(), request) as JsonObject
        check("value" !in absent && ("requiredId" in absent) == nulls)
        val patchNull = SubmitInputDto(requiredId = null, narrow = "future", value = Patch.Null)
        val patchValue = SubmitInputDto(requiredId = "future", narrow = "future", value = Patch.Value("arbitrary"))
        check((codec.encodeToJsonElement(SubmitInputDto.serializer(), patchNull) as JsonObject)["value"] == JsonNull)
        check((codec.encodeToJsonElement(SubmitInputDto.serializer(), patchValue) as JsonObject)["value"] == JsonPrimitive("arbitrary"))
        check(codec.decodeFromString<SubmitInputDto>("""{"requiredId":null,"narrow":"future","value":null}""").value === Patch.Null)
        var expected = ""
        var streams = 0
        var badStream = false
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val http = HttpClient(UUIDStreamingEngine(MockEngineConfig().apply { addHandler { incoming ->
            when (incoming.url.encodedPath) {
                "/echo" -> {
                    check(uuidBody(incoming.body) == expected)
                    respond(expected, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/submit" -> {
                    check(Json.parseToJsonElement(uuidBody(incoming.body)) == Json.parseToJsonElement(expected))
                    respond("", HttpStatusCode.NoContent)
                }
                "/watch" -> {
                    streams++
                    val channel = ByteChannel(autoFlush = true)
                    val frames = if (badStream) "event: id\ndata: []\n\n" else
                        "event: id\r\ndata: null\r\n\r\nevent: id\ndata: \"future\"\n\n"
                    producers.launch {
                        try {
                            for (chunk in frames.toByteArray().asList().chunked(3)) {
                                channel.writeFully(chunk.toByteArray())
                                yield()
                            }
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected UUID route ${incoming.url}")
            }
        } })) {
            install(ContentNegotiation) { json(codec) }
            install(SSE)
        }
        try {
            val api = UuidDefaultsApiClient(http, "https://example.test/", codec).default
            expected = "null"
            check(api.echo(null) == "null") // Ktor String roots retain raw ASCII null.
            expected = "arbitrary string"
            check(api.echo(expected) == expected)
            for (input in listOf(request, patchNull, patchValue)) {
                expected = codec.encodeToString(input)
                api.submit(input)
            }
            val flow = api.watch()
            check(streams == 0)
            repeat(2) {
                val events = withTimeout(3000) { flow.toList() }
                check(events.size == 2 && (events[0] as WatchEventDto.Id).data == null)
                check((events[1] as WatchEventDto.Id).data == "future")
            }
            check(streams == 2)
            badStream = true
            val failure = runCatching { withTimeout(3000) { api.watch().toList() } }.exceptionOrNull()
            check(failure != null && generateSequence(failure) { it.cause }.any { it is SerializationException })
            expected = "reused"
            check(api.echo(expected) == expected)
        } finally { http.close(); producers.cancel() }
    }
}
