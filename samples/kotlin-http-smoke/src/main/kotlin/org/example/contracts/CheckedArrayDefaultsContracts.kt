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
import kotlinx.datetime.LocalDate
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.checkedarrays.CheckedArrays
import org.example.checkedarrays.api.models.PatchInputDto
import org.example.checkedarrays.api.models.WatchEventDto
import org.example.checkedarrays.models.HolderDto
import org.example.checkedarrays.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class CheckedArraysEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun checkedArraysReject(rule: String, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected $rule failure" }
    check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong failure for $rule: $failure" }
}

private fun checkedArraysBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> checkedArraysBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected body $body")
}

suspend fun checkedArrayDefaultsContracts() {
    val value = HolderDto(required = listOf(2), kotlin = "shadow", kotlinx = "shadow")
    check(value.values == listOf(2, null) && value.names == listOf("ready") && value.floats == listOf(1f))
    check(value.dates == listOf(LocalDate.parse("2024-02-29")))
    checkedArraysReject("minItems") { HolderDto(required = listOf(2), dates = emptyList()) }
    checkedArraysReject("maxItems") { HolderDto(required = listOf(2), dates = List(3) { LocalDate.parse("2024-02-29") }) }
    checkedArraysReject("minItems") { HolderDto(required = emptyList()) }
    checkedArraysReject("uniqueItems") { HolderDto(required = listOf(2), values = listOf(2, 2)) }
    checkedArraysReject("multipleOf") { HolderDto(required = listOf(2), values = listOf(3)) }
    checkedArraysReject("minimum") { HolderDto(required = listOf(2), floats = listOf(0f)) }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("{\"required\":[2]}")
        check(missing.values == value.values && missing.names == value.names && missing.dates == value.dates)
        check(codec.decodeFromString<HolderDto>("{\"required\":[2],\"values\":null}").values == null)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("values") == defaults)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded).values == value.values)
        for ((json, rule) in listOf(
            "{\"required\":[]}" to "minItems",
            """{"required":[2],"dates":[]}""" to "minItems",
            "{\"required\":[2],\"values\":[2,2]}" to "uniqueItems",
            "{\"required\":[2],\"values\":[2,4,6,8]}" to "maxItems",
            "{\"required\":[2],\"names\":[\"BAD\"]}" to "pattern",
            "{\"required\":[2],\"values\":[-2]}" to "minimum"
        )) checkedArraysReject(rule) { codec.decodeFromString<HolderDto>(json) }
        val absent = codec.decodeFromString<PatchInputDto>("{}")
        check(absent.values == Patch.Absent && absent.dates == Patch.Absent)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).isEmpty())
        val nullPatch = codec.decodeFromString<PatchInputDto>("{\"values\":null}")
        check(nullPatch.values == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), nullPatch) as JsonObject)["values"] === JsonNull)
        checkedArraysReject("minItems") { PatchInputDto(values = Patch.Value(emptyList())) }
        checkedArraysReject("minItems") { PatchInputDto(dates = Patch.Value(emptyList())) }
        val temporalNull = PatchInputDto(dates = Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), temporalNull) as JsonObject)["dates"] === JsonNull)
        checkedArraysReject("minimum") { codec.decodeFromString<PatchInputDto>("{\"values\":[-1]}") }
        checkedArrayDefaultsWire(codec, value)
        println("CheckedArrayDefaults Json defaults=$defaults nulls=$nulls constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun checkedArrayDefaultsWire(codec: Json, value: HolderDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var temporalBad = false
    var calls = 0
    val engine = CheckedArraysEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        val payload = if (temporalBad) """{"required":[2],"dates":[]}""" else if (bad) "{\"required\":[2],\"values\":[2,2]}" else "{\"required\":[2]}"
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(checkedArraysBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(checkedArraysBody(request.body) == "{}")
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val invalid = bad || temporalBad
                producers.launch { try {
                    val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (invalid) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected checked array request")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = CheckedArrays(http, "http://example.test/", codec).defaults
        check(api.send(value).values == value.values)
        api.patchPatch(PatchInputDto())
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) {
            val data = (withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data
            check(data.values == value.values && data.dates == value.dates)
        }
        bad = true
        checkedArraysReject("uniqueItems") { api.send(value) }
        checkedArraysReject("uniqueItems") { withTimeout(10000) { api.watch().toList() } }
        check(channels.last().isClosedForRead)
        bad = false
        check(api.send(value).names == value.names)
        temporalBad = true
        checkedArraysReject("minItems") { api.send(value) }
        checkedArraysReject("minItems") { withTimeout(10000) { api.watch().toList() } }
        check(channels.last().isClosedForRead)
        temporalBad = false
        check(api.send(value).dates == value.dates)
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
