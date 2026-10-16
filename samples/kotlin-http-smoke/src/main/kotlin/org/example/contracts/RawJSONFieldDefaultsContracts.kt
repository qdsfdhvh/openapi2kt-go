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
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import org.example.rawjson.RawJson
import org.example.rawjson.api.models.SendInlineInputDto
import org.example.rawjson.api.models.WatchHoldersEventDto
import org.example.rawjson.models.HolderDto
import org.example.rawjson.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RawJSONFieldDefaultsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun rawJSONFieldDefaultsReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
}

private fun rawJSONFieldDefaultsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> rawJSONFieldDefaultsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected raw JSON request body $body")
}

private fun rawJSONFieldDefaultsFields(value: HolderDto): Map<String, JsonElement?> = linkedMapOf(
    "open" to value.open, "free" to value.free, "empty" to value.empty,
    "nullableOpen" to value.nullableOpen, "nullableFree" to value.nullableFree,
)

suspend fun rawJSONFieldDefaultsContracts(checked: Boolean) {
    val expected = mapOf(
        "open" to Json.parseToJsonElement("""{"a":1,"b":[true,null]}"""),
        "free" to Json.parseToJsonElement("[1,2,3]"),
        "empty" to Json.parseToJsonElement("""{"x":"y"}"""),
        "nullableOpen" to null,
        "nullableFree" to null,
    )
    val value = HolderDto()
    check(rawJSONFieldDefaultsFields(value) == expected)
    check(value.open is JsonObject)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(rawJSONFieldDefaultsFields(missing) == expected)
        val overridden = codec.decodeFromString<HolderDto>("""{"open":{"z":1},"free":true,"empty":"s"}""")
        check(overridden.open == Json.parseToJsonElement("""{"z":1}""") && overridden.free == Json.parseToJsonElement("true") && overridden.empty == Json.parseToJsonElement("\"s\""))
        check(codec.decodeFromString<HolderDto>("""{"open":null}""").open == null)
        check(rawJSONFieldDefaultsFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), missing))) == expected)
        rawJSONFieldDefaultsReject { codec.decodeFromString<HolderDto>("""{"open":[]}""") }
        val inline = codec.decodeFromString<SendInlineInputDto>("{}")
        check(inline.u == Json.parseToJsonElement("\"x\""))
        check(inline.open == Patch.Absent)
        check("open" !in (codec.encodeToJsonElement(SendInlineInputDto.serializer(), inline) as JsonObject))
        check(codec.decodeFromString<SendInlineInputDto>("""{"open":null}""").open == Patch.Null)
        val replacement = Json.parseToJsonElement("""{"key":1}""") as JsonObject
        check(codec.decodeFromString<SendInlineInputDto>("""{"open":{"key":1}}""").open == Patch.Value(replacement))
        rawJSONFieldDefaultsWire(codec, value, checked)
    }
}

private suspend fun rawJSONFieldDefaultsWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = codec.encodeToString(HolderDto.serializer(), value)
    var calls = 0
    var callbacks = 0
    val engine = RawJSONFieldDefaultsEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(rawJSONFieldDefaultsBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/inline" -> {
                check(codec.parseToJsonElement(rawJSONFieldDefaultsBody(request.body)) == codec.encodeToJsonElement(SendInlineInputDto.serializer(), SendInlineInputDto()))
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                val written = CompletableDeferred<Unit>()
                channels += channel
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    written.complete(Unit)
                } catch (cause: Throwable) { written.completeExceptionally(cause); throw cause } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected raw JSON path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = RawJson(http, "http://example.test/", codec).defaults
        check(rawJSONFieldDefaultsFields(api.sendHolder(value) { callbacks++ }) == rawJSONFieldDefaultsFields(value))
        api.sendInline(SendInlineInputDto()) { callbacks++ }
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watchHolders { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        val event = (withTimeout(10000) { stream.toList() }.single() as WatchHoldersEventDto.Holder).data
        check(rawJSONFieldDefaultsFields(event) == rawJSONFieldDefaultsFields(value))
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
    println("Raw JSON field defaults checked=$checked constructor-serde-HTTP-liveSSE passed")
}
