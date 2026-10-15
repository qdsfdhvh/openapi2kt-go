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
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.rootnullableenumaliases.RootNullableEnumAliasApiClient
import org.example.rootnullableenumaliases.api.models.SubmitInputDto
import org.example.rootnullableenumaliases.api.models.WatchEventDto
import org.example.rootnullableenumaliases.models.AInheritedDto
import org.example.rootnullableenumaliases.models.AFalseDto
import org.example.rootnullableenumaliases.models.ATrueDto
import org.example.rootnullableenumaliases.models.HolderDto
import org.example.rootnullableenumaliases.models.Patch
import org.example.rootnullableenumaliases.models.ZStatusDto
import org.example.rootnullableenumprobes.models.ANearDto as EnumNearDto
import org.example.rootnullableenumprobes.models.ANarrowDto as EnumNarrowDto
import org.example.rootnullableenumprobes.models.AWideDto as EnumWideDto

private fun nullableEnumBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableEnumBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable enum body $body")
}
private suspend fun nullableEnumReject(block: suspend () -> Unit) {
    try { block(); error("Invalid enum kind accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong nullable enum failure $cause" } }
}
suspend fun rootNullableEnumAliasContracts() {
    check(AInheritedDto.entries == ZStatusDto.entries)
    check(AInheritedDto.fromSerialName("ready") == ZStatusDto.READY)
    check(AFalseDto.fromSerialName("future") == ZStatusDto.Unrecognized("future"))
    check(!AInheritedDto.serializer().descriptor.isNullable)
    check(serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable)
    check(serializer<ATrueDto>().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val ready = ZStatusDto.READY
        val future = ZStatusDto.Unrecognized("future")
        val nullable = serializer<AInheritedDto>()
        for (value in listOf(null, ready, future)) {
            check(codec.decodeFromString(nullable, codec.encodeToString(nullable, value)) == value)
            check(codec.decodeFromString(ZStatusDto.serializer().nullable, codec.encodeToString(nullable, value)) == value)
        }
        nullableEnumReject { codec.decodeFromString(nullable, "123") }
        val holder = HolderDto(null, ready, future, null, ready, listOf(null, ready, future), mapOf("nil" to null, "future" to future))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<EnumNearDto>("{\"v\":null}").v == null)
        check(codec.decodeFromString<EnumNarrowDto>("{\"v\":\"future\"}").v.serialName == "future")
        nullableEnumReject { codec.decodeFromString<EnumNarrowDto>("{\"v\":null}") }
        val wide = codec.decodeFromJsonElement(EnumWideDto.serializer(), JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
        check(codec.decodeFromString<EnumWideDto>(codec.encodeToString(wide)) == wide)
        val absent = SubmitInputDto(null, ready)
        val encoded = codec.encodeToJsonElement(absent) as JsonObject
        check("inherited" !in encoded && "wrapper" !in encoded && "forcedTrue" !in encoded)
        check(if (nulls) encoded["requiredNull"] === JsonNull else "requiredNull" !in encoded)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(absent)) == absent)
        if (nulls) nullableEnumReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":\"ready\"}") }
        val clear = absent.copy(inherited = Patch.Null, wrapper = Patch.Null, forcedTrue = Patch.Null)
        val clearJson = codec.encodeToJsonElement(clear) as JsonObject
        check(clearJson["inherited"] === JsonNull && clearJson["wrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(clear)) == clear)
        val value = absent.copy(inherited = Patch.Value(future), wrapper = Patch.Value(ready), forcedTrue = Patch.Value(future))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)) == value)
        nullableEnumReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"ready\",\"inherited\":123}") }
        var expected: JsonElement = JsonNull
        var failure = ""
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/optional" -> { check(codec.parseToJsonElement(nullableEnumBody(request.body)) == expected); respond(if (failure == "kind") "123" else if (failure == "syntax") "{" else expected.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                    "/patch" -> { check(codec.parseToJsonElement(nullableEnumBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true); channels += channel; val broken = failure
                        producers.launch { try {
                            val frame = when (broken) {
                                "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                                "kind" -> "event: nullable\r\ndata: 123\r\n\r\n"
                                "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                                else -> "event: nullable\r\ndata: null\r\n\r\nevent: nullable\r\ndata: \"future\"\r\n\r\nevent: narrow\r\ndata: \"ready\"\r\n\r\n"
                            }
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (broken.isNotEmpty()) awaitCancellation()
                        } finally { channel.close() } }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected nullable enum route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = RootNullableEnumAliasApiClient(http, "http://example.test/", codec).default
            check(api.echo(null) == null && api.optional() == null)
            for (enum in listOf(ready, future)) { expected = codec.encodeToJsonElement(ZStatusDto.serializer(), enum); check(api.echo(enum) == enum && api.echo2(enum) == enum && api.optional(enum) == enum) }
            for (patch in listOf(absent, clear, value)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = codec.encodeToJsonElement(ZStatusDto.serializer(), ready)
            for (bad in listOf("kind", "syntax")) { failure = bad; nullableEnumReject { api.echo(ready) } }
            failure = ""; val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 3 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[1] as WatchEventDto.Nullable).data == future && (events[2] as WatchEventDto.Narrow).data == ready)
            for (bad in listOf("narrow", "kind", "syntax")) { failure = bad; withTimeout(3000) { nullableEnumReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            failure = ""; check(api.echo2(ready) == ready)
        } finally { http.close(); producers.cancel() }
        println("RootNullableEnumAlias Json defaults=$defaults nulls=$nulls checks passed")
    }
}
