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
import org.example.pinned.PinnedClient
import org.example.pinned.api.models.WatchEventDto
import org.example.pinned.models.ChoiceDto
import org.example.pinned.models.RefChoiceDto
import org.example.pinned.models.RootDto
import org.example.pinned.models.AlphaDto
import org.example.pinned.models.InferredDto
import org.example.pinned.models.GroupDto
import org.example.pinned.models.GroupAlphaDto
import org.example.pinned.models.NullableChoiceDto
import org.example.pinned.models.NullNullableChoiceDto
import org.example.pinned.models.WithArgsDto
import org.example.pinned.models.ProductDto
import org.example.pinned.models.TypeDataDto
import org.example.pinned.models.UnknownTypeDataDto
import org.example.pinned.models.MixedTypeDataDto
import org.example.pinned.models.UnknownMixedTypeDataDto
import org.example.pinned.models.TypeGroupDto
import org.example.pinned.models.TypeGroupAlphaDto
import org.example.pinned.models.UnknownTypeGroupDto
import org.example.pinned.models.SealedDto
import org.example.pinned.models.EnvelopeDto
import org.example.pinned.models.UnknownChoiceDto
import org.example.pinned.models.UnknownRefChoiceDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class PinnedEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun pinnedReject(block: () -> Unit) {
    try { block(); error("Expected structured const kind rejection") }
    catch (_: SerializationException) { }
}
private suspend fun pinnedWireReject(block: suspend () -> Unit) {
    try { block(); error("Expected structured const wire rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong wire failure: $cause" }
    }
}
private fun pinnedBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> pinnedBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected structured const body $body")
}

private suspend fun pinnedWire(codec: Json, holder: ChoiceDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var calls = 0

    val engine = PinnedEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/choice" -> {
                    check(pinnedBody(request.body) == codec.encodeToString(holder))
                    respond(if (bad) "{\"tag\":\"one\",\"payload\":[]}" else codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/root" -> {
                    val body = pinnedBody(request.body)
                    respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val invalid = bad
                    producers.launch {
                        try {
                            val payload = if (invalid) "{\"tag\":\"one\",\"payload\":[]}" else codec.encodeToString(holder)
                            val frame = "event: choice\r\nid: one\r\ndata: $payload\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (invalid) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected structured const request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = PinnedClient(http, "http://example.test/", codec).pinned
        check(api.sendChoice(holder).payload == holder.payload)
        val before = calls
        val flow = api.watch()
        check(calls == before)
        repeat(2) {
            val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Choice
            check(event.data.payload == holder.payload && event.id == "one")
        }
        val root = codec.decodeFromString<RootDto>("{\"tag\":\"one\",\"payload\":{\"transport\":true},\"rows\":[3]}")
        check((api.sendRoot(root) as AlphaDto).rows == (root as AlphaDto).rows)
        bad = true
        pinnedWireReject { api.sendChoice(holder) }
        val beforeBad = channels.size
        pinnedWireReject { withTimeout(10000) { api.watch().toList() } }
        check(channels.size == beforeBad + 1 && channels.last().isClosedForRead)
        bad = false
        check(api.sendChoice(holder).payload == holder.payload)
    } finally {
        http.close()
        producers.cancel()
        channels.forEach { it.close() }
    }
}

suspend fun pinnedStructuredConstContracts() {
    val first = ChoiceDto.One()
    val second = ChoiceDto.One()
    check(first !== second && first.payload !== second.payload)
    check(first.payload == JsonObject(mapOf("literal" to JsonPrimitive(true))))
    check(ChoiceDto.One.serializer().descriptor.elementsCount == 2)
    check(ChoiceDto.One.serializer().descriptor.isElementOptional(1))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<ChoiceDto>("{\"tag\":\"one\"}") as ChoiceDto.One
        check(missing.payload == first.payload)
        val changed = codec.decodeFromString<ChoiceDto>("{\"tag\":\"one\",\"payload\":{\"alternative\":[null]}}")
        check(changed is ChoiceDto.One && changed.payload.containsKey("alternative"))
        val wire = codec.encodeToJsonElement(ChoiceDto.serializer(), changed) as JsonObject
        check(wire["payload"] == changed.payload && wire["tag"] == JsonPrimitive("one"))
        check(codec.decodeFromJsonElement(ChoiceDto.serializer(), wire).payload == changed.payload)
        for (wrong in listOf("null", "false", "1", "\"object\"", "[]")) {
            pinnedReject { codec.decodeFromString<ChoiceDto>("{\"tag\":\"one\",\"payload\":$wrong}") }
        }
        val unknownWire = codec.parseToJsonElement("{\"tag\":\"unknown\",\"payload\":null}")
        val unknown = codec.decodeFromJsonElement(ChoiceDto.serializer(), unknownWire) as UnknownChoiceDto
        check(unknown.rawElement == unknownWire && codec.encodeToJsonElement(ChoiceDto.serializer(), unknown) == unknownWire)
        check(runCatching { unknown.payload }.exceptionOrNull() is IllegalStateException)
        val absentTag = codec.decodeFromString<ChoiceDto>("{\"payload\":{}}")
        check(absentTag is UnknownChoiceDto)
        check(runCatching { codec.decodeFromString<ChoiceDto>("null") }.exceptionOrNull() is IllegalArgumentException)
        val ref = codec.decodeFromString<RefChoiceDto>("{\"tag\":\"one\",\"payload\":{\"transport\":true},\"rows\":[3]}") as RefChoiceDto.One
        check(ref.payload.containsKey("transport") && ref.rows == codec.parseToJsonElement("[3]"))
        val wrapped = codec.decodeFromString<RefChoiceDto>("{\"tag\":\"two\",\"rows\":[4]}") as RefChoiceDto.Two
        check(wrapped.rows == codec.parseToJsonElement("[4]") && wrapped.payload["\$ref"] == JsonPrimitive("literal data"))
        pinnedReject { codec.decodeFromString<RefChoiceDto>("{\"tag\":\"one\",\"rows\":null}") }
        check((codec.decodeFromString<RefChoiceDto>("{\"tag\":\"unknown\"}") as UnknownRefChoiceDto).payload == null)
        val args = WithArgsDto.One(label = "constructor")
        check(args.label == "constructor")
        check(args.payload.isEmpty())
        val argsChanged = codec.decodeFromString<WithArgsDto>("{\"tag\":\"one\",\"label\":\"value\",\"payload\":[7]}") as WithArgsDto.One
        check(argsChanged.payload == codec.parseToJsonElement("[7]"))
        val root = codec.decodeFromString<RootDto>("{\"tag\":\"one\",\"payload\":{\"inherited\":true},\"rows\":[9]}") as AlphaDto
        check(root.payload.containsKey("inherited") && root.rows == codec.parseToJsonElement("[9]"))
        val rootBack = codec.decodeFromJsonElement(RootDto.serializer(), codec.encodeToJsonElement(RootDto.serializer(), root)) as AlphaDto
        check(rootBack.rows == root.rows && rootBack.payload == root.payload)
        check(AlphaDto().rows == codec.parseToJsonElement("[1,{\"literal\":true}]"))
        for (wrong in listOf("null", "{}", "true", "1", "\"array\"")) {
            pinnedReject { codec.decodeFromString<RootDto>("{\"tag\":\"one\",\"rows\":$wrong}") }
        }
        val inferred = codec.decodeFromString<InferredDto>("{\"tag\":\"one\",\"payload\":{\"inferred\":true}}")
        check(inferred is InferredDto.InferredAlphaDto && inferred.payload.containsKey("inferred"))
        val group = codec.decodeFromString<GroupDto>("{\"tag\":\"one\",\"payload\":{\"intermediate\":true}}") as GroupAlphaDto
        check(group.payload.containsKey("intermediate"))
        check((codec.decodeFromJsonElement(GroupDto.serializer(), codec.encodeToJsonElement(GroupDto.serializer(), group)) as GroupAlphaDto).payload == group.payload)
        check(codec.decodeFromString<NullableChoiceDto>("null") === NullNullableChoiceDto)
        check(codec.encodeToString(NullableChoiceDto.serializer(), NullNullableChoiceDto) == "null")
        val product = codec.decodeFromString<ProductDto>("{\"tag\":\"one\",\"mode\":\"slow\",\"payload\":{\"alternative\":true}}")
        check(product is ProductDto.One2 && product.payload.containsKey("alternative"))
        val dataType = TypeDataDto.TypeDataAlphaDto()
        val dataTypeWire = codec.encodeToJsonElement(TypeDataDto.serializer(), dataType) as JsonObject
        check(dataTypeWire["type"] == dataType.type)
        val dataTypeBack = codec.decodeFromJsonElement(TypeDataDto.serializer(), dataTypeWire) as UnknownTypeDataDto
        check(dataTypeBack.rawElement == dataTypeWire)
        check(runCatching { dataTypeBack.type }.exceptionOrNull() is IllegalStateException)
        val carrier = codec.decodeFromString<TypeDataDto>("{\"type\":{},\"labels\":{\"key\":\"value\"}}") as UnknownTypeDataDto
        check((carrier.rawElement as JsonObject)["labels"] == JsonObject(mapOf("key" to JsonPrimitive("value"))))
        val mixed = MixedTypeDataDto.MixedTypeDataAlphaDto()
        val mixedWire = codec.encodeToJsonElement(MixedTypeDataDto.serializer(), mixed) as JsonObject
        check(mixedWire["type"] == mixed.type)
        check((codec.decodeFromJsonElement(MixedTypeDataDto.serializer(), mixedWire) as UnknownMixedTypeDataDto).rawElement == mixedWire)
        val typeGroup = TypeGroupAlphaDto()
        val groupWire = codec.encodeToJsonElement(TypeGroupDto.serializer(), typeGroup) as JsonObject
        check(groupWire["type"] == typeGroup.type)
        check((codec.decodeFromJsonElement(TypeGroupDto.serializer(), groupWire) as UnknownTypeGroupDto).rawElement == groupWire)
        val nullTypeGroup = codec.decodeFromString<TypeGroupDto>("null") as UnknownTypeGroupDto
        check(nullTypeGroup.rawElement === JsonNull)
        val sealed = SealedDto.SealedAlphaDto()
        val sealedWire = codec.encodeToJsonElement(SealedDto.serializer(), sealed) as JsonObject
        check(sealedWire["type"] == JsonPrimitive("SealedAlphaDto"))
        val sealedChanged = codec.decodeFromString<SealedDto>("{\"type\":\"SealedAlphaDto\",\"payload\":{\"sealed\":true}}")
        check(sealedChanged is SealedDto.SealedAlphaDto && sealedChanged.payload.containsKey("sealed"))
        val envelope = EnvelopeDto(choice = changed, root = root, inferred = inferred, ref = ref)
        val back = codec.decodeFromString<EnvelopeDto>(codec.encodeToString(envelope))
        check(back.choice?.payload == changed.payload && (back.root as AlphaDto).rows == root.rows)
        pinnedWire(codec, changed)
        println("Pinned structured const defaults=$defaults nulls=$nulls kind/body/API/ref/inherited/product/HTTP/liveSSE passed")
    }
}
