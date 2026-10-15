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
import kotlinx.serialization.SerializationException
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.collectionchains.CollectionChainApiClient
import org.example.collectionchains.api.models.SubmitInputDto
import org.example.collectionchains.api.models.WatchEventDto
import org.example.collectionchains.models.Patch
import org.example.collectionchains.models.RecordDto

private fun collectionChainReject(block: () -> Unit) {
    try { block(); error("Missing collection-chain check") }
    catch (cause: IllegalArgumentException) { check(cause.message?.startsWith("Field value") == true) }
}

private suspend fun collectionChainWireReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing collection-chain wire failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message) == true
        }) { "Wrong collection-chain failure: $cause" }
    }
}

private fun collectionChainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> collectionChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected collection-chain body $body")
}

suspend fun schemaInternalCollectionChainContracts() {
    val values = listOf(null, "ok")
    val entries = mapOf("k" to values)
    val mixed = listOf(null, entries)
    val value = RecordDto(values, entries, inherited = values, mixed = mixed)
    collectionChainReject { RecordDto(emptyList(), entries) }
    collectionChainReject { value.copy(values = listOf("ok", "ok")) }
    collectionChainReject { value.copy(inherited = listOf("a")) }
    collectionChainReject { value.copy(entries = mapOf("k" to emptyList())) }
    collectionChainReject { value.copy(mixed = listOf(emptyMap())) }
    collectionChainReject { value.copy(mixed = listOf(mapOf("k" to listOf("a")))) }
    collectionChainReject { SubmitInputDto(requiredNull = null, requiredFalse = values, inherited = Patch.Value(listOf("a"))) }
    val descriptor = SubmitInputDto.serializer().descriptor
    check(descriptor.getElementDescriptor(2).isNullable && !descriptor.isElementOptional(2))
    check(!descriptor.getElementDescriptor(3).isNullable && !descriptor.isElementOptional(3))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val sseCodec = Json(codec) { ignoreUnknownKeys = false }
        val text = codec.encodeToString(RecordDto.serializer(), value)
        check(codec.decodeFromString(RecordDto.serializer(), text) == value)
        check(codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(text)) == value)
        collectionChainReject { codec.decodeFromString(RecordDto.serializer(), text.replace("\"ok\"", "\"a\"")) }
        collectionChainReject { codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(text.replace("[null,\"ok\"]", "[null,null]"))) }
        val wire = codec.encodeToString(values)
        val mixedWire = codec.encodeToString(mixed)
        var mode = 0
        var calls = 0
        var expectedPatch = ""
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/values" -> {
                        check(collectionChainBody(request.body) == wire)
                        respond(if (mode == 1) "[\"a\"]" else wire,
                            HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(collectionChainBody(request.body) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val payload = when (current) {
                                    1 -> "[{\"k\":[\"a\"]}]"
                                    2 -> "[{\"k\":[null,null]}]"
                                    3 -> "[{}]"
                                    4 -> "[]"
                                    else -> mixedWire
                                }
                                val frame = "event: mixed\r\nid: collection-chain-id\r\ndata: $payload\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected collection-chain dispatch")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = CollectionChainApiClient(http, "http://example.test/", sseCodec).default
            val before = calls
            var callback = false
            collectionChainWireReject("shorter than minLength") { api.echo(listOf("a")) { callback = true } }
            collectionChainWireReject("duplicate entries for uniqueItems") { api.echo(listOf(null, null)) }
            collectionChainWireReject("fewer entries than minItems") { api.echo(emptyList()) }
            check(!callback && calls == before)
            check(api.echo(values) == values)
            mode = 1
            collectionChainWireReject("shorter than minLength") { api.echo(values) }
            mode = 0
            for (state in listOf<Patch<List<String?>>>(Patch.Absent, Patch.Null, Patch.Value(values))) {
                val input = SubmitInputDto(requiredNull = null, requiredFalse = values, inherited = state, wrapped = state)
                val objectValue = codec.encodeToJsonElement(SubmitInputDto.serializer(), input) as JsonObject
                for (key in listOf("inherited", "wrapped")) check(when (state) {
                    Patch.Absent -> key !in objectValue
                    Patch.Null -> objectValue[key] == JsonNull
                    is Patch.Value -> objectValue[key] == codec.parseToJsonElement(wire)
                })
                check(objectValue.containsKey("requiredNull") == nulls)
                check(codec.decodeFromJsonElement(SubmitInputDto.serializer(), objectValue) == input)
                expectedPatch = codec.encodeToString(SubmitInputDto.serializer(), input)
                api.submit(input)
            }
            val coldBefore = calls
            val cold = api.watch()
            check(calls == coldBefore)
            repeat(2) {
                val event = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Mixed }
                check(event.data == mixed && event.id == "collection-chain-id")
            }
            check(calls == coldBefore + 2)
            for ((bad, message) in listOf(1 to "shorter than minLength", 2 to "duplicate entries for uniqueItems", 3 to "fewer entries than minProperties", 4 to "fewer entries than minItems")) {
                mode = bad
                collectionChainWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Invalid referenced collection left still-open input alive" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 1)
            check(api.echo(values) == values)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Collection chain contracts passed: defaults=$defaults/nulls=$nulls; ctor-copy-serde/Patch/HTTP/fragmented live SSE close-reuse-cold")
    }
}
