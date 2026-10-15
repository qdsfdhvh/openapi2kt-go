@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
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
import org.example.allofcounts.ApiClient
import org.example.allofcounts.api.models.PatchInputDto
import org.example.allofcounts.api.models.WatchEventDto
import org.example.allofcounts.models.EnvelopeDto
import org.example.allofcounts.models.NodeDto
import org.example.allofcounts.models.Patch
import org.example.allofcounts.models.RecordDto
import org.example.allofcounts.models.ZeroDto
import org.example.http.limits.models.AllOfCountHelperShadow
import org.example.http.limits.models.AllOfCountNestedShadow
import org.example.http.limits.models.AllOfCountOpenapiKtPropertyCountSerializer

private var allOfCountAssertions = 0
private fun allOfCountVerify(value: Boolean, label: String) {
    allOfCountAssertions++
    check(value) { label }
}
private fun allOfCountReject(message: String, block: () -> Unit) {
    allOfCountAssertions++
    try { block(); error("Expected DTO property count rejection") }
    catch (cause: SerializationException) { check(cause.message == message) { "Wrong wrapper error: $cause" } }
}
private suspend fun allOfCountWireReject(message: String, block: suspend () -> Unit) {
    allOfCountAssertions++
    try { block(); error("Expected wire property count rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message == message }) { "Wrong wire error: $cause" }
    }
}
private fun allOfCountBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> allOfCountBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected DTO count body $body")
}
private fun <T> allOfCountParity(codec: Json, serializer: KSerializer<T>, automatic: KSerializer<T>, value: T): JsonObject {
    val expected = codec.encodeToJsonElement(automatic, value) as JsonObject
    allOfCountVerify(codec.encodeToJsonElement(serializer, value) == expected, "actual delegate element parity")
    val text = codec.encodeToString(serializer, value)
    allOfCountVerify(text == codec.encodeToString(automatic, value), "actual delegate exact bytes")
    allOfCountVerify(codec.decodeFromString(serializer, text) == value, "string roundtrip")
    allOfCountVerify(codec.decodeFromJsonElement(serializer, expected) == value, "element roundtrip")
    return expected
}
private fun <T> allOfCountInputReject(codec: Json, serializer: KSerializer<T>, source: String, message: String) {
    allOfCountReject(message) { codec.decodeFromString(serializer, source) }
    allOfCountReject(message) { codec.decodeFromJsonElement(serializer, codec.parseToJsonElement(source)) }
}
private object AllOfCountNonJsonEncoder : AbstractEncoder() {
    override val serializersModule = EmptySerializersModule()
    override fun encodeValue(value: Any): Unit = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeEncoder = error("Non-JSON delegate reached")
}
private object AllOfCountNonJsonDecoder : AbstractDecoder() {
    override val serializersModule = EmptySerializersModule()
    override fun decodeElementIndex(descriptor: SerialDescriptor): Int = error("Non-JSON delegate reached")
    override fun decodeValue(): Any = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeDecoder = error("Non-JSON delegate reached")
}

suspend fun allOfDTOPropertyCountContracts() {
    val descriptor = RecordDto.serializer().descriptor
    allOfCountVerify(descriptor === RecordDto.generatedSerializer().descriptor, "retained descriptor identity")
    allOfCountVerify((0 until descriptor.elementsCount).map(descriptor::getElementName) == listOf("value", "kind"), "winning declaration order")
    val request = PatchInputDto.serializer().descriptor
    allOfCountVerify(!request.isElementOptional(1) && request.getElementDescriptor(1).isNullable, "required nullable no default")
    allOfCountVerify(PatchInputDto.Payload.serializer().descriptor === PatchInputDto.Payload.generatedSerializer().descriptor, "nested Patch auto descriptor")
    allOfCountReject("Property counts require JSON.") { RecordDto.serializer().serialize(AllOfCountNonJsonEncoder, RecordDto("ok")) }
    allOfCountReject("Property counts require JSON.") { RecordDto.serializer().deserialize(AllOfCountNonJsonDecoder) }
    allOfCountReject("Property counts require JSON.") { ZeroDto.serializer().serialize(AllOfCountNonJsonEncoder, ZeroDto()) }
    allOfCountReject("Property counts require JSON.") { ZeroDto.serializer().deserialize(AllOfCountNonJsonDecoder) }
    // Object counts are not ctor/copy barriers; all prior primitive checks remain.
    for (bad in listOf("a", "abcd", "12")) {
        try { RecordDto(bad); error("Lost primitive constructor barrier") }
        catch (cause: IllegalArgumentException) { allOfCountVerify(cause.message?.startsWith("Field value") == true, "safe primitive ctor cause") }
        try { RecordDto("ok").copy(value = bad); error("Lost primitive copy barrier") }
        catch (cause: IllegalArgumentException) { allOfCountVerify(cause.message?.startsWith("Field value") == true, "safe primitive copy cause") }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val strict = Json(codec) { ignoreUnknownKeys = false }
        val ordinary = RecordDto("ok")
        val obj = allOfCountParity(codec, RecordDto.serializer(), RecordDto.generatedSerializer(), ordinary)
        allOfCountVerify(obj.size == 2 && obj["kind"] == JsonPrimitive("record"), "member declares one field but whole output two valid")
        allOfCountInputReject(codec, RecordDto.serializer(), "{}", "Object violates minProperties.")
        allOfCountInputReject(codec, RecordDto.serializer(), """{"value":"ok"}""", "Object violates minProperties.")
        allOfCountVerify(codec.decodeFromString(RecordDto.generatedSerializer(), """{"value":"ok"}""") == ordinary, "missing const auto default does not backfill counted input")
        val sourceOnly = """{"value":"ok","unknown":null}"""
        allOfCountVerify(codec.decodeFromString(RecordDto.serializer(), sourceOnly) == ordinary, "unknown/null member counted once before loss")
        allOfCountVerify(codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(sourceOnly)) == ordinary, "incoming element whole object")
        try { strict.decodeFromString(RecordDto.serializer(), sourceOnly); error("Lost strict unknown rejection") }
        catch (cause: SerializationException) { allOfCountVerify(cause.message?.contains("unknown") == true, "valid count still uses caller strict delegate") }
        val overridden = codec.decodeFromString(RecordDto.serializer(), """{"value":"ok","kind":"other"}""")
        allOfCountVerify(overridden.kind == "other", "ordinary decoded const override retained")
        allOfCountVerify((codec.encodeToJsonElement(RecordDto.serializer(), overridden) as JsonObject)["kind"] == JsonPrimitive("other"), "override not repaired")
        allOfCountVerify(overridden.copy().kind == "record", "copy retains old body const behavior")
        for (json in listOf(codec, strict)) allOfCountInputReject(json, RecordDto.serializer(), """{"value":"private-input","kind":"x","u":null}""", "Object violates maxProperties.")
        for (input in listOf("null", "[]", "17", "\"private-input\"")) allOfCountInputReject(codec, RecordDto.serializer(), input, "Expected a JSON object for property counts.")
        allOfCountParity(codec, ZeroDto.serializer(), ZeroDto.generatedSerializer(), ZeroDto())
        // Conflicting member nullability uses the existing first-inline/root
        // approximation, not conjunction; zero-count null skips object checking.
        val nullableSerializer = EnvelopeDto.Nullable.serializer().nullable
        allOfCountVerify(codec.decodeFromString(nullableSerializer, "null") == null, "first inline nullable permits null despite later false")
        allOfCountVerify(codec.encodeToJsonElement(nullableSerializer, null) == JsonNull, "null never counted as one-member object")
        allOfCountVerify(codec.decodeFromString(EnvelopeDto.Nullable.serializer(), "{}") == EnvelopeDto.Nullable(), "nonnull zero whole object")
        allOfCountInputReject(codec, EnvelopeDto.Nullable.serializer(), """{"v":null}""", "Object violates maxProperties.")
        val empty = EnvelopeDto.Nullable()
        val autoEmpty = codec.encodeToJsonElement(EnvelopeDto.Nullable.generatedSerializer(), empty) as JsonObject
        if (autoEmpty.isEmpty()) allOfCountParity(codec, EnvelopeDto.Nullable.serializer(), EnvelopeDto.Nullable.generatedSerializer(), empty)
        else allOfCountReject("Object violates maxProperties.") { codec.encodeToString(EnvelopeDto.Nullable.serializer(), empty.copy()) }
        allOfCountVerify(codec.decodeFromString(EnvelopeDto.serializer(), """{"nullable":null,"rootTrue":null}""").nullable == null, "nested nullable nominal consumers")
        var chain = NodeDto("leaf")
        repeat(12) { chain = NodeDto("node", chain) }
        allOfCountParity(codec, NodeDto.serializer(), NodeDto.generatedSerializer(), chain)
        allOfCountInputReject(codec, NodeDto.serializer(), """{"value":"ok","next":{}}""", "Object violates minProperties.")
        val unknownNode = codec.decodeFromString(NodeDto.serializer(), """{"unknown":null}""")
        allOfCountVerify(unknownNode == NodeDto(), "raw count before recursive optional loss")
        if ((codec.encodeToJsonElement(NodeDto.generatedSerializer(), unknownNode) as JsonObject).isEmpty()) {
            allOfCountReject("Object violates minProperties.") { codec.encodeToString(NodeDto.serializer(), unknownNode.copy()) }
            allOfCountReject("Object violates minProperties.") { codec.encodeToJsonElement(NodeDto.serializer(), unknownNode) }
        } else allOfCountParity(codec, NodeDto.serializer(), NodeDto.generatedSerializer(), unknownNode)
        val envelope = EnvelopeDto(records = listOf(ordinary), index = mapOf("key" to ordinary))
        allOfCountVerify(codec.decodeFromString(EnvelopeDto.serializer(), codec.encodeToString(EnvelopeDto.serializer(), envelope)) == envelope, "list/map references share nominal codec")
        allOfCountInputReject(codec, EnvelopeDto.serializer(), """{"records":[{}]}""", "Object violates minProperties.")
        allOfCountInputReject(codec, EnvelopeDto.serializer(), """{"index":{"key":{}}}""", "Object violates minProperties.")
        val payload = PatchInputDto.Payload("ok")
        allOfCountVerify(allOfCountParity(codec, PatchInputDto.Payload.serializer(), PatchInputDto.Payload.generatedSerializer(), payload).size == 2, "Patch Absent plus ALWAYS const legal2")
        for (state in listOf<Patch<String>>(Patch.Null, Patch.Value("ok"))) {
            val tooWide = payload.copy(note = state)
            allOfCountVerify((codec.encodeToJsonElement(PatchInputDto.Payload.generatedSerializer(), tooWide) as JsonObject).size == 3, "leaf-valid ctor/copy third member3")
            allOfCountReject("Object violates maxProperties.") { codec.encodeToString(PatchInputDto.Payload.serializer(), tooWide) }
            allOfCountReject("Object violates maxProperties.") { codec.encodeToJsonElement(PatchInputDto.Payload.serializer(), tooWide) }
        }
        val patches = listOf(PatchInputDto(payload, "required"), PatchInputDto(payload, null, Patch.Null), PatchInputDto(payload, "required", Patch.Value("replacement")))
        for (p in patches) {
            val encoded = allOfCountParity(codec, PatchInputDto.serializer(), PatchInputDto.generatedSerializer(), p)
            allOfCountVerify(encoded.containsKey("requiredNull") == (p.requiredNull != null || nulls), "required nullable caller Json")
            allOfCountVerify(when (p.note) { Patch.Absent -> !encoded.containsKey("note"); Patch.Null -> encoded["note"] == JsonNull; is Patch.Value -> encoded["note"] == JsonPrimitive("replacement") }, "Patch three states unchanged")
        }
        val missingRequired = """{"payload":{"value":"ok","kind":"patch"}}"""
        if (nulls) {
            try { codec.decodeFromString(PatchInputDto.serializer(), missingRequired); error("Missing required nullable accepted") }
            catch (cause: MissingFieldException) { allOfCountVerify(cause.missingFields == listOf("requiredNull"), "required nullable missing") }
        } else allOfCountVerify(codec.decodeFromString(PatchInputDto.serializer(), missingRequired).requiredNull == null, "existing implicit null inference")
        allOfCountParity(codec, AllOfCountHelperShadow.serializer(), AllOfCountHelperShadow.generatedSerializer(), AllOfCountHelperShadow())
        allOfCountParity(codec, AllOfCountNestedShadow.serializer(), AllOfCountNestedShadow.generatedSerializer(), AllOfCountNestedShadow(openapiKtPropertyCountSerializer = AllOfCountNestedShadow.OpenapiKtPropertyCountSerializer()))
        allOfCountParity(codec, AllOfCountOpenapiKtPropertyCountSerializer.serializer(), AllOfCountOpenapiKtPropertyCountSerializer.generatedSerializer(), AllOfCountOpenapiKtPropertyCountSerializer())
        allOfCountWire(codec, strict, ordinary, patches, payload)
        println("AllOf DTO property count contracts passed: defaults=$defaults/nulls=$nulls; whole JSON/retained serde/Patch/nullable/primitive barriers/HTTP/live SSE")
    }
    println("AllOf DTO property count assertions=$allOfCountAssertions; composed120LongDouble/240mixed boundaries compilation-only")
}

private suspend fun allOfCountWire(codec: Json, strict: Json, ordinary: RecordDto, patches: List<PatchInputDto>, payload: PatchInputDto.Payload) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var responseMode = 0
    var badSSE = false
    var unknownSSE = false
    var calls = 0
    var expectedPatch = ""
    val engine = StreamingTestEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/records" -> {
                    allOfCountVerify(allOfCountBody(request.body) == codec.encodeToString(RecordDto.generatedSerializer(), ordinary), "actual delegated HTTP exact bytes")
                    respond(when (responseMode) { 1 -> """{"value":"ok"}"""; 2 -> """{"value":"private-input","kind":"x","u":null}"""; 3 -> """{"value":"ok","unknown":null}"""; else -> """{"value":"ok","kind":"record"}""" }, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> { allOfCountVerify(allOfCountBody(request.body) == expectedPatch, "HTTP nested Patch actual body"); respond("", HttpStatusCode.NoContent) }
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
        val api = ApiClient(http, "http://example.test/", strict).counts
        allOfCountVerify(api.send(ordinary) == ordinary, "typed HTTP legal2 response")
        for (p in patches) { expectedPatch = codec.encodeToString(PatchInputDto.generatedSerializer(), p); api.patchPatch(p) }
        for (state in listOf<Patch<String>>(Patch.Null, Patch.Value("ok"))) {
            var callback = false
            val before = calls
            allOfCountWireReject("Object violates maxProperties.") { api.patchPatch(PatchInputDto(payload.copy(note = state), "required")) { callback = true } }
            allOfCountVerify(callback && calls == before, "callback ran but NO engine dispatch for leaf-valid member3")
        }
        responseMode = 1
        allOfCountWireReject("Object violates minProperties.") { api.send(ordinary) }
        responseMode = 2
        allOfCountWireReject("Object violates maxProperties.") { api.send(ordinary) }
        responseMode = 3
        allOfCountVerify(api.send(ordinary) == ordinary, "HTTP permissive plugin Json after raw count")
        responseMode = 0
        val before = calls
        val cold = api.watch()
        allOfCountVerify(calls == before, "cold Flow no dispatch")
        val first = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
        allOfCountVerify(first.data == ordinary && first.id == "transport-id", "SSE counts data only, not event/id")
        allOfCountVerify(calls == before + 1, "first cold collection")
        allOfCountVerify(withTimeout(10_000) { cold.toList().size } == 1 && calls == before + 2, "independent recollection")
        badSSE = true
        allOfCountWireReject("Object violates maxProperties.") { withTimeout(10_000) { api.watch().toList() } }
        allOfCountVerify(channels.last().isClosedForRead, "invalid known fragmented JSON closes STILL OPEN input")
        badSSE = false
        unknownSSE = true
        try { withTimeout(10_000) { api.watch().toList() }; error("Lost distinct SSE strict Json") }
        catch (cause: Throwable) { allOfCountVerify(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message?.contains("unknown") == true }, "SSE Json distinct from HTTP") }
        allOfCountVerify(channels.last().isClosedForRead, "delegate failure closes live input")
        unknownSSE = false
        allOfCountVerify(withTimeout(10_000) { api.watch().toList().size } == 1, "SSE reuse after failure")
        allOfCountVerify(api.send(ordinary) == ordinary, "HTTP caller client reusable")
    } finally { http.close(); engine.close(); producers.cancel() }
}
