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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.arraycomponents.ArrayComponentApiClient
import org.example.arraycomponents.api.models.SubmitInputDto
import org.example.arraycomponents.api.models.WatchEventDto
import org.example.arraycomponents.models.ACarrierDto
import org.example.arraycomponents.models.Patch
import org.example.arraycomponents.models.RecordDto
import org.example.arraycomponents.models.RowsDto
import org.example.arraycomponents.models.RowsDtoInner
import org.example.arraycomponents.models.StatesDto
import org.example.arraycomponents.models.StatesDtoInner
import org.example.arraycomponents.models.ZValuesDto
import org.example.arrayoriginal.models.Record as OriginalArray

private fun arrayReject(block: () -> Unit) {
    try { block(); error("Missing array component check") }
    catch (cause: IllegalArgumentException) { check(cause.message?.startsWith("Field value") == true) }
}

private suspend fun arrayWireReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing array component wire failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message) == true
        }) { "Wrong array component failure: $cause" }
    }
}

private fun arrayBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> arrayBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected array component body $body")
}

suspend fun arrayComponentContracts() {
    val values: ZValuesDto = listOf(null, "ok")
    val raw: RecordDto = listOf("raw")
    val original: OriginalArray = listOf("raw")
    val states: StatesDto = listOf(StatesDtoInner.OPEN, StatesDtoInner.Unrecognized("later"))
    val rows: RowsDto = listOf(RowsDtoInner("parent", children = listOf(RowsDtoInner("child"))))
    val value = ACarrierDto(values, listOf(values), mapOf("k" to values), states, rows)
    arrayReject { ACarrierDto(emptyList()) }
    arrayReject { value.copy(values = listOf("ok", "ok")) }
    arrayReject { value.copy(nested = listOf(listOf("a"))) }
    arrayReject { value.copy(entries = mapOf("k" to listOf(null, null))) }
    arrayReject { value.copy(rows = emptyList()) }
    arrayReject { RowsDtoInner("") }
    arrayReject { rows.single().copy(children = emptyList()) }
    arrayReject { SubmitInputDto(requiredNull = null, values = Patch.Value(listOf("a"))) }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val sseCodec = Json(codec) { ignoreUnknownKeys = false }
        val text = codec.encodeToString(ACarrierDto.serializer(), value)
        check(codec.decodeFromString(ACarrierDto.serializer(), text) == value)
        check(codec.decodeFromJsonElement(ACarrierDto.serializer(), codec.parseToJsonElement(text)) == value)
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(raw)) == raw)
        check(codec.decodeFromString<OriginalArray>(codec.encodeToString(original)) == original)
        check(codec.decodeFromString<StatesDto>(codec.encodeToString(states)) == states)
        check(codec.decodeFromString<RowsDto>(codec.encodeToString(rows)) == rows)
        arrayReject { codec.decodeFromString(ACarrierDto.serializer(), text.replace("\"ok\"", "\"a\"")) }
        arrayReject { codec.decodeFromJsonElement(ACarrierDto.serializer(), codec.parseToJsonElement(text.replace("[null,\"ok\"]", "[null,null]"))) }
        // Transparent typealiases cannot attach independent constructor/serde checks.
        val unchecked: ZValuesDto = codec.decodeFromString("[\"a\"]")
        check(unchecked == listOf("a"))
        arrayReject { ACarrierDto(unchecked) }
        val wire = codec.encodeToString(values)
        val rowsWire = codec.encodeToString(rows)
        var mode = 0
        var calls = 0
        var expectedPatch = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/values" -> {
                        check(arrayBody(request.body) == wire)
                        respond(if (mode == 1) "[\"a\"]" else wire,
                            HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/rows" -> {
                        check(arrayBody(request.body) == rowsWire)
                        respond(if (mode == 1) rowsWire.replace("\"parent\"", "\"\"") else rowsWire,
                            HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(arrayBody(request.body) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val payload = when (current) {
                                    1 -> "[\"a\"]"
                                    2 -> "[null,null]"
                                    3 -> "[]"
                                    4 -> "[\"ok\",\"yes\",\"no\",\"hey\"]"
                                    else -> wire
                                }
                                val frame = "event: values\r\nid: array-id\r\ndata: $payload\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected array component dispatch")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ArrayComponentApiClient(http, "http://example.test/", sseCodec).default
            val before = calls
            var callback = false
            arrayWireReject("shorter than minLength") { api.echo(unchecked) { callback = true } }
            arrayWireReject("duplicate entries for uniqueItems") { api.echo(listOf(null, null)) }
            arrayWireReject("fewer entries than minItems") { api.echoRows(emptyList()) }
            check(!callback && calls == before)
            check(api.echo(values) == values)
            check(api.echoRows(rows) == rows)
            mode = 1
            arrayWireReject("shorter than minLength") { api.echo(values) }
            arrayWireReject("shorter than minLength") { api.echoRows(rows) }
            mode = 0
            for (state in listOf<Patch<ZValuesDto>>(Patch.Absent, Patch.Null, Patch.Value(values))) {
                val input = SubmitInputDto(requiredNull = null, values = state)
                val objectValue = codec.encodeToJsonElement(SubmitInputDto.serializer(), input) as JsonObject
                check(when (state) {
                    Patch.Absent -> "values" !in objectValue
                    Patch.Null -> objectValue["values"] == JsonNull
                    is Patch.Value -> objectValue["values"] == codec.parseToJsonElement(wire)
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
                val event = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Values }
                check(event.data == values && event.id == "array-id")
            }
            check(calls == coldBefore + 2)
            for ((bad, message) in listOf(1 to "shorter than minLength", 2 to "duplicate entries for uniqueItems", 3 to "fewer entries than minItems", 4 to "more entries than maxItems")) {
                mode = bad
                arrayWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Invalid array alias left still-open stream alive" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 1)
            check(api.echo(values) == values)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Array component contracts passed: defaults=$defaults/nulls=$nulls; aliases/peers/recursion/ctor-copy-serde/Patch/HTTP/live SSE close-reuse-cold")
    }
}
