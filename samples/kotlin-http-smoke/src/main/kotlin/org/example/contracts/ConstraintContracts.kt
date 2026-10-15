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
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import org.example.constraints.ApiClient
import org.example.constraints.api.models.PatchInputDto
import org.example.constraints.api.models.WatchEventDto
import org.example.constraints.models.CountDto
import org.example.constraints.models.EdgeDto
import org.example.constraints.models.NameDto
import org.example.constraints.models.Patch
import org.example.constraints.models.RateDto
import org.example.constraints.models.RecordDto
import org.example.constraints.models.ScaleDto
import org.example.constraints.models.TinyDto
import java.util.concurrent.CopyOnWriteArrayList

private suspend fun validationFailure(block: suspend () -> Unit) {
    try { block(); error("Expected intrinsic wrapper validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            it is IllegalArgumentException && it.message?.startsWith("rawValue ") == true
        }) { "Validation cause lost: $cause" }
    }
}

private fun constraintBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> constraintBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected constrained body $body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun constraintContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // JSON Schema character counts, not UTF16 code units or grapheme clusters.
        for (text in listOf("a", "🍃", "a🍃", "e\u0301")) {
            check(codec.decodeFromString(NameDto.serializer(), codec.encodeToString(NameDto.serializer(), NameDto(text))).rawValue == text)
        }
        for (bad in listOf("", "abc", "🍃🍃🍃")) {
            validationFailure { NameDto(bad) }
            validationFailure { codec.decodeFromString(NameDto.serializer(), JsonPrimitive(bad).toString()) }
        }
        for (value in listOf(-2L, 0L, 2L)) check(codec.decodeFromString(CountDto.serializer(), value.toString()).rawValue == value)
        for (bad in listOf(-3L, 3L)) {
            validationFailure { CountDto(bad) }
            validationFailure { codec.decodeFromString(CountDto.serializer(), bad.toString()) }
        }
        check(EdgeDto(Long.MIN_VALUE + 2).rawValue == Long.MIN_VALUE + 2)
        check(EdgeDto(Long.MAX_VALUE - 2).rawValue == Long.MAX_VALUE - 2)
        validationFailure { EdgeDto(Long.MIN_VALUE) }
        validationFailure { codec.decodeFromString(EdgeDto.serializer(), Long.MAX_VALUE.toString()) }
        check(TinyDto(1).rawValue == 1 && TinyDto(3).rawValue == 3)
        validationFailure { TinyDto(0) }
        check(RateDto(-0.25).rawValue == -0.25 && RateDto(0.25).rawValue == 0.25)
        for (bad in listOf(0.5, Double.NaN, Double.POSITIVE_INFINITY, Double.NEGATIVE_INFINITY)) validationFailure { RateDto(bad) }
        val special = Json(codec) { allowSpecialFloatingPointValues = true }
        validationFailure { special.decodeFromString(RateDto.serializer(), "NaN") }
        check(ScaleDto(0.1f).rawValue == 0.1f)
        check(ScaleDto(-1e-20f).rawValue == -1e-20f)
        validationFailure { ScaleDto(0.2f) }
        validationFailure { ScaleDto(Float.fromBits(0.1f.toBits() + 1)) }
        // Mapped Float semantics: source JSON precision can round to the bound.
        check(codec.decodeFromString(ScaleDto.serializer(), "0.100000001").rawValue == 0.1f)
        validationFailure { ScaleDto(Float.NaN) }
        val record = RecordDto(NameDto("🍃"), CountDto(2))
        check(codec.decodeFromString(RecordDto.serializer(), codec.encodeToString(RecordDto.serializer(), record)) == record)
        validationFailure { codec.decodeFromString(RecordDto.serializer(), "{\"name\":\"abc\",\"count\":2}") }
        validationFailure { codec.decodeFromString(ListSerializer(NameDto.serializer()), "[\"a\",\"abc\"]") }
        validationFailure { codec.decodeFromString(MapSerializer(String.serializer(), CountDto.serializer()), "{\"k\":3}") }
        validationFailure { codec.decodeFromString(PatchInputDto.serializer(), "{\"name\":\"abc\"}") }
        check(codec.encodeToJsonElement(PatchInputDto.serializer(), PatchInputDto()) == buildJsonObject {})
        check(codec.encodeToJsonElement(PatchInputDto.serializer(), PatchInputDto(Patch.Null)) == buildJsonObject { put("name", JsonNull) })
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var mode = "good"
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when {
                    request.url.encodedPath.startsWith("/name/") -> {
                        check(request.url.encodedPath == "/name/%F0%9F%8D%83")
                        check(constraintBody(request.body) == "\"🍃\"")
                        respond(if (mode == "good") "\"🍃\"" else "\"abc\"", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    request.url.encodedPath == "/count" -> respond(if (mode == "good") "2" else "3", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    request.url.encodedPath == "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val bad = mode != "good"
                        producers.launch {
                            try {
                                val frame = "data: \"${if (bad) "abc" else "🍃"}\"\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (bad) awaitCancellation() // still-open input must be closed on semantic decode failure
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    request.url.encodedPath == "/patch" -> {
                        check(codec.parseToJsonElement(constraintBody(request.body)) == buildJsonObject { put("name", "🍃") })
                        respond("", HttpStatusCode.NoContent)
                    }
                    else -> error("Unexpected constrained request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).checks
            check(api.name(NameDto("🍃"), NameDto("🍃")) == NameDto("🍃"))
            mode = "bad"
            validationFailure { api.name(NameDto("🍃"), NameDto("🍃")) }
            validationFailure { api.count() }
            validationFailure { api.watch().toList() }
            check(channels.last().isClosedForRead)
            mode = "good"
            check((api.watch().toList().single() as WatchEventDto.Message).data == NameDto("🍃"))
            check(api.count() == CountDto(2)) // bad HTTP/SSE payloads do not close caller client
            api.patchPatch(PatchInputDto(Patch.Value(NameDto("🍃"))))
            check(calls == 7)
            println("Constraint contracts passed: defaults=$defaults/nulls=$nulls; Unicode lengths/exactLong/finiteFloatDouble/init+serde/ref/list/map/Patch/invalidHTTP-SSE/client recovery; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
