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
import org.example.elementchecks.ApiClient
import org.example.elementchecks.api.models.PatchInputDto
import org.example.elementchecks.api.models.WatchEventDto
import org.example.elementchecks.models.BagDto
import org.example.elementchecks.models.Patch
import org.example.elementchecks.models.RecordDto

private suspend fun elementFailure(block: suspend () -> Unit) {
    try { block(); error("Expected element validation failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            it is IllegalArgumentException && it.message?.startsWith("Field value ") == true
        }) { "Element validation cause lost: $cause" }
    }
}

private fun elementBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> elementBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected element content $body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun elementConstraintContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // Collection aliases do not create an independently checking serializer.
        val unchecked = codec.decodeFromString<BagDto>("{\"raw\":[\"\"]}")
        check(unchecked["raw"] == listOf(""))
        val bag: BagDto = mapOf("x" to listOf("a", null), "nil" to null, "empty" to emptyList())
        val record = RecordDto(
            rows = listOf(mapOf("arbitrary key 🌿" to listOf("a", null, "bc"))),
            counts = mapOf("x" to listOf(-2L, 2L)), bag = bag,
            ratios = listOf(-1f, null, 1f), folds = listOf("🌿🌿", null, "e\u0301"),
            children = listOf(RecordDto.ChildrenInner(listOf("🌿🌿"))),
        )
        val encoded = codec.encodeToString(RecordDto.serializer(), record)
        check(codec.decodeFromString(RecordDto.serializer(), encoded) == record)
        check(record.copy(rows = emptyList(), counts = emptyMap(), bag = emptyMap(), ratios = null, folds = emptyList()).rows.isEmpty())
        check(record.copy(ratios = listOf(null)).ratios == listOf(null))
        elementFailure { record.copy(rows = listOf(mapOf("x" to listOf("")))) }
        elementFailure { record.copy(counts = mapOf("x" to listOf(Long.MAX_VALUE))) }
        elementFailure { record.copy(bag = mapOf("x" to listOf("BAD"))) }
        elementFailure { record.copy(ratios = listOf(Float.NaN)) }
        elementFailure { record.copy(ratios = listOf(Float.NEGATIVE_INFINITY)) }
        elementFailure { record.copy(folds = listOf("🌿🌿🌿")) }
        elementFailure { RecordDto.ChildrenInner(listOf("")) }
        elementFailure { codec.decodeFromString(RecordDto.serializer(), encoded.replace("bc", "BAD")) }
        val patches = listOf(PatchInputDto(), PatchInputDto(Patch.Null), PatchInputDto(Patch.Value(listOf(listOf("a", null), emptyList()))))
        for (p in patches) check(codec.decodeFromString(PatchInputDto.serializer(), codec.encodeToString(PatchInputDto.serializer(), p)) == p)
        elementFailure { PatchInputDto(Patch.Value(listOf(listOf("")))) }
        elementFailure { codec.decodeFromString(PatchInputDto.serializer(), "{\"rows\":[[\"BAD\"]]}") }
        check(WatchEventDto.Message(null).data == null)
        elementFailure { WatchEventDto.Message(listOf(mapOf("x" to listOf("BAD")))) }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var bad = false
        var nullEvent = false
        var calls = 0
        val rowsPayload = codec.parseToJsonElement(encoded).jsonObject.getValue("rows").toString()
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/record" -> {
                        check(codec.decodeFromString(RecordDto.serializer(), elementBody(request.body)) == record)
                        respond(if (bad) encoded.replace("bc", "BAD") else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(codec.decodeFromString(PatchInputDto.serializer(), elementBody(request.body)) in patches)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/params" -> {
                        check(request.url.parameters.getAll("ids") == listOf("a", "bc"))
                        check(request.url.parameters["x"] == "2")
                        check(request.headers["idsHeader"] == "a,bc")
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/values" -> respond(if (bad) "[{\"x\":3}]" else elementBody(request.body), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val invalid = bad
                        val payload = if (invalid) "[{\"x\":[\"BAD\"]}]" else if (nullEvent) "null" else rowsPayload
                        producers.launch {
                            try {
                                for (byte in "data: $payload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (invalid) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected element request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http, "http://example.test/", codec).elements
            val before = calls
            elementFailure { api.params(ids = listOf("")) { error("Invalid element reached callback") } }
            elementFailure { api.params(counts = mapOf("x" to Long.MIN_VALUE)) }
            elementFailure { api.params(idsHeader = listOf("BAD")) }
            elementFailure { api.values(listOf(mapOf("x" to 3L))) }
            check(calls == before) { "Invalid element sent a network request" }
            check(api.record(record) == record)
            for (p in patches) api.patchPatch(p)
            api.params(ids = listOf("a", null, "bc"), counts = mapOf("x" to 2L), idsHeader = listOf("a", "bc"))
            val values = listOf(mapOf("x" to 2L, "nil" to null))
            check(api.values(values) == values)
            check(api.values(emptyList()).isEmpty())
            bad = true
            elementFailure { api.record(record) }
            elementFailure { api.values(values) }
            elementFailure { api.watch().toList() }
            check(channels.last().isClosedForRead) { "Invalid element SSE did not close live input" }
            bad = false
            check((api.watch().toList().single() as WatchEventDto.Message).data == record.rows)
            nullEvent = true
            check((api.watch().toList().single() as WatchEventDto.Message).data == null)
            check(api.record(record) == record)
            check(calls == 13)
            println("Element contracts passed: defaults=$defaults/nulls=$nulls; nestedDTO/list-map/nullable/empty/fold/Patch/ctor-copy-serde/preflight-postflight/liveSSE-close/recovery; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
