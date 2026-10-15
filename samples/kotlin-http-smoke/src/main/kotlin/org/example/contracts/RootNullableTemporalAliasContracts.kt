@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, kotlin.time.ExperimentalTime::class)
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
import kotlinx.serialization.builtins.serializer
import org.example.rootnullabletemporalaliases.models.NativeMaybeDto
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.rootnullabletemporalaliases.RootNullableTemporalAliasApiClient
import org.example.rootnullabletemporalaliases.api.models.SubmitInputDto
import org.example.rootnullabletemporalaliases.api.models.WatchEventDto
import org.example.rootnullabletemporalaliases.models.AInheritedDto
import org.example.rootnullabletemporalaliases.models.AFalseDto
import org.example.rootnullabletemporalaliases.models.ATrueDto
import org.example.rootnullabletemporalaliases.models.HolderDto
import org.example.rootnullabletemporalaliases.models.Patch
import org.example.rootnullabletemporalaliases.models.ZStampDto
import org.example.rootnullabletemporalaliases.models.ZDayDto
import org.example.rootnullabletemporalaliases.models.LInheritedDto
import org.example.rootnullabletemporalaliases.models.LFalseDto
import org.example.rootnullabletemporalprobes.models.ANearDto as TemporalNearDto
import org.example.rootnullabletemporalprobes.models.ANarrowDto as TemporalNarrowDto
import org.example.rootnullabletemporalprobes.models.AWideDto as TemporalWideDto

private fun nullableTemporalBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableTemporalBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable value-class body $body")
}
private suspend fun nullableTemporalReject(block: suspend () -> Unit) {
    try { block(); error("Invalid value-class kind accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong nullable value-class failure $cause" } }
}
private suspend fun nullableTemporalBoundsReject(block: suspend () -> Unit) {
    try { block(); error("Invalid wrapped value accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException || it is SerializationException }) { "Wrong bounds failure $cause" } }
}
suspend fun rootNullableTemporalAliasContracts() {
    check(AInheritedDto.parse("2024-02-29T12:34:56Z") == ZStampDto.parse("2024-02-29T12:34:56Z"))
    check(NativeMaybeDto.parse("2024-02-29T12:34:56Z") == AFalseDto.parse("2024-02-29T12:34:56Z"))
    check(!AInheritedDto.serializer().descriptor.isNullable && serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable && serializer<ATrueDto>().descriptor.isNullable)
    check(!LInheritedDto.serializer().descriptor.isNullable && serializer<LInheritedDto>().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val ready = ZStampDto.parse("2024-02-29T12:34:56Z")
        val future = ZStampDto.parse("2024-03-01T12:34:56Z")
        val day = ZDayDto.parse("2024-02-29")
        val nullable = serializer<AInheritedDto>()
        for (value in listOf(null, day)) check(codec.decodeFromString<LInheritedDto>(codec.encodeToString<LInheritedDto>(value)) == value)
        nullableTemporalBoundsReject { codec.decodeFromString<LInheritedDto>("\"2024-02-30\"") }
        nullableTemporalBoundsReject { AInheritedDto.parse("invalid") }
        nullableTemporalBoundsReject { LInheritedDto.parse("2024-02-30") }
        nullableTemporalBoundsReject { codec.decodeFromString<AInheritedDto>("\"\"") }
        for (value in listOf(null, ready, future)) {
            check(codec.decodeFromString(nullable, codec.encodeToString(nullable, value)) == value)
            check(codec.decodeFromString(ZStampDto.serializer().nullable, codec.encodeToString(nullable, value)) == value)
        }
        nullableTemporalReject { codec.decodeFromString(nullable, "123") }
        val holder = HolderDto(null, ready, future, null, ready, listOf(null, ready, future), mapOf("nil" to null, "future" to future), null, day, day, ready)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<TemporalNearDto>("{\"v\":null}").v == null)
        check(codec.decodeFromString<TemporalNarrowDto>("{\"v\":\"2024-03-01T12:34:56Z\"}").v == future)
        nullableTemporalReject { codec.decodeFromString<TemporalNarrowDto>("{\"v\":null}") }
        val wide = codec.decodeFromJsonElement(TemporalWideDto.serializer(), JsonObject((0 until 240).associate { "p$it" to if (it == 239) codec.encodeToJsonElement(ZStampDto.serializer(), ready) else JsonNull }))
        check(wide.p0 == null && wide.p239 == ready)
        check(codec.decodeFromString<TemporalWideDto>(codec.encodeToString(wide)) == wide)
        val absent = SubmitInputDto(null, ready)
        val encoded = codec.encodeToJsonElement(absent) as JsonObject
        check("inherited" !in encoded && "wrapper" !in encoded && "forcedTrue" !in encoded)
        check(if (nulls) encoded["requiredNull"] === JsonNull else "requiredNull" !in encoded)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(absent)) == absent)
        if (nulls) nullableTemporalReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":\"2024-02-29T12:34:56Z\"}") }
        val clear = absent.copy(inherited = Patch.Null, wrapper = Patch.Null, forcedTrue = Patch.Null, day = Patch.Null, dayTrue = Patch.Null)
        val clearJson = codec.encodeToJsonElement(clear) as JsonObject
        check(clearJson["inherited"] === JsonNull && clearJson["wrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(clear)) == clear)
        val value = absent.copy(inherited = Patch.Value(future), wrapper = Patch.Value(ready), forcedTrue = Patch.Value(future), day = Patch.Value(day), dayTrue = Patch.Value(day))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)) == value)
        nullableTemporalReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"2024-02-29T12:34:56Z\",\"inherited\":123}") }
        nullableTemporalBoundsReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"2024-02-29T12:34:56Z\",\"day\":\"2024-02-30\"}") }
        var expected: JsonElement = JsonNull
        var failure = ""
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/optional", "/day", "/day-direct" -> { check(codec.parseToJsonElement(nullableTemporalBody(request.body)) == expected); respond(if (failure == "kind") "123" else if (failure == "bounds") "\"\"" else if (failure == "dayBounds") "\"2024-02-30\"" else if (failure == "syntax") "{" else expected.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                    "/query" -> { check(request.url.parameters["day"] == day.toString()); check(request.url.parameters["since"] == (if (failure == "noSince") null else ready.toString())); respond("", HttpStatusCode.NoContent) }
                    "/patch" -> { check(codec.parseToJsonElement(nullableTemporalBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true); channels += channel; val broken = failure
                        producers.launch { try {
                            val frame = when (broken) {
                                "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                                "bounds" -> "event: nullable\r\ndata: \"\"\r\n\r\n"
                                "dayBounds" -> "event: day\r\ndata: \"2024-02-30\"\r\n\r\n"
                                "dayNarrow" -> "event: dayNarrow\r\ndata: null\r\n\r\n"
                                "kind" -> "event: nullable\r\ndata: 123\r\n\r\n"
                                "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                                else -> "event: nullable\r\ndata: null\r\n\r\nevent: nullable\r\ndata: \"2024-03-01T12:34:56Z\"\r\n\r\nevent: narrow\r\ndata: \"2024-02-29T12:34:56Z\"\r\n\r\nevent: day\r\ndata: null\r\n\r\nevent: dayNarrow\r\ndata: \"2024-02-29\"\r\n\r\n"
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
            val api = RootNullableTemporalAliasApiClient(http, "http://example.test/", codec).default
            check(api.echo(null) == null && api.optional() == null)
            api.query(day = day, since = ready)
            failure = "noSince"; api.query(day = day); failure = ""
            for (wrapped in listOf(ready, future)) { expected = codec.encodeToJsonElement(ZStampDto.serializer(), wrapped); check(api.echo(wrapped) == wrapped && api.echo2(wrapped) == wrapped && api.optional(wrapped) == wrapped) }
            expected = JsonNull; check(api.echoDay(null) == null)
            expected = codec.encodeToJsonElement(ZDayDto.serializer(), day); check(api.echoDay(day) == day && api.echoDay2(day) == day)
            for (patch in listOf(absent, clear, value)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = codec.encodeToJsonElement(ZStampDto.serializer(), ready)
            for (bad in listOf("kind", "syntax")) { failure = bad; nullableTemporalReject { api.echo(ready) } }
            failure = "bounds"; nullableTemporalBoundsReject { api.echo(ready) }
            expected = codec.encodeToJsonElement(ZDayDto.serializer(), day); failure = "dayBounds"; nullableTemporalBoundsReject { api.echoDay(day) }
            expected = codec.encodeToJsonElement(ZStampDto.serializer(), ready)
            failure = ""; val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 5 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[1] as WatchEventDto.Nullable).data == future && (events[2] as WatchEventDto.Narrow).data == ready)
            check((events[3] as WatchEventDto.Day).data == null && (events[4] as WatchEventDto.DayNarrow).data == day)
            for (bad in listOf("bounds", "dayBounds")) { failure = bad; withTimeout(3000) { nullableTemporalBoundsReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            for (bad in listOf("narrow", "dayNarrow", "kind", "syntax")) { failure = bad; withTimeout(3000) { nullableTemporalReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            failure = ""; check(api.echo2(ready) == ready)
        } finally { http.close(); producers.cancel() }
        println("RootNullableTemporalAlias Json defaults=$defaults nulls=$nulls checks passed")
    }
}
