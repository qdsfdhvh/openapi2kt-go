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
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import org.example.aliaschecks.ApiClient
import org.example.aliaschecks.api.models.PatchInputDto
import org.example.aliaschecks.api.models.WatchEventDto
import org.example.aliaschecks.models.IdDto
import org.example.aliaschecks.models.LinkDto
import org.example.aliaschecks.models.Patch
import org.example.aliaschecks.models.RecordDto

private suspend fun aliasFailure(block: suspend () -> Unit) {
    try { block(); error("Expected alias validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            it is IllegalArgumentException && it.message?.startsWith("Field value ") == true
        }) { "Alias validation cause lost: $cause" }
    }
}

private fun aliasBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> aliasBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected alias content $body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun aliasConstraintContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // Transparent aliases cannot validate arbitrary caller-owned creation/serde.
        val unchecked: IdDto = "BAD"
        val link: LinkDto = unchecked
        check(codec.decodeFromString(String.serializer(), "\"BAD\"") == link)
        val record = RecordDto("a", -2L, listOf("a", "bc"), mapOf("n" to 2L))
        val encoded = codec.encodeToString(RecordDto.serializer(), record)
        check(codec.decodeFromString(RecordDto.serializer(), encoded) == record)
        aliasFailure { record.copy(id = "BAD") }
        aliasFailure { record.copy(count = Long.MAX_VALUE) }
        aliasFailure { record.copy(ids = listOf("")) }
        aliasFailure { record.copy(counts = mapOf("n" to Long.MIN_VALUE)) }
        aliasFailure { codec.decodeFromString(RecordDto.serializer(), encoded.replace("bc", "BAD")) }
        val patches = listOf(PatchInputDto(), PatchInputDto(Patch.Null), PatchInputDto(Patch.Value("ab")))
        for (p in patches) check(codec.decodeFromString(PatchInputDto.serializer(), codec.encodeToString(PatchInputDto.serializer(), p)) == p)
        aliasFailure { PatchInputDto(Patch.Value("BAD")) }
        aliasFailure { codec.decodeFromString(PatchInputDto.serializer(), "{\"id\":\"BAD\"}") }
        aliasFailure { WatchEventDto.Message("BAD") }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var bad = false
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/v/a" -> {
                        check(request.url.parameters["count"] in listOf(null, "2"))
                        respond(if (bad) "3" else "2", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/id" -> {
                        // Bare String aliases keep Ktor's raw text wire, not JSON quotes.
                        check(aliasBody(request.body) == "ab")
                        respond(if (bad) "BAD" else "ab", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/record" -> {
                        check(codec.decodeFromString(RecordDto.serializer(), aliasBody(request.body)) == record)
                        respond(if (bad) encoded.replace("bc", "BAD") else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(codec.decodeFromString(PatchInputDto.serializer(), aliasBody(request.body)) in patches)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/params" -> {
                        check(request.url.parameters.getAll("ids") == listOf("a", "bc"))
                        check(request.url.parameters["n"] == "2")
                        check(request.url.parameters["ratio"] == "1.0")
                        check(request.headers["code"] == "ab")
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/values" -> respond(if (bad) "[{\"x\":\"BAD\"}]" else aliasBody(request.body), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val invalid = bad
                        producers.launch {
                            try {
                                for (byte in "data: \"${if (invalid) "BAD" else "ab"}\"\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (invalid) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected alias request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).aliases
            val before = calls
            aliasFailure { api.count("BAD") { error("Invalid preflight reached callback") } }
            aliasFailure { api.count("a", 3L) }
            aliasFailure { api.id("BAD") }
            aliasFailure { api.params(ids = listOf("BAD")) }
            aliasFailure { api.params(counts = mapOf("n" to 3L)) }
            aliasFailure { api.params(code = "BAD") }
            aliasFailure { api.params(ratio = Float.NaN) }
            aliasFailure { api.params(ratio = Float.POSITIVE_INFINITY) }
            aliasFailure { api.values(listOf(mapOf("x" to "BAD"))) }
            check(calls == before) { "Invalid alias sent network request" }
            check(api.count("a", 2L) == 2L)
            check(api.id("ab") == "ab")
            check(api.record(record) == record)
            for (p in patches) api.patchPatch(p)
            api.params(ids = listOf("a", null, "bc"), counts = mapOf("n" to 2L), code = "ab", ratio = 1f)
            val values = listOf(mapOf("x" to "ab", "nil" to null))
            check(api.values(values) == values)
            bad = true
            aliasFailure { api.count("a") }
            aliasFailure { api.id("ab") }
            aliasFailure { api.record(record) }
            aliasFailure { api.values(values) }
            aliasFailure { api.watch().toList() }
            check(channels.last().isClosedForRead) { "Invalid alias SSE did not close live input" }
            bad = false
            check((api.watch().toList().single() as WatchEventDto.Message).data == "ab")
            check(api.id("ab") == "ab")
            check(calls == 15)
            println("Alias contracts passed: defaults=$defaults/nulls=$nulls; ctor/copy/serde/Patch/elements/preflight/postflight/rawString/liveSSE-close/recovery; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
