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
import kotlinx.serialization.KSerializer
import kotlinx.serialization.MissingFieldException
import kotlinx.serialization.SerializationException
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.descriptors.SerialDescriptor
import kotlinx.serialization.encoding.AbstractDecoder
import kotlinx.serialization.encoding.AbstractEncoder
import kotlinx.serialization.encoding.CompositeDecoder
import kotlinx.serialization.encoding.CompositeEncoder
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.modules.EmptySerializersModule
import org.example.referencedcounts.ReferencedCounts
import org.example.referencedcounts.api.models.PatchInputDto
import org.example.referencedcounts.api.models.WatchEventDto
import org.example.referencedcounts.models.EnvelopeDto
import org.example.referencedcounts.models.NodeDto
import org.example.referencedcounts.models.Patch
import org.example.referencedcounts.models.RecordDto
import org.example.referencedcounts.models.ZeroDto
import org.example.referencedcounts.models.WithDefaultDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedCountsEngine(config: MockEngineConfig) : MockEngine(config) {
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


private var referencedCountAssertions = 0
private fun referencedCountVerify(value: Boolean, label: String) {
    referencedCountAssertions++
    check(value) { label }
}
private fun referencedCountReject(message: String, block: () -> Unit) {
    referencedCountAssertions++
    try { block(); error("Expected DTO property count rejection") }
    catch (cause: SerializationException) { check(cause.message == message) { "Wrong wrapper error: $cause" } }
}
private suspend fun referencedCountWireReject(message: String, block: suspend () -> Unit) {
    referencedCountAssertions++
    try { block(); error("Expected wire property count rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message == message }) { "Wrong wire error: $cause" }
    }
}
private fun referencedCountBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedCountBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected DTO count body $body")
}
private fun <T> referencedCountParity(codec: Json, serializer: KSerializer<T>, automatic: KSerializer<T>, value: T): JsonObject {
    val expected = codec.encodeToJsonElement(automatic, value) as JsonObject
    referencedCountVerify(codec.encodeToJsonElement(serializer, value) == expected, "actual delegate element parity")
    val text = codec.encodeToString(serializer, value)
    referencedCountVerify(text == codec.encodeToString(automatic, value), "actual delegate exact bytes")
    referencedCountVerify(codec.decodeFromString(serializer, text) == value, "string roundtrip")
    referencedCountVerify(codec.decodeFromJsonElement(serializer, expected) == value, "element roundtrip")
    return expected
}
private fun <T> referencedCountInputReject(codec: Json, serializer: KSerializer<T>, source: String, message: String) {
    referencedCountReject(message) { codec.decodeFromString(serializer, source) }
    referencedCountReject(message) { codec.decodeFromJsonElement(serializer, codec.parseToJsonElement(source)) }
}
private object ReferencedCountNonJsonEncoder : AbstractEncoder() {
    override val serializersModule = EmptySerializersModule()
    override fun encodeValue(value: Any): Unit = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeEncoder = error("Non-JSON delegate reached")
}
private object ReferencedCountNonJsonDecoder : AbstractDecoder() {
    override val serializersModule = EmptySerializersModule()
    override fun decodeElementIndex(descriptor: SerialDescriptor): Int = error("Non-JSON delegate reached")
    override fun decodeValue(): Any = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeDecoder = error("Non-JSON delegate reached")
}

suspend fun referencedAllOfCountsContracts() {
    val descriptor = RecordDto.serializer().descriptor
    referencedCountVerify(descriptor === RecordDto.generatedSerializer().descriptor, "retained descriptor identity")
    referencedCountVerify((0 until descriptor.elementsCount).map(descriptor::getElementName) == listOf("value", "kind"), "winning declaration order")
    val request = PatchInputDto.serializer().descriptor
    referencedCountVerify(!request.isElementOptional(1) && request.getElementDescriptor(1).isNullable, "required nullable no default")
    referencedCountVerify(PatchInputDto.Payload.serializer().descriptor === PatchInputDto.Payload.generatedSerializer().descriptor, "nested Patch auto descriptor")
    referencedCountReject("Property counts require JSON.") { RecordDto.serializer().serialize(ReferencedCountNonJsonEncoder, RecordDto("ok")) }
    referencedCountReject("Property counts require JSON.") { RecordDto.serializer().deserialize(ReferencedCountNonJsonDecoder) }
    referencedCountReject("Property counts require JSON.") { ZeroDto.serializer().serialize(ReferencedCountNonJsonEncoder, ZeroDto()) }
    referencedCountReject("Property counts require JSON.") { ZeroDto.serializer().deserialize(ReferencedCountNonJsonDecoder) }
    // Object counts are not ctor/copy barriers; all prior primitive checks remain.
    for (bad in listOf("a", "abcd", "12")) {
        try { RecordDto(bad); error("Lost primitive constructor barrier") }
        catch (cause: IllegalArgumentException) { referencedCountVerify(cause.message?.startsWith("Field value") == true, "safe primitive ctor cause") }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val strict = Json(codec) { ignoreUnknownKeys = false }
        val defaulted = WithDefaultDto()
        referencedCountParity(codec, WithDefaultDto.serializer(), WithDefaultDto.generatedSerializer(), defaulted)
        referencedCountInputReject(codec, WithDefaultDto.serializer(), "{}", "Object violates minProperties.")
        referencedCountVerify(codec.decodeFromString<WithDefaultDto>("{\"unknown\":null}") == defaulted, "count source before default and unknown loss")
        referencedCountInputReject(codec, WithDefaultDto.serializer(), "{\"state\":\"ready\",\"kind\":\"defaulted\",\"unknown\":null}", "Object violates maxProperties.")
        val ordinary = RecordDto("ok")
        val obj = referencedCountParity(codec, RecordDto.serializer(), RecordDto.generatedSerializer(), ordinary)
        referencedCountVerify(obj.size == 2 && obj["kind"] == JsonPrimitive("record"), "member declares one field but whole output two valid")
        referencedCountInputReject(codec, RecordDto.serializer(), "{}", "Object violates minProperties.")
        referencedCountInputReject(codec, RecordDto.serializer(), """{"value":"ok"}""", "Object violates minProperties.")
        referencedCountVerify(codec.decodeFromString(RecordDto.generatedSerializer(), """{"value":"ok"}""") == ordinary, "missing const auto default does not backfill counted input")
        val sourceOnly = """{"value":"ok","unknown":null}"""
        referencedCountVerify(codec.decodeFromString(RecordDto.serializer(), sourceOnly) == ordinary, "unknown/null member counted once before loss")
        referencedCountVerify(codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(sourceOnly)) == ordinary, "incoming element whole object")
        try { strict.decodeFromString(RecordDto.serializer(), sourceOnly); error("Lost strict unknown rejection") }
        catch (cause: SerializationException) { referencedCountVerify(cause.message?.contains("unknown") == true, "valid count still uses caller strict delegate") }
        val overridden = codec.decodeFromString(RecordDto.serializer(), """{"value":"ok","kind":"other"}""")
        referencedCountVerify(overridden.kind == "other", "ordinary decoded const override retained")
        referencedCountVerify((codec.encodeToJsonElement(RecordDto.serializer(), overridden) as JsonObject)["kind"] == JsonPrimitive("other"), "override not repaired")
        referencedCountVerify(RecordDto(overridden.value).kind == "record", "reconstruction retains existing body const behavior")
        for (json in listOf(codec, strict)) referencedCountInputReject(json, RecordDto.serializer(), """{"value":"private-input","kind":"x","u":null}""", "Object violates maxProperties.")
        for (input in listOf("null", "[]", "17", "\"private-input\"")) referencedCountInputReject(codec, RecordDto.serializer(), input, "Expected a JSON object for property counts.")
        referencedCountParity(codec, ZeroDto.serializer(), ZeroDto.generatedSerializer(), ZeroDto())
        // Conflicting member nullability uses the existing first-inline/root
        // approximation, not conjunction; zero-count null skips object checking.
        val nullableSerializer = EnvelopeDto.Nullable.serializer().nullable
        referencedCountVerify(codec.decodeFromString(nullableSerializer, "null") == null, "first inline nullable permits null despite later false")
        referencedCountVerify(codec.encodeToJsonElement(nullableSerializer, null) == JsonNull, "null never counted as one-member object")
        referencedCountVerify(codec.decodeFromString(EnvelopeDto.Nullable.serializer(), "{}") == EnvelopeDto.Nullable(), "nonnull zero whole object")
        referencedCountInputReject(codec, EnvelopeDto.Nullable.serializer(), """{"v":null}""", "Object violates maxProperties.")
        val empty = EnvelopeDto.Nullable()
        val autoEmpty = codec.encodeToJsonElement(EnvelopeDto.Nullable.generatedSerializer(), empty) as JsonObject
        if (autoEmpty.isEmpty()) referencedCountParity(codec, EnvelopeDto.Nullable.serializer(), EnvelopeDto.Nullable.generatedSerializer(), empty)
        else referencedCountReject("Object violates maxProperties.") { codec.encodeToString(EnvelopeDto.Nullable.serializer(), empty) }
        referencedCountVerify(codec.decodeFromString(EnvelopeDto.serializer(), """{"nullable":null,"rootTrue":null}""").nullable == null, "nested nullable nominal consumers")
        var chain = NodeDto("leaf")
        repeat(12) { chain = NodeDto("node", chain) }
        referencedCountParity(codec, NodeDto.serializer(), NodeDto.generatedSerializer(), chain)
        referencedCountInputReject(codec, NodeDto.serializer(), """{"value":"ok","next":{}}""", "Object violates minProperties.")
        val unknownNode = codec.decodeFromString(NodeDto.serializer(), """{"unknown":null}""")
        referencedCountVerify(unknownNode == NodeDto(), "raw count before recursive optional loss")
        if ((codec.encodeToJsonElement(NodeDto.generatedSerializer(), unknownNode) as JsonObject).isEmpty()) {
            referencedCountReject("Object violates minProperties.") { codec.encodeToString(NodeDto.serializer(), unknownNode) }
            referencedCountReject("Object violates minProperties.") { codec.encodeToJsonElement(NodeDto.serializer(), unknownNode) }
        } else referencedCountParity(codec, NodeDto.serializer(), NodeDto.generatedSerializer(), unknownNode)
        val envelope = EnvelopeDto(records = listOf(ordinary), index = mapOf("key" to ordinary))
        referencedCountVerify(codec.decodeFromString(EnvelopeDto.serializer(), codec.encodeToString(EnvelopeDto.serializer(), envelope)) == envelope, "list/map references share nominal codec")
        referencedCountInputReject(codec, EnvelopeDto.serializer(), """{"records":[{}]}""", "Object violates minProperties.")
        referencedCountInputReject(codec, EnvelopeDto.serializer(), """{"index":{"key":{}}}""", "Object violates minProperties.")
        val payload = PatchInputDto.Payload("ok")
        referencedCountVerify(referencedCountParity(codec, PatchInputDto.Payload.serializer(), PatchInputDto.Payload.generatedSerializer(), payload).size == 2, "Patch Absent plus ALWAYS const legal2")
        for (state in listOf<Patch<String>>(Patch.Null, Patch.Value("ok"))) {
            val tooWide = PatchInputDto.Payload(payload.value, state)
            referencedCountVerify((codec.encodeToJsonElement(PatchInputDto.Payload.generatedSerializer(), tooWide) as JsonObject).size == 3, "leaf-valid ctor/copy third member3")
            referencedCountReject("Object violates maxProperties.") { codec.encodeToString(PatchInputDto.Payload.serializer(), tooWide) }
            referencedCountReject("Object violates maxProperties.") { codec.encodeToJsonElement(PatchInputDto.Payload.serializer(), tooWide) }
        }
        val patches = listOf(PatchInputDto(payload, "required"), PatchInputDto(payload, null, Patch.Null), PatchInputDto(payload, "required", Patch.Value("replacement")))
        for (p in patches) {
            val encoded = referencedCountParity(codec, PatchInputDto.serializer(), PatchInputDto.generatedSerializer(), p)
            referencedCountVerify(encoded.containsKey("requiredNull") == (p.requiredNull != null || nulls), "required nullable caller Json")
            referencedCountVerify(when (p.note) { Patch.Absent -> !encoded.containsKey("note"); Patch.Null -> encoded["note"] == JsonNull; is Patch.Value -> encoded["note"] == JsonPrimitive("replacement") }, "Patch three states unchanged")
        }
        val missingRequired = """{"payload":{"value":"ok","kind":"patch"}}"""
        if (nulls) {
            try { codec.decodeFromString(PatchInputDto.serializer(), missingRequired); error("Missing required nullable accepted") }
            catch (cause: MissingFieldException) { referencedCountVerify(cause.missingFields == listOf("requiredNull"), "required nullable missing") }
        } else referencedCountVerify(codec.decodeFromString(PatchInputDto.serializer(), missingRequired).requiredNull == null, "existing implicit null inference")
        referencedCountWire(codec, strict, ordinary, patches, payload)
        println("ReferencedAllOfCounts Json defaults=$defaults nulls=$nulls original-wholeobject-delegate-default-const-Patch-null-HTTP-liveSSE passed")
    }
    println("AllOf DTO property count assertions=$referencedCountAssertions; referenced/nested whole-object sources retained")
}

private suspend fun referencedCountWire(codec: Json, strict: Json, ordinary: RecordDto, patches: List<PatchInputDto>, payload: PatchInputDto.Payload) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var responseMode = 0
    var badSSE = false
    var unknownSSE = false
    var calls = 0
    var expectedPatch = ""
    val engine = ReferencedCountsEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/records" -> {
                    referencedCountVerify(referencedCountBody(request.body) == codec.encodeToString(RecordDto.generatedSerializer(), ordinary), "actual delegated HTTP exact bytes")
                    respond(when (responseMode) { 1 -> """{"value":"ok"}"""; 2 -> """{"value":"private-input","kind":"x","u":null}"""; 3 -> """{"value":"ok","unknown":null}"""; else -> """{"value":"ok","kind":"record"}""" }, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> { referencedCountVerify(referencedCountBody(request.body) == expectedPatch, "HTTP nested Patch actual body"); respond("", HttpStatusCode.NoContent) }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val bad = badSSE
                    val unknown = unknownSSE
                    producers.launch {
                        try {
                            val data = if (bad) """{"value":"private-input","kind":"x","u":null}""" else if (unknown) """{"value":"ok","unknown":null}""" else """{"value":"ok","kind":"record"}"""
                            val frame = "event: record\r\nid: transport-id\r\ndata: $data\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (bad || unknown) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected AllOf count request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ReferencedCounts(http, "http://example.test/", strict).counts
        referencedCountVerify(api.send(ordinary) == ordinary, "typed HTTP legal2 response")
        for (p in patches) { expectedPatch = codec.encodeToString(PatchInputDto.generatedSerializer(), p); api.patchPatch(p) }
        for (state in listOf<Patch<String>>(Patch.Null, Patch.Value("ok"))) {
            var callback = false
            val before = calls
            referencedCountWireReject("Object violates maxProperties.") { api.patchPatch(PatchInputDto(PatchInputDto.Payload(payload.value, state), "required")) { callback = true } }
            referencedCountVerify(callback && calls == before, "callback ran but NO engine dispatch for leaf-valid member3")
        }
        responseMode = 1
        referencedCountWireReject("Object violates minProperties.") { api.send(ordinary) }
        responseMode = 2
        referencedCountWireReject("Object violates maxProperties.") { api.send(ordinary) }
        responseMode = 3
        referencedCountVerify(api.send(ordinary) == ordinary, "HTTP permissive plugin Json after raw count")
        responseMode = 0
        val before = calls
        val cold = api.watch()
        referencedCountVerify(calls == before, "cold Flow no dispatch")
        val first = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
        referencedCountVerify(first.data == ordinary && first.id == "transport-id", "SSE counts data only, not event/id")
        referencedCountVerify(calls == before + 1, "first cold collection")
        referencedCountVerify(withTimeout(10_000) { cold.toList().size } == 1 && calls == before + 2, "independent recollection")
        badSSE = true
        referencedCountWireReject("Object violates maxProperties.") { withTimeout(10_000) { api.watch().toList() } }
        referencedCountVerify(channels.last().isClosedForRead, "invalid known fragmented JSON closes STILL OPEN input")
        badSSE = false
        unknownSSE = true
        try { withTimeout(10_000) { api.watch().toList() }; error("Lost distinct SSE strict Json") }
        catch (cause: Throwable) { referencedCountVerify(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message?.contains("unknown") == true }, "SSE Json distinct from HTTP") }
        referencedCountVerify(channels.last().isClosedForRead, "delegate failure closes live input")
        unknownSSE = false
        referencedCountVerify(withTimeout(10_000) { api.watch().toList().size } == 1, "SSE reuse after failure")
        referencedCountVerify(api.send(ordinary) == ordinary, "HTTP caller client reusable")
    } finally { http.close(); engine.close(); producers.cancel() }
}
