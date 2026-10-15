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
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import org.example.dtochecks.ApiClient
import org.example.dtochecks.api.models.InlineChecksInputDto
import org.example.dtochecks.api.models.InlineChecksResponseDto
import org.example.dtochecks.api.models.WatchEventDto
import org.example.dtochecks.models.PlainDto
import org.example.dtochecks.models.BoxDto
import org.example.dtochecks.models.Patch
import java.util.concurrent.CopyOnWriteArrayList

private suspend fun dtoFailure(block: suspend () -> Unit) {
    try { block(); error("Expected DTO validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            it is IllegalArgumentException && it.message?.startsWith("Field value ") == true
        }) { "Missing DTO validation cause: $cause" }
    }
}

private fun plain(name: String = "🍃", count: Int = 2): PlainDto =
    PlainDto(name = name, count = count, value = "xy", length = 0, `when` = "z")

private fun dtoBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> dtoBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected DTO content $body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun dtoConstraintContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val valid = plain()
        val encoded = codec.encodeToString(PlainDto.serializer(), valid)
        check(codec.decodeFromString(PlainDto.serializer(), encoded) == valid)
        for (text in listOf("a", "🍃", "a🍃", "e\u0301")) check(plain(text).name == text)
        for (bad in listOf("", "abc", "🍃🍃🍃")) {
            dtoFailure { plain(bad) }
            dtoFailure { valid.copy(name = bad) }
            dtoFailure { codec.decodeFromString(PlainDto.serializer(), encoded.replace("\"🍃\"", JsonPrimitive(bad).toString())) }
        }
        dtoFailure { plain(count = 3) }
        dtoFailure { valid.copy(length = -1) }
        dtoFailure { valid.copy(value = "abc") }
        dtoFailure { valid.copy(`when` = "") }
        dtoFailure { valid.copy(requiredNullable = "abc") }
        dtoFailure { valid.copy(edge = Long.MIN_VALUE) }
        check(valid.copy(edge = Long.MIN_VALUE + 2).edge == Long.MIN_VALUE + 2)
        check(valid.copy(edge = Long.MAX_VALUE - 2).edge == Long.MAX_VALUE - 2)
        check(valid.copy(requiredNullable = null, edge = null).requiredNullable == null)
        check(PlainDto.Child(0.1f).scale == 0.1f)
        dtoFailure { PlainDto.Child(Float.fromBits(0.1f.toBits() + 1)) }
        dtoFailure { PlainDto.Child(Float.NaN) }
        val special = Json(codec) { allowSpecialFloatingPointValues = true }
        dtoFailure { special.decodeFromString(PlainDto.Child.serializer(), "{\"scale\":NaN}") }
        val boxed = BoxDto(valid, listOf(valid), mapOf("k" to valid))
        val boxJson = codec.encodeToString(BoxDto.serializer(), boxed)
        check(codec.decodeFromString(BoxDto.serializer(), boxJson) == boxed)
        for (field in listOf("record", "records", "mapped")) {
            val badJson = buildJsonObject {
                put("record", codec.encodeToJsonElement(PlainDto.serializer(), valid))
                put("records", buildJsonArray { add(codec.encodeToJsonElement(PlainDto.serializer(), valid)) })
                put("mapped", buildJsonObject { put("k", codec.encodeToJsonElement(PlainDto.serializer(), valid)) })
                val bad = codec.parseToJsonElement(encoded.replace("\"🍃\"", "\"abc\""))
                when (field) {
                    "record" -> put(field, bad)
                    "records" -> put(field, buildJsonArray { add(bad) })
                    else -> put(field, buildJsonObject { put("k", bad) })
                }
            }
            dtoFailure { codec.decodeFromJsonElement(BoxDto.serializer(), badJson) }
        }
        val absent = InlineChecksInputDto(rate = 0.5)
        val nil = InlineChecksInputDto(name = Patch.Null, rate = 0.5)
        val some = InlineChecksInputDto(name = Patch.Value("🍃"), rate = 0.5)
        for (input in listOf(absent, nil, some)) check(codec.decodeFromString(InlineChecksInputDto.serializer(), codec.encodeToString(InlineChecksInputDto.serializer(), input)) == input)
        dtoFailure { InlineChecksInputDto(name = Patch.Value("abc"), rate = 0.5) }
        dtoFailure { absent.copy(name = Patch.Value("")) }
        dtoFailure { codec.decodeFromString(InlineChecksInputDto.serializer(), "{\"name\":\"abc\",\"rate\":0.5}") }
        dtoFailure { InlineChecksInputDto(rate = Double.POSITIVE_INFINITY) }
        dtoFailure { special.decodeFromString(InlineChecksInputDto.serializer(), "{\"rate\":NaN}") }
        dtoFailure { codec.decodeFromString(InlineChecksResponseDto.serializer(), "{\"value\":\"abc\"}") }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var bad = false
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/round" -> {
                        check(codec.decodeFromString(PlainDto.serializer(), dtoBody(request.body)) == valid)
                        respond(if (bad) encoded.replace("\"🍃\"", "\"abc\"") else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> {
                        val element = codec.parseToJsonElement(dtoBody(request.body)) as JsonObject
                        check(element["rate"] == JsonPrimitive(0.5))
                        check(element["name"] == null || element["name"] == JsonNull || element["name"] == JsonPrimitive("🍃"))
                        respond("{\"value\":\"${if (bad) "abc" else "🍃"}\"}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val invalid = bad
                        producers.launch {
                            try {
                                val frame = "data: {\"value\":\"${if (invalid) "abc" else "🍃"}\"}\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (invalid) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected DTO request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).models
            check(api.round(valid) == valid)
            for (input in listOf(absent, nil, some)) check(api.inlineChecks(input).value == "🍃")
            bad = true
            dtoFailure { api.round(valid) }
            dtoFailure { api.inlineChecks(some) }
            dtoFailure { api.watch().toList() }
            check(channels.last().isClosedForRead)
            bad = false
            check((api.watch().toList().single() as WatchEventDto.Message).data.value == "🍃")
            check(api.round(valid) == valid)
            check(calls == 9)
            println("DTO constraint contracts passed: defaults=$defaults/nulls=$nulls; init+copy+serde/Unicode/exactLong/finiteFloatDouble/nestedDTO/list-mapDTO/nullable/Patch/HTTP-SSE validation/client recovery; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
