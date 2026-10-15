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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import org.example.countchecks.ApiClient
import org.example.countchecks.api.models.PatchInputDto
import org.example.countchecks.api.models.WatchEventDto
import org.example.countchecks.models.BagDto
import org.example.countchecks.models.Patch
import org.example.countchecks.models.RecordDto

private suspend fun countFailure(block: suspend () -> Unit) {
    try { block(); error("Expected count validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException && it.message?.startsWith("Field value ") == true }) { "Count cause lost: $cause" }
    }
}
private fun countBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> countBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected count body $body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun collectionCountContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val unchecked = codec.decodeFromString<BagDto>("{}")
        check(unchecked.isEmpty())
        val record = RecordDto(rows = listOf(mapOf("key" to listOf("a", null))), bag = mapOf("x" to listOf(0, 1)), optional = null)
        val encoded = codec.encodeToString(RecordDto.serializer(), record)
        check(codec.decodeFromString<RecordDto>(encoded) == record)
        check(record.copy(optional = listOf("a", "b")).optional?.size == 2)
        countFailure { record.copy(rows = emptyList()) }
        countFailure { record.copy(rows = listOf(emptyMap())) }
        countFailure { record.copy(rows = listOf(mapOf("x" to emptyList()))) }
        countFailure { record.copy(rows = listOf(mapOf("x" to listOf("")))) }
        countFailure { record.copy(bag = emptyMap()) }
        countFailure { record.copy(bag = mapOf("x" to listOf(-1))) }
        check(record.copy(raw = codec.parseToJsonElement("{\"x\":null}").jsonObject).raw?.size == 1)
        countFailure { record.copy(raw = codec.parseToJsonElement("{}").jsonObject) }
        countFailure { record.copy(raw = codec.parseToJsonElement("{\"x\":1,\"y\":2,\"z\":3}").jsonObject) }
        countFailure { record.copy(optional = emptyList()) }
        countFailure { record.copy(optional = listOf("a", "b", "c")) }
        countFailure { codec.decodeFromString<RecordDto>(encoded.replace("[0,1]", "[]")) }
        val patches = listOf(PatchInputDto(), PatchInputDto(Patch.Null), PatchInputDto(Patch.Value(listOf("a"))))
        for (p in patches) check(codec.decodeFromString<PatchInputDto>(codec.encodeToString(PatchInputDto.serializer(), p)) == p)
        countFailure { PatchInputDto(Patch.Value(emptyList())) }
        countFailure { codec.decodeFromString<PatchInputDto>("{\"rows\":[]}") }
        countFailure { WatchEventDto.Message(emptyList()) }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var bad = false
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/values" -> respond(if (bad) "[]" else countBody(request.body), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    "/patch" -> respond("", HttpStatusCode.NoContent)
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val invalid = bad
                        producers.launch {
                            try {
                                val frame = if (invalid) "data: []\r\n\r\n" else "data: [1,2]\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (invalid) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected count request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).counts
            val before = calls
            countFailure { api.values(emptyList()) { error("Invalid count reached callback") } }
            countFailure { api.values(listOf(emptyMap())) }
            check(calls == before)
            val values = listOf(mapOf("x" to 1))
            check(api.values(values) == values)
            for (p in patches) api.patchPatch(p)
            bad = true
            countFailure { api.values(values) }
            countFailure { api.watch().toList() }
            check(channels.last().isClosedForRead)
            bad = false
            check((api.watch().toList().single() as WatchEventDto.Message).data == listOf(1, 2))
            check(api.values(values) == values)
            check(calls == 8)
            println("Collection count contracts passed: defaults=$defaults/nulls=$nulls; container-element/ctor-copy-serde/alias/Patch/null/preflight-postflight/liveSSE-close/recovery; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
