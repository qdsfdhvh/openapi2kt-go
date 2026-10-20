@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlin.uuid.Uuid
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
import io.ktor.utils.io.ClosedByteChannelException
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.nullableuuid.NullableUuidApiClient
import org.example.nullableuuid.api.models.PatchUuidInputDto
import org.example.nullableuuid.api.models.WatchUuidEventDto
import org.example.nullableuuid.models.AggregateDto
import org.example.nullableuuid.models.DefaultsDto
import org.example.nullableuuid.models.BoundedDto
import org.example.nullableuuid.models.BridgeDto
import org.example.nullableuuid.models.HolderDto
import org.example.nullableuuid.models.MaybeDto
import org.example.nullableuuid.models.OwnDto
import org.example.nullableuuid.models.Patch
import org.example.nullableuuid.models.ReopenedDto
import org.example.nullableuuid.models.`MaybeDto default`
import org.example.nullableuuid.models.`NullableValueDto default`
import org.example.nullableuuid.models.`BoundedDto default`

private fun nullableUuidBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableUuidBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic UUID body")
}

suspend fun nullableUuidContracts(checked: Boolean = true) = withTimeout(20000) {
    val canonical = "550e8400-e29b-41d4-a716-446655440000"
    val id = Uuid.parse(canonical)
    val other = Uuid.parse("650e8400-e29b-41d4-a716-446655440000")
    check(serializer<MaybeDto>().descriptor.isNullable && serializer<OwnDto>().descriptor.isNullable)
    check(serializer<ReopenedDto>().descriptor.isNullable && !serializer<BridgeDto>().descriptor.isNullable)
    check(`MaybeDto default`() == null && `NullableValueDto default`() == id)
    check(`BoundedDto default`() == id)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<MaybeDto>("null") == null)
        check(codec.decodeFromString<OwnDto>("null") == null)
        for (text in listOf(canonical, canonical.uppercase(), canonical.replace("-", ""))) {
            val value = codec.decodeFromString<MaybeDto>(codec.encodeToString(text))
            check(value == id && codec.encodeToString(value) == codec.encodeToString(canonical))
        }
        for (bad in listOf("", "not-a-uuid", "quote\"\\\né", "{$canonical}", "urn:uuid:$canonical", " $canonical")) {
            check(runCatching { codec.decodeFromString<MaybeDto>(codec.encodeToString(bad)) }.isFailure)
            // Explicit native String overrides retain arbitrary text and escaping.
            check(codec.decodeFromString<DefaultsDto>("""{"plain":${codec.encodeToString(bad)}}""").plain == bad)
        }
        check(runCatching { codec.decodeFromString<BridgeDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<MaybeDto>("42") }.isFailure)
        val defaulted = codec.decodeFromString<DefaultsDto>("{}")
        check(defaulted.id == id && defaulted.constant == id && defaulted.referenced == id)
        check(defaulted.rows == listOf(id, null) && defaulted.nested == listOf(listOf(id)))
        check(defaulted.values == mapOf("key" to id) && defaulted.kotlin == "shadow")
        check(codec.decodeFromString<DefaultsDto>(codec.encodeToString(defaulted)).id == id)
        // Transparent aliases expose the builtin codec; consumers own constraints.
        check(codec.decodeFromString<BoundedDto>(codec.encodeToString(other)) == other)
        val holder = HolderDto(narrow = id, bridge = id, rows = listOf(null, id), values = mapOf("key" to id))
        val decoded = codec.decodeFromString<HolderDto>(codec.encodeToString(holder))
        check(decoded.value == null && decoded.own == null && decoded.bounded == null)
        check(decoded.rows == holder.rows && decoded.values == holder.values)
        val absent = PatchUuidInputDto(requiredNull = null, requiredFalse = id)
        val patches = listOf(absent,
            PatchUuidInputDto(requiredNull = null, requiredFalse = id, value = Patch.Null, bounded = Patch.Null),
            PatchUuidInputDto(requiredNull = null, requiredFalse = id, value = Patch.Value(id), bounded = Patch.Value(id)))
        check("value" !in codec.encodeToJsonElement(absent).jsonObject)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["value"] == JsonNull)
        for (patch in patches) {
            val roundtrip = codec.decodeFromString<PatchUuidInputDto>(codec.encodeToString(patch))
            check(roundtrip.requiredNull == null && roundtrip.requiredFalse == id && roundtrip.value == patch.value)
        }
        check(runCatching { codec.decodeFromString<PatchUuidInputDto>("""{"requiredNull":null,"requiredFalse":null}""") }.isFailure)
        if (nulls) check(runCatching { codec.decodeFromString<PatchUuidInputDto>("""{"requiredFalse":"$canonical"}""") }.isFailure)
        check(AggregateDto().id == id)
        if (checked) {
            check(runCatching { AggregateDto(id = other) }.isFailure)
            check(runCatching { HolderDto(narrow = id, bridge = id, bounded = other) }.isFailure)
            check(runCatching { PatchUuidInputDto(requiredNull = null, requiredFalse = id, bounded = Patch.Value(other)) }.isFailure)
            check(runCatching { codec.decodeFromString<HolderDto>("""{"narrow":"$canonical","bridge":"$canonical","bounded":"$other"}""") }.isFailure)
        }
        var expected = "null"
        var failure = ""
        var calls = 0
        var expectedPatch = codec.encodeToJsonElement(absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = BuiltinChainStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo", "/narrow", "/bound" -> {
                    check(nullableUuidBody(request.body) == expected)
                    val response = when (failure) {
                        "bounds" -> codec.encodeToString(other)
                        "lexical" -> "\"not-a-uuid\""
                        "null" -> "null"
                        "uppercase" -> codec.encodeToString(canonical.uppercase())
                        else -> expected
                    }
                    respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/parameters/$canonical" -> {
                    check(request.url.parameters["lookup"] == canonical)
                    check(request.url.parameters.getAll("ids") == listOf(canonical, other.toString()))
                    check(request.headers["trace"] == canonical)
                    respond("", HttpStatusCode.NoContent)
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(nullableUuidBody(request.body)) == expectedPatch)
                    respond("", HttpStatusCode.NoContent)
                }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels.add(channel)
                    val frames = when (failure) {
                        "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                        "kind" -> "event: nullable\r\ndata: {}\r\n\r\n"
                        "lexical" -> "event: nullable\r\ndata: \"not-a-uuid\"\r\n\r\n"
                        "bounds" -> "event: bounded\r\ndata: \"$other\"\r\n\r\n"
                        else -> "event: nullable\r\ndata: null\r\n\r\nevent: nullable\r\ndata: \"$canonical\"\r\n\r\nevent: narrow\r\ndata: \"$canonical\"\r\n\r\nevent: bounded\r\ndata: null\r\n\r\nevent: bounded\r\ndata: \"$canonical\"\r\n\r\n"
                    }
                    writers.add(scope.launch {
                        try { for (byte in frames.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() }
                        catch (closed: ClosedByteChannelException) { if (!channel.isClosedForWrite) throw closed }
                    })
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic UUID route")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = NullableUuidApiClient(http, "https://example.test/", codec).default
            expected = "null"; check(api.echoUuid(null) == null && api.echoBounded(null) == null)
            expected = codec.encodeToString(id)
            check(api.echoUuid(id) == id && api.echoNarrow(id) == id && api.echoBounded(id) == id)
            failure = "uppercase"; check(api.echoUuid(id) == id); failure = ""
            for (bad in listOf("lexical", "null")) {
                failure = bad; check(runCatching { api.echoNarrow(id) }.isFailure)
            }
            failure = ""
            api.uuidParameters(id = id, lookup = id, trace = id, ids = listOf(id, other))
            for (patch in patches) { expectedPatch = codec.encodeToJsonElement(patch); api.patchUuid(patch) }
            if (checked) {
                val before = calls; check(runCatching { api.echoBounded(other) }.isFailure); check(calls == before)
                failure = "bounds"; check(runCatching { api.echoBounded(id) }.isFailure); failure = ""
            }
            val before = calls; val flow = api.watchUuid(); check(calls == before)
            repeat(2) {
                val events = flow.take(5).toList()
                check((events[0] as WatchUuidEventDto.Nullable).data == null)
                check((events[1] as WatchUuidEventDto.Nullable).data == id)
                check((events[2] as WatchUuidEventDto.Narrow).data == id)
                check((events[3] as WatchUuidEventDto.Bounded).data == null && (events[4] as WatchUuidEventDto.Bounded).data == id)
                check(channels.last().isClosedForWrite)
            }
            for (bad in if (checked) listOf("narrow", "kind", "lexical", "bounds") else listOf("narrow", "kind", "lexical")) {
                failure = bad; check(runCatching { flow.take(1).toList() }.isFailure); check(channels.last().isClosedForWrite)
            }
            failure = ""; check((flow.take(1).toList().single() as WatchUuidEventDto.Nullable).data == null)
            check(channels.last().isClosedForWrite)
            check(api.echoUuid(id) == id)
        } finally { http.close(); scope.cancel(); writers.forEach { it.join() } }
    }
}
