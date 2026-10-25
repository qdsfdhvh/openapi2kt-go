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
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.discarded.DiscardedClient
import org.example.discarded.api.models.SendInlineInputDto
import org.example.discarded.api.models.WatchEventDto
import org.example.discarded.models.ExtraDto
import org.example.discarded.models.ExtraArrayDto
import org.example.discarded.models.RequiredChoiceDto
import org.example.discarded.models.ChoiceDto
import org.example.discarded.models.SingletonDto
import org.example.discarded.models.RootDto
import org.example.discarded.models.AlphaDto
import org.example.discarded.models.BodyDto
import org.example.discarded.models.DictionaryDto
import org.example.discarded.models.ContainersDto
import org.example.discarded.models.NullableExtraDto
import org.example.discarded.models.UnknownChoiceDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class DiscardedEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun discardedReject(block: () -> Unit) {
    try { block(); error("Expected structured const kind rejection") }
    catch (_: SerializationException) { }
}
private suspend fun discardedWireReject(block: suspend () -> Unit) {
    try { block(); error("Expected structured const wire rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong wire failure: $cause" }
    }
}
private fun discardedBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> discardedBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected structured const body $body")
}

suspend fun discardedStructuredConstContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val strict = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(ExtraDto().value == null && ExtraArrayDto().value == null)
        check(codec.decodeFromString<NullableExtraDto>("null") == null)
        val missing = codec.decodeFromString<ExtraDto>("{}")
        check(missing.value == null)
        val choice = ChoiceDto.One(payload = "winner")
        check(choice.payload == "winner" && ChoiceDto.One.serializer().descriptor.elementsCount == 2)
        for (extra in listOf("null", "true", "1", "\"text\"", "{}", "[]", "{\"$" + "ref\":\"literal data\"}")) {
            val input = "{\"value\":\"kept\",\"extra\":$extra}"
            val value = codec.decodeFromString<ExtraDto>(input)
            check(value.value == "kept")
            val wire = codec.encodeToJsonElement(ExtraDto.serializer(), value) as JsonObject
            check(wire["value"] == JsonPrimitive("kept") && !wire.containsKey("extra"))
            check(codec.decodeFromString<ExtraArrayDto>(input).value == "kept")
            discardedReject { strict.decodeFromString<ExtraDto>(input) }
            discardedReject { strict.decodeFromString<ExtraArrayDto>(input) }
            val branchInput = "{\"tag\":\"one\",\"payload\":\"winner\",\"extra\":$extra}"
            val branch = codec.decodeFromString<ChoiceDto>(branchInput) as ChoiceDto.One
            check(branch.payload == "winner")
            check(!(codec.encodeToJsonElement(ChoiceDto.serializer(), branch) as JsonObject).containsKey("extra"))
            discardedReject { strict.decodeFromString<ChoiceDto>(branchInput) }
            // Scalar-only singleton codecs retain their historical consume-all path.
            check(strict.decodeFromString<SingletonDto>("{\"tag\":\"one\",\"extra\":$extra}") === SingletonDto.One)
            val hierarchyInput = "{\"tag\":\"one\",\"value\":\"winner\",\"extra\":$extra}"
            val leaf = codec.decodeFromString<RootDto>(hierarchyInput) as AlphaDto
            check(leaf.value == "winner")
            check(!(codec.encodeToJsonElement(RootDto.serializer(), leaf) as JsonObject).containsKey("extra"))
            discardedReject { strict.decodeFromString<RootDto>(hierarchyInput) }
        }
        discardedReject { codec.decodeFromString<ChoiceDto>("{\"tag\":\"one\",\"payload\":{}}") }
        check(AlphaDto(value = "constructor").value == "constructor")
        val required = RequiredChoiceDto.One(payload = "constructor")
        val requiredDescriptor = RequiredChoiceDto.One.serializer().descriptor
        check(!requiredDescriptor.isElementOptional(requiredDescriptor.getElementIndex("payload")))
        check(required.payload == "constructor")
        for (badWinner in listOf("{\"tag\":\"one\"}", "{\"tag\":\"one\",\"payload\":null}", "{\"tag\":\"one\",\"payload\":{}}")) {
            discardedReject { codec.decodeFromString<RequiredChoiceDto>(badWinner) }
        }
        check((codec.decodeFromString<RequiredChoiceDto>("{\"tag\":\"one\",\"payload\":\"required\"}") as RequiredChoiceDto.One).payload == "required")
        val raw = codec.parseToJsonElement("{\"tag\":\"unknown\",\"extra\":[1]}")
        val unknown = codec.decodeFromJsonElement(ChoiceDto.serializer(), raw) as UnknownChoiceDto
        check(unknown.rawElement == raw && codec.encodeToJsonElement(ChoiceDto.serializer(), unknown) == raw)
        val body = codec.decodeFromString<BodyDto>("{\"extra\":false}")
        check(body.payload == JsonObject(mapOf("kept" to JsonPrimitive(true))))
        check((codec.encodeToJsonElement(BodyDto.serializer(), body) as JsonObject)["payload"] == body.payload)
        val dictionary = codec.decodeFromString<DictionaryDto>("{\"key\":{\"alternative\":null}}")
        check(dictionary["key"] == JsonObject(mapOf("alternative" to JsonNull)))
        discardedReject { codec.decodeFromString<DictionaryDto>("{\"key\":[]}") }
        val envelope = ContainersDto(extra = ExtraDto("value"), choice = choice, root = AlphaDto("value"))
        check(codec.decodeFromString<ContainersDto>(codec.encodeToString(envelope)).choice?.payload == "winner")
        discardedWire(codec, choice)
        println("Discarded structured const defaults=$defaults nulls=$nulls winner/strict/ignored/Unknown/body/dictionary/HTTP/liveSSE passed")
    }
}

private suspend fun discardedWire(codec: Json, choice: ChoiceDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var calls = 0
    val engine = DiscardedEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/choice" -> {
                    check(discardedBody(request.body) == codec.encodeToString(choice))
                    val result = if (bad) "{\"tag\":\"one\",\"payload\":{}}" else "{\"tag\":\"one\",\"payload\":\"winner\",\"extra\":[1]}"
                    respond(result, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/inline" -> {
                    check(discardedBody(request.body) == codec.encodeToString(SendInlineInputDto("inline")))
                    respond("", HttpStatusCode.NoContent)
                }
                "/extra" -> {
                    check(discardedBody(request.body) == codec.encodeToString(ExtraDto("sent")))
                    respond("{\"value\":\"kept\",\"extra\":false}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/root" -> respond("{\"tag\":\"one\",\"value\":\"kept\",\"extra\":[]}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val invalid = bad
                    producers.launch {
                        try {
                            val payload = if (invalid) "{\"tag\":\"one\",\"payload\":{}}" else "{\"tag\":\"one\",\"payload\":\"winner\",\"extra\":{}}"
                            val frame = "event: choice\r\nid: one\r\ndata: $payload\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (invalid) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected discarded const request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = DiscardedClient(http, "http://example.test/", codec).discarded
        api.sendInline(SendInlineInputDto("inline"))
        check(api.sendExtra(ExtraDto("sent")).value == "kept")
        check(api.sendChoice(choice).payload == "winner")
        check((api.sendRoot(AlphaDto("sent")) as AlphaDto).value == "kept")
        val before = calls
        val flow = api.watch()
        check(calls == before)
        repeat(2) {
            val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Choice
            check(event.data.payload == "winner" && event.id == "one")
        }
        bad = true
        discardedWireReject { api.sendChoice(choice) }
        val beforeBad = channels.size
        discardedWireReject { withTimeout(10000) { api.watch().toList() } }
        check(channels.size == beforeBad + 1 && channels.last().isClosedForRead)
        bad = false
        check(api.sendChoice(choice).payload == "winner")
    } finally {
        http.close()
        producers.cancel()
        channels.forEach { it.close() }
    }
}
