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
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.collectiondtos.ApiClient
import org.example.collectiondtos.api.models.SubmitInputDto
import org.example.collectiondtos.api.models.WatchEventDto
import org.example.collectiondtos.models.ConsumerDto
import org.example.collectiondtos.models.NestedOwnerKeyDto
import org.example.collectiondtos.models.Patch
import org.example.collectiondtos.models.RootMapDtoValue
import org.example.collectiondtos.models.RootRowsDto
import org.example.collectiondtos.models.RootRowsDtoInner

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class CollectionDtoStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun collectionDtoBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> collectionDtoBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected collection DTO body $content")
}

private suspend fun collectionDtoReject(block: suspend () -> Unit) {
    val error = runCatching { block() }.exceptionOrNull()
    check(error != null && generateSequence(error) { it.cause }.any {
        it is kotlinx.serialization.SerializationException || it is IllegalArgumentException
    }) { "Expected DTO/codec failure, not a stalled stream or engine assertion: $error" }
}

suspend fun collectionOwnedDTOContracts(checked: Boolean = false) {
    val child = NestedOwnerKeyDto.BatchesInnerValueInner.Child("nested")
    val mixed = NestedOwnerKeyDto.BatchesInnerValueInner("mixed", child)
    val row = NestedOwnerKeyDto.RowsInner("row")
    val root = RootRowsDtoInner("valid", listOf(RootRowsDtoInner("leaf")))
    val mapping = RootMapDtoValue("mapping")
    val entry = NestedOwnerKeyDto.EntriesValue("entry")
    val value = ConsumerDto(row = row, root = root, narrowed = mapping,
        mixed = mixed, child = child, rows = listOf(root), mapping = mapOf("k" to row))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<ConsumerDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<RootRowsDtoInner>(codec.encodeToString(root)) == root)
        check(codec.decodeFromString<RootRowsDto>(codec.encodeToString<RootRowsDto>(listOf(root))) == listOf(root))
        check(codec.decodeFromString<RootRowsDto>("null") == null)
        check(codec.decodeFromString(RootMapDtoValue.serializer().nullable, "null") == null)
        check(RootRowsDtoInner.serializer().descriptor.serialName.endsWith("RootRowsDtoInner"))
        collectionDtoReject { codec.decodeFromString<RootRowsDtoInner>("null") }
        collectionDtoReject { codec.decodeFromString<ConsumerDto>("{\"row\":null}") }
        if (checked) {
            collectionDtoReject { RootRowsDtoInner("") }
            collectionDtoReject { RootMapDtoValue("") }
            collectionDtoReject { NestedOwnerKeyDto.RowsInner("") }
            collectionDtoReject { codec.decodeFromString<RootRowsDtoInner>("{\"id\":\"\"}") }
        }
        var mode = 0
        var calls = 0
        var patchWire = ""
        val wire = codec.encodeToString(value)
        val rootWire = codec.encodeToString(root)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = CollectionDtoStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(collectionDtoBody(request.body) == wire)
                        respond(if (mode == 1) "{}" else wire, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/direct" -> {
                        check(collectionDtoBody(request.body) == rootWire)
                        respond(if (mode == 1) "{}" else rootWire, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/submit" -> {
                        check(collectionDtoBody(request.body) == patchWire)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val payload = if (current == 0) rootWire else if (current == 2) "{\"id\":\"\"}" else "{}"
                                val frame = "event: row\r\nid: collection-id\r\ndata: $payload\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected collection DTO route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).default
            check(api.echo(value) == value)
            check(api.direct(root) == root)
            mode = 1
            collectionDtoReject { api.echo(value) }
            collectionDtoReject { api.direct(root) }
            mode = 0
            for (state in listOf<Patch<NestedOwnerKeyDto.EntriesValue>>(Patch.Absent, Patch.Null, Patch.Value(entry))) {
                val input = SubmitInputDto(optional = state, requiredNull = null, requiredFalse = mapping)
                val encoded = codec.encodeToJsonElement(SubmitInputDto.serializer(), input) as JsonObject
                check(when (state) {
                    Patch.Absent -> "optional" !in encoded
                    Patch.Null -> encoded["optional"] == JsonNull
                    is Patch.Value -> encoded["optional"] == codec.encodeToJsonElement(NestedOwnerKeyDto.EntriesValue.serializer(), entry)
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
                val event = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Row }
                check(event.data == root && event.id == "collection-id")
            }
            check(calls == before + 2)
            for (bad in if (checked) listOf(1, 2) else listOf(1)) {
                mode = bad
                collectionDtoReject { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Collection DTO failure left stream open" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 1)
            check(api.echo(value) == value)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Collection-owned DTO contracts PASS: defaults=$defaults/nulls=$nulls checked=$checked; canonical peers/mixed-owner/recursion/serde/Patch/HTTP/live SSE close-reuse-cold")
    }
}
