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
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
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
import kotlinx.coroutines.yield
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.jsonObject
import org.example.mapunique.MapUniqueApiClient
import org.example.mapunique.api.models.PatchInputDto
import org.example.mapunique.api.models.WatchEventDto
import org.example.mapunique.models.StateDto
import org.example.mapunique.models.`DefaultStructuralDto default`
import org.example.mapunique.models.HolderDto
import org.example.mapunique.models.Patch
import org.example.mapunique.models.`DefaultValuesDto default`
import org.example.mapunique.models.`DefaultIdMapsDto default`
import kotlin.time.Instant
import kotlin.uuid.Uuid
import kotlinx.datetime.LocalDate
import java.util.concurrent.CopyOnWriteArrayList

private fun mapUniqueBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> mapUniqueBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic Map unique body")
}

private suspend fun mapUniqueExpect(checked: Boolean, keyword: String = "uniqueItems", action: suspend () -> Unit) {
    if (!checked) { action(); return }
    try { action(); error("Missing Map uniqueItems rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(keyword) == true
        }) { "Wrong Map uniqueness failure: $cause" }
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class MapUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

suspend fun mapUniqueContracts(checked: Boolean = true) = withTimeout(20000) {
    val directId = linkedMapOf("a" to "one", "b" to null)
    val id = linkedMapOf("a" to "one", "b" to null)
    val same = linkedMapOf("b" to null, "a" to "one")
    val directValues = listOf(directId, null, mapOf("c" to "two"))
    val values = listOf(id, null, mapOf("c" to "two"))
    val uuid = Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
    val date = LocalDate.parse("2024-02-29")
    val instant = Instant.parse("2026-01-02T03:04:05Z")
    val instantMaps = listOf(linkedMapOf("a" to instant, "b" to null), null)
    val duplicateInstantMaps = listOf(mapOf("a" to instant, "b" to null), linkedMapOf("b" to null, "a" to Instant.parse("2026-01-02T05:04:05+02:00")))
    check(id == same)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        structuralUniqueContracts(codec, checked)
        val holder = HolderDto(directValues, values, mapOf("ids" to values), ids = listOf(mapOf("a" to uuid), null), dates = listOf(mapOf("a" to date), emptyMap()), instants = instantMaps, listRows = listOf(listOf(1, null), listOf(null, 1)), mixedRows = listOf(mapOf("a" to listOf(instant, null))), aliasRows = listOf(listOf(1), listOf(2)))
        val encoded = codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(encoded) == holder)
        check(holder.defaultInstantMaps == instantMaps)
        check(`DefaultIdMapsDto default`() == listOf(mapOf("a" to uuid), null))
        mapUniqueExpect(checked) { HolderDto(emptyList(), ids = listOf(mapOf("a" to uuid), mapOf("a" to Uuid.parse("550E8400E29B41D4A716446655440000")))) }
        mapUniqueExpect(checked) { HolderDto(emptyList(), dates = listOf(mapOf("a" to date), mapOf("a" to date))) }
        mapUniqueExpect(checked) { HolderDto(emptyList(), instants = duplicateInstantMaps) }
        mapUniqueExpect(checked) { PatchInputDto(instants = Patch.Value(duplicateInstantMaps)) }
        HolderDto(emptyList(), instants = listOf(mapOf("a" to Instant.parse("2026-01-02T03:04:05.123000001Z")), mapOf("a" to Instant.parse("2026-01-02T03:04:05.123000002Z"))))
        for ((field, duplicate) in listOf(
            "ids" to "[{\"a\":\"550e8400-e29b-41d4-a716-446655440000\"},{\"a\":\"550E8400E29B41D4A716446655440000\"}]",
            "dates" to "[{\"a\":\"2024-02-29\"},{\"a\":\"2024-02-29\"}]",
            "instants" to "[{\"a\":\"2026-01-02T03:04:05.123Z\"},{\"a\":\"2026-01-02T05:04:05.123000000+02:00\"}]"
        )) { mapUniqueExpect(checked) { codec.decodeFromString<HolderDto>("{\"directValues\":[],\"$field\":$duplicate}") } }
        HolderDto(listOf(emptyMap(), mapOf("a" to null), mapOf("a" to "null")))
        val collisionA = mapOf("Aa" to "x")
        val collisionB = mapOf("BB" to "x")
        check(collisionA.hashCode() == collisionB.hashCode() && collisionA != collisionB)
        HolderDto(listOf(collisionA, collisionB))
        HolderDto(emptyList(), ints = listOf(mapOf("a" to Int.MIN_VALUE), mapOf("a" to Int.MAX_VALUE)), longs = listOf(mapOf("a" to Long.MIN_VALUE), mapOf("a" to Long.MAX_VALUE)), booleans = listOf(mapOf("a" to true), mapOf("a" to false), mapOf("a" to null)))
        mapUniqueExpect(checked) { HolderDto(emptyList(), ints = listOf(mapOf("a" to 1), mapOf("a" to 1))) }
        mapUniqueExpect(checked) { HolderDto(emptyList(), longs = listOf(mapOf("a" to Long.MAX_VALUE), mapOf("a" to Long.MAX_VALUE))) }
        mapUniqueExpect(checked) { HolderDto(emptyList(), booleans = listOf(mapOf("a" to true), mapOf("a" to true))) }
        mapUniqueExpect(checked) { codec.decodeFromString<HolderDto>("{\"directValues\":[],\"ints\":[{\"a\":0},{\"a\":-0}]}") }
        check(id.keys.toList() == listOf("a", "b") && same.keys.toList() == listOf("b", "a"))
        mapUniqueExpect(checked, "minProperties") { HolderDto(emptyList(), bounded = listOf(emptyMap())) }
        mapUniqueExpect(checked, "minLength") { HolderDto(emptyList(), bounded = listOf(mapOf("a" to ""))) }
        val bridged = HolderDto(directValues, bridged = listOf(id, emptyMap()))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(bridged)) == bridged)
        check(holder.defaultDirectValues == listOf(directId, null))
        check(holder.defaultValues == listOf(id, null))
        check(holder.defaultNested == listOf(listOf(id, null)))
        check(`DefaultValuesDto default`() == listOf(id, null))
        HolderDto(emptyList(), emptyList(), emptyMap())
        for (duplicate in listOf(listOf(directId, directId), listOf(null, null))) {
            mapUniqueExpect(checked) { HolderDto(duplicate) }
            mapUniqueExpect(checked) { PatchInputDto(directValues = Patch.Value(duplicate)) }
        }
        mapUniqueExpect(checked) { HolderDto(directValues, listOf(id, same)) }
        mapUniqueExpect(checked) { HolderDto(directValues, bridged = listOf(id, same)) }
        mapUniqueExpect(checked) { HolderDto(directValues, nested = mapOf("bad" to listOf(id, same))) }
        mapUniqueExpect(checked) { HolderDto(directValues, defaultNested = listOf(listOf(id, same))) }
        val badValues = "[{\"a\":\"one\",\"b\":null},{\"b\":null,\"a\":\"one\"}]"
        val bad = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("values" to codec.parseToJsonElement(badValues))).toString()
        mapUniqueExpect(checked) { codec.decodeFromString<HolderDto>(bad) }
        val badBridge = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("bridged" to codec.parseToJsonElement(badValues))).toString()
        val badInstants = "[{\"a\":\"2026-01-02T03:04:05Z\",\"b\":null},{\"b\":null,\"a\":\"2026-01-02T05:04:05+02:00\"}]"
        val badSpecial = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("instants" to codec.parseToJsonElement(badInstants))).toString()
        mapUniqueExpect(checked) { codec.decodeFromString<HolderDto>(badBridge) }
        PatchInputDto(); PatchInputDto(directValues = Patch.Null, values = Patch.Null)

        val badStructuralValues = "[[1,null],[1,null]]"
        val badStructural = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("listRows" to codec.parseToJsonElement(badStructuralValues))).toString()
        var expectedStructural: List<List<Int?>>? = listOf(listOf(1, null), listOf(2))
        var mode = 0
        var calls = 0
        var expectedValues: List<Map<String, String?>?>? = values
        var expectedInstants: List<Map<String, Instant?>?>? = instantMaps
        var expectedPatch = PatchInputDto()
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val jobs = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = MapUniqueEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(mapUniqueBody(request.body)) == codec.parseToJsonElement(encoded))
                    respond(if (mode == 1) bad else if (mode == 2) badSpecial else if (mode == 3) badStructural else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/values" -> {
                    check(codec.parseToJsonElement(mapUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedValues)))
                    respond(if (mode == 1) badValues else codec.encodeToString(expectedValues), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(mapUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedPatch)))
                    respond("", HttpStatusCode.NoContent)
                }
                "/instants" -> {
                    check(codec.parseToJsonElement(mapUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedInstants)))
                    respond(if (mode == 2) badInstants else codec.encodeToString(expectedInstants), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/structural" -> {
                    check(codec.parseToJsonElement(mapUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedStructural)))
                    respond(if (mode == 3) badStructuralValues else codec.encodeToString(expectedStructural), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/watch" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val payload = if (mode == 1) bad else if (mode == 2) badSpecial else if (mode == 3) badStructural else encoded
                    jobs += scope.launch {
                        for (byte in "event: sample\r\ndata: $payload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        awaitCancellation()
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic Map unique URL")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = MapUniqueApiClient(client, "https://example.test/", codec).default
            check(api.echo(holder) == holder)
            check(api.values(values) == values)
            check(api.instants(instantMaps) == instantMaps)
            check(api.structural(expectedStructural) == expectedStructural)
            expectedStructural = null; check(api.structural(null) == null)
            expectedStructural = listOf(listOf(1, null), listOf(2))
            expectedInstants = null; check(api.instants(null) == null); expectedInstants = instantMaps
            expectedValues = null; check(api.values(null) == null)
            expectedValues = values
            for (patch in listOf(PatchInputDto(), PatchInputDto(directValues = Patch.Null, instants = Patch.Null), PatchInputDto(directValues = Patch.Value(directValues), values = Patch.Value(values), instants = Patch.Value(instantMaps)))) {
                expectedPatch = patch; api.patchPatch(patch)
            }
            check(codec.parseToJsonElement(codec.encodeToString(PatchInputDto(directValues = Patch.Null))).jsonObject["directValues"] == JsonNull)
            val before = calls
            var callback = false
            expectedValues = listOf(id, same)
            mapUniqueExpect(checked) { api.values(listOf(id, same)) { callback = true } }
            if (checked) check(!callback && calls == before)
            val beforeSpecial = calls
            callback = false; expectedInstants = duplicateInstantMaps
            mapUniqueExpect(checked) { api.instants(duplicateInstantMaps) { callback = true } }
            if (checked) check(!callback && calls == beforeSpecial)
            expectedInstants = instantMaps
            val beforeStructural = calls
            callback = false
            expectedStructural = listOf(listOf(1, null), listOf(1, null))
            mapUniqueExpect(checked) { api.structural(expectedStructural) { callback = true } }
            if (checked) check(!callback && calls == beforeStructural)
            expectedStructural = listOf(listOf(1, null), listOf(2))
            expectedValues = values
            mode = 1
            mapUniqueExpect(checked) { api.echo(holder) }
            mapUniqueExpect(checked) { api.values(values) }
            mode = 2
            mapUniqueExpect(checked) { api.instants(instantMaps) }
            mapUniqueExpect(checked) { api.echo(holder) }
            mode = 3
            mapUniqueExpect(checked) { api.structural(expectedStructural) }
            mapUniqueExpect(checked) { api.echo(holder) }
            mode = 0
            val count = calls
            val cold = api.watch(); check(calls == count)
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 1
            mapUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 2
            mapUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 3
            mapUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 0
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            check(api.echo(holder) == holder)
        } finally { client.close(); scope.cancel(); jobs.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
        println("MapUnique checked=$checked defaults=$defaults nulls=$nulls Map-value-equality-null-defaults-alias-Patch-preflight-HTTP-liveSSEclose-cold-reuse passed")
    }
}

private suspend fun structuralUniqueContracts(codec: Json, checked: Boolean) {
    val instant = Instant.parse("2026-01-02T03:04:05Z")
    HolderDto(emptyList(), listRows = listOf(null, emptyList(), listOf(null), listOf(1, 2), listOf(2, 1)))
    HolderDto(emptyList(), mixedRows = listOf(null, emptyMap(), mapOf("a" to null), mapOf("a" to emptyList()), mapOf("a" to listOf(null))))
    val first = linkedMapOf("a" to listOf(instant), "b" to null)
    val second = linkedMapOf("b" to null, "a" to listOf(Instant.parse("2026-01-02T05:04:05+02:00")))
    check(first == second)
    mapUniqueExpect(checked) { HolderDto(emptyList(), mixedRows = listOf(first, second)) }
    mapUniqueExpect(checked) { PatchInputDto(mixedRows = Patch.Value(listOf(first, second))) }
    for (rows in listOf(listOf(listOf(1), listOf(1)), listOf(null, null))) {
        mapUniqueExpect(checked) { HolderDto(emptyList(), listRows = rows) }
        mapUniqueExpect(checked) { PatchInputDto(listRows = Patch.Value(rows)) }
    }
    PatchInputDto(listRows = Patch.Null); PatchInputDto(listRows = Patch.Absent)
    mapUniqueExpect(checked) { HolderDto(emptyList(), aliasRows = listOf(listOf(1), listOf(1))) }
    mapUniqueExpect(checked) { HolderDto(emptyList(), floatRows = listOf(listOf(0.0f), listOf(-0.0f))) }
    mapUniqueExpect(checked) { HolderDto(emptyList(), floatRows = listOf(listOf(Float.NaN))) }
    mapUniqueExpect(checked) { HolderDto(emptyList(), doubleMaps = listOf(mapOf("a" to 0.0), mapOf("a" to -0.0))) }
    mapUniqueExpect(checked) { HolderDto(emptyList(), doubleMaps = listOf(mapOf("a" to Double.POSITIVE_INFINITY))) }
    val known = codec.decodeFromString<StateDto>("\"a\"")
    mapUniqueExpect(checked) { HolderDto(emptyList(), enumRows = listOf(listOf(known), listOf(StateDto.Unrecognized("a")))) }
    HolderDto(emptyList(), enumRows = listOf(listOf(known), listOf(StateDto.Unrecognized("other"))))
    for ((field, duplicate) in listOf(
        "listRows" to "[[1,null],[1,null]]",
        "mixedRows" to "[{\"a\":[\"2026-01-02T03:04:05Z\"],\"b\":null},{\"b\":null,\"a\":[\"2026-01-02T05:04:05+02:00\"]}]",
        "floatRows" to "[[16777216],[16777217]]",
        "doubleMaps" to "[{\"a\":0.0},{\"a\":-0.0}]",
        "enumRows" to "[[\"a\"],[\"a\"]]",
        "aliasRows" to "[[1],[1]]"
    )) mapUniqueExpect(checked) { codec.decodeFromString<HolderDto>("{\"directValues\":[],\"$field\":$duplicate}") }
    check(HolderDto(emptyList()).defaultListRows == listOf(listOf(1), listOf(2)))
    check(`DefaultStructuralDto default`() == listOf(listOf(1), listOf(2)))
    val a = listOf(0, 31)
    val b = listOf(1, 0)
    check(a.hashCode() == b.hashCode() && a != b)
    HolderDto(emptyList(), listRows = listOf(a, b))
    HolderDto(emptyList(), mixedRows = listOf(mapOf("a" to listOf(instant)), mapOf("a" to listOf(Instant.parse("2026-01-02T03:04:05.000000001Z")))))
    println("Structural unique typed-list-map-alias-float-enum-null-defaults-Patch passed checked=$checked")
}
