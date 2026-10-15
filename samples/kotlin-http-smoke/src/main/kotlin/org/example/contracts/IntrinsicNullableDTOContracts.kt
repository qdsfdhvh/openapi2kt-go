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
import org.example.intrinsicnullabledtos.IntrinsicNullableDTOApiClient
import org.example.intrinsicnullabledtos.api.models.SubmitInputDto
import org.example.intrinsicnullabledtos.api.models.WatchEventDto
import org.example.intrinsicnullabledtos.models.AInheritedDto
import org.example.intrinsicnullabledtos.models.AFalseDto
import org.example.intrinsicnullabledtos.models.ATrueDto
import org.example.intrinsicnullabledtos.models.HolderDto
import org.example.intrinsicnullabledtos.models.Patch
import org.example.intrinsicnullabledtos.models.ZRecordDto
import org.example.intrinsicnullabledtos.models.ZOtherDto
import org.example.intrinsicnullableprobes.models.ANearDto as NullableNearDto
import org.example.intrinsicnullableprobes.models.ANarrowDto as NarrowNearDto
import org.example.intrinsicnullableprobes.models.AWideDto as NullableWideDto
import org.example.intrinsicnullableprobes.models.ADeepCheckedDto as IntrinsicDeepCheckedDto

private fun intrinsicNullableBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> intrinsicNullableBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable object body $body")
}
private suspend fun intrinsicNullableReject(block: suspend () -> Unit) {
    try { block(); error("Invalid nullable DTO accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException || it is IllegalArgumentException }) { "Wrong nullable DTO failure $cause" } }
}
suspend fun intrinsicNullableDTOContracts() {
    val physical: ZRecordDto = ZRecordDto("ok")
    check(!ZRecordDto.serializer().descriptor.isNullable)
    check(serializer<ZRecordDto?>().descriptor.isNullable)
    val made = AInheritedDto("ok")
    check(made == physical)
    check(made == ZRecordDto("ok"))
    check(made.copy(id = "next") == AFalseDto("next"))
    check(!AInheritedDto.serializer().descriptor.isNullable)
    check(serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable)
    check(serializer<ATrueDto>().descriptor.isNullable)
    intrinsicNullableReject { AInheritedDto("") }
    intrinsicNullableReject { made.copy(id = "") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val nullableSerializer = serializer<AInheritedDto>()
        check(codec.decodeFromString(nullableSerializer, "null") == null)
        check(codec.encodeToString(nullableSerializer, null) == "null")
        check(codec.decodeFromString(ZRecordDto.serializer().nullable, "null") == null)
        check(codec.decodeFromString<ZOtherDto?>("null") == null)
        check(codec.decodeFromString<ZOtherDto>(codec.encodeToString(ZOtherDto("other"))) == ZOtherDto("other"))
        val record = AFalseDto("ok", child = AInheritedDto("child"))
        check(codec.decodeFromString(nullableSerializer, codec.encodeToString(nullableSerializer, record)) == record)
        intrinsicNullableReject { codec.decodeFromString(nullableSerializer, "{\"id\":\"\"}") }
        val holder = HolderDto(null, record, null, null, record, listOf(null, record), mapOf("nil" to null, "value" to record), null, record, null, record, null, ZOtherDto("ok"))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<NullableNearDto>("{\"v\":null}").v == null)
        var checked: JsonElement = codec.parseToJsonElement("\"ok\"")
        var invalidChecked: JsonElement = codec.parseToJsonElement("\"\"")
        repeat(16) { checked = kotlinx.serialization.json.JsonArray(listOf(checked)); invalidChecked = kotlinx.serialization.json.JsonArray(listOf(invalidChecked)) }
        val deep = codec.decodeFromJsonElement(IntrinsicDeepCheckedDto.serializer(), JsonObject(mapOf("checked" to checked)))
        check(codec.decodeFromString<IntrinsicDeepCheckedDto>(codec.encodeToString(deep)) == deep)
        check(codec.decodeFromString<IntrinsicDeepCheckedDto?>("null") == null)
        intrinsicNullableReject { codec.decodeFromJsonElement(IntrinsicDeepCheckedDto.serializer(), JsonObject(mapOf("checked" to invalidChecked))) }
        val near = codec.decodeFromString<NarrowNearDto>("{\"v\":{\"id\":\"ok\"}}")
        check(near.v.id == "ok")
        intrinsicNullableReject { codec.decodeFromString<NarrowNearDto>("{\"v\":null}") }
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
        if (nulls) intrinsicNullableReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":{\"id\":\"ok\"}}") }
        val clear = absent.copy(inherited = Patch.Null, forcedTrue = Patch.Null, wrapper = Patch.Null)
        val clearJson = codec.encodeToJsonElement(clear) as JsonObject
        check(clearJson["inherited"] === JsonNull && clearJson["wrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(clear)) == clear)
        check("direct" !in absentJson && "directWrapper" !in absentJson)
        val directClear = absent.copy(direct = Patch.Null, directWrapper = Patch.Null)
        val directClearJson = codec.encodeToJsonElement(directClear) as JsonObject
        check(directClearJson["direct"] === JsonNull && directClearJson["directWrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(directClear)) == directClear)
        val value = absent.copy(inherited = Patch.Value(record), forcedTrue = Patch.Value(record), wrapper = Patch.Value(record))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)) == value)
        intrinsicNullableReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":{\"id\":\"ok\"},\"inherited\":{\"id\":\"\"}}") }
        var calls = 0
        var expected: JsonElement = JsonNull
        var failure = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/optional" -> { check(codec.parseToJsonElement(intrinsicNullableBody(request.body)) == expected); respond(if (failure == "dto") "{\"id\":\"\"}" else if (failure == "syntax") "{" else expected.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                    "/patch" -> { check(codec.parseToJsonElement(intrinsicNullableBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true); channels += channel
                        val broken = failure
                        producers.launch { try {
                            val frame = when (broken) {
                                "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                                "dto" -> "event: nullable\r\ndata: {\"id\":\"\"}\r\n\r\n"
                                "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                                else -> "event: nullable\r\nid: alias-id\r\ndata: null\r\n\r\nevent: nullable\r\ndata: {\"id\":\"ok\"}\r\n\r\nevent: narrow\r\ndata: {\"id\":\"ok\"}\r\n\r\nevent: direct\r\ndata: null\r\n\r\nevent: direct\r\ndata: {\"id\":\"ok\"}\r\n\r\n"
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
            val api = IntrinsicNullableDTOApiClient(http, "http://example.test/", codec).default
            check(api.echo(null) == null)
            check(api.optional() == null)
            expected = codec.encodeToJsonElement(record)
            check(api.echo(record) == record && api.echo2(record) == record && api.optional(record) == record)
            val directValue = absent.copy(direct = Patch.Value(record), directWrapper = Patch.Value(record))
            val directNull = absent.copy(direct = Patch.Null, directWrapper = Patch.Null)
            for (patch in listOf(absent, clear, value, directValue, directNull)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = codec.encodeToJsonElement(record)
            for (bad in listOf("dto", "syntax")) { failure = bad; intrinsicNullableReject { api.echo(record) } }
            failure = ""
            val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 5 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[3] as WatchEventDto.Direct).data == null)
            check((events[4] as WatchEventDto.Direct).data == AFalseDto("ok"))
            check((events[1] as WatchEventDto.Nullable).data == AFalseDto("ok"))
            check((events[2] as WatchEventDto.Narrow).data == AFalseDto("ok"))
            for (bad in listOf("narrow", "dto", "syntax")) { failure = bad; withTimeout(3000) { intrinsicNullableReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            failure = ""; check(api.echo2(record) == record)
        } finally { http.close(); producers.cancel() }
        println("IntrinsicNullableDTO Json defaults=$defaults nulls=$nulls checks passed")
    }
}
