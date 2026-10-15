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
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.nullableallof.NullableAllOfDTOApiClient
import org.example.nullableallof.api.models.SubmitInputDto
import org.example.nullableallof.api.models.WatchEventDto
import org.example.nullableallof.models.AInheritedDto
import org.example.nullableallof.models.AFalseDto
import org.example.nullableallof.models.ATrueDto
import org.example.nullableallof.models.HolderDto
import org.example.nullableallof.models.Patch
import org.example.nullableallof.models.ZRecordDto
import org.example.nullableallof.models.ZOtherDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullableAllOfStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun nullableAllOfBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableAllOfBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic nullable allOf body")
}
private suspend fun nullableAllOfReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing nullable allOf rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is SerializationException || it is IllegalArgumentException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong nullable allOf failure: $cause" }
    }
}

suspend fun nullableAllOfDTOContracts(checked: Boolean = false) = withTimeout(30000) {
    val physical: ZRecordDto = AInheritedDto("ok")
    check(!ZRecordDto.serializer().descriptor.isNullable)
    check(serializer<ZRecordDto?>().descriptor.isNullable)
    check(!AInheritedDto.serializer().descriptor.isNullable)
    check(serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable)
    check(serializer<ATrueDto>().descriptor.isNullable)
    if (checked) nullableAllOfReject("minLength") { AInheritedDto("") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val nullable = serializer<AInheritedDto>()
        check(codec.decodeFromString(nullable, "null") == null)
        check(codec.encodeToString(nullable, null) == "null")
        check(codec.decodeFromString(ZRecordDto.serializer().nullable, "null") == null)
        nullableAllOfReject("null") { codec.decodeFromString(ZRecordDto.serializer(), "null") }
        nullableAllOfReject("null") { codec.decodeFromString<AFalseDto>("null") }
        val record = AFalseDto("ok", child = physical)
        val wire = codec.encodeToJsonElement(record)
        check(codec.encodeToJsonElement(codec.decodeFromString(nullable, codec.encodeToString(nullable, record))) == wire)
        check(codec.decodeFromString<ZOtherDto?>("null") == null)
        val other = ZOtherDto("other")
        check(other.stamp == "shared")
        check((codec.encodeToJsonElement(other) as JsonObject).getValue("stamp") == codec.parseToJsonElement("\"shared\""))
        val holder = HolderDto(null, record, null, null, record, listOf(null, record), mapOf("nil" to null, "value" to record), null, record, null, record, null, other)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))) == codec.encodeToJsonElement(holder))
        nullableAllOfReject("id") { codec.decodeFromString(nullable, "{}") }
        if (checked) nullableAllOfReject("minLength") { codec.decodeFromString(nullable, "{\"id\":\"\"}") }
        val absent = SubmitInputDto(null, record)
        val absentWire = codec.encodeToJsonElement(absent) as JsonObject
        check("inherited" !in absentWire && "wrapper" !in absentWire && "direct" !in absentWire)
        check(if (nulls) absentWire["requiredNull"] == JsonNull else "requiredNull" !in absentWire)
        if (nulls) nullableAllOfReject("requiredNull") { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":{\"id\":\"ok\"}}") }
        val clear = SubmitInputDto(null, record, inherited = Patch.Null, forcedTrue = Patch.Null, wrapper = Patch.Null, direct = Patch.Null, directWrapper = Patch.Null)
        val assigned = SubmitInputDto(null, record, inherited = Patch.Value(record), forcedTrue = Patch.Value(record), wrapper = Patch.Value(record), direct = Patch.Value(record), directWrapper = Patch.Value(record))
        for (input in listOf(absent, clear, assigned)) {
            check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) == codec.encodeToJsonElement(input))
        }
        check((codec.encodeToJsonElement(clear) as JsonObject).getValue("direct") == JsonNull)
        if (checked) nullableAllOfReject("minLength") { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":{\"id\":\"ok\"},\"inherited\":{\"id\":\"\"}}") }
        var calls = 0
        var expected: JsonElement = JsonNull
        var failure = ""
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = NullableAllOfStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/alias", "/direct", "/optional" -> {
                    check(codec.parseToJsonElement(nullableAllOfBody(request.body)) == expected)
                    val response = when (failure) { "dto" -> "{\"id\":\"\"}"; "syntax" -> "{"; "narrow" -> "null"; else -> expected.toString() }
                    respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(nullableAllOfBody(request.body)) == expected)
                    respond("", HttpStatusCode.NoContent)
                }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val mode = failure; val keepOpen = endless || mode.isNotEmpty()
                    producers.launch { try {
                        val frame = when (mode) {
                            "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                            "dto" -> "event: nullable\r\ndata: {\"id\":\"\"}\r\n\r\n"
                            "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                            else -> "event: nullable\r\nid: alias-id\r\ndata: null\r\n\r\nevent: nullable\r\ndata: {\"id\":\"ok\"}\r\n\r\nevent: narrow\r\ndata: {\"id\":\"ok\"}\r\n\r\nevent: direct\r\ndata: null\r\n\r\nevent: direct\r\ndata: {\"id\":\"ok\"}\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected nullable allOf synthetic route")
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = NullableAllOfDTOApiClient(http, "https://example.test/", codec).default
            check(api.echo(null) == null && api.optional() == null)
            expected = wire
            check(codec.encodeToJsonElement(api.echo(record)) == wire)
            check(codec.encodeToJsonElement(api.echo2(record)) == wire)
            check(codec.encodeToJsonElement(api.optional(record)) == wire)
            for (patch in listOf(absent, clear, assigned)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = wire
            failure = "syntax"; nullableAllOfReject("JSON") { api.echo(record) }
            if (checked) { failure = "dto"; nullableAllOfReject("minLength") { api.echo(record) } }
            failure = "narrow"; nullableAllOfReject("null") { api.echo2(record) }
            failure = ""
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val events = withTimeout(3000) { cold.toList() }
                check(events.size == 5 && (events[0] as WatchEventDto.Nullable).data == null)
                check((events[3] as WatchEventDto.Direct).data == null)
                check((events[1] as WatchEventDto.Nullable).data!!.id == "ok")
                check((events[2] as WatchEventDto.Narrow).data.id == "ok")
                check((events[4] as WatchEventDto.Direct).data!!.id == "ok")
            }
            for ((bad, message) in listOf("narrow" to "null", "syntax" to "JSON") + if (checked) listOf("dto" to "minLength") else emptyList()) {
                failure = bad
                withTimeout(3000) { nullableAllOfReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            failure = ""; endless = true
            withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false
            check(withTimeout(3000) { cold.toList() }.size == 5)
            check(codec.encodeToJsonElement(api.echo2(record)) == wire)
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("NullableAllOfDTO Json defaults=$defaults nulls=$nulls original-allOf-constructor-canonicalCodec-nullViews-Patch-HTTP-SSEclose-cancel-cold-reuse passed")
    }
}
