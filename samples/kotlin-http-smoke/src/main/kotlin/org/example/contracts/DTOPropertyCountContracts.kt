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
import org.example.dtocounts.ApiClient
import org.example.dtocounts.api.models.PatchInputDto
import org.example.dtocounts.api.models.WatchEventDto
import org.example.dtocounts.models.EnvelopeDto
import org.example.dtocounts.models.NodeDto
import org.example.dtocounts.models.Patch
import org.example.dtocounts.models.RecordDto
import org.example.dtocounts.models.ShadowDto
import org.example.dtocounts.models.ZeroDto
import org.example.dtocountlimit.ApiClient as LimitClient
import org.example.dtocountlimit.api.models.LimitedInputDto
import org.example.dtocountlimit.models.Patch as LimitPatch

private var dtoAssertions = 0
private fun dtoVerify(value: Boolean, label: String) {
    dtoAssertions++
    check(value) { label }
}
private fun dtoReject(message: String, block: () -> Unit) {
    dtoAssertions++
    try { block(); error("Expected DTO property count rejection") }
    catch (cause: SerializationException) { check(cause.message == message) { "Wrong wrapper error: $cause" } }
}
private suspend fun dtoWireReject(message: String, block: suspend () -> Unit) {
    dtoAssertions++
    try { block(); error("Expected wire property count rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message == message }) { "Wrong wire error: $cause" }
    }
}
private fun dtoBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> dtoBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected DTO count body $body")
}
private fun <T> dtoParity(codec: Json, serializer: KSerializer<T>, automatic: KSerializer<T>, value: T): JsonObject {
    val expected = codec.encodeToJsonElement(automatic, value) as JsonObject
    dtoVerify(codec.encodeToJsonElement(serializer, value) == expected, "actual delegate element parity")
    val text = codec.encodeToString(serializer, value)
    dtoVerify(text == codec.encodeToString(automatic, value), "actual delegate exact bytes")
    dtoVerify(codec.decodeFromString(serializer, text) == value, "string roundtrip")
    dtoVerify(codec.decodeFromJsonElement(serializer, expected) == value, "element roundtrip")
    return expected
}
private fun <T> dtoInputReject(codec: Json, serializer: KSerializer<T>, source: String, message: String) {
    dtoReject(message) { codec.decodeFromString(serializer, source) }
    dtoReject(message) { codec.decodeFromJsonElement(serializer, codec.parseToJsonElement(source)) }
}
private object DTOCountNonJsonEncoder : AbstractEncoder() {
    override val serializersModule = EmptySerializersModule()
    override fun encodeValue(value: Any): Unit = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeEncoder = error("Non-JSON delegate reached")
}
private object DTOCountNonJsonDecoder : AbstractDecoder() {
    override val serializersModule = EmptySerializersModule()
    override fun decodeElementIndex(descriptor: SerialDescriptor): Int = error("Non-JSON delegate reached")
    override fun decodeValue(): Any = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeDecoder = error("Non-JSON delegate reached")
}

suspend fun dtoPropertyCountContracts() {
    dtoVerify(KotlinVersion.CURRENT.toString() == System.getProperty("openapiKt.expectedKotlin", "2.4.10"), "declared Kotlin profile")
    val descriptor = RecordDto.serializer().descriptor
    dtoVerify(descriptor === RecordDto.generatedSerializer().descriptor, "automatic descriptor identity")
    dtoVerify((0 until descriptor.elementsCount).map(descriptor::getElementName) == listOf("id", "optional", "kind"), "descriptor declaration order")
    val requestDescriptor = PatchInputDto.serializer().descriptor
    dtoVerify(!requestDescriptor.isElementOptional(0) && requestDescriptor.getElementDescriptor(0).isNullable, "required nullable unchanged")
    dtoVerify(LimitedInputDto.serializer().descriptor === LimitedInputDto.generatedSerializer().descriptor, "Patch descriptor unchanged")
    dtoReject("Property counts require JSON.") { RecordDto.serializer().serialize(DTOCountNonJsonEncoder, RecordDto("private-input")) }
    dtoReject("Property counts require JSON.") { RecordDto.serializer().deserialize(DTOCountNonJsonDecoder) }
    dtoReject("Property counts require JSON.") { ZeroDto.serializer().serialize(DTOCountNonJsonEncoder, ZeroDto()) }
    dtoReject("Property counts require JSON.") { ZeroDto.serializer().deserialize(DTOCountNonJsonDecoder) }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val strict = Json(codec) { ignoreUnknownKeys = false }
        val ordinary = RecordDto("ok")
        val objectValue = dtoParity(codec, RecordDto.serializer(), RecordDto.generatedSerializer(), ordinary)
        dtoVerify(objectValue.size == 2 + (if (defaults && nulls) 1 else 0), "actual outgoing defaults/null size")
        dtoVerify(objectValue["kind"] == JsonPrimitive("record"), "const ALWAYS unchanged")
        for (input in listOf("""{"id":"ok"}""", """{"id":"ok","optional":null}""", """{"id":"ok","unknown":null}""")) {
            val element = codec.parseToJsonElement(input)
            dtoVerify(codec.decodeFromString(RecordDto.serializer(), input) == codec.decodeFromJsonElement(RecordDto.generatedSerializer(), element), "raw input before defaults/unknown loss")
            dtoVerify(codec.decodeFromJsonElement(RecordDto.serializer(), element) == ordinary, "element incoming")
        }
        val overridden = codec.decodeFromString<RecordDto>("""{"id":"ok","kind":"overridden"}""")
        dtoVerify(overridden.kind == "overridden", "decoded ordinary const override")
        dtoVerify((codec.encodeToJsonElement(RecordDto.serializer(), overridden) as JsonObject)["kind"] == JsonPrimitive("overridden"), "const override output not repaired")
        dtoVerify(overridden.copy().kind == "record", "copy body const remains existing behavior")
        dtoInputReject(codec, RecordDto.serializer(), "{}", "Object violates minProperties.")
        val unknownWide = """{"id":"private-input","u1":null,"u2":null,"u3":null}"""
        for (json in listOf(codec, strict)) dtoInputReject(json, RecordDto.serializer(), unknownWide, "Object violates maxProperties.")
        for (input in listOf("null", "[]", "19", "\"private-input\"")) dtoInputReject(codec, RecordDto.serializer(), input, "Expected a JSON object for property counts.")
        // Caller unknown handling remains automatic once the count is valid.
        try { strict.decodeFromString<RecordDto>("""{"id":"ok","unknown":null}"""); error("Strict caller lost unknown rejection") }
        catch (cause: SerializationException) { dtoVerify(cause.message?.contains("unknown") == true, "delegate unknown behavior") }
        val sourceOnly = codec.decodeFromString<NodeDto>("""{"unknown":null}""")
        dtoVerify(sourceOnly == NodeDto(), "incoming unknown/null each count once before loss")
        val sourceOutput = codec.encodeToJsonElement(NodeDto.generatedSerializer(), sourceOnly) as JsonObject
        if (sourceOutput.isEmpty()) {
            dtoReject("Object violates minProperties.") { codec.encodeToString(NodeDto.serializer(), sourceOnly.copy()) }
            dtoReject("Object violates minProperties.") { codec.encodeToJsonElement(NodeDto.serializer(), NodeDto()) }
        } else dtoParity(codec, NodeDto.serializer(), NodeDto.generatedSerializer(), sourceOnly)
        dtoVerify(codec.decodeFromString(NodeDto.serializer().nullable, "null") == null, "whole nullable object uses nullable codec")
        dtoVerify(codec.encodeToJsonElement(NodeDto.serializer().nullable, null) == JsonNull, "whole nullable encoding")
        var chain = NodeDto("leaf")
        repeat(12) { chain = NodeDto("node", chain) }
        val chainObject = dtoParity(codec, NodeDto.serializer(), NodeDto.generatedSerializer(), chain)
        var cursor: JsonElement? = chainObject
        var nodes = 0
        while (cursor is JsonObject) { dtoVerify(cursor.size in 1..2, "each recursive object own count"); nodes++; cursor = cursor["next"] }
        dtoVerify(nodes == 13, "nominal recursion without self delegate")
        dtoInputReject(codec, NodeDto.serializer(), """{"value":"ok","next":{}}""", "Object violates minProperties.")
        if (sourceOutput.isEmpty()) dtoReject("Object violates minProperties.") { codec.encodeToString(NodeDto.serializer(), NodeDto("parent", NodeDto())) }
        val envelope = EnvelopeDto(listOf(ordinary), mapOf("key" to ordinary))
        dtoVerify(codec.decodeFromString<EnvelopeDto>(codec.encodeToString(EnvelopeDto.serializer(), envelope)) == envelope, "typed list/map consumer shares nominal wrapper")
        dtoInputReject(codec, EnvelopeDto.serializer(), """{"records":[{}]}""", "Object violates minProperties.")
        dtoInputReject(codec, EnvelopeDto.serializer(), """{"index":{"key":{}}}""", "Object violates minProperties.")
        dtoParity(codec, ZeroDto.serializer(), ZeroDto.generatedSerializer(), ZeroDto())
        dtoParity(codec, ShadowDto.serializer(), ShadowDto.generatedSerializer(), ShadowDto())
        val patches = listOf(PatchInputDto("required"), PatchInputDto(null, Patch.Null), PatchInputDto("required", Patch.Value("replacement")))
        for (p in patches) {
            val encoded = dtoParity(codec, PatchInputDto.serializer(), PatchInputDto.generatedSerializer(), p)
            dtoVerify(encoded.containsKey("requiredNull") == (p.requiredNull != null || nulls), "required nullable caller config")
            dtoVerify(when (p.note) { Patch.Absent -> !encoded.containsKey("note"); Patch.Null -> encoded["note"] == JsonNull; is Patch.Value -> encoded["note"] == JsonPrimitive("replacement") }, "generated Patch 3 states")
        }
        if (nulls) {
            try { codec.decodeFromString<PatchInputDto>("""{"kind":"patch"}"""); error("Missing required nullable accepted") }
            catch (cause: MissingFieldException) { dtoVerify(cause.missingFields == listOf("requiredNull"), "automatic required nullable") }
        } else dtoVerify(codec.decodeFromString<PatchInputDto>("""{"kind":"patch"}""").requiredNull == null, "automatic null inference")
        val limited = LimitedInputDto("ok")
        val limitedObject = dtoParity(codec, LimitedInputDto.serializer(), LimitedInputDto.generatedSerializer(), limited)
        dtoVerify(limitedObject.size == 2, "real Patch Absent and const count")
        dtoInputReject(codec, LimitedInputDto.serializer(), """{"id":"ok"}""", "Object violates minProperties.")
        val filled = codec.decodeFromString(LimitedInputDto.generatedSerializer(), """{"id":"ok"}""")
        dtoVerify(filled.kind == "limited", "missing const delegate default does not retroactively satisfy input")
        for (state in listOf<LimitPatch<String>>(LimitPatch.Null, LimitPatch.Value("private-input"))) {
            val tooWide = limited.copy(note = state)
            dtoVerify((codec.encodeToJsonElement(LimitedInputDto.generatedSerializer(), tooWide) as JsonObject).size == 3, "ctor/copy are not barriers; real delegated output size")
            dtoReject("Object violates maxProperties.") { codec.encodeToString(LimitedInputDto.serializer(), tooWide) }
            dtoReject("Object violates maxProperties.") { codec.encodeToJsonElement(LimitedInputDto.serializer(), tooWide) }
        }
        dtoCountWire(codec, strict, ordinary, patches, limited)
        println("DTO property count contracts passed: defaults=$defaults/nulls=$nulls; real generated string/element serde/Patch/const/nullable/13-level recursion/nonJSON/HTTP/liveSSE")
    }
    println("DTO property count assertions=$dtoAssertions; 120Long/240boxed counted DTO boundaries are compilation-only")
}

private suspend fun dtoCountWire(codec: Json, strict: Json, ordinary: RecordDto, patches: List<PatchInputDto>, limited: LimitedInputDto) {
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
                    dtoVerify(dtoBody(request.body) == codec.encodeToString(RecordDto.generatedSerializer(), ordinary), "HTTP actual delegated outgoing bytes")
                    respond(when (responseMode) { 1 -> "{}"; 2 -> """{"id":"private-input","u1":null,"u2":null,"u3":null}"""; 3 -> """{"id":"ok","unknown":null}"""; else -> """{"id":"ok"}""" }, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> { dtoVerify(dtoBody(request.body) == expectedPatch, "HTTP Patch actual bytes"); respond("", HttpStatusCode.NoContent) }
                "/limited" -> { dtoVerify(dtoBody(request.body) == codec.encodeToString(LimitedInputDto.generatedSerializer(), limited), "HTTP legal limited exact body"); respond("", HttpStatusCode.NoContent) }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val bad = badSSE
                    val unknown = unknownSSE
                    producers.launch {
                        try {
                            val data = if (bad) """{"id":"private-input","u1":null,"u2":null,"u3":null}""" else if (unknown) """{"id":"ok","unknown":null}""" else """{"id":"ok"}"""
                            val frame = "event: record\r\nid: transport-id\r\ndata: $data\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (bad || unknown) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected DTO property count request")
            }
        }
    })
    // HTTP serde follows plugin Json. SSE gets a distinct, strict caller Json.
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ApiClient(http, "http://example.test/", strict).counts
        val limit = LimitClient(http, "http://example.test/").limit
        dtoVerify(api.send(ordinary).kind == "record", "HTTP response missing const defaults after valid raw count")
        for (p in patches) { expectedPatch = codec.encodeToString(PatchInputDto.generatedSerializer(), p); api.patchPatch(p) }
        limit.limited(limited)
        for (state in listOf<LimitPatch<String>>(LimitPatch.Null, LimitPatch.Value("private-input"))) {
            var callback = false
            val before = calls
            dtoWireReject("Object violates maxProperties.") { limit.limited(limited.copy(note = state)) { callback = true } }
            dtoVerify(callback && calls == before, "callback before serialization count error; NO engine dispatch")
        }
        responseMode = 1
        dtoWireReject("Object violates minProperties.") { api.send(ordinary) }
        responseMode = 2
        dtoWireReject("Object violates maxProperties.") { api.send(ordinary) }
        responseMode = 3
        dtoVerify(api.send(ordinary) == ordinary, "HTTP unknown handling after valid raw count uses plugin Json")
        responseMode = 0
        val before = calls
        val cold = api.watch()
        dtoVerify(calls == before, "cold Flow no dispatch")
        val first = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
        dtoVerify(first.data == ordinary && first.id == "transport-id", "SSE counts data not event/id")
        dtoVerify(calls == before + 1, "cold first collection opens stream")
        dtoVerify(withTimeout(10_000) { cold.toList().size } == 1 && calls == before + 2, "cold recollection opens independent stream")
        badSSE = true
        dtoWireReject("Object violates maxProperties.") { withTimeout(10_000) { api.watch().toList() } }
        dtoVerify(channels.last().isClosedForRead, "bad known data closes STILL OPEN fragmented channel")
        badSSE = false
        unknownSSE = true
        try { withTimeout(10_000) { api.watch().toList() }; error("SSE caller Json lost strict unknown rejection") }
        catch (cause: Throwable) { dtoVerify(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message?.contains("unknown") == true }, "SSE distinct strict Json, not permissive HTTP codec") }
        dtoVerify(channels.last().isClosedForRead, "delegate failure also closes live SSE input")
        unknownSSE = false
        dtoVerify(withTimeout(10_000) { api.watch().toList().size } == 1, "SSE client reusable after decode rejection")
        dtoVerify(api.send(ordinary) == ordinary, "HTTP client reusable")
    } finally { http.close(); engine.close(); producers.cancel() }
}
