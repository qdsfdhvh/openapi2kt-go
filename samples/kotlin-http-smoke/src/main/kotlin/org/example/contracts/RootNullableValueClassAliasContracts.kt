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
import org.example.rootnullablevalueclassaliases.RootNullableValueClassAliasApiClient
import org.example.rootnullablevalueclassaliases.api.models.SubmitInputDto
import org.example.rootnullablevalueclassaliases.api.models.WatchEventDto
import org.example.rootnullablevalueclassaliases.models.AInheritedDto
import org.example.rootnullablevalueclassaliases.models.AFalseDto
import org.example.rootnullablevalueclassaliases.models.ATrueDto
import org.example.rootnullablevalueclassaliases.models.HolderDto
import org.example.rootnullablevalueclassaliases.models.Patch
import org.example.rootnullablevalueclassaliases.models.ZNameDto
import org.example.rootnullablevalueclassaliases.models.ZCountDto
import org.example.rootnullablevalueclassaliases.models.LInheritedDto
import org.example.rootnullablevalueclassaliases.models.LFalseDto
import org.example.rootnullablevalueclassprobes.models.AUnboxedDto as ValueClassUnboxedDto
import org.example.rootnullablevalueclassprobes.models.ANearDto as ValueClassNearDto
import org.example.rootnullablevalueclassprobes.models.ANarrowDto as ValueClassNarrowDto
import org.example.rootnullablevalueclassprobes.models.AWideDto as ValueClassWideDto

private fun nullableValueClassBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableValueClassBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable value-class body $body")
}
private suspend fun nullableValueClassReject(block: suspend () -> Unit) {
    try { block(); error("Invalid value-class kind accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong nullable value-class failure $cause" } }
}
private suspend fun nullableValueClassBoundsReject(block: suspend () -> Unit) {
    try { block(); error("Invalid wrapped value accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException }) { "Wrong bounds failure $cause" } }
}
suspend fun rootNullableValueClassAliasContracts() {
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
        nullableValueClassBoundsReject { codec.decodeFromString<LInheritedDto>("0") }
        nullableValueClassBoundsReject { AInheritedDto("") }
        nullableValueClassBoundsReject { LInheritedDto(0L) }
        nullableValueClassBoundsReject { codec.decodeFromString<AInheritedDto>("\"\"") }
        for (value in listOf(null, ready, future)) {
            check(codec.decodeFromString(nullable, codec.encodeToString(nullable, value)) == value)
            check(codec.decodeFromString(ZNameDto.serializer().nullable, codec.encodeToString(nullable, value)) == value)
        }
        nullableValueClassReject { codec.decodeFromString(nullable, "123") }
        val holder = HolderDto(null, ready, future, null, ready, listOf(null, ready, future), mapOf("nil" to null, "future" to future), null, count, count)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<ValueClassNearDto>("{\"v\":null}").v == null)
        check(codec.decodeFromString<ValueClassNarrowDto>("{\"v\":\"future\"}").v.rawValue == "future")
        nullableValueClassReject { codec.decodeFromString<ValueClassNarrowDto>("{\"v\":null}") }
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
        if (nulls) nullableValueClassReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":\"ready\"}") }
        val clear = absent.copy(inherited = Patch.Null, wrapper = Patch.Null, forcedTrue = Patch.Null, count = Patch.Null, countTrue = Patch.Null)
        val clearJson = codec.encodeToJsonElement(clear) as JsonObject
        check(clearJson["inherited"] === JsonNull && clearJson["wrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(clear)) == clear)
        val value = absent.copy(inherited = Patch.Value(future), wrapper = Patch.Value(ready), forcedTrue = Patch.Value(future), count = Patch.Value(count), countTrue = Patch.Value(count))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)) == value)
        nullableValueClassReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"ready\",\"inherited\":123}") }
        nullableValueClassBoundsReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"ready\",\"count\":0}") }
        var expected: JsonElement = JsonNull
        var failure = ""
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/optional", "/count", "/count-direct" -> { check(codec.parseToJsonElement(nullableValueClassBody(request.body)) == expected); respond(if (failure == "kind") "123" else if (failure == "bounds") "\"\"" else if (failure == "countBounds") "0" else if (failure == "syntax") "{" else expected.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                    "/patch" -> { check(codec.parseToJsonElement(nullableValueClassBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
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
            val api = RootNullableValueClassAliasApiClient(http, "http://example.test/", codec).default
            check(api.echo(null) == null && api.optional() == null)
            for (wrapped in listOf(ready, future)) { expected = codec.encodeToJsonElement(ZNameDto.serializer(), wrapped); check(api.echo(wrapped) == wrapped && api.echo2(wrapped) == wrapped && api.optional(wrapped) == wrapped) }
            expected = JsonNull; check(api.echoCount(null) == null)
            expected = codec.parseToJsonElement("3"); check(api.echoCount(count) == count && api.echoCount2(count) == count)
            for (patch in listOf(absent, clear, value)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = codec.encodeToJsonElement(ZNameDto.serializer(), ready)
            for (bad in listOf("kind", "syntax")) { failure = bad; nullableValueClassReject { api.echo(ready) } }
            failure = "bounds"; nullableValueClassBoundsReject { api.echo(ready) }
            expected = codec.parseToJsonElement("3"); failure = "countBounds"; nullableValueClassBoundsReject { api.echoCount(count) }
            expected = codec.encodeToJsonElement(ZNameDto.serializer(), ready)
            failure = ""; val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 5 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[1] as WatchEventDto.Nullable).data == future && (events[2] as WatchEventDto.Narrow).data == ready)
            check((events[3] as WatchEventDto.Count).data == null && (events[4] as WatchEventDto.CountNarrow).data == count)
            for (bad in listOf("bounds", "countBounds")) { failure = bad; withTimeout(3000) { nullableValueClassBoundsReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            for (bad in listOf("narrow", "countNarrow", "kind", "syntax")) { failure = bad; withTimeout(3000) { nullableValueClassReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            failure = ""; check(api.echo2(ready) == ready)
        } finally { http.close(); producers.cancel() }
        println("RootNullableValueClassAlias Json defaults=$defaults nulls=$nulls checks passed")
    }
}
