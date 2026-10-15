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
import kotlinx.serialization.json.encodeToJsonElement
import org.example.nullablearrays.NullableArrayApiClient
import org.example.nullablearrays.api.models.SubmitInputDto
import org.example.nullablearrays.api.models.WatchEventDto
import org.example.nullablearrays.models.ACarrierDto
import org.example.nullablearrays.models.Patch
import org.example.nullablearrays.models.RowsDto
import org.example.nullablearrays.models.RowsDtoInner
import org.example.nullablearrays.models.StatesDto
import org.example.nullablearrays.models.StatesDtoInner
import org.example.nullablearrays.models.ZValuesDto

private fun nullableArrayReject(block: () -> Unit) {
    try { block(); error("Missing nullable array validation") }
    catch (cause: IllegalArgumentException) { check(cause.message?.startsWith("Field value") == true) }
}

private suspend fun nullableArrayWireReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing nullable array wire failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong nullable array failure: $cause" }
    }
}

private fun nullableArrayBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableArrayBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable array request body $body")
}

suspend fun nullableArrayComponentContracts() {
    val rows = listOf(RowsDtoInner("parent", children = listOf(RowsDtoInner("child"))))
    val states = listOf(StatesDtoInner.OPEN, StatesDtoInner.Unrecognized("later"))
    val values = listOf(null, "ok")
    val value = ACarrierDto(narrow = rows, wrappedNarrow = rows, states = states, values = values, nested = listOf(null, values))
    nullableArrayReject { value.copy(narrow = emptyList()) }
    nullableArrayReject { value.copy(inherited = emptyList()) }
    nullableArrayReject { value.copy(wrappedNarrow = emptyList()) }
    nullableArrayReject { value.copy(values = listOf("a")) }
    nullableArrayReject { value.copy(nested = listOf(listOf(null, null))) }
    nullableArrayReject { RowsDtoInner("") }
    val shared = ACarrierDto.serializer().descriptor
    check(shared.getElementDescriptor(shared.getElementIndex("inherited")).isNullable)
    check(shared.isElementOptional(shared.getElementIndex("inherited")))
    check(!shared.getElementDescriptor(shared.getElementIndex("narrow")).isNullable)
    check(!shared.isElementOptional(shared.getElementIndex("narrow")))
    val request = SubmitInputDto.serializer().descriptor
    check(request.getElementDescriptor(request.getElementIndex("requiredNull")).isNullable)
    check(!request.isElementOptional(request.getElementIndex("requiredNull")))
    check(!request.getElementDescriptor(request.getElementIndex("requiredNarrow")).isNullable)
    check(!request.isElementOptional(request.getElementIndex("requiredNarrow")))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val streamCodec = Json(codec) { ignoreUnknownKeys = false }
        check(codec.decodeFromString<RowsDto>("null") == null)
        check(codec.decodeFromString<StatesDto>("null") == null)
        check(codec.decodeFromString<ZValuesDto>("null") == null)
        val standalone: ZValuesDto = codec.decodeFromString("[\"a\"]")
        check(standalone == listOf("a")) // Native aliases have no standalone validation barrier.
        nullableArrayReject { value.copy(values = standalone!!) }
        check(codec.decodeFromString<RowsDto>(codec.encodeToString<RowsDto>(rows)) == rows)
        check(codec.decodeFromString<StatesDto>(codec.encodeToString<StatesDto>(states)) == states)
        val text = codec.encodeToString(ACarrierDto.serializer(), value)
        check(codec.decodeFromString(ACarrierDto.serializer(), text) == value)
        check(codec.decodeFromJsonElement(ACarrierDto.serializer(), codec.parseToJsonElement(text)) == value)
        nullableArrayReject { codec.decodeFromString(ACarrierDto.serializer(), text.replace("\"ok\"", "\"a\"")) }
        nullableArrayWireReject("null") {
            codec.decodeFromString(ACarrierDto.serializer(), text.replace("\"values\":[null,\"ok\"]", "\"values\":null"))
        }
        val rowsWire = codec.encodeToString(rows)
        val statesWire = codec.encodeToString(states)
        var mode = 0
        var calls = 0
        var expectedBody = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { httpRequest ->
                calls++
                when (httpRequest.url.encodedPath) {
                    "/rows", "/narrow" -> {
                        check(nullableArrayBody(httpRequest.body) == expectedBody)
                        val response = when (mode) {
                            1 -> "null"
                            2 -> "[]"
                            3 -> rowsWire.replace("\"parent\"", "\"\"")
                            else -> expectedBody
                        }
                        respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/states" -> {
                        check(nullableArrayBody(httpRequest.body) == statesWire)
                        respond(statesWire, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(nullableArrayBody(httpRequest.body) == expectedBody)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val frame = when (current) {
                                    1 -> "event: narrow\r\ndata: null\r\n\r\n"
                                    2 -> "event: rows\r\ndata: []\r\n\r\n"
                                    3 -> "event: narrow\r\ndata: ${rowsWire.replace("\"parent\"", "\"\"")}\r\n\r\n"
                                    4 -> "event: narrow\r\ndata: [{\"id\":\"valid\",\"extra\":true}]\r\n\r\n"
                                    else -> "event: rows\r\nid: nullable-id\r\ndata: null\r\n\r\nevent: narrow\r\ndata: $rowsWire\r\n\r\n"
                                }
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected nullable array route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = NullableArrayApiClient(http, "http://example.test/", streamCodec).default
            val before = calls
            var callback = false
            nullableArrayWireReject("minItems") { api.echoNarrow(emptyList()) { callback = true } }
            check(!callback && calls == before)
            expectedBody = "null"
            check(api.echoRows(null) == null)
            expectedBody = rowsWire
            check(api.echoNarrow(rows) == rows)
            check(api.echoRows(rows) == rows)
            check(api.echoStates(states) == states)
            for ((bad, message) in listOf(1 to "null", 2 to "minItems", 3 to "minLength")) {
                mode = bad
                nullableArrayWireReject(message) { api.echoNarrow(rows) }
            }
            mode = 1
            check(api.echoRows(rows) == null)
            mode = 0
            for (state in listOf<Patch<List<String?>>>(Patch.Absent, Patch.Null, Patch.Value(values))) {
                val input = SubmitInputDto(optional = state, wrapped = state, requiredNull = null, requiredNarrow = rows)
                val encoded = codec.encodeToJsonElement(SubmitInputDto.serializer(), input) as JsonObject
                for (name in listOf("optional", "wrapped")) {
                    check(when (state) {
                        Patch.Absent -> name !in encoded
                        Patch.Null -> encoded[name] == JsonNull
                        is Patch.Value -> encoded[name] == codec.encodeToJsonElement(state.value)
                    })
                }
                check(encoded.containsKey("requiredNull") == nulls)
                check(codec.decodeFromJsonElement(SubmitInputDto.serializer(), encoded) == input)
                expectedBody = codec.encodeToString(SubmitInputDto.serializer(), input)
                api.submit(input)
                nullableArrayReject { input.copy(optional = Patch.Value(listOf("a"))) }
            }
            val beforeStream = calls
            val cold = api.watch()
            check(calls == beforeStream)
            repeat(2) {
                val events = withTimeout(10_000) { cold.toList() }
                check((events[0] as WatchEventDto.Rows).data == null)
                check((events[1] as WatchEventDto.Narrow).data == rows)
                check(events.all { it.id == "nullable-id" })
            }
            check(calls == beforeStream + 2)
            for ((bad, message) in listOf(1 to "null", 2 to "minItems", 3 to "minLength", 4 to "extra")) {
                mode = bad
                nullableArrayWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Nullable array failure left live stream open" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 2)
            expectedBody = rowsWire
            check(api.echoNarrow(rows) == rows)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Nullable array contracts passed: defaults=$defaults/nulls=$nulls; root/item/null-view/descriptor/ctor-copy-serde/Patch/HTTP/live SSE close-reuse-cold")
    }
}
