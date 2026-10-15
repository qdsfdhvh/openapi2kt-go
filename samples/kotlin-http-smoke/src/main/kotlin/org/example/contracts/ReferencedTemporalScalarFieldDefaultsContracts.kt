@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, kotlin.time.ExperimentalTime::class)
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
import kotlin.time.Instant
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.datetime.LocalDate
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.serializer
import org.example.reftemporal.ReferencedTemporal
import org.example.reftemporal.api.models.PatchInputDto
import org.example.reftemporal.api.models.WatchEventDto
import org.example.reftemporal.models.AliasDto
import org.example.reftemporal.models.BridgeDto
import org.example.reftemporal.models.DayDto
import org.example.reftemporal.models.HolderDto
import org.example.reftemporal.models.Patch
import org.example.reftemporal.models.RequiredNullableDto
import org.example.reftemporal.models.StampDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RefTemporalEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun refTemporalReject(block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected temporal decode failure" }
}

private fun refTemporalBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> refTemporalBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedTemporalScalarFieldDefaultsContracts(checked: Boolean) {
    val day = LocalDate.parse("2024-02-29")
    val stamp = Instant.parse("2024-02-29T10:04:56.123456789Z")
    val midnight = Instant.parse("2024-02-29T00:00:00Z")
    val value = HolderDto(required = day, kotlin = "shadow", kotlinx = "shadow")
    val alias: AliasDto = value.stamp
    val builtin: Instant? = alias
    val canonical: StampDto = builtin
    val bridge: BridgeDto = checkNotNull(value.bridge)
    val dateAlias: DayDto = value.required
    check(canonical == stamp && bridge == midnight && dateAlias == day)
    check(value.day == day && value.internal == midnight && value.nil == null)
    check(serializer<AliasDto>().descriptor.isNullable && serializer<StampDto>().descriptor.isNullable)
    check(!AliasDto.serializer().descriptor.isNullable && !serializer<BridgeDto>().descriptor.isNullable)
    check(AliasDto.parse("2024-02-29T12:34:56.123456789+02:30") == stamp)
    check(HolderDto::class.java.getDeclaredMethod("getRequired").returnType == LocalDate::class.java)
    check(HolderDto::class.java.getDeclaredMethod("getStamp").returnType == Instant::class.java)
    check(HolderDto.serializer().descriptor.serialName == "org.example.reftemporal.models.HolderDto")
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("""{"required":"2024-02-29"}""")
        check(missing.day == day && missing.stamp == stamp && missing.internal == midnight && missing.bridge == midnight)
        refTemporalReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":"2024-02-29","day":null,"stamp":null}""").let { it.day == null && it.stamp == null })
        val replacement = codec.decodeFromString<HolderDto>("""{"required":"2024-02-29","day":"2024-03-01","stamp":"2024-03-01T00:00:00Z"}""")
        check(replacement.day == LocalDate.parse("2024-03-01") && replacement.stamp == Instant.parse("2024-03-01T00:00:00Z"))
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("stamp") == defaults && encoded.containsKey("day") == defaults)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire) == requiredNull)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.stamp == Patch.Absent && absent.clear == Patch.Absent && absent.ordinary == midnight)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "stamp" || it == "clear" })
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"stamp":null,"clear":null}""")
        check(cleared.stamp == Patch.Null && cleared.clear == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["stamp"] === JsonNull)
        val update = PatchInputDto(required = midnight, stamp = Patch.Value(stamp))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}") }.isSuccess == !nulls)
        for (bad in listOf("""{"required":null}""", """{"required":"2023-02-29"}""", """{"required":"2024-02-29","stamp":{}}""", """{"required":"2024-02-29","stamp":"invalid"}""")) {
            refTemporalReject { codec.decodeFromString<HolderDto>(bad) }
        }
        refTemporalWire(codec, missing, midnight, stamp)
        println("Referenced temporal defaults Json defaults=$defaults nulls=$nulls checked=$checked ABI-constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun refTemporalWire(codec: Json, value: HolderDto, midnight: Instant, stamp: Instant) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":"2024-02-29"}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = RefTemporalEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(refTemporalBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(request.method.value == "PATCH")
                check(codec.parseToJsonElement(refTemporalBody(request.body)) == expectedPatch)
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
            else -> error("Unexpected referenced temporal path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ReferencedTemporal(http, "http://example.test/", codec).defaults
        check(api.send(value).stamp == stamp)
        for (patch in listOf(PatchInputDto(required = midnight), PatchInputDto(required = null, stamp = Patch.Null), PatchInputDto(required = midnight, stamp = Patch.Value(stamp)))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.stamp == stamp) }
        for (bad in listOf("""{"required":null}""", """{"required":"2023-02-29"}""", """{"required":"2024-02-29","stamp":{}}""", """{"required":"2024-02-29","stamp":"invalid"}""", "{")) {
            payload = bad
            failing = true
            refTemporalReject { api.send(value) }
            refTemporalReject { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":"2024-02-29"}"""
            failing = false
            check(api.send(value).bridge == midnight)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.stamp == stamp)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
