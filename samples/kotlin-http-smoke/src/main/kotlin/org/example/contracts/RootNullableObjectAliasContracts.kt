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
import org.example.rootnullableobjectaliases.RootNullableObjectAliasApiClient
import org.example.rootnullableobjectaliases.api.models.SubmitInputDto
import org.example.rootnullableobjectaliases.api.models.WatchEventDto
import org.example.rootnullableobjectaliases.models.AInheritedDto
import org.example.rootnullableobjectaliases.models.AFalseDto
import org.example.rootnullableobjectaliases.models.ATrueDto
import org.example.rootnullableobjectaliases.models.HolderDto
import org.example.rootnullableobjectaliases.models.Patch
import org.example.rootnullableobjectaliases.models.ZRecordDto
import org.example.rootnullableobjectprobes.models.ANearDto as NullableNearDto
import org.example.rootnullableobjectprobes.models.ANarrowDto as NarrowNearDto
import org.example.rootnullableobjectprobes.models.AWideDto as NullableWideDto

private fun nullableObjectBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableObjectBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable object body $body")
}
private suspend fun nullableObjectReject(block: suspend () -> Unit) {
    try { block(); error("Invalid nullable DTO accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException || it is IllegalArgumentException }) { "Wrong nullable DTO failure $cause" } }
}
suspend fun rootNullableObjectAliasContracts() {
    val made = AInheritedDto("ok")
    check(made == ZRecordDto("ok"))
    check(made.copy(id = "next") == AFalseDto("next"))
    check(!AInheritedDto.serializer().descriptor.isNullable)
    check(serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable)
    check(serializer<ATrueDto>().descriptor.isNullable)
    nullableObjectReject { AInheritedDto("") }
    nullableObjectReject { made.copy(id = "") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val nullableSerializer = serializer<AInheritedDto>()
        check(codec.decodeFromString(nullableSerializer, "null") == null)
        check(codec.encodeToString(nullableSerializer, null) == "null")
        check(codec.decodeFromString(ZRecordDto.serializer().nullable, "null") == null)
        val record = AFalseDto("ok", child = AInheritedDto("child"))
        check(codec.decodeFromString(nullableSerializer, codec.encodeToString(nullableSerializer, record)) == record)
        nullableObjectReject { codec.decodeFromString(nullableSerializer, "{\"id\":\"\"}") }
        val holder = HolderDto(null, record, null, null, record, listOf(null, record), mapOf("nil" to null, "value" to record))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<NullableNearDto>("{\"v\":null}").v == null)
        val near = codec.decodeFromString<NarrowNearDto>("{\"v\":{\"id\":\"ok\"}}")
        check(near.v.id == "ok")
        nullableObjectReject { codec.decodeFromString<NarrowNearDto>("{\"v\":null}") }
        val wideRaw = JsonObject((0 until 240).associate { "p$it" to if (it % 2 == 0) JsonNull else codec.parseToJsonElement("{\"id\":\"ok\"}") })
        val wide = codec.decodeFromJsonElement(NullableWideDto.serializer(), wideRaw)
        check(wide.p0 == null && wide.p239.id == "ok")
        val wideEncoded = codec.encodeToJsonElement(wide) as JsonObject
        check(wideEncoded["p239"] is JsonObject)
        check(if (nulls && defaults) wideEncoded["p0"] === JsonNull else "p0" !in wideEncoded)
        val absent = SubmitInputDto(null, record)
        val absentJson = codec.encodeToJsonElement(absent) as JsonObject
        check("inherited" !in absentJson && "forcedTrue" !in absentJson && "wrapper" !in absentJson)
        check(if (nulls) absentJson["requiredNull"] === JsonNull else "requiredNull" !in absentJson)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(absent)) == absent)
        if (nulls) nullableObjectReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":{\"id\":\"ok\"}}") }
        val clear = absent.copy(inherited = Patch.Null, forcedTrue = Patch.Null, wrapper = Patch.Null)
        val clearJson = codec.encodeToJsonElement(clear) as JsonObject
        check(clearJson["inherited"] === JsonNull && clearJson["wrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(clear)) == clear)
        val value = absent.copy(inherited = Patch.Value(record), forcedTrue = Patch.Value(record), wrapper = Patch.Value(record))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)) == value)
        nullableObjectReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":{\"id\":\"ok\"},\"inherited\":{\"id\":\"\"}}") }
        var calls = 0
        var expected: JsonElement = JsonNull
        var failure = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/optional" -> { check(codec.parseToJsonElement(nullableObjectBody(request.body)) == expected); respond(if (failure == "dto") "{\"id\":\"\"}" else if (failure == "syntax") "{" else expected.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                    "/patch" -> { check(codec.parseToJsonElement(nullableObjectBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true); channels += channel
                        val broken = failure
                        producers.launch { try {
                            val frame = when (broken) {
                                "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                                "dto" -> "event: nullable\r\ndata: {\"id\":\"\"}\r\n\r\n"
                                "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                                else -> "event: nullable\r\nid: alias-id\r\ndata: null\r\n\r\nevent: nullable\r\ndata: {\"id\":\"ok\"}\r\n\r\nevent: narrow\r\ndata: {\"id\":\"ok\"}\r\n\r\n"
                            }
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (broken.isNotEmpty()) awaitCancellation()
                        } finally { channel.close() } }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected nullable DTO route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = RootNullableObjectAliasApiClient(http, "http://example.test/", codec).default
            check(api.echo(null) == null)
            check(api.optional() == null)
            expected = codec.encodeToJsonElement(record)
            check(api.echo(record) == record && api.echo2(record) == record && api.optional(record) == record)
            for (patch in listOf(absent, clear, value)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = codec.encodeToJsonElement(record)
            for (bad in listOf("dto", "syntax")) { failure = bad; nullableObjectReject { api.echo(record) } }
            failure = ""
            val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 3 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[1] as WatchEventDto.Nullable).data == AFalseDto("ok"))
            check((events[2] as WatchEventDto.Narrow).data == AFalseDto("ok"))
            for (bad in listOf("narrow", "dto", "syntax")) { failure = bad; withTimeout(3000) { nullableObjectReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            failure = ""; check(api.echo2(record) == record)
        } finally { http.close(); producers.cancel() }
        println("RootNullableObjectAlias Json defaults=$defaults nulls=$nulls checks passed")
    }
}
