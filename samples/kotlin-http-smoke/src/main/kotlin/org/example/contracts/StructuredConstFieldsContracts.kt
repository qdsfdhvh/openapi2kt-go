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
import org.example.structuredconst.StructuredConsts
import org.example.structuredconst.api.models.PatchInputDto
import org.example.structuredconst.api.models.WatchEventDto
import org.example.structuredconst.models.ContainersDto
import org.example.structuredconst.models.HolderDto
import org.example.structuredconst.models.MergedDto
import org.example.structuredconst.models.NestedDto
import org.example.structuredconst.models.OnlyDto
import org.example.structuredconst.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StructuredConstsEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun structuredConstReject(block: () -> Unit) {
    try { block(); error("Expected structured const kind rejection") }
    catch (_: SerializationException) { }
}
private suspend fun structuredConstWireReject(block: suspend () -> Unit) {
    try { block(); error("Expected structured const wire rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong wire failure: $cause" }
    }
}
private fun structuredConstBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> structuredConstBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected structured const body $body")
}

suspend fun structuredConstContracts() {
    val holder = HolderDto(label = "ready", kotlinx = "shadow", kotlin = "shadow")
    check(holder.payload["\$ref"] == JsonPrimitive("literal data"))
    check(holder.payload["type"] == JsonPrimitive("data"))
    check((holder.payload["values"] as JsonArray).first() === JsonNull)
    check(holder.payload["quoted"] == JsonPrimitive("quote \" and \$value"))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val omitted = codec.decodeFromString<HolderDto>("{}")
        check(omitted.payload == holder.payload && omitted.rows == holder.rows)
        val wire = codec.encodeToJsonElement(HolderDto.serializer(), holder) as JsonObject
        check(wire["payload"] == holder.payload && wire["rows"] == holder.rows)
        val decoded = codec.decodeFromJsonElement(HolderDto.serializer(), wire)
        check(decoded.payload == holder.payload && decoded.rows == holder.rows && decoded.label == "ready")
        for (wrong in listOf("null", "false", "0", "\"object\"", "[]")) {
            structuredConstReject { codec.decodeFromString<HolderDto>("{\"payload\":$wrong}") }
        }
        for (wrong in listOf("null", "true", "1", "\"array\"", "{}")) {
            structuredConstReject { codec.decodeFromString<HolderDto>("{\"rows\":$wrong}") }
        }
        // Existing body-const serde preserves a supplied value of the same kind.
        val alternative = codec.decodeFromString<HolderDto>("{\"payload\":{\"other\":null},\"rows\":[2,3]}")
        check(alternative.payload == codec.parseToJsonElement("{\"other\":null}"))
        check(alternative.rows == codec.parseToJsonElement("[2,3]"))
        val changed = codec.encodeToJsonElement(HolderDto.serializer(), alternative) as JsonObject
        check(changed["payload"] == alternative.payload && changed["rows"] == alternative.rows)
        val only = codec.decodeFromString<OnlyDto>("{}")
        val onlyWire = codec.encodeToJsonElement(OnlyDto.serializer(), only) as JsonObject
        check(onlyWire["payload"] == codec.parseToJsonElement("{\"nested\":[1,null]}"))
        check(onlyWire["rows"] == codec.parseToJsonElement("[true,\"value\"]"))
        val nested = NestedDto(child = NestedDto.Child(kotlinx = true))
        check(codec.decodeFromString<NestedDto>(codec.encodeToString(nested)).child?.rows == codec.parseToJsonElement("[1,2]"))
        val containers = ContainersDto(holder = holder, rows = listOf(holder), map = mapOf("one" to holder))
        val back = codec.decodeFromString<ContainersDto>(codec.encodeToString(containers))
        check(back.holder?.payload == holder.payload && back.rows?.single()?.rows == holder.rows && back.map?.get("one")?.payload == holder.payload)
        check(MergedDto().payload == codec.parseToJsonElement("{\"source\":\"first\"}"))
        val patches = listOf(PatchInputDto(), PatchInputDto(Patch.Null), PatchInputDto(Patch.Value("change")))
        for (patch in patches) {
            val patchWire = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            check(patchWire["payload"] == codec.parseToJsonElement("{\"patch\":true}"))
            check(patchWire["rows"] == codec.parseToJsonElement("[null]"))
            check(when (patch.label) { Patch.Absent -> !patchWire.containsKey("label"); Patch.Null -> patchWire["label"] === JsonNull; is Patch.Value -> patchWire["label"] == JsonPrimitive("change") })
            check(codec.decodeFromJsonElement(PatchInputDto.serializer(), patchWire).label == patch.label)
        }
        structuredConstWire(codec, holder, patches)
        println("StructuredConstFields Json defaults=$defaults nulls=$nulls typed-body-default-alternative-kind-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun structuredConstWire(codec: Json, holder: HolderDto, patches: List<PatchInputDto>) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var calls = 0
    var patchWire = ""
    val engine = StructuredConstsEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/holder" -> {
                    check(structuredConstBody(request.body) == codec.encodeToString(holder))
                    respond(if (bad) "{\"payload\":[]}" else codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(structuredConstBody(request.body) == patchWire)
                    respond("", HttpStatusCode.NoContent)
                }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val invalid = bad
                    producers.launch {
                        try {
                            val payload = if (invalid) "{\"rows\":{}}" else codec.encodeToString(holder)
                            val frame = "event: holder\r\nid: one\r\ndata: $payload\r\n\r\n"
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
        val api = StructuredConsts(http, "http://example.test/", codec).structuredConst
        check(api.send(holder).payload == holder.payload)
        for (patch in patches) {
            patchWire = codec.encodeToString(patch)
            api.patchPatch(patch)
        }
        val before = calls
        val flow = api.watch()
        check(calls == before)
        repeat(2) {
            val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Holder
            check(event.data.payload == holder.payload && event.data.rows == holder.rows && event.id == "one")
        }
        bad = true
        structuredConstWireReject { api.send(holder) }
        val beforeBad = channels.size
        structuredConstWireReject { withTimeout(10000) { api.watch().toList() } }
        check(channels.size == beforeBad + 1 && channels.last().isClosedForRead)
        bad = false
        check(api.send(holder).rows == holder.rows)
    } finally {
        http.close()
        producers.cancel()
        channels.forEach { it.close() }
    }
}
