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
import kotlinx.coroutines.yield
import kotlinx.serialization.SerializationException
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.builtins.ByteArraySerializer
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.rootnullablebytesaliases.RootNullableBytesAliasApiClient
import org.example.rootnullablebytesaliases.api.models.SubmitInputDto
import org.example.rootnullablebytesaliases.api.models.WatchEventDto
import org.example.rootnullablebytesaliases.models.AInheritedDto
import org.example.rootnullablebytesaliases.models.AFalseDto
import org.example.rootnullablebytesaliases.models.ATrueDto
import org.example.rootnullablebytesaliases.models.HolderDto
import org.example.rootnullablebytesaliases.models.LInheritedDto
import org.example.rootnullablebytesaliases.models.NativeMaybeDto
import org.example.rootnullablebytesaliases.models.Patch
import org.example.rootnullablebytesaliases.models.ZBytesDto
import org.example.rootnullablebytesprobes.models.ANearDto as BytesNearDto
import org.example.rootnullablebytesprobes.models.ANarrowDto as BytesNarrowDto
import org.example.rootnullablebytesprobes.models.AWideDto as BytesWideDto

private fun nullableBytesBody(body: OutgoingContent): ByteArray = when (body) {
    is OutgoingContent.ContentWrapper -> nullableBytesBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes()
    else -> error("Unexpected nullable bytes body type ${body::class}")
}
private suspend fun nullableBytesReject(block: suspend () -> Unit) {
    try { block(); error("Invalid byte JSON accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong byte JSON failure $cause" } }
}
suspend fun rootNullableBytesAliasContracts() {
    check(!ByteArraySerializer().descriptor.isNullable && serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable && serializer<ATrueDto>().descriptor.isNullable)
    check(serializer<LInheritedDto>().descriptor.isNullable && serializer<NativeMaybeDto>().descriptor.isNullable)
    val ready = byteArrayOf(-128, -1, 0, 127)
    val future = byteArrayOf(1, 2)
    val binary = byteArrayOf(4)
    val nullBytes = "null".encodeToByteArray()
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val nullable = serializer<AInheritedDto>()
        for (value in listOf(null, ready, future)) {
            val text = codec.encodeToString(nullable, value)
            check(codec.parseToJsonElement(text) == (value?.let { JsonArray(it.map { n -> JsonPrimitive(n) }) } ?: JsonNull))
            val back = codec.decodeFromString(nullable, text)
            val adapter = codec.decodeFromString(ByteArraySerializer().nullable, text)
            check(if (value == null) back == null && adapter == null else back?.contentEquals(value) == true && adapter?.contentEquals(value) == true)
        }
        for (bad in listOf("{}", "\"base64\"", "[128]", "[-129]", "[null]")) nullableBytesReject { codec.decodeFromString(nullable, bad) }
        val holder = HolderDto(null, ready, future, null, ready, listOf(null, ready, future), mapOf("nil" to null, "future" to future), null, binary, binary, ready)
        val holderTree = codec.encodeToJsonElement(holder)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))) == holderTree)
        check(codec.decodeFromString<BytesNearDto>("{\"v\":null}").v == null)
        check(codec.decodeFromString<BytesNarrowDto>("{\"v\":[1,2]}").v.contentEquals(future))
        nullableBytesReject { codec.decodeFromString<BytesNarrowDto>("{\"v\":null}") }
        val wide = codec.decodeFromJsonElement(BytesWideDto.serializer(), JsonObject((0 until 240).associate { "p$it" to if (it == 239) codec.encodeToJsonElement(ByteArraySerializer(), ready) else JsonNull }))
        check(wide.p0 == null && wide.p239?.contentEquals(ready) == true)
        check(codec.encodeToJsonElement(codec.decodeFromString<BytesWideDto>(codec.encodeToString(wide.copy(p239 = future)))) == codec.encodeToJsonElement(wide.copy(p239 = future)))
        val absent = SubmitInputDto(null, ready)
        val absentTree = codec.encodeToJsonElement(absent) as JsonObject
        check("inherited" !in absentTree && "wrapper" !in absentTree)
        check(if (nulls) absentTree["requiredNull"] === JsonNull else "requiredNull" !in absentTree)
        if (nulls) nullableBytesReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":[-128,-1,0,127]}") }
        val clear = absent.copy(inherited = Patch.Null, wrapper = Patch.Null, forcedTrue = Patch.Null, binary = Patch.Null, binaryTrue = Patch.Null)
        check((codec.encodeToJsonElement(clear) as JsonObject)["inherited"] === JsonNull)
        val value = absent.copy(inherited = Patch.Value(future), wrapper = Patch.Value(ready), forcedTrue = Patch.Value(future), binary = Patch.Value(binary), binaryTrue = Patch.Value(binary))
        check((codec.encodeToJsonElement(value) as JsonObject)["inherited"] == JsonArray(listOf(JsonPrimitive(1), JsonPrimitive(2))))
        for (patch in listOf(absent, clear, value)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(patch))) == codec.encodeToJsonElement(patch))
        for (bad in listOf("{}", "[128]", "[null]")) nullableBytesReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":[1],\"inherited\":$bad}") }
        var expectedRaw = nullBytes
        var expectedPatch = codec.encodeToJsonElement(absent)
        var rawResponse: ByteArray? = null
        var failure = ""
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/alias", "/direct", "/optional", "/binary", "/binary-direct" -> { check(nullableBytesBody(request.body).contentEquals(expectedRaw)); respond(rawResponse ?: expectedRaw, HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                "/patch" -> { check(codec.parseToJsonElement(nullableBytesBody(request.body).decodeToString()) == expectedPatch); respond("", HttpStatusCode.NoContent) }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel; val broken = failure
                    producers.launch { try {
                        val frame = when (broken) {
                            "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                            "range" -> "event: nullable\r\ndata: [128]\r\n\r\n"
                            "kind" -> "event: nullable\r\ndata: {}\r\n\r\n"
                            "element" -> "event: binary\r\ndata: [null]\r\n\r\n"
                            "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                            else -> "event: nullable\r\ndata: null\r\n\r\nevent: nullable\r\ndata: [1,2]\r\n\r\nevent: narrow\r\ndata: [-128,-1,0,127]\r\n\r\nevent: binary\r\ndata: null\r\n\r\nevent: binaryNarrow\r\ndata: [4]\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (broken.isNotEmpty()) awaitCancellation()
                    } finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected nullable bytes route")
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = RootNullableBytesAliasApiClient(http, "http://example.test/", codec).default
            check(api.echo(null)?.contentEquals(nullBytes) == true && api.optional()?.contentEquals(nullBytes) == true)
            for (bytes in listOf(ready, future, nullBytes)) { expectedRaw = bytes; check(api.echo(bytes)?.contentEquals(bytes) == true && api.echo2(bytes).contentEquals(bytes) && api.optional(bytes)?.contentEquals(bytes) == true) }
            expectedRaw = nullBytes; check(api.echoBinary(null)?.contentEquals(nullBytes) == true)
            expectedRaw = binary; check(api.echoBinary(binary)?.contentEquals(binary) == true && api.echoBinary2(binary).contentEquals(binary))
            for (patch in listOf(absent, clear, value)) { expectedPatch = codec.encodeToJsonElement(patch); api.submit(patch) }
            // Raw HTTP responses are bytes, not JSON-array/null decoding. Preserve
            // malformed JSON byte strings too, unlike SSE/model JSON payloads.
            expectedRaw = ready
            for (raw in listOf("{}", "[128]", "{")) { rawResponse = raw.encodeToByteArray(); check(api.echo(ready)?.contentEquals(rawResponse) == true) }
            rawResponse = null
            val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 5 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[1] as WatchEventDto.Nullable).data?.contentEquals(future) == true && (events[2] as WatchEventDto.Narrow).data.contentEquals(ready))
            check((events[3] as WatchEventDto.Binary).data == null && (events[4] as WatchEventDto.BinaryNarrow).data.contentEquals(binary))
            for (bad in listOf("range", "kind", "element", "narrow", "syntax")) { failure = bad; withTimeout(3000) { nullableBytesReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() } }
            failure = ""; check(api.echo2(ready).contentEquals(ready))
        } finally { http.close(); producers.cancel() }
        println("RootNullableBytesAlias Json defaults=$defaults nulls=$nulls signedarray-model-Patch-SSE/rawHTTP-nullASCII-not-KotlinNull checks passed")
    }
}
