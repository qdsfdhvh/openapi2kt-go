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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.referencedbounds.ReferencedBounds
import org.example.referencedbounds.api.models.PatchInputDto
import org.example.referencedbounds.api.models.WatchEventDto
import org.example.referencedbounds.models.Patch
import org.example.referencedbounds.models.RecordDto
import org.example.referencedbounds.models.ZNullableDto
import org.example.referencedbounds.models.ZNumbersDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedBoundsEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun referencedBoundsReject(keyword: String, block: () -> Unit) {
    val error = runCatching(block).exceptionOrNull()
    check(error != null && generateSequence(error) { it.cause }.any {
        it is IllegalArgumentException && it.message?.startsWith("Field value") == true && it.message?.contains(keyword) == true
    }) { "Missing retained source check $keyword: $error" }
}
private suspend fun referencedBoundsWireReject(keyword: String, block: suspend () -> Unit) {
    var failure: Throwable? = null
    try { block() } catch (error: Throwable) { failure = error }
    check(failure != null && generateSequence(failure) { it.cause }.any {
        it is IllegalArgumentException && it.message?.startsWith("Field value") == true && it.message?.contains(keyword) == true
    }) { "Missing retained wire check $keyword: $failure" }
}
private fun referencedBoundsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedBoundsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected referenced bounds body")
}

suspend fun referencedAllOfBoundsContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val record = RecordDto("ab")
        check(codec.encodeToString(RecordDto.serializer(), record) == "{\"value\":\"ab\"}")
        check(codec.decodeFromString<RecordDto>("{\"value\":\"ab\"}") == record)
        for ((value, keyword) in listOf("a" to "minLength", "abcd" to "maxLength", "12" to "pattern")) {
            referencedBoundsReject(keyword) { RecordDto(value) }
            referencedBoundsReject(keyword) { codec.decodeFromString<RecordDto>("{\"value\":\"$value\"}") }
            referencedBoundsReject(keyword) { codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement("{\"value\":\"$value\"}")) }
        }
        check(codec.decodeFromString<ZNullableDto>(codec.encodeToString(ZNullableDto.serializer(), ZNullableDto(null))) == ZNullableDto(null))
        check(codec.decodeFromString<ZNullableDto>("{\"value\":\"ab\"}") == ZNullableDto("ab"))
        referencedBoundsReject("minLength") { ZNullableDto("a") }
        referencedBoundsReject("maxLength") { codec.decodeFromString<ZNullableDto>("{\"value\":\"abcd\"}") }
        val payloads = listOf(PatchInputDto.Payload(requiredNull = null), PatchInputDto.Payload(Patch.Null, null), PatchInputDto.Payload(Patch.Value("ab"), "abc"))
        for (payload in payloads) {
            val element = codec.encodeToJsonElement(PatchInputDto.Payload.serializer(), payload) as JsonObject
            check(element.containsKey("requiredNull") == (nulls || payload.requiredNull != null))
            when (payload.value) {
                Patch.Absent -> check(!element.containsKey("value"))
                Patch.Null -> check(element["value"] == JsonNull)
                is Patch.Value -> check(element["value"] == JsonPrimitive("ab"))
            }
            check(codec.decodeFromJsonElement(PatchInputDto.Payload.serializer(), element) == payload)
        }
        val descriptor = PatchInputDto.Payload.serializer().descriptor
        check(descriptor.isElementOptional(0) && !descriptor.isElementOptional(1) && descriptor.getElementDescriptor(1).isNullable)
        referencedBoundsReject("minLength") { PatchInputDto.Payload(Patch.Value("a"), null) }
        referencedBoundsReject("maxLength") { PatchInputDto.Payload(requiredNull = "abcd") }
        referencedBoundsReject("minLength") { codec.decodeFromString<PatchInputDto.Payload>("{\"requiredNull\":null,\"value\":\"a\"}") }
        for (long in listOf(Long.MIN_VALUE, Long.MAX_VALUE)) {
            val numbers = ZNumbersDto("shadow", long, 2, 0.5, 0.4f, 0.4)
            check(codec.decodeFromString<ZNumbersDto>(codec.encodeToString(ZNumbersDto.serializer(), numbers)) == numbers)
            for ((from, to, keyword) in listOf(Triple("\"steps\":2", "\"steps\":0", "exclusiveMinimum"), Triple("\"steps\":2", "\"steps\":8", "exclusiveMaximum"), Triple("\"steps\":2", "\"steps\":3", "multipleOf"), Triple("\"floatTie\":0.4", "\"floatTie\":0.3", "exclusiveMinimum"), Triple("\"doubleTie\":0.4", "\"doubleTie\":0.3", "exclusiveMinimum"), Triple("\"fraction\":0.5", "\"fraction\":0.3", "multipleOf"))) {
                referencedBoundsReject(keyword) { codec.decodeFromString<ZNumbersDto>(codec.encodeToString(ZNumbersDto.serializer(), numbers).replace(from, to)) }
            }
        }
        referencedBoundsReject("finite") { ZNumbersDto("shadow", 0, 2, Double.NaN, 0.4f, 0.4) }
        referencedBoundsReject("exclusiveMinimum") { ZNumbersDto("shadow", 0, 2, 0.5, 0.3f, 0.4) }
        referencedBoundsReject("multipleOf") { ZNumbersDto("shadow", 0, 2, 0.5, 0.4f, 0.4, 0.3) }
        referencedBoundsHttp(codec, record, payloads.map { PatchInputDto(it) })
        println("ReferencedAllOfBounds Json defaults=$defaults nulls=$nulls original-source-conjunction-null-Patch-numeric-HTTP-liveSSE passed")
    }
}

private suspend fun referencedBoundsHttp(codec: Json, record: RecordDto, patches: List<PatchInputDto>) {
    var calls = 0
    var badResponse = false
    var badSSE = false
    var expectedPatch = ""
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val engine = ReferencedBoundsEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/records" -> {
                    check(referencedBoundsBody(request.body) == codec.encodeToString(RecordDto.serializer(), record))
                    respond(if (badResponse) "{\"value\":\"a\"}" else "{\"value\":\"ab\"}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> { check(referencedBoundsBody(request.body) == expectedPatch); respond("", HttpStatusCode.NoContent) }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val bad = badSSE
                    producers.launch {
                        try {
                            val frame = "event: record\r\nid: referenced-id\r\ndata: {\"value\":\"${if (bad) "a" else "ab"}\"}\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (bad) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected referenced bounds request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ReferencedBounds(http, "http://example.test/", codec).bounds
        check(api.send(record) == record)
        for (patch in patches) { expectedPatch = codec.encodeToString(PatchInputDto.serializer(), patch); api.patchPatch(patch) }
        val beforeBad = calls
        var callback = false
        referencedBoundsWireReject("minLength") { api.send(RecordDto("a")) { callback = true } }
        check(!callback && calls == beforeBad)
        badResponse = true
        referencedBoundsWireReject("minLength") { api.send(record) }
        badResponse = false
        val before = calls
        val cold = api.watch()
        check(calls == before)
        val first = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
        check(first.data == record && first.id == "referenced-id")
        check(withTimeout(10_000) { cold.toList().size } == 1 && calls == before + 2)
        badSSE = true
        referencedBoundsWireReject("minLength") { withTimeout(10_000) { api.watch().toList() } }
        check(channels.last().isClosedForRead)
        badSSE = false
        check(withTimeout(10_000) { api.watch().toList().size } == 1)
        check(api.send(record) == record)
    } finally { http.close(); engine.close(); producers.cancel() }
}
