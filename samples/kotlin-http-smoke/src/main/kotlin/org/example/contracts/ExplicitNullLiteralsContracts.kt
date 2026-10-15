@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
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
import kotlinx.serialization.json.JsonPrimitive
import org.example.nullliterals.NullLiterals
import org.example.nullliterals.api.models.PatchPatchInputDto
import org.example.nullliterals.api.models.WatchEventDto
import org.example.nullliterals.models.ChoiceDto
import org.example.nullliterals.models.EnumChoiceDto
import org.example.nullliterals.models.HolderDto
import org.example.nullliterals.models.NullAliasDto
import org.example.nullliterals.models.OnlyDto
import org.example.nullliterals.models.Patch
import org.example.nullliterals.models.PinnedNullDto
import org.example.nullliterals.models.UnknownChoiceDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullLiteralsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers,
            response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers,
            response.version, body, response.callContext)
    }
}

private fun nullLiteralReject(block: () -> Unit) {
    try { block(); error("Expected null-only codec rejection") }
    catch (_: SerializationException) { }
}
private suspend fun nullLiteralWireReject(block: suspend () -> Unit) {
    try { block(); error("Expected null-only wire rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong wire failure: $cause" }
    }
}
private fun nullLiteralBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullLiteralBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected null literal body $body")
}

suspend fun explicitNullLiteralContracts() {
    val pinned: PinnedNullDto = JsonNull
    val borrowed: NullAliasDto = JsonNull
    check(pinned === borrowed)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        check(codec.decodeFromString<NullAliasDto>("null") === JsonNull)
        for (wrong in listOf("\"null\"", "false", "0", "{}", "[]")) {
            nullLiteralReject { codec.decodeFromString<NullAliasDto>(wrong) }
            nullLiteralReject { codec.decodeFromString(HolderDto.serializer(), "{\"requiredNull\":$wrong}") }
            nullLiteralReject { codec.decodeFromString(HolderDto.serializer(), "{\"requiredNull\":null,\"fixed\":$wrong}") }
            nullLiteralReject { codec.decodeFromString(HolderDto.serializer(), "{\"requiredNull\":null,\"rows\":[$wrong]}") }
            nullLiteralReject { codec.decodeFromString(HolderDto.serializer(), "{\"requiredNull\":null,\"map\":{\"a\":$wrong}}") }
        }
        val holder = HolderDto(kotlinx = "shadow", requiredNull = JsonNull, optional = JsonNull,
            rows = listOf(JsonNull), map = mapOf("a" to JsonNull), nested = HolderDto.Nested(true))
        val text = codec.encodeToString(HolderDto.serializer(), holder)
        val encoded = codec.parseToJsonElement(text) as JsonObject
        check(encoded["requiredNull"] === JsonNull && encoded["fixed"] === JsonNull && encoded["borrowedFixed"] === JsonNull)
        check((encoded["nested"] as JsonObject)["fixed"] === JsonNull)
        val decoded = codec.decodeFromString(HolderDto.serializer(), text)
        check(decoded.requiredNull === JsonNull && decoded.fixed === JsonNull && decoded.borrowedFixed === JsonNull)
        check(decoded.optional == null && decoded.rows?.single() === JsonNull && decoded.map?.get("a") === JsonNull)
        check(decoded.nested?.fixed === JsonNull && decoded.kotlinx == "shadow")
        val absent = codec.decodeFromString(HolderDto.serializer(), "{\"requiredNull\":null}")
        check(absent.optional == null && absent.fixed === JsonNull)
        if (nulls || !JsonNull.serializer().descriptor.isNullable) nullLiteralReject { codec.decodeFromString(HolderDto.serializer(), "{}") }
        else check(codec.decodeFromString(HolderDto.serializer(), "{}").requiredNull === JsonNull)
        val only = codec.encodeToString(OnlyDto.serializer(), OnlyDto())
        check(codec.parseToJsonElement(only) == codec.parseToJsonElement("{\"fixed\":null,\"kotlinx\":\"shadow\"}"))
        check(codec.decodeFromString(OnlyDto.serializer(), "{}").fixed === JsonNull)
        val singleton = codec.encodeToString(ChoiceDto.serializer(), ChoiceDto.Null)
        check(codec.parseToJsonElement(singleton) == codec.parseToJsonElement("{\"tag\":null,\"kotlinx\":\"shadow\",\"kotlin\":\"shadow\"}"))
        check(codec.decodeFromString(ChoiceDto.serializer(), singleton) === ChoiceDto.Null)
        // Existing constant-only singleton dispatch uses canonical token text.
        check(codec.decodeFromString(ChoiceDto.serializer(), "{\"tag\":\"null\"}") === ChoiceDto.Null)
        val unknown = codec.parseToJsonElement("{\"tag\":\"future\",\"extra\":[null,\"null\"]}")
        val raw = codec.decodeFromJsonElement(ChoiceDto.serializer(), unknown) as UnknownChoiceDto
        check(raw.rawElement == unknown && codec.encodeToJsonElement(ChoiceDto.serializer(), raw) == unknown)
        check(codec.decodeFromString(EnumChoiceDto.serializer(), "{\"tag\":null}") is EnumChoiceDto.Null)
        nullLiteralReject { codec.decodeFromString(EnumChoiceDto.serializer(), "{\"tag\":\"null\"}") }
        val patches = listOf(PatchPatchInputDto(JsonNull), PatchPatchInputDto(null, Patch.Null), PatchPatchInputDto(JsonNull, Patch.Value(JsonNull)))
        for (patch in patches) {
            val wire = codec.encodeToJsonElement(PatchPatchInputDto.serializer(), patch) as JsonObject
            check(wire["fixed"] === JsonNull)
            check(when (patch.note) { Patch.Absent -> !wire.containsKey("note"); Patch.Null -> wire["note"] === JsonNull; is Patch.Value -> wire["note"] === JsonNull })
            val back = codec.decodeFromJsonElement(PatchPatchInputDto.serializer(), wire)
            check(back.note === if (patch.note === Patch.Absent) Patch.Absent else Patch.Null)
        }
        nullLiteralWire(codec, holder, patches)
        println("ExplicitNullLiterals Json defaults=$defaults nulls=$nulls canonical-JsonNull-const-shadow-singleton-Patch-alias-HTTP-liveSSE passed")
    }
}

private suspend fun nullLiteralWire(codec: Json, holder: HolderDto, patches: List<PatchPatchInputDto>) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var calls = 0
    var patchWire = ""
    val engine = NullLiteralsEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/holder" -> {
                    check(nullLiteralBody(request.body) == codec.encodeToString(HolderDto.serializer(), holder))
                    respond(if (bad) "{\"requiredNull\":\"null\"}" else codec.encodeToString(HolderDto.serializer(), holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/literal" -> { check(nullLiteralBody(request.body) == "null"); respond(if (bad) "\"null\"" else "null", HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                "/inline" -> { check(nullLiteralBody(request.body) == "null"); respond("legacy Unit ignores this body", HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                "/query" -> { check(request.url.parameters["q"] == "null"); respond("", HttpStatusCode.NoContent) }
                "/patch" -> { check(nullLiteralBody(request.body) == patchWire); respond("", HttpStatusCode.NoContent) }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val invalid = bad
                    producers.launch {
                        try {
                            val frame = if (invalid) "event: vocabulary\r\ndata: \"null\"\r\n\r\n" else "event: pinned\r\nid: first\r\ndata: null\r\n\r\nevent: vocabulary\r\nid: second\r\ndata: null\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (invalid) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected null literal request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = NullLiterals(http, "http://example.test/", codec).nullLiterals
        check(api.literal(JsonNull) === JsonNull)
        check(api.inline(JsonNull) == Unit)
        api.query(JsonNull)
        check(api.send(holder).requiredNull === JsonNull)
        for (p in patches) { patchWire = codec.encodeToString(PatchPatchInputDto.serializer(), p); api.patchPatch(p) }
        var callback = false
        check(api.literal(JsonNull) { callback = true } === JsonNull && callback)
        bad = true
        nullLiteralWireReject { api.literal(JsonNull) }
        nullLiteralWireReject { api.send(holder) }
        bad = false
        val before = calls
        val cold = api.watch()
        check(calls == before)
        val first = withTimeout(10_000) { cold.toList() }
        check(first.size == 2 && (first[0] as WatchEventDto.Pinned).data === JsonNull && (first[1] as WatchEventDto.Vocabulary).data === JsonNull)
        check(first[0].id == "first" && first[1].id == "second")
        check(calls == before + 1 && withTimeout(10_000) { cold.toList().size } == 2 && calls == before + 2)
        bad = true
        nullLiteralWireReject { withTimeout(10_000) { api.watch().toList() } }
        check(channels.last().isClosedForRead)
        bad = false
        check(withTimeout(10_000) { api.watch().toList().size } == 2 && api.literal(JsonNull) === JsonNull)
    } finally { http.close(); engine.close(); producers.cancel() }
}
