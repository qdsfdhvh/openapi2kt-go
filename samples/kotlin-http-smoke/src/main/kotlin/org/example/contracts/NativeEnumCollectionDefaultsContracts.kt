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
import org.example.enumcollections.EnumCollections
import org.example.enumcollections.api.models.PatchInputDto
import org.example.enumcollections.api.models.WatchEventDto
import org.example.enumcollections.models.HolderDto
import org.example.enumcollections.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class EnumCollectionEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun enumCollectionReject(rule: String, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected $rule failure" }
    check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun enumCollectionBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> enumCollectionBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun nativeEnumCollectionDefaultsContracts(checked: Boolean) {
    val value = HolderDto(required = listOf(2), kotlin = "shadow", kotlinx = "shadow")
    check(value.codes == if (checked) listOf(2, null) else listOf(1, null, 2))
    check(value.flags == mapOf("a/~" to true, "\$ref" to true))
    check(value.longs == listOf(Long.MIN_VALUE, Long.MAX_VALUE))
    check(value.floats == mapOf("half" to 0.5f) && value.doubles == listOf(0.5))
    check(value.empty == emptyList<Boolean>() && value.emptyMap == emptyMap<String, Int>() && value.nil == null)
    check(HolderDto.Nested().values == mapOf("one" to 1, "nil" to null))
    val second = HolderDto(required = listOf(2))
    check(value.codes !== second.codes && value.flags !== second.flags)
    if (checked) {
        enumCollectionReject("enum") { HolderDto(required = listOf(3)) }
        enumCollectionReject("enum") { HolderDto(required = listOf(1), flags = mapOf("no" to false)) }
        enumCollectionReject("enum") { HolderDto(required = listOf(1), doubles = listOf(0.25)) }
        enumCollectionReject("minimum") { HolderDto(required = listOf(1), codes = listOf(1)) }
        enumCollectionReject("minItems") { HolderDto(required = emptyList()) }
        enumCollectionReject("maxProperties") { HolderDto(required = listOf(1), flags = mapOf("a" to true, "b" to true, "c" to true)) }
    } else {
        check(HolderDto(required = listOf(3), codes = listOf(3), flags = mapOf("no" to false)).required == listOf(3))
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("""{"required":[2]}""")
        check(missing.codes == value.codes && missing.flags == value.flags && missing.longs == value.longs)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>("""{"required":[2],"codes":null,"flags":null}""").let { it.codes == null && it.flags == null })
        check(codec.decodeFromString<HolderDto>("""{"required":[2],"codes":[2],"flags":{"new":true}}""").let { it.codes == listOf(2) && it.flags == mapOf("new" to true) })
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("codes") == defaults && encoded.containsKey("flags") == defaults)
        if (defaults) {
            check(encoded["codes"] == codec.parseToJsonElement(if (checked) "[2,null]" else "[1,null,2]"))
            check(encoded["flags"] == codec.parseToJsonElement("""{"a/~":true,"${'$'}ref":true}"""))
        }
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.codes == Patch.Absent && absent.flags == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "codes" || it == "flags" })
        val update = PatchInputDto(required = listOf(2), codes = Patch.Value(listOf(2)), flags = Patch.Value(mapOf("yes" to true)))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val explicitNull = codec.decodeFromString<PatchInputDto>("""{"required":[2],"codes":null,"flags":null}""")
        check(explicitNull.codes == Patch.Null && explicitNull.flags == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), explicitNull) as JsonObject)["codes"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        if (checked) {
            enumCollectionReject("enum") { codec.decodeFromString<HolderDto>("""{"required":[3]}""") }
            enumCollectionReject("enum") { codec.decodeFromString<HolderDto>("""{"required":[2],"flags":{"no":false}}""") }
            enumCollectionReject("enum") { PatchInputDto(required = listOf(2), codes = Patch.Value(listOf(3))) }
            enumCollectionReject("minItems") { PatchInputDto(required = listOf(2), codes = Patch.Value(emptyList())) }
        } else {
            check(codec.decodeFromString<HolderDto>("""{"required":[3],"flags":{"no":false}}""").flags == mapOf("no" to false))
        }
        enumCollectionWire(codec, missing, checked)
        println("Native enum collection defaults Json defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun enumCollectionWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":[2]}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = EnumCollectionEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(enumCollectionBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(enumCollectionBody(request.body)) == expectedPatch)
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
            else -> error("Unexpected enum collection path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = EnumCollections(http, "http://example.test/", codec).defaults
        check(api.send(value).codes == value.codes)
        for (patch in listOf(
            PatchInputDto(required = listOf(2)),
            PatchInputDto(required = listOf(2), codes = Patch.Null, flags = Patch.Null),
            PatchInputDto(required = listOf(2), codes = Patch.Value(listOf(2)), flags = Patch.Value(mapOf("yes" to true)))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.flags == value.flags) }
        if (checked) {
            for ((bad, rule) in listOf(
                """{"required":[3]}""" to "enum",
                """{"required":[2],"flags":{"no":false}}""" to "enum",
                """{"required":[2],"codes":[1]}""" to "minimum",
                """{"required":[]}""" to "minItems"
            )) {
                payload = bad
                failing = true
                enumCollectionReject(rule) { api.send(value) }
                enumCollectionReject(rule) { withTimeout(10000) { api.watch().toList() } }
                check(channels.last().isClosedForRead)
                payload = """{"required":[2]}"""
                failing = false
                check(api.send(value).codes == value.codes)
                check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.codes == value.codes)
            }
        } else {
            payload = """{"required":[3],"flags":{"no":false}}"""
            check(api.send(value).flags == mapOf("no" to false))
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.required == listOf(3))
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
