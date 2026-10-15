package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.MapSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import org.example.patterns.ApiClient
import org.example.patterns.api.models.PatchInputDto
import org.example.patterns.api.models.WatchEventDto
import org.example.patterns.models.CodeDto
import org.example.patterns.models.SearchDto
import org.example.patterns.models.EscapedDto
import org.example.patterns.models.RecordDto
import org.example.patterns.models.Patch
import java.util.concurrent.CopyOnWriteArrayList

private suspend fun patternFailure(block: suspend () -> Unit) {
    try { block(); error("Expected pattern validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            it is IllegalArgumentException && it.message?.endsWith("does not match pattern.") == true
        }) { "Pattern cause lost: $cause" }
    }
}

private fun patternBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> patternBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected pattern content $body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun patternContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(CodeDto("AB123").rawValue == "AB123")
        val escaped = EscapedDto("a\"\\")
        check(codec.decodeFromString(EscapedDto.serializer(), codec.encodeToString(EscapedDto.serializer(), escaped)) == escaped)
        patternFailure { EscapedDto("not a quote/backslash") }
        for (bad in listOf("ab123", "AB", "XAB123X", "")) {
            patternFailure { CodeDto(bad) }
            patternFailure { codec.decodeFromString(CodeDto.serializer(), JsonPrimitive(bad).toString()) }
        }
        // JSON Schema pattern is substring matching, not whole String.matches.
        for (text in listOf("cat", "🌿cat🌿", "hotdog")) check(SearchDto(text).rawValue == text)
        patternFailure { SearchDto("bird") }
        val record = RecordDto(value = "xxaaabyy", kotlin = "ok", code = CodeDto("AB123"))
        val encoded = codec.encodeToString(RecordDto.serializer(), record)
        check(codec.decodeFromString(RecordDto.serializer(), encoded) == record)
        check(record.copy(note = "hotdog").note == "hotdog")
        check(record.copy(note = null).note == null)
        patternFailure { record.copy(value = "xyz") }
        patternFailure { record.copy(kotlin = "bad") }
        patternFailure { record.copy(note = "bird") }
        patternFailure { codec.decodeFromString(RecordDto.serializer(), encoded.replace("xxaaabyy", "xyz")) }
        patternFailure { codec.decodeFromString(ListSerializer(CodeDto.serializer()), "[\"AB123\",\"bad\"]") }
        patternFailure { codec.decodeFromString(MapSerializer(String.serializer(), CodeDto.serializer()), "{\"k\":\"bad\"}") }
        val absent = PatchInputDto()
        val nil = PatchInputDto(Patch.Null)
        val some = PatchInputDto(Patch.Value("xaaabx"))
        for (input in listOf(absent, nil, some)) check(codec.decodeFromString(PatchInputDto.serializer(), codec.encodeToString(PatchInputDto.serializer(), input)) == input)
        patternFailure { PatchInputDto(Patch.Value("bad")) }
        patternFailure { codec.decodeFromString(PatchInputDto.serializer(), "{\"value\":\"bad\"}") }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var bad = false
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/code" -> respond("\"${if (bad) "bad" else "AB123"}\"", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    "/record" -> {
                        check(codec.decodeFromString(RecordDto.serializer(), patternBody(request.body)) == record)
                        respond(if (bad) encoded.replace("xxaaabyy", "xyz") else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        val input = codec.decodeFromString(PatchInputDto.serializer(), patternBody(request.body))
                        check(input in listOf(absent, nil, some))
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val invalid = bad
                        producers.launch {
                            try {
                                val frame = "data: \"${if (invalid) "bad" else "AB123"}\"\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (invalid) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected pattern request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).patterns
            check(api.code() == CodeDto("AB123"))
            check(api.record(record) == record)
            for (input in listOf(absent, nil, some)) api.patchPatch(input)
            bad = true
            patternFailure { api.code() }
            patternFailure { api.record(record) }
            patternFailure { api.watch().toList() }
            check(channels.last().isClosedForRead)
            bad = false
            check((api.watch().toList().single() as WatchEventDto.Message).data == CodeDto("AB123"))
            check(api.code() == CodeDto("AB123"))
            check(calls == 10)
            println("Pattern contracts passed: defaults=$defaults/nulls=$nulls; ctor/copy/serde/substring/anchors/namespace-field/ref/list/map/Patch/invalidHTTP-SSE/client recovery; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
