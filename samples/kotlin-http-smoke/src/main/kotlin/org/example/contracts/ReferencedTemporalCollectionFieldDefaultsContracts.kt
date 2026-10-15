@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, kotlin.time.ExperimentalTime::class)
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
import kotlinx.datetime.LocalDate
import kotlinx.serialization.serializer
import org.example.referencedtemporaldefaults.models.RequiredNullableTemporalDto
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.referencedtemporaldefaults.ReferencedTemporalDefaults
import org.example.referencedtemporaldefaults.api.models.PatchInputDto
import org.example.referencedtemporaldefaults.api.models.WatchEventDto
import org.example.referencedtemporaldefaults.models.HolderDto
import org.example.referencedtemporaldefaults.models.Patch
import org.example.referencedtemporaldefaults.models.RequiredNullableDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedTemporalCollectionDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun referencedTemporalCollectionDefaultReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun referencedTemporalCollectionDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedTemporalCollectionDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedTemporalCollectionFieldDefaultsContracts(checked: Boolean) {
    val value = HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = listOf(listOf(2L)), kotlin = "shadow", kotlinx = "shadow")
    val day = LocalDate.parse("2024-02-29")
    val stamp = Instant.parse("2024-02-29T12:34:56.123456789+02:30")
    val dates: org.example.referencedtemporaldefaults.models.DaysDto? = value.namedDateList
    val stamps: org.example.referencedtemporaldefaults.models.StampsDto? = value.namedInstantMap
    val builtin: List<LocalDate?>? = dates
    check(builtin == listOf(day, null))
    check(stamps == mapOf("z" to null, "a/~" to stamp, "\$ref" to Instant.parse("2024-02-29T18:00:00+18:00")))
    check(value.dateRows == listOf(listOf(day, null), emptyList(), null))
    check(value.dateListMaps == listOf(mapOf("a/~" to day, "z" to null), null, emptyMap()))
    check(value.instantMapLists == mapOf("a" to listOf(stamp, null), "empty" to emptyList(), "nil" to null))
    check(value.instantMapMaps == mapOf("a" to mapOf("stamp" to stamp), "empty" to emptyMap()))
    check(value.borrowed == listOf(mapOf("stamp" to Instant.parse("2024-02-29T00:00:00-18:00"))))
    check(value.emptyDates == emptyList<LocalDate?>() && value.dateNil == null)
    check(serializer<org.example.referencedtemporaldefaults.models.DaysDto>().descriptor.serialName == "kotlin.collections.ArrayList")
    check(serializer<org.example.referencedtemporaldefaults.models.StampsDto>().descriptor.serialName == "kotlin.collections.LinkedHashMap")
    val descriptor = HolderDto.serializer().descriptor
    val requiredDateIndex = descriptor.getElementIndex("requiredDays")
    check(!descriptor.isElementOptional(requiredDateIndex))
    check(descriptor.getElementDescriptor(requiredDateIndex).getElementDescriptor(0).getElementDescriptor(0).serialName == "kotlinx.datetime.LocalDate")
    if (checked) {
        referencedTemporalCollectionDefaultReject("maxItems") { HolderDto(requiredDays = value.requiredDays, required = value.required, namedDateList = List(4) { day }) }
        referencedTemporalCollectionDefaultReject("minItems") { HolderDto(requiredDays = listOf(emptyList()), required = value.required) }
    }
    check(value.rows == listOf(listOf(Long.MIN_VALUE, null, Long.MAX_VALUE), null, emptyList()))
    check(value.listMaps == listOf(mapOf("z" to null, "a/~" to "quote\"\$😀", "\$ref" to "data"), null, emptyMap()))
    check(value.mapLists == mapOf("z" to null, "b" to emptyList(), "a" to listOf(true, null, false)))
    val actualRows: org.example.referencedtemporaldefaults.models.RowsDto? = value.rows
    val actualMaps: org.example.referencedtemporaldefaults.models.MapMapsDto? = value.mapMaps
    check(actualRows == value.repeatedRows && actualMaps == value.mapMaps)
    check(value.rows !== value.repeatedRows)
    check(value.mapMaps == mapOf("z" to emptyMap(), "a" to mapOf("z" to null, "a" to 1.0f)))
    check(value.ints == listOf(mapOf("min" to Int.MIN_VALUE, "max" to Int.MAX_VALUE)))
    check(value.doubles == mapOf("one" to listOf(1.25, -0.0)))
    check(value.empty == emptyList<List<String>>() && value.nil == null)
    check(HolderDto.Nested().value == listOf(listOf("nested")))
    val second = HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = value.required)
    check(value.rows !== second.rows && value.rows?.first() !== second.rows?.first())
    check(value.mapMaps !== second.mapMaps && value.mapMaps?.get("a") !== second.mapMaps?.get("a"))
    check(value.empty !== second.empty)
    check(value.namedDateList !== second.namedDateList && value.dateRows?.first() !== second.dateRows?.first())
    check(value.instantMapMaps !== second.instantMapMaps && value.instantMapMaps?.get("a") !== second.instantMapMaps?.get("a"))
    check(value.emptyDates !== second.emptyDates)
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    if (checked) {
        referencedTemporalCollectionDefaultReject("minItems") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = emptyList()) }
        referencedTemporalCollectionDefaultReject("minItems") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = listOf(emptyList())) }
        referencedTemporalCollectionDefaultReject("minimum") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = listOf(listOf(-1L))) }
        referencedTemporalCollectionDefaultReject("uniqueItems") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = listOf(listOf(1L, 1L))) }
        referencedTemporalCollectionDefaultReject("maxItems") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = value.required, rows = listOf(List(4) { 1L })) }
        referencedTemporalCollectionDefaultReject("minLength") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = value.required, listMaps = listOf(mapOf("a" to ""))) }
        referencedTemporalCollectionDefaultReject("maxProperties") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = value.required, listMaps = listOf(mapOf("a" to "a", "b" to "b", "c" to "c", "d" to "d"))) }
        referencedTemporalCollectionDefaultReject("minimum") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = value.required, mapMaps = mapOf("a" to mapOf("b" to 0.0f))) }
        referencedTemporalCollectionDefaultReject("maxProperties") { HolderDto(requiredDays = listOf(listOf(LocalDate.parse("2024-02-29"))), required = value.required, mapMaps = mapOf("a" to mapOf("a" to 1.0f, "b" to 2.0f, "c" to 3.0f))) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]]}""").namedInstantMap == value.namedInstantMap)
        check(codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]],"namedDateList":null,"namedInstantMap":null}""").let { it.namedDateList == null && it.namedInstantMap == null })
        check(codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]],"namedDateList":["2024-03-01"],"namedInstantMap":{"a":"2024-03-01T00:00:00Z"}}""").namedDateList == listOf(LocalDate.parse("2024-03-01")))
        val temporalNull = RequiredNullableTemporalDto(value = null)
        check(!RequiredNullableTemporalDto.serializer().descriptor.isElementOptional(0))
        val temporalNullWire = codec.encodeToJsonElement(RequiredNullableTemporalDto.serializer(), temporalNull) as JsonObject
        check(temporalNullWire.containsKey("value") == nulls)
        check(runCatching { codec.decodeFromString<RequiredNullableTemporalDto>("{}") }.isSuccess == !nulls)
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire) == requiredNull)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val missing = codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]]}""")
        check(missing.rows == value.rows && missing.mapMaps == value.mapMaps && missing.listMaps == value.listMaps)
        referencedTemporalCollectionDefaultReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]],"rows":null,"mapMaps":null}""").let { it.rows == null && it.mapMaps == null })
        check(codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]],"rows":[null,[],[null,5]],"mapMaps":{"b":{"c":2}}}""").let { it.rows == listOf(null, emptyList(), listOf(null, 5L)) && it.mapMaps == mapOf("b" to mapOf("c" to 2.0f)) })
        check(codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]],"mapMaps":{"a":{"a":1,"z":null},"z":{}}}""").mapMaps == missing.mapMaps)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("rows") == defaults && encoded.containsKey("mapMaps") == defaults)
        check(encoded.containsKey("namedDateList") == defaults && encoded.containsKey("namedInstantMap") == defaults)
        if (defaults) check(encoded["namedInstantMap"] == codec.parseToJsonElement("""{"${'$'}ref":"2024-02-29T00:00:00Z","a/~":"2024-02-29T10:04:56.123456789Z","z":null}"""))
        if (defaults) {
            check(encoded["rows"] == codec.parseToJsonElement("""[[-9223372036854775808,null,9223372036854775807],null,[]]"""))
            check(encoded["mapMaps"] == codec.parseToJsonElement("""{"a":{"a":1.0,"z":null},"z":{}}"""))
        }
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        referencedTemporalCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[null]}""") }
        referencedTemporalCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[null]]}""") }
        referencedTemporalCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]],"listMaps":[{"bad":1}]}""") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent && absent.labels == Patch.Absent && absent.days == Patch.Absent && absent.stamps == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "rows" || it == "labels" })
        val update = PatchInputDto(required = listOf(listOf(2L)), rows = Patch.Value(listOf(listOf(3L))), labels = Patch.Value(mapOf("a" to listOf("yes"))), days = Patch.Value(listOf(LocalDate.parse("2024-02-29"))), stamps = Patch.Value(mapOf("a" to Instant.parse("2024-02-29T00:00:00Z"))))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":null,"labels":null}""")
        check(cleared.rows == Patch.Null && cleared.labels == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["rows"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        if (checked) {
            referencedTemporalCollectionDefaultReject("minimum") { codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[-1]]}""") }
            referencedTemporalCollectionDefaultReject("minItems") { codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[]]}""") }
            referencedTemporalCollectionDefaultReject("uniqueItems") { codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[1,1]]}""") }
            referencedTemporalCollectionDefaultReject("minLength") { codec.decodeFromString<HolderDto>("""{"requiredDays":[["2024-02-29"]],"required":[[2]],"listMaps":[{"bad":""}]}""") }
            referencedTemporalCollectionDefaultReject("minItems") { PatchInputDto(required = null, rows = Patch.Value(listOf(emptyList()))) }
        }
        referencedTemporalCollectionDefaultWire(codec, missing, checked)
        println("Referenced temporal collection defaults Json defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun referencedTemporalCollectionDefaultWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"requiredDays":[["2024-02-29"]],"required":[[2]]}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = ReferencedTemporalCollectionDefaultEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(referencedTemporalCollectionDefaultBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(referencedTemporalCollectionDefaultBody(request.body)) == expectedPatch)
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
            else -> error("Unexpected nested default path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ReferencedTemporalDefaults(http, "http://example.test/", codec).defaults
        var callbacks = 0
        val beforeCall = calls
        check(api.send(value, requestBuilder = { callbacks++ }).rows == value.rows)
        check(callbacks == 1 && calls == beforeCall + 1)
        if (checked) {
            val mutableDays = mutableListOf(LocalDate.parse("2024-02-29"))
            repeat(3) { mutableDays += LocalDate.parse("2024-03-01") }
            val beforeRejected = calls
            referencedTemporalCollectionDefaultReject("maxItems") {
                val pending = HolderDto(requiredDays = value.requiredDays, required = value.required, namedDateList = mutableDays)
                api.send(pending, requestBuilder = { callbacks++ })
            }
            check(callbacks == 1 && calls == beforeRejected)
        }
        for (patch in listOf(
            PatchInputDto(required = listOf(listOf(2L))),
            PatchInputDto(required = null, rows = Patch.Null, labels = Patch.Null, days = Patch.Null, stamps = Patch.Null),
            PatchInputDto(required = listOf(listOf(2L)), rows = Patch.Value(listOf(listOf(3L))), labels = Patch.Value(mapOf("a" to listOf("yes"))), days = Patch.Value(listOf(LocalDate.parse("2024-02-29"))), stamps = Patch.Value(mapOf("a" to Instant.parse("2024-02-29T00:00:00Z"))))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.rows == value.rows) }
        val invalid = mutableListOf<Pair<String, String?>>(
            """{"requiredDays":[["2024-02-29"]],"required":[[null]]}""" to null,
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"rows":[{}]}""" to null,
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"listMaps":[{"bad":false}]}""" to null,
            "{" to null)
        invalid += listOf(
            """{"requiredDays":[["2023-02-29"]],"required":[[2]]}""" to null,
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"namedInstantMap":{"bad":"2023-02-29T00:00:00Z"}}""" to null,
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"dateRows":[[true]]}""" to null)
        if (checked) invalid += listOf(
            """{"requiredDays":[["2024-02-29"]],"required":[[]]}""" to "minItems",
            """{"requiredDays":[["2024-02-29"]],"required":[[-1]]}""" to "minimum",
            """{"requiredDays":[["2024-02-29"]],"required":[[1,1]]}""" to "uniqueItems",
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"listMaps":[{"bad":""}]}""" to "minLength",
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"mapMaps":{"a":{"b":0}}}""" to "minimum",
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"mapMaps":{"a":{"a":1,"b":2,"c":3}}}""" to "maxProperties",
            """{"requiredDays":[["2024-02-29"]],"required":[[2]],"rows":[[1,2,3,4]]}""" to "maxItems")
        for ((bad, rule) in invalid) {
            payload = bad
            failing = true
            val beforeFailure = calls
            val beforeCallbacks = callbacks
            referencedTemporalCollectionDefaultReject(rule) { api.send(value, requestBuilder = { callbacks++ }) }
            check(calls == beforeFailure + 1 && callbacks == beforeCallbacks + 1)
            referencedTemporalCollectionDefaultReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"requiredDays":[["2024-02-29"]],"required":[[2]]}"""
            failing = false
            check(api.send(value).mapMaps == value.mapMaps)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.mapMaps == value.mapMaps)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
