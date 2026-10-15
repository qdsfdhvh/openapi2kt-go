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
import kotlinx.serialization.json.JsonObject
import org.example.uniquechecks.ApiClient
import org.example.uniquechecks.api.models.PatchInputDto
import org.example.uniquechecks.api.models.WatchEventDto
import org.example.uniquechecks.models.BagDto
import org.example.uniquechecks.models.Patch
import org.example.uniquechecks.models.RecordDto

private suspend fun uniqueFailure(block: suspend () -> Unit) {
    try { block(); error("Expected collection validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException && it.message?.startsWith("Field value ") == true }) { "Validation cause lost: $cause" }
    }
}
private fun uniqueBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> uniqueBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected uniqueness body $body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun collectionUniqueContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val strings = mutableListOf("z", "é", "e\u0301", "a")
        val ints = mutableListOf(Int.MAX_VALUE, Int.MIN_VALUE, 0)
        val longs = mutableListOf(Long.MAX_VALUE, Long.MIN_VALUE, 0L)
        val bools = mutableListOf(true, false)
        val nullable = mutableListOf("b", null, "a")
        val record = RecordDto(strings = strings, ints = ints, longs = longs, bools = bools, nullable = nullable,
            aliases = listOf("ok", null), nullableLongs = listOf(Long.MAX_VALUE, null, Long.MIN_VALUE), bag = mapOf("x" to listOf(2, null, 1)), optional = null,
            kotlin = "scope", value = "scope", size = "scope", compareTo = "scope", openapiKtUniqueItems = "scope",
            noop = listOf(listOf("same"), listOf("same")))
        check(strings == listOf("z", "é", "e\u0301", "a") && ints == listOf(Int.MAX_VALUE, Int.MIN_VALUE, 0))
        check(longs == listOf(Long.MAX_VALUE, Long.MIN_VALUE, 0L) && bools == listOf(true, false) && nullable == listOf("b", null, "a"))
        val encoded = codec.encodeToString(RecordDto.serializer(), record)
        check(codec.decodeFromString<RecordDto>(encoded) == record)
        check(record.copy(ints = emptyList(), longs = listOf(0L), bools = emptyList(), nullable = listOf(null)).ints.isEmpty())
        check(record.copy(strings = listOf("A", "a")).strings == listOf("A", "a"))
        check(record.copy(bools = listOf(false)).bools == listOf(false))
        uniqueFailure { record.copy(strings = listOf("a", "a")) }
        uniqueFailure { record.copy(ints = listOf(Int.MIN_VALUE, Int.MIN_VALUE)) }
        uniqueFailure { record.copy(longs = listOf(Long.MAX_VALUE, Long.MAX_VALUE)) }
        uniqueFailure { record.copy(bools = listOf(false, false)) }
        uniqueFailure { record.copy(nullable = listOf(null, null)) }
        uniqueFailure { record.copy(nullable = listOf("a", null, "a")) }
        uniqueFailure { record.copy(aliases = listOf("x", "x")) }
        uniqueFailure { record.copy(aliases = listOf("")) }
        uniqueFailure { record.copy(aliases = listOf(null, null)) }
        uniqueFailure { record.copy(aliases = emptyList()) }
        uniqueFailure { record.copy(strings = listOf("")) }
        uniqueFailure { record.copy(nullableLongs = listOf(Long.MIN_VALUE, null, Long.MIN_VALUE)) }
        uniqueFailure { record.copy(nullableLongs = listOf(null, null)) }
        check(record.copy(nullableLongs = listOf(null)).nullableLongs == listOf(null))
        uniqueFailure { record.copy(bag = mapOf("x" to listOf(null, null))) }
        uniqueFailure { record.copy(bag = mapOf("x" to emptyList())) }
        uniqueFailure { record.copy(optional = listOf("x", "x")) }
        check(record.copy(optional = emptyList()).optional?.isEmpty() == true)
        uniqueFailure { record.copy(strings = emptyList()) }
        uniqueFailure { record.copy(strings = listOf("a", "b", "c", "d", "e")) }
        // Equality follows parsed Strings, not JSON escape spelling.
        for ((field, bad) in listOf("strings" to "[\"a\",\"\\u0061\"]", "ints" to "[1,1]", "longs" to "[9223372036854775807,9223372036854775807]", "bools" to "[true,true]", "nullable" to "[null,null]", "nullableLongs" to "[null,null]")) {
            val parsed = codec.parseToJsonElement(encoded) as JsonObject
            val replaced = JsonObject(parsed + (field to codec.parseToJsonElement(bad)))
            uniqueFailure { codec.decodeFromString<RecordDto>(replaced.toString()) }
        }
        // Transparent collection serialization and later mutation are not barriers.
        check(codec.decodeFromString<BagDto>("{\"x\":[1,1]}")["x"] == listOf(1, 1))
        val patches = listOf(PatchInputDto(), PatchInputDto(Patch.Null), PatchInputDto(Patch.Value(listOf("a"))))
        for (p in patches) check(codec.decodeFromString<PatchInputDto>(codec.encodeToString(PatchInputDto.serializer(), p)) == p)
        uniqueFailure { PatchInputDto(Patch.Value(listOf("a", "a"))) }
        uniqueFailure { codec.decodeFromString<PatchInputDto>("{\"rows\":[\"a\",\"a\"]}") }
        uniqueFailure { WatchEventDto.Message(listOf(null, null)) }
        uniqueFailure { WatchEventDto.Message(listOf(true, true)) }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var bad = false
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/values" -> respond(if (bad) "[1,1]" else uniqueBody(request.body), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    "/patch" -> respond("", HttpStatusCode.NoContent)
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val invalid = bad
                        producers.launch {
                            try {
                                val frame = if (invalid) "data: [null,null]\r\n\r\n" else "data: [true,null,false]\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (invalid) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected unique request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).unique
            val before = calls
            uniqueFailure { api.values(listOf(1, 1)) { error("Invalid input reached callback") } }
            uniqueFailure { api.values(listOf(null, null)) }
            uniqueFailure { api.values(listOf(-1)) }
            uniqueFailure { api.patchPatch(PatchInputDto(Patch.Value(listOf("x", "x")))) { error("Invalid Patch reached callback") } }
            check(calls == before)
            val values = mutableListOf(2, null, 1)
            check(api.values(values) == values && values == listOf(2, null, 1))
            for (p in patches) api.patchPatch(p)
            bad = true
            uniqueFailure { api.values(values) }
            uniqueFailure { api.watch().toList() }
            check(channels.last().isClosedForRead)
            bad = false
            check((api.watch().toList().single() as WatchEventDto.Message).data == listOf(true, null, false))
            check(api.values(values) == values)
            check(calls == 8)
            println("Collection unique contracts passed: defaults=$defaults/nulls=$nulls; native equality/null/Unicode/ctor-copy-serde/alias/Patch/no-mutation/preflight-postflight/fragmented-liveSSE-close-reuse; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
