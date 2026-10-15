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
import io.ktor.utils.io.ClosedByteChannelException
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.yield
import kotlinx.serialization.SerializationException
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.coverchains.RecursiveCoverReferenceChainsApiClient
import org.example.coverchains.api.models.SubmitInputDto
import org.example.coverchains.api.models.WatchEventDto
import org.example.coverchains.api.models.WatchNarrowEventDto
import org.example.coverchains.api.models.WatchTrueEventDto
import org.example.coverchains.models.AgainBridgeDto
import org.example.coverchains.models.HolderDto
import org.example.coverchains.models.JSONChoiceDto
import org.example.coverchains.models.Patch
import org.example.coverchains.models.CoverBridgeDto
import org.example.coverchains.models.CoverFalseBridgeDto
import org.example.coverchains.models.SelfBridgeDto
import org.example.coverchains.models.SelfFalseBridgeDto

// Exercise Ktor's actual fragmented parser in an independent fresh project.
@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RecursiveCoverStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun recursiveCoverBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> recursiveCoverBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic recursive cover body ${body::class}")
}
private suspend fun recursiveCoverReject(block: suspend () -> Unit) {
    try { block(); error("Malformed recursive cover JSON accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) }
}

suspend fun recursiveCoverReferenceChainsContracts() = withTimeout(20000) {
    check(serializer<CoverBridgeDto>().descriptor.isNullable)
    check(serializer<AgainBridgeDto>().descriptor.isNullable)
    check(serializer<SelfBridgeDto>().descriptor.isNullable)
    check(!serializer<CoverFalseBridgeDto>().descriptor.isNullable)
    check(!serializer<SelfFalseBridgeDto>().descriptor.isNullable)
    val future = JsonObject(mapOf("unknown" to JsonPrimitive("kept")))
    val kinds = listOf(JsonNull, JsonPrimitive("text"), JsonPrimitive(42), JsonPrimitive(true), JsonArray(listOf(JsonNull, future)), future)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        for (value in listOf(null) + kinds) {
            val inherited: CoverBridgeDto = value
            val again: AgainBridgeDto = inherited
            val any: SelfBridgeDto = again
            val expected = if (value === JsonNull) null else value
            val text = codec.encodeToString(serializer<CoverBridgeDto>(), inherited)
            check(codec.decodeFromString<CoverBridgeDto>(text) == expected)
            check(codec.decodeFromString<AgainBridgeDto>(codec.encodeToString(again)) == expected)
            check(codec.decodeFromString<SelfBridgeDto>(codec.encodeToString(any)) == expected)
            check(codec.decodeFromString(JsonElement.serializer().nullable, text) == expected)
        }
        // Cover aliases retain the caller's backing numeric policy, including
        // exponent spelling/precision changes; they add no numeric conversion.
        for (literal in listOf("1e99", "18446744073709551617")) {
            val number = codec.parseToJsonElement(literal)
            val backing = codec.encodeToString(JsonElement.serializer(), number)
            check(codec.encodeToString<CoverFalseBridgeDto>(number) == backing)
            check(codec.encodeToString<SelfFalseBridgeDto>(number) == backing)
            check(codec.decodeFromString<CoverFalseBridgeDto>(backing) == codec.decodeFromString(JsonElement.serializer(), backing))
        }
        for (value in kinds) {
            val raw: CoverFalseBridgeDto = value
            val any: SelfFalseBridgeDto = raw
            check(codec.decodeFromString<CoverFalseBridgeDto>(codec.encodeToString(raw)) == value)
            check(codec.decodeFromString<SelfFalseBridgeDto>(codec.encodeToString(any)) == value)
        }
        val holder = HolderDto(inherited = null, forcedTrue = future, forcedFalse = JsonNull, wrapper = null,
            wrapperFalse = JsonNull, items = listOf(null, future), values = JsonObject(mapOf("nil" to JsonNull, "future" to future)),
            requiredNull = null, any = future, anyFalse = JsonNull)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))) == codec.encodeToJsonElement(holder))
        recursiveCoverReject { codec.decodeFromString<HolderDto>("{}") }
        for (tag in listOf("a", "b")) for (value in kinds) {
            val input = JsonObject(mapOf("kind" to JsonPrimitive(tag), "value" to value))
            val choice = codec.decodeFromString<JSONChoiceDto>(input.toString())
            check(choice.value == value)
            check(codec.decodeFromString<JSONChoiceDto>(codec.encodeToString(choice)).value == choice.value)
            check(choice.kind == tag)
        }
        val absent = SubmitInputDto(items = emptyList(), values = JsonObject(emptyMap()), requiredNull = null)
        check("inherited" !in (codec.encodeToJsonElement(absent) as JsonObject))
        if (nulls) recursiveCoverReject { codec.decodeFromString<SubmitInputDto>("{\"items\":[],\"values\":{}}") }
        val clear = SubmitInputDto(items = emptyList(), values = JsonObject(emptyMap()), requiredNull = null, inherited = Patch.Null, wrapper = Patch.Null)
        val assigned = SubmitInputDto(items = emptyList(), values = JsonObject(emptyMap()), requiredNull = null, inherited = Patch.Value(future), wrapper = Patch.Value(JsonNull))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(assigned)).wrapper === Patch.Null)
        for (p in listOf(absent, clear, assigned)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(p))) == codec.encodeToJsonElement(p))
        var expected: JsonElement = JsonNull
        var syntax = false
        var endless = false
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = RecursiveCoverStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/watch", "/watch-narrow", "/watch-true" -> {
                    val event = when (request.url.encodedPath) { "/watch-narrow" -> "narrow"; "/watch-true" -> "any"; else -> "raw" }
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val broken = syntax; val keepOpen = endless
                    producers.launch { try {
                        val frame = if (broken) "event: $event\r\ndata: {\r\n\r\n" else kinds.joinToString("") { "event: $event\r\ndata: ${codec.encodeToString(JsonElement.serializer(), it)}\r\n\r\n" }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (broken || keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) {
                        // The client closes a live fragmented writer on cancellation.
                        check(channel.isClosedForWrite)
                    } finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                "/query" -> {
                    check(request.url.parameters["any"] == expected.toString())
                    check(request.url.parameters["value"] == if (expected === JsonNull) null else expected.toString())
                    respond("", HttpStatusCode.NoContent)
                }
                "/submit" -> { check(codec.parseToJsonElement(recursiveCoverBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
                else -> {
                    check(codec.parseToJsonElement(recursiveCoverBody(request.body)) == expected)
                    respond(codec.encodeToString(JsonElement.serializer(), expected), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = RecursiveCoverReferenceChainsApiClient(http, "https://example.test/", codec).default
            check(api.echo(null) == null && api.echoTrue(null) == null)
            for (value in kinds) {
                expected = value
                check(api.echo(value) == if (value === JsonNull) null else value)
                check(api.echo2(value) == value)
                check(api.echoTrue(value) == if (value === JsonNull) null else value)
                api.query(value = if (value === JsonNull) null else value, any = value)
            }
            for (p in listOf(absent, clear, assigned)) { expected = codec.encodeToJsonElement(p); api.submit(p) }
            val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }; check(events.size == kinds.size)
            events.forEachIndexed { i, event -> check((event as WatchEventDto.Raw).data == if (kinds[i] === JsonNull) null else kinds[i]) }
            val any = withTimeout(3000) { api.watchTrue().toList() }; check(any.size == kinds.size)
            any.forEachIndexed { i, event -> check((event as WatchTrueEventDto.Any).data == if (kinds[i] === JsonNull) null else kinds[i]) }
            val narrow = withTimeout(3000) { api.watchNarrow().toList() }; check((narrow.first() as WatchNarrowEventDto.Narrow).data === JsonNull)
            syntax = true
            for (stream in listOf(0, 1, 2)) {
                withTimeout(3000) { recursiveCoverReject { when (stream) { 0 -> flow.toList(); 1 -> api.watchNarrow().toList(); else -> api.watchTrue().toList() } } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            syntax = false; endless = true
            withTimeout(3000) { flow.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false; check(withTimeout(3000) { flow.toList() }.size == kinds.size)
            expected = future; check(api.echo2(future) == future && api.echoTrue(future) == future)
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("RecursiveCoverReferenceChains Json defaults=$defaults nulls=$nulls graph-SCC-allkinds-JsonNull-collapse-Patch-union-HTTP-query-SSEclose-cancel-reuse-cold passed")
    }
}
