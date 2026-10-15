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
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlin.time.Instant
import kotlinx.datetime.LocalDate
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.mapdefaults.MapDefaults
import org.example.mapdefaults.api.models.PatchInputDto
import org.example.mapdefaults.api.models.WatchEventDto
import org.example.mapdefaults.models.HolderDto
import org.example.mapdefaults.models.Patch
import org.example.mapdefaults.models.RequiredDatesDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class MapDefaultsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun mapDefaultsReject(rule: String, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected $rule failure" }
    check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong failure for $rule: $failure" }
}

private suspend fun temporalMapsReject(block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null && generateSequence(failure) { it.cause }.any {
        it is SerializationException || it is IllegalArgumentException
    }) { "Expected temporal codec failure, got $failure" }
}

private fun mapDefaultsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> mapDefaultsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected body $body")
}

suspend fun primitiveMapDefaultsContracts(checked: Boolean = true) {
    val value = HolderDto(required = mapOf("count" to 2), kotlin = "shadow", kotlinx = "shadow")
    check(value.values == mapOf("a/~" to 2, "nil" to null))
    check(value.names == mapOf("\$ref" to "#/data", "quote\"" to "a\n\$😀"))
    check(value.longs == mapOf("min" to Long.MIN_VALUE))
    check(value.floats == mapOf("one" to 1f) && value.doubles == mapOf("half" to 0.5))
    check(value.flags == mapOf("yes" to true) && value.empty == emptyMap<String, String>() && value.nullable == null)
    val day = LocalDate.parse("2024-02-29")
    val stamp = Instant.parse("2024-02-29T10:04:56.123456789Z")
    check(value.dates == mapOf("a/~" to day, "nil" to null))
    check(value.times == mapOf("stamp" to stamp))
    check(value.edgeTimes == mapOf(
        "fraction" to Instant.parse("2024-02-29T12:34:56.1Z"),
        "east" to Instant.parse("2024-02-28T18:34:56Z"),
        "west" to Instant.parse("2024-03-01T06:34:56Z"),
        "zero" to Instant.parse("2024-02-29T12:34:56Z"),
        "year" to Instant.parse("0000-01-01T00:00:00Z"),
        "nil" to null
    ))
    check(value.emptyDates == emptyMap<String, LocalDate>() && value.nullableDates == null)
    val second = HolderDto(required = mapOf("count" to 2))
    check(value.values !== second.values && value.empty !== second.empty)
    check(value.dates !== second.dates && value.times !== second.times && value.emptyDates !== second.emptyDates)
    if (checked) {
        mapDefaultsReject("minProperties") { HolderDto(required = emptyMap()) }
        mapDefaultsReject("maxProperties") { HolderDto(required = mapOf("count" to 2), values = mapOf("a" to 2, "b" to 4, "c" to 6, "d" to 8)) }
        mapDefaultsReject("multipleOf") { HolderDto(required = mapOf("count" to 2), values = mapOf("a" to 3)) }
        mapDefaultsReject("minimum") { HolderDto(required = mapOf("count" to 2), floats = mapOf("a" to 0f)) }
        mapDefaultsReject("minProperties") { HolderDto(required = mapOf("count" to 2), dates = emptyMap()) }
        mapDefaultsReject("maxProperties") { HolderDto(required = mapOf("count" to 2), times = mapOf("a" to stamp, "b" to stamp, "c" to stamp)) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("""{"required":{"count":2}}""")
        check(missing.values == value.values && missing.names == value.names)
        check(missing.dates == value.dates && missing.times == value.times && missing.edgeTimes == value.edgeTimes)
        check(codec.decodeFromString<HolderDto>("""{"required":{"count":2},"dates":null,"times":null}""").let { it.dates == null && it.times == null })
        val replaced = codec.decodeFromString<HolderDto>("""{"required":{"count":2},"dates":{"new":"2024-03-01"},"times":{"new":"2024-02-29T12:34:56+02:30"}}""")
        check(replaced.dates == mapOf("new" to LocalDate.parse("2024-03-01")))
        check(replaced.times == mapOf("new" to Instant.parse("2024-02-29T10:04:56Z")))
        for (invalid in listOf(""""dates":{"a/~":"2023-02-29"}""", """"times":{"stamp":0}""", """"times":{"stamp":null}""", """"dates":[]""")) {
            temporalMapsReject { codec.decodeFromString<HolderDto>("""{"required":{"count":2},$invalid}""") }
        }
        temporalMapsReject { codec.decodeFromString<RequiredDatesDto>("{}") }
        check(codec.decodeFromString<RequiredDatesDto>("""{"dates":{"supplied":"2024-03-01"}}""").dates == mapOf("supplied" to LocalDate.parse("2024-03-01")))
        check(codec.decodeFromString<HolderDto>("""{"required":{"count":2},"values":null}""").values == null)
        check(codec.decodeFromString<HolderDto>("""{"required":{"count":2},"values":{"custom":4}}""").values == mapOf("custom" to 4))
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"required":{"count":2},"values":[]}""") }.isFailure)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("values") == defaults)
        check(encoded.containsKey("dates") == defaults && encoded.containsKey("times") == defaults)
        if (defaults) {
            check((encoded["dates"] as JsonObject)["a/~"] == codec.parseToJsonElement("\"2024-02-29\""))
            check((encoded["times"] as JsonObject)["stamp"] == codec.parseToJsonElement("\"2024-02-29T10:04:56.123456789Z\""))
            check((encoded["dates"] as JsonObject)["nil"] === JsonNull)
        }
        val roundTrip = codec.decodeFromJsonElement(HolderDto.serializer(), encoded)
        check(roundTrip.values == value.values && roundTrip.dates == value.dates && roundTrip.times == value.times)
        val absent = codec.decodeFromString<PatchInputDto>("{}")
        check(absent.values == Patch.Absent && absent.dates == Patch.Absent && absent.times == Patch.Absent)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).isEmpty())
        val nullPatch = codec.decodeFromString<PatchInputDto>("""{"values":null}""")
        check(nullPatch.values == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), nullPatch) as JsonObject)["values"] === JsonNull)
        val update = PatchInputDto(values = Patch.Value(mapOf("count" to 2)))
        check(codec.encodeToJsonElement(PatchInputDto.serializer(), update) == codec.parseToJsonElement("""{"values":{"count":2}}"""))
        val temporalNull = codec.decodeFromString<PatchInputDto>("""{"dates":null,"times":null}""")
        check(temporalNull.dates == Patch.Null && temporalNull.times == Patch.Null)
        check(codec.encodeToJsonElement(PatchInputDto.serializer(), temporalNull) == codec.parseToJsonElement("""{"dates":null,"times":null}"""))
        val temporalUpdate = PatchInputDto(dates = Patch.Value(mapOf("day" to day, "nil" to null)), times = Patch.Value(mapOf("stamp" to stamp)))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), temporalUpdate)) == temporalUpdate)
        if (checked) {
            mapDefaultsReject("minProperties") { PatchInputDto(dates = Patch.Value(emptyMap())) }
            mapDefaultsReject("maxProperties") { codec.decodeFromString<HolderDto>("""{"required":{"count":2},"dates":{"a":null,"b":null,"c":null}}""") }
            mapDefaultsReject("minProperties") { PatchInputDto(values = Patch.Value(emptyMap())) }
            mapDefaultsReject("minimum") { codec.decodeFromString<PatchInputDto>("""{"values":{"a":-1}}""") }
            mapDefaultsReject("multipleOf") { codec.decodeFromString<HolderDto>("""{"required":{"count":2},"values":{"a":3}}""") }
        }
        mapDefaultsWire(codec, value, checked)
        println("PrimitiveMapDefaults Json defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun mapDefaultsWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var temporalBad: String? = null
    var expectedPatch: JsonObject = JsonObject(emptyMap())
    var calls = 0
    val engine = MapDefaultsEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        val payload = temporalBad ?: if (bad) """{"required":{"count":2},"values":{"a":3}}""" else """{"required":{"count":2}}"""
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(mapDefaultsBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(mapDefaultsBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val invalid = bad || temporalBad != null
                producers.launch { try {
                    val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (invalid) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected map defaults request")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = MapDefaults(http, "http://example.test/", codec).defaults
        val response = api.send(value)
        check(response.values == value.values && response.dates == value.dates && response.times == value.times)
        api.patchPatch(PatchInputDto())
        for (patch in listOf(
            PatchInputDto(dates = Patch.Null, times = Patch.Null),
            PatchInputDto(dates = Patch.Value(value.dates!!), times = Patch.Value(value.times!!))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) {
            val data = (withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data
            check(data.values == value.values && data.dates == value.dates && data.times == value.times)
        }
        for ((payload, rule) in listOf(
            """{"required":{"count":2},"dates":{"a/~":"2023-02-29"}}""" to null,
            """{"required":{"count":2},"times":{"stamp":null}}""" to null,
            """{"required":{"count":2},"dates":{}}""" to "minProperties",
            """{"required":{"count":2},"times":{"a":"2024-02-29T12:34:56Z","b":"2024-02-29T12:34:56Z","c":"2024-02-29T12:34:56Z"}}""" to "maxProperties"
        )) {
            if (rule != null && !checked) continue
            temporalBad = payload
            if (rule == null) {
                temporalMapsReject { api.send(value) }
                temporalMapsReject { withTimeout(10000) { api.watch().toList() } }
            } else {
                mapDefaultsReject(rule) { api.send(value) }
                mapDefaultsReject(rule) { withTimeout(10000) { api.watch().toList() } }
            }
            check(channels.last().isClosedForRead)
            temporalBad = null
            check(api.send(value).times == value.times)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.dates == value.dates)
        }
        if (checked) {
            bad = true
            mapDefaultsReject("multipleOf") { api.send(value) }
            mapDefaultsReject("multipleOf") { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            bad = false
            check(api.send(value).names == value.names)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
