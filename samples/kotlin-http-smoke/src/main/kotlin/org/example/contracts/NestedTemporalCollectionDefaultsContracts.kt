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
import kotlin.time.Instant
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.datetime.LocalDate
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.nestedtemporal.NestedTemporal
import org.example.nestedtemporal.api.models.PatchInputDto
import org.example.nestedtemporal.api.models.WatchEventDto
import org.example.nestedtemporal.models.HolderDto
import org.example.nestedtemporal.models.Patch
import org.example.nestedtemporal.models.RequiredNullableDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NestedTemporalEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nestedTemporalReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun nestedTemporalBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nestedTemporalBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun nestedTemporalCollectionDefaultsContracts(checked: Boolean) {
    val day = LocalDate.parse("2024-02-29")
    val stamp = Instant.parse("2024-02-29T00:00:00Z")
    val required = listOf(listOf(day))
    val value = HolderDto(required = required, kotlin = "shadow", kotlinx = "shadow")
    val actualDates: List<List<LocalDate?>?>? = value.rows
    val actualStamps: List<Map<String, Instant?>?>? = value.listMaps
    check(actualDates == listOf(listOf(day, null, LocalDate.parse("0000-01-01")), null, emptyList()))
    check(actualStamps == listOf(mapOf("z" to null, "a/~" to Instant.parse("2024-02-29T10:04:56.123456789Z"), "\$ref" to stamp), null, emptyMap()))
    check(value.mapLists == mapOf("z" to null, "b" to emptyList(), "a" to listOf(day, null)))
    check(value.mapMaps == mapOf("z" to emptyMap(), "a" to mapOf("z" to null, "a" to Instant.parse("2024-02-28T18:34:56Z"), "b" to Instant.parse("2024-03-01T06:34:56Z"))))
    check(value.mixed == listOf(mapOf("a" to listOf(mapOf("b" to day)))))
    check(value.builtin == mapOf("a" to mapOf("b" to 1.0f)))
    check(value.empty == emptyList<List<LocalDate>>() && value.nil == null)
    check(HolderDto.Nested().value == required)
    val second = HolderDto(required = required)
    check(value.rows !== second.rows && value.rows?.first() !== second.rows?.first())
    check(value.listMaps !== second.listMaps && value.listMaps?.first() !== second.listMaps?.first())
    check(value.mapLists !== second.mapLists && value.mapLists?.get("a") !== second.mapLists?.get("a"))
    check(value.mapMaps !== second.mapMaps && value.mapMaps?.get("a") !== second.mapMaps?.get("a"))
    check(value.empty !== second.empty && value.mixed?.first()?.get("a") !== second.mixed?.first()?.get("a"))
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    check(HolderDto.serializer().descriptor.serialName == "org.example.nestedtemporal.models.HolderDto")
    if (checked) {
        nestedTemporalReject("minItems") { HolderDto(required = emptyList()) }
        nestedTemporalReject("minItems") { HolderDto(required = listOf(emptyList())) }
        nestedTemporalReject("maxItems") { HolderDto(required = required, rows = listOf(List(4) { day })) }
        nestedTemporalReject("maxProperties") { HolderDto(required = required, listMaps = listOf(mapOf("a" to stamp, "b" to stamp, "c" to stamp, "d" to stamp))) }
        nestedTemporalReject("maxProperties") { HolderDto(required = required, mapMaps = mapOf("a" to mapOf("a" to stamp, "b" to stamp, "c" to stamp, "d" to stamp))) }
        nestedTemporalReject("minimum") { HolderDto(required = required, builtin = mapOf("a" to mapOf("b" to -1.0f))) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire) == requiredNull)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val missing = codec.decodeFromString<HolderDto>("""{"required":[["2024-02-29"]]}""")
        check(missing.rows == value.rows && missing.mapMaps == value.mapMaps && missing.listMaps == value.listMaps && missing.mixed == value.mixed)
        nestedTemporalReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":[["2024-02-29"]],"rows":null,"mapMaps":null}""").let { it.rows == null && it.mapMaps == null })
        check(codec.decodeFromString<HolderDto>("""{"required":[["2024-02-29"]],"rows":[null,[],[null,"2024-03-01"]],"mapMaps":{"b":{"c":"2024-02-29T00:00:00Z"}}}""").let { it.rows == listOf(null, emptyList(), listOf(null, LocalDate.parse("2024-03-01"))) && it.mapMaps == mapOf("b" to mapOf("c" to stamp)) })
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("rows") == defaults && encoded.containsKey("mapMaps") == defaults)
        if (defaults) {
            check(encoded["rows"] == codec.parseToJsonElement("""[["2024-02-29",null,"0000-01-01"],null,[]]"""))
            check(encoded["listMaps"] == codec.parseToJsonElement("""[{"${'$'}ref":"2024-02-29T00:00:00Z","a/~":"2024-02-29T10:04:56.123456789Z","z":null},null,{}]"""))
        }
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        nestedTemporalReject { codec.decodeFromString<HolderDto>("""{"required":[[null]]}""") }
        nestedTemporalReject { codec.decodeFromString<HolderDto>("""{"required":[["2023-02-29"]]}""") }
        nestedTemporalReject { codec.decodeFromString<HolderDto>("""{"required":[["2024-02-29"]],"listMaps":[{"bad":"invalid"}]}""") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent && absent.stamps == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "rows" || it == "stamps" })
        val update = PatchInputDto(required = required, rows = Patch.Value(required), stamps = Patch.Value(mapOf("a" to listOf(stamp))))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":null,"stamps":null}""")
        check(cleared.rows == Patch.Null && cleared.stamps == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["rows"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        if (checked) {
            nestedTemporalReject("minItems") { codec.decodeFromString<HolderDto>("""{"required":[[]]}""") }
            nestedTemporalReject("maxItems") { codec.decodeFromString<HolderDto>("""{"required":[["2024-02-29"]],"rows":[["2024-02-29","2024-02-29","2024-02-29","2024-02-29"]]}""") }
            nestedTemporalReject("minimum") { codec.decodeFromString<HolderDto>("""{"required":[["2024-02-29"]],"builtin":{"a":{"b":-1}}}""") }
            nestedTemporalReject("minItems") { PatchInputDto(required = null, rows = Patch.Value(listOf(emptyList()))) }
        }
        nestedTemporalWire(codec, missing, checked)
        println("Nested temporal defaults Json defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun nestedTemporalWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val good = """{"required":[["2024-02-29"]]}"""
    var payload = good
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NestedTemporalEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nestedTemporalBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nestedTemporalBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (holdOpen) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected nested temporal path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = NestedTemporal(http, "http://example.test/", codec).defaults
        check(api.send(value).rows == value.rows)
        for (patch in listOf(PatchInputDto(required = value.required), PatchInputDto(required = null, rows = Patch.Null, stamps = Patch.Null), PatchInputDto(required = value.required, rows = Patch.Value(value.required)))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.rows == value.rows) }
        val invalid = mutableListOf<Pair<String, String?>>(
            """{"required":[[null]]}""" to null,
            """{"required":[["2023-02-29"]]}""" to null,
            """{"required":[["2024-02-29"]],"rows":[{}]}""" to null,
            """{"required":[["2024-02-29"]],"listMaps":[{"bad":"invalid"}]}""" to null,
            "{" to null)
        if (checked) invalid += listOf(
            """{"required":[[]]}""" to "minItems",
            """{"required":[["2024-02-29"]],"rows":[["2024-02-29","2024-02-29","2024-02-29","2024-02-29"]]}""" to "maxItems",
            """{"required":[["2024-02-29"]],"builtin":{"a":{"b":-1}}}""" to "minimum",
            """{"required":[["2024-02-29"]],"mapMaps":{"a":{"a":null,"b":null,"c":null,"d":null}}}""" to "maxProperties")
        for ((bad, rule) in invalid) {
            payload = bad
            failing = true
            nestedTemporalReject(rule) { api.send(value) }
            nestedTemporalReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = good
            failing = false
            check(api.send(value).mapMaps == value.mapMaps)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.mapMaps == value.mapMaps)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
