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
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.chains.ScalarChainApiClient
import org.example.chains.api.models.SubmitInputDto
import org.example.chains.api.models.WatchEventDto
import org.example.chains.models.Patch
import org.example.chains.models.RecordDto

private fun chainReject(block: () -> Unit) {
    try { block(); error("Missing scalar-chain check") }
    catch (cause: IllegalArgumentException) { check(cause.message?.startsWith("Field value") == true) }
}
private suspend fun chainWireReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing scalar-chain wire failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message) == true
        }) { "Wrong scalar-chain failure: $cause" }
    }
}
private fun chainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> chainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected scalar-chain body $body")
}

suspend fun schemaInternalScalarChainContracts() {
    val value = RecordDto("ok", null, listOf(null, "ok"), mapOf("key" to null, "value" to "ok"))
    chainReject { RecordDto("a", null, value.list, value.map) }
    chainReject { value.copy(nullable = "a") }
    chainReject { value.copy(list = listOf("ok", "ok")) }
    chainReject { value.copy(list = listOf("a")) }
    chainReject { value.copy(map = mapOf("key" to "a")) }
    chainReject { SubmitInputDto(null, "ok", inherited = Patch.Value("a")) }
    val descriptor = SubmitInputDto.serializer().descriptor
    check(descriptor.getElementDescriptor(0).isNullable && !descriptor.isElementOptional(0))
    check(!descriptor.getElementDescriptor(1).isNullable && !descriptor.isElementOptional(1))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val strict = Json(codec) { ignoreUnknownKeys = false }
        val text = codec.encodeToString(RecordDto.serializer(), value)
        check(codec.decodeFromString(RecordDto.serializer(), text) == value)
        check(codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(text)) == value)
        chainReject { codec.decodeFromString(RecordDto.serializer(), text.replace("\"value\":\"ok\"", "\"value\":\"a\"")) }
        chainReject { codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(text.replace("\"ok\"", "\"a\""))) }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var mode = 0
        var calls = 0
        var expectedPatch = ""
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/records" -> {
                        check(request.url.parameters["code"] == "ok")
                        check(chainBody(request.body) == text)
                        respond(if (mode == 1) text.replace("\"value\":\"ok\"", "\"value\":\"a\"") else text,
                            HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/submit" -> {
                        check(chainBody(request.body) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val payload = when (current) {
                                    1 -> text.replace("\"value\":\"ok\"", "\"value\":\"a\"")
                                    2 -> text.replace("[null,\"ok\"]", "[null,null]")
                                    3 -> text.dropLast(1) + ",\"unknown\":null}"
                                    else -> text
                                }
                                val frame = "event: record\r\nid: chain-id\r\ndata: $payload\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected scalar-chain dispatch")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ScalarChainApiClient(http, "http://example.test/", strict).default
            val before = calls
            var callback = false
            chainWireReject("Field value is shorter than minLength.") { api.echo("a", value) { callback = true } }
            check(!callback && calls == before)
            check(api.echo("ok", value) == value)
            mode = 1
            chainWireReject("Field value is shorter than minLength.") { api.echo("ok", value) }
            mode = 0
            for (state in listOf<Patch<String>>(Patch.Absent, Patch.Null, Patch.Value("ok"))) {
                val input = SubmitInputDto(null, "ok", inherited = state, wrapped = state)
                val objectValue = codec.encodeToJsonElement(SubmitInputDto.serializer(), input) as JsonObject
                for (key in listOf("inherited", "wrapped")) check(when (state) {
                    Patch.Absent -> key !in objectValue
                    Patch.Null -> objectValue[key] == JsonNull
                    is Patch.Value -> objectValue[key] == JsonPrimitive("ok")
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
                val event = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
                check(event.data == value && event.id == "chain-id")
            }
            check(calls == coldBefore + 2)
            for ((bad, message) in listOf(1 to "shorter than minLength", 2 to "duplicate entries for uniqueItems", 3 to "unknown")) {
                mode = bad
                chainWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Failed chain payload left still-open input alive" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 1)
            check(api.echo("ok", value) == value)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Scalar chain contracts passed: defaults=$defaults/nulls=$nulls; ctor-copy-serde/Patch/source HTTP/fragmented live SSE close-reuse-cold")
    }
}
