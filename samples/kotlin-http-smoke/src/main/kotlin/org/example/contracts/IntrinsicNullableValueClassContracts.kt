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
import org.example.intrinsicnullablevalueclasses.IntrinsicNullableValueClassApiClient
import org.example.intrinsicnullablevalueclasses.api.models.SubmitInputDto
import org.example.intrinsicnullablevalueclasses.api.models.WatchEventDto
import org.example.intrinsicnullablevalueclasses.models.AInheritedDto
import org.example.intrinsicnullablevalueclasses.models.AFalseDto
import org.example.intrinsicnullablevalueclasses.models.ATrueDto
import org.example.intrinsicnullablevalueclasses.models.HolderDto
import org.example.intrinsicnullablevalueclasses.models.Patch
import org.example.intrinsicnullablevalueclasses.models.ZNameDto
import org.example.intrinsicnullablevalueclasses.models.ZCountDto
import org.example.intrinsicnullablevalueclasses.models.LInheritedDto
import org.example.intrinsicnullablevalueclasses.models.LFalseDto
import org.example.intrinsicnullablevalueclassprobes.models.AUnboxedDto as ValueClassUnboxedDto
import org.example.intrinsicnullablevalueclassprobes.models.ANearDto as ValueClassNearDto
import org.example.intrinsicnullablevalueclassprobes.models.ANarrowDto as ValueClassNarrowDto
import org.example.intrinsicnullablevalueclassprobes.models.AWideDto as ValueClassWideDto

private fun intrinsicValueClassBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> intrinsicValueClassBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable value-class body $body")
}
private suspend fun intrinsicValueClassReject(block: suspend () -> Unit) {
    try { block(); error("Invalid value-class kind accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong nullable value-class failure $cause" } }
}
private suspend fun intrinsicValueClassBoundsReject(block: suspend () -> Unit) {
    try { block(); error("Invalid wrapped value accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException }) { "Wrong bounds failure $cause" } }
}
suspend fun intrinsicNullableValueClassContracts() {
    check(!ZNameDto.serializer().descriptor.isNullable && !ZCountDto.serializer().descriptor.isNullable)
    check(serializer<ZNameDto?>().descriptor.isNullable && serializer<ZCountDto?>().descriptor.isNullable)
    check(AInheritedDto("ready").rawValue == "ready")
    check(AFalseDto("future") == ZNameDto("future"))
    check(LInheritedDto(3L).rawValue == 3L && LFalseDto(3L) == ZCountDto(3L))
    check(!LInheritedDto.serializer().descriptor.isNullable && serializer<LInheritedDto>().descriptor.isNullable)
    check(!AInheritedDto.serializer().descriptor.isNullable)
    check(serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable)
    check(serializer<ATrueDto>().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val ready = ZNameDto("ready")
        val future = ZNameDto("future")
        val count = ZCountDto(3L)
        val nullable = serializer<AInheritedDto>()
        for (value in listOf(null, count)) check(codec.decodeFromString<LInheritedDto>(codec.encodeToString<LInheritedDto>(value)) == value)
        intrinsicValueClassBoundsReject { codec.decodeFromString<LInheritedDto>("0") }
        intrinsicValueClassBoundsReject { AInheritedDto("") }
        intrinsicValueClassBoundsReject { LInheritedDto(0L) }
        intrinsicValueClassBoundsReject { codec.decodeFromString<AInheritedDto>("\"\"") }
        for (value in listOf(null, ready, future)) {
            check(codec.decodeFromString(nullable, codec.encodeToString(nullable, value)) == value)
            check(codec.decodeFromString(ZNameDto.serializer().nullable, codec.encodeToString(nullable, value)) == value)
        }
        intrinsicValueClassReject { codec.decodeFromString(nullable, "123") }
        val holder = HolderDto(null, ready, future, null, ready, listOf(null, ready, future), mapOf("nil" to null, "future" to future), null, count, count, null, ready, null, ready, null, count)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<ValueClassNearDto>("{\"v\":null}").v == null)
        check(codec.decodeFromString<ValueClassNarrowDto>("{\"v\":\"future\"}").v.rawValue == "future")
        intrinsicValueClassReject { codec.decodeFromString<ValueClassNarrowDto>("{\"v\":null}") }
        val wide = codec.decodeFromJsonElement(ValueClassWideDto.serializer(), JsonObject((0 until 240).associate { "p$it" to if (it == 239) codec.parseToJsonElement("3") else JsonNull }))
        check(wide.p0 == null && wide.p239?.rawValue == 3L)
        val unboxed = codec.decodeFromJsonElement(ValueClassUnboxedDto.serializer(), JsonObject((0 until 120).associate { "p$it" to codec.parseToJsonElement("3") }))
        check(unboxed.p0.rawValue == 3L && unboxed.p119.rawValue == 3L)
        check(codec.decodeFromString<ValueClassUnboxedDto>(codec.encodeToString(unboxed)) == unboxed)
        check(codec.decodeFromString<ValueClassWideDto>(codec.encodeToString(wide)) == wide)
        val absent = SubmitInputDto(null, ready)
        val encoded = codec.encodeToJsonElement(absent) as JsonObject
        check("inherited" !in encoded && "wrapper" !in encoded && "forcedTrue" !in encoded)
        check(if (nulls) encoded["requiredNull"] === JsonNull else "requiredNull" !in encoded)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(absent)) == absent)
        if (nulls) intrinsicValueClassReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":\"ready\"}") }
        val clear = absent.copy(inherited = Patch.Null, wrapper = Patch.Null, forcedTrue = Patch.Null, count = Patch.Null, countTrue = Patch.Null)
        val clearJson = codec.encodeToJsonElement(clear) as JsonObject
        check(clearJson["inherited"] === JsonNull && clearJson["wrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(clear)) == clear)
        val directClear = absent.copy(direct = Patch.Null, directWrapper = Patch.Null, directCount = Patch.Null)
        val directValue = absent.copy(direct = Patch.Value(future), directWrapper = Patch.Value(ready), directCount = Patch.Value(count))
        for (direct in listOf(directClear, directValue)) check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(direct)) == direct)
        intrinsicValueClassBoundsReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"ready\",\"directCount\":0}") }
        val value = absent.copy(inherited = Patch.Value(future), wrapper = Patch.Value(ready), forcedTrue = Patch.Value(future), count = Patch.Value(count), countTrue = Patch.Value(count))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)) == value)
        intrinsicValueClassReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"ready\",\"inherited\":123}") }
        intrinsicValueClassBoundsReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"ready\",\"count\":0}") }
        var expected: JsonElement = JsonNull
        var failure = ""
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/optional", "/count", "/count-direct" -> { check(codec.parseToJsonElement(intrinsicValueClassBody(request.body)) == expected); respond(if (failure == "kind") "123" else if (failure == "bounds") "\"\"" else if (failure == "countBounds") "0" else if (failure == "syntax") "{" else expected.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                    "/patch" -> { check(codec.parseToJsonElement(intrinsicValueClassBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true); channels += channel; val broken = failure
                        producers.launch { try {
                            val frame = when (broken) {
                                "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                                "bounds" -> "event: nullable\r\ndata: \"\"\r\n\r\n"
                                "countBounds" -> "event: count\r\ndata: 0\r\n\r\n"
                                "countNarrow" -> "event: countNarrow\r\ndata: null\r\n\r\n"
                                "kind" -> "event: nullable\r\ndata: 123\r\n\r\n"
                                "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                                else -> "event: nullable\r\ndata: null\r\n\r\nevent: nullable\r\ndata: \"future\"\r\n\r\nevent: narrow\r\ndata: \"ready\"\r\n\r\nevent: count\r\ndata: null\r\n\r\nevent: countNarrow\r\ndata: 3\r\n\r\n"
                            }
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (broken.isNotEmpty()) awaitCancellation()
                        } finally { channel.close() } }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected nullable value-class route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = IntrinsicNullableValueClassApiClient(http, "http://example.test/", codec).default
            check(api.echo(null) == null && api.optional() == null)
            for (wrapped in listOf(ready, future)) { expected = codec.encodeToJsonElement(ZNameDto.serializer(), wrapped); check(api.echo(wrapped) == wrapped && api.echo2(wrapped) == wrapped && api.optional(wrapped) == wrapped) }
            expected = JsonNull; check(api.echoCount(null) == null)
            expected = codec.parseToJsonElement("3"); check(api.echoCount(count) == count && api.echoCount2(count) == count)
            for (patch in listOf(absent, clear, value, directClear, directValue)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = codec.encodeToJsonElement(ZNameDto.serializer(), ready)
            for (bad in listOf("kind", "syntax")) { failure = bad; intrinsicValueClassReject { api.echo(ready) } }
            failure = "bounds"; intrinsicValueClassBoundsReject { api.echo(ready) }
            expected = codec.parseToJsonElement("3"); failure = "countBounds"; intrinsicValueClassBoundsReject { api.echoCount(count) }
            expected = codec.encodeToJsonElement(ZNameDto.serializer(), ready)
            failure = ""; val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 5 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[1] as WatchEventDto.Nullable).data == future && (events[2] as WatchEventDto.Narrow).data == ready)
            check((events[3] as WatchEventDto.Count).data == null && (events[4] as WatchEventDto.CountNarrow).data == count)
            for (bad in listOf("bounds", "countBounds")) { failure = bad; withTimeout(3000) { intrinsicValueClassBoundsReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            for (bad in listOf("narrow", "countNarrow", "kind", "syntax")) { failure = bad; withTimeout(3000) { intrinsicValueClassReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            failure = ""; check(api.echo2(ready) == ready)
        } finally { http.close(); producers.cancel() }
        println("IntrinsicNullableValueClass Json defaults=$defaults nulls=$nulls checks passed")
    }
}
