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
import kotlinx.serialization.descriptors.SerialDescriptor
import kotlinx.serialization.encoding.AbstractDecoder
import kotlinx.serialization.encoding.AbstractEncoder
import kotlinx.serialization.encoding.CompositeDecoder
import kotlinx.serialization.encoding.CompositeEncoder
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.modules.EmptySerializersModule
import kotlinx.serialization.serializer
import org.example.directcounts.DirectCounts
import org.example.directcounts.api.models.PatchInputDto
import org.example.directcounts.api.models.WatchEventDto
import org.example.directcounts.models.CheckedDto
import org.example.directcounts.models.DefaultedDto
import org.example.directcounts.models.HolderDto
import org.example.directcounts.models.InheritedDto
import org.example.directcounts.models.NarrowDto
import org.example.directcounts.models.NestedAliasDto
import org.example.directcounts.models.NestedUseDto
import org.example.directcounts.models.Patch
import org.example.directcounts.models.RecordDto
import org.example.directcounts.models.RenullDto
import org.example.directcounts.models.WithConstDto
import org.example.directcounts.models.ZeroDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class DirectCountsEngine(config: MockEngineConfig) : MockEngine(config) {
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

private var directCountsAssertions = 0
private fun directCountVerify(value: Boolean, label: String) {
    directCountsAssertions++
    check(value) { label }
}
private fun directCountReject(message: String, block: () -> Unit) {
    directCountsAssertions++
    try { block(); error("Expected direct nullable object count rejection") }
    catch (cause: SerializationException) { check(cause.message == message) { "Unexpected count cause: $cause" } }
}
private suspend fun directCountWireReject(message: String, block: suspend () -> Unit) {
    directCountsAssertions++
    try { block(); error("Expected wire count rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message == message }) { "Unexpected wire cause: $cause" }
    }
}
private fun <T> directCountParity(codec: Json, retained: KSerializer<T>, automatic: KSerializer<T>, value: T): JsonObject {
    val expected = codec.encodeToJsonElement(automatic, value) as JsonObject
    directCountVerify(codec.encodeToJsonElement(retained, value) == expected, "actual generated delegate object parity")
    val text = codec.encodeToString(retained, value)
    directCountVerify(text == codec.encodeToString(automatic, value), "actual generated delegate exact bytes")
    directCountVerify(codec.decodeFromString(retained, text) == value, "retained string roundtrip")
    directCountVerify(codec.decodeFromJsonElement(retained, expected) == value, "retained element roundtrip")
    return expected
}
private fun <T> directCountInputReject(codec: Json, serializer: KSerializer<T>, source: String, message: String) {
    directCountReject(message) { codec.decodeFromString(serializer, source) }
    directCountReject(message) { codec.decodeFromJsonElement(serializer, codec.parseToJsonElement(source)) }
}
private object DirectCountNonJsonEncoder : AbstractEncoder() {
    override val serializersModule = EmptySerializersModule()
    override fun encodeValue(value: Any): Unit = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeEncoder = error("Non-JSON delegate reached")
}
private object DirectCountNonJsonDecoder : AbstractDecoder() {
    override val serializersModule = EmptySerializersModule()
    override fun decodeElementIndex(descriptor: SerialDescriptor): Int = error("Non-JSON delegate reached")
    override fun decodeValue(): Any = error("Non-JSON automatic delegate reached")
    override fun beginStructure(descriptor: SerialDescriptor): CompositeDecoder = error("Non-JSON delegate reached")
}

suspend fun directNullableComponentCountsContracts() {
    val physical = RecordDto.serializer().descriptor
    directCountVerify(physical === RecordDto.generatedSerializer().descriptor && !physical.isNullable, "original non-null physical descriptor identity")
    directCountVerify(physical.elementsCount == 1 && physical.getElementName(0) == "value" && physical.isElementOptional(0), "original constructor/default descriptor")
    directCountVerify(InheritedDto.serializer().descriptor === physical, "nullable alias companion stays canonical non-null")
    directCountVerify(serializer<InheritedDto>().descriptor.isNullable && serializer<RenullDto>().descriptor.isNullable, "reified inherited and restored nullable alias codecs")
    directCountVerify(!serializer<NarrowDto>().descriptor.isNullable, "nearest false genuine physical view")
    val requestDescriptor = PatchInputDto.serializer().descriptor
    directCountVerify(!requestDescriptor.isElementOptional(0) && requestDescriptor.getElementDescriptor(0).isNullable, "request required nullable no default")
    directCountVerify(requestDescriptor.isElementOptional(1) && requestDescriptor.isElementOptional(2), "Patch versus narrowed optional presence")
    directCountReject("Property counts require JSON.") { RecordDto.serializer().serialize(DirectCountNonJsonEncoder, RecordDto("ok")) }
    directCountReject("Property counts require JSON.") { RecordDto.serializer().deserialize(DirectCountNonJsonDecoder) }
    directCountReject("Property counts require JSON.") { ZeroDto.serializer().serialize(DirectCountNonJsonEncoder, ZeroDto()) }
    for (bad in listOf("a", "abcd")) {
        try { CheckedDto(bad); error("Lost original primitive constructor check") }
        catch (cause: IllegalArgumentException) { directCountVerify(cause.message?.startsWith("Field value") == true, "original safe primitive check") }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val strict = Json(codec) { ignoreUnknownKeys = false }
        val record = RecordDto("ok")
        directCountVerify(directCountParity(codec, RecordDto.serializer(), RecordDto.generatedSerializer(), record).size == 1, "ordinary direct output one member")
        for (nullable in listOf(RecordDto.serializer().nullable, serializer<InheritedDto>(), serializer<RenullDto>())) {
            directCountVerify(codec.decodeFromString(nullable, "null") == null, "root null skips count min/max")
            directCountVerify(codec.encodeToJsonElement(nullable, null) == JsonNull, "root null original nullable encoder")
        }
        directCountVerify(codec.decodeFromString(ZeroDto.serializer().nullable, "null") == null, "zero-object null is not one member")
        directCountInputReject(codec, serializer<NarrowDto>(), "null", "Expected a JSON object for property counts.")
        for (input in listOf("[]", "1", "\"text\"")) directCountInputReject(codec, RecordDto.serializer().nullable, input, "Expected a JSON object for property counts.")
        directCountInputReject(codec, RecordDto.serializer(), "{}", "Object violates minProperties.")
        directCountVerify(codec.decodeFromString(RecordDto.serializer(), "{\"unknown\":null}") == RecordDto(), "unknown explicit null counts before unknown loss")
        directCountVerify(codec.decodeFromString(RecordDto.serializer(), "{\"value\":null}") == RecordDto(), "serialized null key is one physical member")
        for (json in listOf(codec, strict)) directCountInputReject(json, RecordDto.serializer(), "{\"value\":\"ok\",\"u\":null,\"v\":null}", "Object violates maxProperties.")
        try { strict.decodeFromString(RecordDto.serializer(), "{\"value\":\"ok\",\"unknown\":null}"); error("Lost strict delegate unknown behavior") }
        catch (cause: SerializationException) { directCountVerify(cause.message?.contains("unknown") == true, "count-valid object still uses caller strict delegate") }
        // These constructors are legal. Their actual encoded presence decides
        // counts; neither constructor field count nor decode defaults repairs it.
        val empty = RecordDto()
        val actualEmpty = codec.encodeToJsonElement(RecordDto.generatedSerializer(), empty) as JsonObject
        directCountVerify(actualEmpty.size == (if (defaults && nulls) 1 else 0), "unchanged encodeDefaults/explicitNulls physical presence")
        if (actualEmpty.isEmpty()) {
            directCountReject("Object violates minProperties.") { codec.encodeToString(RecordDto.serializer(), empty) }
            directCountReject("Object violates minProperties.") { codec.encodeToJsonElement(RecordDto.serializer(), empty) }
        } else directCountParity(codec, RecordDto.serializer(), RecordDto.generatedSerializer(), empty)
        val defaulted = DefaultedDto()
        directCountInputReject(codec, DefaultedDto.serializer(), "{}", "Object violates minProperties.")
        directCountVerify(codec.decodeFromString(DefaultedDto.generatedSerializer(), "{}") == defaulted, "automatic default is not counted input")
        directCountVerify(codec.decodeFromString(DefaultedDto.serializer(), "{\"unknown\":null}") == defaulted, "original unknown count precedes default application")
        if (defaults) directCountParity(codec, DefaultedDto.serializer(), DefaultedDto.generatedSerializer(), defaulted)
        else directCountReject("Object violates minProperties.") { codec.encodeToString(DefaultedDto.serializer(), defaulted) }
        val withConst = WithConstDto("ok")
        directCountVerify(directCountParity(codec, WithConstDto.serializer(), WithConstDto.generatedSerializer(), withConst).size == 2, "always encoded const joins whole physical object")
        directCountInputReject(codec, WithConstDto.serializer(), "{\"value\":\"ok\"}", "Object violates minProperties.")
        directCountVerify(codec.decodeFromString(WithConstDto.serializer(), "{\"value\":\"ok\",\"unknown\":null}") == withConst, "missing const is supplied only after raw count")
        val overridden = codec.decodeFromString(WithConstDto.serializer(), "{\"value\":\"ok\",\"kind\":\"other\"}")
        directCountVerify(overridden.kind == "other" && (codec.encodeToJsonElement(WithConstDto.serializer(), overridden) as JsonObject)["kind"] == JsonPrimitive("other"), "ordinary decoded const override remains")
        val zero = ZeroDto()
        directCountInputReject(codec, ZeroDto.serializer(), "{\"value\":null}", "Object violates maxProperties.")
        directCountVerify(codec.decodeFromString(ZeroDto.serializer(), "{}") == zero, "zero incoming object allowed")
        if (defaults && nulls) directCountReject("Object violates maxProperties.") { codec.encodeToString(ZeroDto.serializer(), zero) }
        else directCountParity(codec, ZeroDto.serializer(), ZeroDto.generatedSerializer(), zero)
        directCountReject("Object violates maxProperties.") { codec.encodeToString(ZeroDto.serializer(), ZeroDto("legal constructor")) }
        val holder = HolderDto(narrow = record, direct = null, alias = record, records = listOf(null, record), index = mapOf("null" to null, "value" to record))
        directCountVerify(codec.decodeFromString(HolderDto.serializer(), codec.encodeToString(HolderDto.serializer(), holder)) == holder, "nested refs and independent List/Map null codecs")
        directCountInputReject(codec, HolderDto.serializer(), "{\"narrow\":{\"value\":\"ok\"},\"records\":[{}]}", "Object violates minProperties.")
        directCountInputReject(codec, HolderDto.serializer(), "{\"narrow\":{\"value\":\"ok\"},\"index\":{\"x\":{}}}", "Object violates minProperties.")
        directCountVerify(codec.decodeFromString(NestedUseDto.serializer(), "{\"value\":{}}").value == NestedAliasDto(), "internal nested reference uses original physical owner")
        directCountInputReject(codec, NestedUseDto.serializer(), "{\"value\":{\"a\":null,\"b\":null}}", "Object violates maxProperties.")
        val patches = listOf(PatchInputDto(null), PatchInputDto(record, Patch.Null), PatchInputDto(record, Patch.Value(record)))
        for (p in patches) {
            val encoded = codec.encodeToJsonElement(PatchInputDto.serializer(), p) as JsonObject
            directCountVerify(encoded.containsKey("requiredNull") == (p.requiredNull != null || nulls), "required nullable caller settings retained")
            directCountVerify(when (p.note) { Patch.Absent -> !encoded.containsKey("note"); Patch.Null -> encoded["note"] == JsonNull; is Patch.Value -> encoded["note"] is JsonObject }, "Absent/Null/Value exact bare wire")
            directCountVerify(codec.decodeFromJsonElement(PatchInputDto.serializer(), encoded) == p, "Patch presence roundtrip")
        }
        if (nulls) {
            try { codec.decodeFromString(PatchInputDto.serializer(), "{}"); error("Lost required nullable missing policy") }
            catch (cause: MissingFieldException) { directCountVerify(cause.missingFields == listOf("requiredNull"), "original missing-key cause") }
        } else directCountVerify(codec.decodeFromString(PatchInputDto.serializer(), "{}").requiredNull == null, "original implicit missing null policy")
        if (actualEmpty.isEmpty()) directCountReject("Object violates minProperties.") {
            codec.encodeToString(PatchInputDto.serializer(), PatchInputDto(null, Patch.Value(empty)))
        }
        directCountWire(codec, strict, record, patches)
        println("DirectNullableComponentCounts defaults=$defaults nulls=$nulls original-physical-object-null-delegate-presence-HTTP-fragmentedSSE passed")
    }
    println("Direct nullable component count assertions=$directCountsAssertions")
}

private fun directCountBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> directCountBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected direct nullable count body $body")
}
private suspend fun directCountWire(codec: Json, strict: Json, record: RecordDto, patches: List<PatchInputDto>) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var response = "{\"value\":\"ok\"}"
    var frameData = "{\"value\":\"ok\"}"
    var holdOpen = false
    var calls = 0
    var expectedBody = ""
    val engine = DirectCountsEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/records", "/narrow", "/zero", "/checked" -> {
                    directCountVerify(directCountBody(request.body) == expectedBody, "actual typed HTTP caller-codec body bytes")
                    respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> { directCountVerify(directCountBody(request.body) == expectedBody, "HTTP Patch physical body"); respond("", HttpStatusCode.NoContent) }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val data = frameData
                    val live = holdOpen
                    producers.launch {
                        try {
                            val frame = "event: record\r\nid: transport-id\r\ndata: $data\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (live) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected direct count operation")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = DirectCounts(http, "http://example.test/", strict).counts
        expectedBody = codec.encodeToString(RecordDto.generatedSerializer(), record)
        directCountVerify(api.send(record) == record && api.sendNarrow(record) == record, "nullable and narrowed typed HTTP value codecs")
        response = "null"; expectedBody = "null"
        directCountVerify(api.send(null) == null && api.sendZero(null) == null, "typed root HTTP null skips both minimum and zero bounds")
        expectedBody = codec.encodeToString(RecordDto.generatedSerializer(), record)
        directCountWireReject("Expected a JSON object for property counts.") { api.sendNarrow(record) }
        for ((source, message) in listOf("{}" to "Object violates minProperties.", "{\"value\":\"ok\",\"u\":null,\"v\":null}" to "Object violates maxProperties.")) {
            response = source
            directCountWireReject(message) { api.send(record) }
        }
        response = "{\"value\":\"ok\",\"unknown\":null}"
        directCountVerify(api.send(record) == record, "HTTP permissive caller delegate after count")
        response = "{\"value\":\"ok\"}"
        // Count-invalid values are valid constructors. Callback precedes codec
        // rejection, and the request never reaches the engine.
        var callback = false; val before = calls
        directCountWireReject("Object violates maxProperties.") { api.sendZero(ZeroDto("extra")) { callback = true } }
        directCountVerify(callback && calls == before, "codec count failure after callback without engine dispatch")
        var primitiveCallback = false
        try { api.sendChecked(CheckedDto("x")) { primitiveCallback = true }; error("Lost primitive constructor authority") }
        catch (cause: IllegalArgumentException) { directCountVerify(!primitiveCallback && calls == before, "existing primitive construction fails before callback/dispatch") }
        for (p in patches) { expectedBody = codec.encodeToString(PatchInputDto.serializer(), p); api.patchPatch(p) }
        val coldBefore = calls; val cold = api.watch()
        directCountVerify(calls == coldBefore, "SSE Flow creation is cold")
        val first = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
        directCountVerify(first.data == record && first.id == "transport-id", "SSE counts data only")
        directCountVerify(withTimeout(10_000) { cold.toList().size } == 1 && calls == coldBefore + 2, "fresh cold recollection")
        frameData = "null"
        directCountVerify((withTimeout(10_000) { api.watch().toList().single() } as WatchEventDto.Record).data == null, "nullable SSE data original null codec")
        holdOpen = true
        for ((source, message) in listOf("{}" to "Object violates minProperties.", "{\"value\":\"ok\",\"u\":null,\"v\":null}" to "Object violates maxProperties.")) {
            frameData = source
            directCountWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
            directCountVerify(channels.last().isClosedForRead, "count failure closes still-open fragmented input")
        }
        frameData = "{\"value\":\"ok\",\"unknown\":null}"
        try { withTimeout(10_000) { api.watch().toList() }; error("Lost separate strict SSE Json") }
        catch (cause: Throwable) { directCountVerify(generateSequence(cause) { it.cause }.any { it is SerializationException && it.message?.contains("unknown") == true }, "SSE delegate uses its own strict Json") }
        directCountVerify(channels.last().isClosedForRead, "delegate failure closes still-open input")
        holdOpen = false; frameData = "{\"value\":\"ok\"}"
        directCountVerify(withTimeout(10_000) { api.watch().toList().size } == 1, "SSE client reused after failure")
        expectedBody = codec.encodeToString(RecordDto.generatedSerializer(), record)
        directCountVerify(api.send(record) == record, "caller HTTP client remains reusable")
    } finally { http.close(); engine.close(); producers.cancel() }
}
