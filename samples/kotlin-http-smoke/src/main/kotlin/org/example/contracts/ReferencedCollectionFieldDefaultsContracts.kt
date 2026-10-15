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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.referenceddefaults.ReferencedDefaults
import org.example.referenceddefaults.api.models.PatchInputDto
import org.example.referenceddefaults.api.models.WatchEventDto
import org.example.referenceddefaults.models.HolderDto
import org.example.referenceddefaults.models.Patch
import org.example.referenceddefaults.models.RequiredNullableDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun referencedDefaultReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun referencedDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedCollectionFieldDefaultsContracts(checked: Boolean) {
    val value = HolderDto(required = listOf(listOf(2L)), kotlin = "shadow", kotlinx = "shadow")
    check(value.rows == listOf(listOf(Long.MIN_VALUE, null, Long.MAX_VALUE), null, emptyList()))
    check(value.listMaps == listOf(mapOf("z" to null, "a/~" to "quote\"\$😀", "\$ref" to "data"), null, emptyMap()))
    check(value.mapLists == mapOf("z" to null, "b" to emptyList(), "a" to listOf(true, null, false)))
    val actualRows: org.example.referenceddefaults.models.RowsDto? = value.rows
    val actualMaps: org.example.referenceddefaults.models.MapMapsDto? = value.mapMaps
    check(actualRows == value.repeatedRows && actualMaps == value.mapMaps)
    check(value.rows !== value.repeatedRows)
    check(value.mapMaps == mapOf("z" to emptyMap(), "a" to mapOf("z" to null, "a" to 1.0f)))
    check(value.ints == listOf(mapOf("min" to Int.MIN_VALUE, "max" to Int.MAX_VALUE)))
    check(value.doubles == mapOf("one" to listOf(1.25, -0.0)))
    check(value.empty == emptyList<List<String>>() && value.nil == null)
    check(HolderDto.Nested().value == listOf(listOf("nested")))
    val second = HolderDto(required = value.required)
    check(value.rows !== second.rows && value.rows?.first() !== second.rows?.first())
    check(value.mapMaps !== second.mapMaps && value.mapMaps?.get("a") !== second.mapMaps?.get("a"))
    check(value.empty !== second.empty)
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    if (checked) {
        referencedDefaultReject("minItems") { HolderDto(required = emptyList()) }
        referencedDefaultReject("minItems") { HolderDto(required = listOf(emptyList())) }
        referencedDefaultReject("minimum") { HolderDto(required = listOf(listOf(-1L))) }
        referencedDefaultReject("uniqueItems") { HolderDto(required = listOf(listOf(1L, 1L))) }
        referencedDefaultReject("maxItems") { HolderDto(required = value.required, rows = listOf(List(4) { 1L })) }
        referencedDefaultReject("minLength") { HolderDto(required = value.required, listMaps = listOf(mapOf("a" to ""))) }
        referencedDefaultReject("maxProperties") { HolderDto(required = value.required, listMaps = listOf(mapOf("a" to "a", "b" to "b", "c" to "c", "d" to "d"))) }
        referencedDefaultReject("minimum") { HolderDto(required = value.required, mapMaps = mapOf("a" to mapOf("b" to 0.0f))) }
        referencedDefaultReject("maxProperties") { HolderDto(required = value.required, mapMaps = mapOf("a" to mapOf("a" to 1.0f, "b" to 2.0f, "c" to 3.0f))) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire) == requiredNull)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val missing = codec.decodeFromString<HolderDto>("""{"required":[[2]]}""")
        check(missing.rows == value.rows && missing.mapMaps == value.mapMaps && missing.listMaps == value.listMaps)
        referencedDefaultReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":[[2]],"rows":null,"mapMaps":null}""").let { it.rows == null && it.mapMaps == null })
        check(codec.decodeFromString<HolderDto>("""{"required":[[2]],"rows":[null,[],[null,5]],"mapMaps":{"b":{"c":2}}}""").let { it.rows == listOf(null, emptyList(), listOf(null, 5L)) && it.mapMaps == mapOf("b" to mapOf("c" to 2.0f)) })
        check(codec.decodeFromString<HolderDto>("""{"required":[[2]],"mapMaps":{"a":{"a":1,"z":null},"z":{}}}""").mapMaps == missing.mapMaps)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("rows") == defaults && encoded.containsKey("mapMaps") == defaults)
        if (defaults) {
            check(encoded["rows"] == codec.parseToJsonElement("""[[-9223372036854775808,null,9223372036854775807],null,[]]"""))
            check(encoded["mapMaps"] == codec.parseToJsonElement("""{"a":{"a":1.0,"z":null},"z":{}}"""))
        }
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        referencedDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[null]}""") }
        referencedDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[[null]]}""") }
        referencedDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[[2]],"listMaps":[{"bad":1}]}""") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent && absent.labels == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "rows" || it == "labels" })
        val update = PatchInputDto(required = listOf(listOf(2L)), rows = Patch.Value(listOf(listOf(3L))), labels = Patch.Value(mapOf("a" to listOf("yes"))))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":null,"labels":null}""")
        check(cleared.rows == Patch.Null && cleared.labels == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["rows"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        if (checked) {
            referencedDefaultReject("minimum") { codec.decodeFromString<HolderDto>("""{"required":[[-1]]}""") }
            referencedDefaultReject("minItems") { codec.decodeFromString<HolderDto>("""{"required":[[]]}""") }
            referencedDefaultReject("uniqueItems") { codec.decodeFromString<HolderDto>("""{"required":[[1,1]]}""") }
            referencedDefaultReject("minLength") { codec.decodeFromString<HolderDto>("""{"required":[[2]],"listMaps":[{"bad":""}]}""") }
            referencedDefaultReject("minItems") { PatchInputDto(required = null, rows = Patch.Value(listOf(emptyList()))) }
        }
        referencedDefaultWire(codec, missing, checked)
        println("Referenced collection defaults Json defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun referencedDefaultWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":[[2]]}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = ReferencedDefaultEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(referencedDefaultBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(referencedDefaultBody(request.body)) == expectedPatch)
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
        val api = ReferencedDefaults(http, "http://example.test/", codec).defaults
        check(api.send(value).rows == value.rows)
        for (patch in listOf(
            PatchInputDto(required = listOf(listOf(2L))),
            PatchInputDto(required = null, rows = Patch.Null, labels = Patch.Null),
            PatchInputDto(required = listOf(listOf(2L)), rows = Patch.Value(listOf(listOf(3L))), labels = Patch.Value(mapOf("a" to listOf("yes"))))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.rows == value.rows) }
        val invalid = mutableListOf<Pair<String, String?>>(
            """{"required":[[null]]}""" to null,
            """{"required":[[2]],"rows":[{}]}""" to null,
            """{"required":[[2]],"listMaps":[{"bad":false}]}""" to null,
            "{" to null)
        if (checked) invalid += listOf(
            """{"required":[[]]}""" to "minItems",
            """{"required":[[-1]]}""" to "minimum",
            """{"required":[[1,1]]}""" to "uniqueItems",
            """{"required":[[2]],"listMaps":[{"bad":""}]}""" to "minLength",
            """{"required":[[2]],"mapMaps":{"a":{"b":0}}}""" to "minimum",
            """{"required":[[2]],"mapMaps":{"a":{"a":1,"b":2,"c":3}}}""" to "maxProperties",
            """{"required":[[2]],"rows":[[1,2,3,4]]}""" to "maxItems")
        for ((bad, rule) in invalid) {
            payload = bad
            failing = true
            referencedDefaultReject(rule) { api.send(value) }
            referencedDefaultReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":[[2]]}"""
            failing = false
            check(api.send(value).mapMaps == value.mapMaps)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.mapMaps == value.mapMaps)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
