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
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.builtins.nullable
import org.example.internaldtos.ApiClient
import org.example.internaldtos.api.models.SubmitInputDto
import org.example.internaldtos.api.models.WatchEventDto
import org.example.internaldtos.models.ConsumerDto
import org.example.internaldtos.models.Patch
import org.example.internaldtos.models.TreeKeyDto

// A real Ktor parser over fragmented channels, not a replacement SSE decoder.
@OptIn(io.ktor.utils.io.InternalAPI::class)
private class InternalDtoStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun internalDtoBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> internalDtoBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected internal DTO body $content")
}

private suspend fun internalDtoReject(block: suspend () -> Unit) {
    val error = runCatching { block() }.exceptionOrNull()
    check(error != null && generateSequence(error) { it.cause }.any {
        it is kotlinx.serialization.SerializationException || it is IllegalArgumentException
    }) { "Expected a DTO/codec failure, not a stalled stream or engine assertion: $error" }
}

suspend fun schemaInternalDTOContracts(checked: Boolean = false) {
    val child = TreeKeyDto.Node.Child("nested")
    val node = TreeKeyDto.Node("valid", peer = TreeKeyDto.Maybe("peer"), child = child)
    val peer = TreeKeyDto.Maybe("label", back = node)
    val value = ConsumerDto(node = node, narrowed = peer, nodes = listOf(node), mapping = mapOf("k" to node))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<TreeKeyDto.Node>(codec.encodeToString(node)) == node)
        check(codec.decodeFromString<ConsumerDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString(TreeKeyDto.Maybe.serializer().nullable, "null") == null)
        check(TreeKeyDto.Node.serializer().descriptor.serialName.endsWith("TreeKeyDto.Node"))
        internalDtoReject { codec.decodeFromString<ConsumerDto>("{\"node\":null}") }
        if (checked) {
            internalDtoReject { TreeKeyDto.Node("") }
            internalDtoReject { codec.decodeFromString<TreeKeyDto.Node>("{\"id\":\"\"}") }
        }
        var mode = 0
        var calls = 0
        var patchWire = ""
        val wire = codec.encodeToString(value)
        val nodeWire = codec.encodeToString(node)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = InternalDtoStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(internalDtoBody(request.body) == wire)
                        respond(if (mode == 1) "{}" else wire, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/direct" -> {
                        check(internalDtoBody(request.body) == nodeWire)
                        respond(if (mode == 1) "{}" else nodeWire, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/submit" -> {
                        check(internalDtoBody(request.body) == patchWire)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val payload = if (current == 0) nodeWire else if (current == 2) "{\"id\":\"\"}" else "{}"
                                val frame = "event: node\r\nid: internal-id\r\ndata: $payload\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected internal DTO route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).default
            check(api.echo(value) == value)
            check(api.direct(node) == node)
            mode = 1
            internalDtoReject { api.echo(value) }
            internalDtoReject { api.direct(node) }
            mode = 0
            for (state in listOf<Patch<TreeKeyDto.Maybe>>(Patch.Absent, Patch.Null, Patch.Value(peer))) {
                val input = SubmitInputDto(optional = state, requiredNull = null, requiredFalse = peer)
                val encoded = codec.encodeToJsonElement(SubmitInputDto.serializer(), input) as JsonObject
                check(when (state) {
                    Patch.Absent -> "optional" !in encoded
                    Patch.Null -> encoded["optional"] == JsonNull
                    is Patch.Value -> encoded["optional"] == codec.encodeToJsonElement(TreeKeyDto.Maybe.serializer(), peer)
                })
                check(encoded.containsKey("requiredNull") == nulls)
                check(codec.decodeFromJsonElement(SubmitInputDto.serializer(), encoded) == input)
                patchWire = codec.encodeToString(input)
                api.submit(input)
            }
            val before = calls
            val cold = api.watch()
            check(calls == before)
            repeat(2) {
                val event = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Node }
                check(event.data == node && event.id == "internal-id")
            }
            check(calls == before + 2)
            for (bad in if (checked) listOf(1, 2) else listOf(1)) {
                mode = bad
                internalDtoReject { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Canonical DTO failure left stream open" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 1)
            check(api.echo(value) == value)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Internal DTO contracts PASS: defaults=$defaults/nulls=$nulls checked=$checked; canonical recursion/serde/Patch/HTTP/live SSE close-reuse-cold")
    }
}
