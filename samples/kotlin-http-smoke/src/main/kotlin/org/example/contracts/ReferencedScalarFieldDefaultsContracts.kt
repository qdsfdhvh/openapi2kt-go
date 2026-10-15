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
import kotlinx.coroutines.CancellationException
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
import kotlinx.serialization.serializer
import org.example.refscalars.ReferencedScalars
import org.example.refscalars.api.models.PatchInputDto
import org.example.refscalars.api.models.WatchEventDto
import org.example.refscalars.models.AliasDto
import org.example.refscalars.models.CountDto
import org.example.refscalars.models.HolderDto
import org.example.refscalars.models.Patch
import org.example.refscalars.models.RequiredNullableDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RefScalarEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun refScalarReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun refScalarBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> refScalarBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedScalarFieldDefaultsContracts(checked: Boolean) {
    val value = HolderDto(required = 2L, kotlin = "shadow")
    val alias: AliasDto = value.count
    val builtin: Long? = alias
    val canonical: CountDto = builtin
    check(canonical == 2L)
    check(value.label == "quote\"\$😀" && value.flag == false)
    check(value.integer == Int.MIN_VALUE && value.minimum == Long.MIN_VALUE)
    check(value.ratio == 1.0f && value.doubleValue == 1.25 && value.internal == 3L && value.nil == null)
    check(serializer<AliasDto>().descriptor.serialName == "kotlin.Long?")
    check(serializer<AliasDto>().descriptor.isNullable)
    check(HolderDto.serializer().descriptor.serialName == "org.example.refscalars.models.HolderDto")
    check(HolderDto::class.java.getDeclaredMethod("getRequired").returnType == java.lang.Long.TYPE)
    check(HolderDto::class.java.getDeclaredMethod("getCount").returnType == java.lang.Long::class.java)
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    // Independent transparent aliases/codecs retain their original unchecked contract.
    val unchecked: AliasDto = 99L
    check(Json.decodeFromString<AliasDto>("99") == unchecked)
    if (checked) {
        refScalarReject("maximum") { HolderDto(required = 12L) }
        refScalarReject("multipleOf") { HolderDto(required = 3L) }
        refScalarReject("maxLength") { HolderDto(required = 2L, label = "a".repeat(33)) }
        refScalarReject("minimum") { HolderDto(required = 2L, ratio = 0.0f) }
        refScalarReject("exclusiveMinimum") { HolderDto(required = 2L, doubleValue = 1.0) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("""{"required":2}""")
        check(missing.count == 2L && missing.label == value.label && missing.minimum == Long.MIN_VALUE)
        refScalarReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":2,"count":null,"label":null}""").let { it.count == null && it.label == null })
        check(codec.decodeFromString<HolderDto>("""{"required":2,"count":4,"label":"value","flag":true,"ratio":1.5}""").let { it.count == 4L && it.label == "value" && it.flag == true && it.ratio == 1.5f })
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("count") == defaults && encoded.containsKey("minimum") == defaults)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire) == requiredNull)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.count == Patch.Absent && absent.clear == Patch.Absent && absent.ordinary == 2L)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "count" || it == "clear" })
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"count":null,"clear":null}""")
        check(cleared.count == Patch.Null && cleared.clear == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["count"] === JsonNull)
        val update = PatchInputDto(required = 2L, count = Patch.Value(4L))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}") }.isSuccess == !nulls)
        refScalarReject { codec.decodeFromString<HolderDto>("""{"required":null}""") }
        refScalarReject { codec.decodeFromString<HolderDto>("""{"required":2,"flag":{}}""") }
        if (checked) {
            refScalarReject("maximum") { codec.decodeFromString<HolderDto>("""{"required":12}""") }
            refScalarReject("multipleOf") { codec.decodeFromString<HolderDto>("""{"required":3}""") }
            refScalarReject("maximum") { PatchInputDto(required = null, count = Patch.Value(12L)) }
        }
        refScalarWire(codec, missing, checked)
        println("Referenced scalar defaults Json defaults=$defaults nulls=$nulls checked=$checked ABI-constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun refScalarWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":2}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = RefScalarEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(refScalarBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(refScalarBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (holdOpen) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected referenced scalar path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ReferencedScalars(http, "http://example.test/", codec).defaults
        check(api.send(value).count == 2L)
        for (patch in listOf(PatchInputDto(required = 2L), PatchInputDto(required = null, count = Patch.Null), PatchInputDto(required = 2L, count = Patch.Value(4L)))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        if (checked) {
            val before = calls
            var callbacks = 0
            // The original DTO constructor is the nominal validation barrier.
            refScalarReject("maximum") { api.send(HolderDto(required = 12L)) { callbacks++ } }
            check(calls == before && callbacks == 0)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.count == 2L) }
        val invalid = mutableListOf<Pair<String, String?>>(
            """{"required":null}""" to null,
            """{"required":2,"flag":{}}""" to null,
            "{" to null)
        if (checked) invalid += listOf(
            """{"required":12}""" to "maximum",
            """{"required":3}""" to "multipleOf",
            """{"required":2,"ratio":0}""" to "minimum",
            """{"required":2,"doubleValue":1}""" to "exclusiveMinimum")
        for ((bad, rule) in invalid) {
            payload = bad
            failing = true
            refScalarReject(rule) { api.send(value) }
            refScalarReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":2}"""
            failing = false
            check(api.send(value).minimum == Long.MIN_VALUE)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.ratio == 1.0f)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
