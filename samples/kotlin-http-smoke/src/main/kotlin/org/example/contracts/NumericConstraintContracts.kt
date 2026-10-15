package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.http.HttpStatusCode
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
import org.example.numericchecks.SyntheticNumericBoundsApiClient
import org.example.numericchecks.api.models.PatchInputDto
import org.example.numericchecks.api.models.WatchEventDto
import org.example.numericchecks.models.EvenDto
import org.example.numericchecks.models.Patch
import org.example.numericchecks.models.RecordDto
import org.example.numericchecks.models.WindowDto
import org.example.numericlegacy.models.RecordDto as LegacyRecordDto
import org.example.numericlegacy.models.WindowDto as LegacyWindowDto

private suspend fun numericFailure(label: String = "Field value ", block: suspend () -> Unit) {
    try { block(); error("Expected numeric validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            it is IllegalArgumentException && it.message?.startsWith(label) == true
        }) { "Numeric validation cause lost: $cause" }
    }
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun numericConstraintContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(LegacyWindowDto(4).rawValue == 4)
        check(LegacyRecordDto(4).value == 4)
        check(LegacyRecordDto(4, Double.NaN).free?.isNaN() == true) // false is a no-op
        numericFailure("rawValue ") { LegacyWindowDto(2) }
        numericFailure("rawValue ") { LegacyWindowDto(8) }
        numericFailure { LegacyRecordDto(3) }
        numericFailure { codec.decodeFromString<LegacyRecordDto>("{\"value\":8}") }
        check(WindowDto(Long.MIN_VALUE + 2).rawValue == Long.MIN_VALUE + 2)
        numericFailure("rawValue ") { WindowDto(Long.MIN_VALUE) }
        numericFailure("rawValue ") { WindowDto(Long.MAX_VALUE) }
        numericFailure("rawValue ") { WindowDto(3) }
        numericFailure("rawValue ") { codec.decodeFromString<WindowDto>("3") }
        val unchecked: EvenDto = codec.decodeFromString<EvenDto>("3")
        check(unchecked == 3L) // transparent serializer is intentionally unchecked
        val record = RecordDto(kotlin = "shadow", even = Long.MIN_VALUE, steps = 4, fraction = 0.5, matrix = mapOf("arbitrary key" to listOf(2, null, 4)))
        val encoded = codec.encodeToString(RecordDto.serializer(), record)
        check(codec.decodeFromString<RecordDto>(encoded) == record)
        check(record.copy(matrix = emptyMap()).matrix.isEmpty())
        numericFailure { record.copy(even = 3) }
        numericFailure { record.copy(steps = 2) }
        numericFailure { record.copy(steps = 10) }
        numericFailure { record.copy(steps = 3) }
        numericFailure { record.copy(fraction = 0.0) }
        numericFailure { record.copy(fraction = 1.0) }
        numericFailure { record.copy(fraction = 0.3) }
        numericFailure { record.copy(fraction = Double.NaN) }
        numericFailure { record.copy(fraction = Double.POSITIVE_INFINITY) }
        check(record.copy(tenthsDouble = 0.1, tenthsFloat = 0.1f).tenthsDouble == 0.1)
        // Strict mapped remainder, not decimal arithmetic or hidden epsilon.
        numericFailure { record.copy(tenthsDouble = 0.3) }
        numericFailure { record.copy(tenthsFloat = 0.3f) }
        numericFailure { record.copy(matrix = mapOf("x" to listOf(3))) }
        numericFailure { codec.decodeFromString<RecordDto>(encoded.replace("\"steps\":4", "\"steps\":3")) }
        val patches = listOf(PatchInputDto(), PatchInputDto(Patch.Null), PatchInputDto(Patch.Value(4)))
        for (p in patches) check(codec.decodeFromString<PatchInputDto>(codec.encodeToString(PatchInputDto.serializer(), p)) == p)
        numericFailure { PatchInputDto(Patch.Value(3)) }
        numericFailure { codec.decodeFromString<PatchInputDto>("{\"steps\":2}") }
        numericFailure { WatchEventDto.Message(listOf(3)) }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var bad = false
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/count" -> {
                        check(request.url.parameters["value"] == Long.MIN_VALUE.toString())
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/patch" -> respond("", HttpStatusCode.NoContent)
                    "/echo" -> respond(if (bad) "{\"x\":[3]}" else "{\"x\":[2,4]}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val invalid = bad
                        producers.launch {
                            try {
                                val frame = if (invalid) "data: [3]\r\n\r\n" else "data: [2,4]\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (invalid) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected numeric request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = SyntheticNumericBoundsApiClient(http, "http://example.test/", codec).numbers
            val before = calls
            numericFailure { api.count(3) { error("Invalid numeric value reached callback") } }
            numericFailure { api.echo(listOf(3)) }
            check(calls == before)
            api.count(Long.MIN_VALUE)
            for (p in patches) api.patchPatch(p)
            check(api.echo(listOf(Long.MIN_VALUE, 2)) == mapOf("x" to listOf(2, 4)))
            bad = true
            numericFailure { api.echo(listOf(2)) }
            numericFailure { api.watch().toList() }
            check(channels.last().isClosedForRead)
            bad = false
            check((api.watch().toList().single() as WatchEventDto.Message).data == listOf(2, 4))
            api.count(Long.MIN_VALUE)
            check(calls == 9)
            println("Numeric contracts passed: defaults=$defaults/nulls=$nulls; exclusive/multipleOf/extrema/namespace/ctor-copy-serde/Patch/typed-preflight-postflight/liveSSE-close/recovery; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
