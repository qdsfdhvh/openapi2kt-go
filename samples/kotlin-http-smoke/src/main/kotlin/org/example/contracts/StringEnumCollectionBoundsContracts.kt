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
import org.example.stringenumbounds.StringEnumBounds
import org.example.stringenumbounds.api.models.PatchInputDto
import org.example.stringenumbounds.api.models.WatchEventDto
import org.example.stringenumbounds.models.HolderDto
import org.example.stringenumbounds.models.Patch
import org.example.stringenumbounds.models.RequiredMapsDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StringEnumBoundsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun stringEnumBoundsReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected $rule failure" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun stringEnumBoundsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> stringEnumBoundsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun stringEnumCollectionBoundsContracts(checked: Boolean) {
    val value = HolderDto(required = listOf(HolderDto.RequiredInner.TWO), kotlin = "shadow", kotlinx = "shadow", labelsInner = "shadow")
    check(value.codes == listOf(HolderDto.CodesInner.ONE, null, HolderDto.CodesInner.TWO))
    check(value.labels == mapOf("a/~" to HolderDto.LabelsValue.YES, "\$ref" to HolderDto.LabelsValue.NO))
    check(value.special?.single()?.serialName == "quote\"\$😀")
    check(value.empty == emptyList<HolderDto.EmptyInner>() && value.emptyMap == emptyMap<String, HolderDto.EmptyMapValue>() && value.nil == null)
    check(HolderDto.Nested().labels == mapOf("one" to HolderDto.Nested.LabelsValue.NESTED, "nil" to null))
    check(HolderDto.CodesInner.entries.map { it.serialName } == listOf("one", "two", "x", "toolong", "BAD"))
    check(HolderDto.CodesInner.fromSerialName("future") == HolderDto.CodesInner.Unrecognized("future"))
    check(HolderDto.CodesInner.serializer().descriptor.serialName == "org.example.stringenumbounds.models.HolderDto.CodesInner")
    val second = HolderDto(required = value.required)
    check(value.codes !== second.codes && value.labels !== second.labels && value.empty !== second.empty && value.emptyMap !== second.emptyMap)
    val future = HolderDto(required = listOf(HolderDto.RequiredInner.Unrecognized("future")), codes = listOf(null, HolderDto.CodesInner.Unrecognized("future")), labels = mapOf("future" to HolderDto.LabelsValue.Unrecognized("future")), bareCodes = listOf(HolderDto.BareCodesInner.Unrecognized("future")), bareLabels = mapOf("substring" to HolderDto.BareLabelsValue.Unrecognized("futurecat"), "nil" to null))
    val invalid = listOf("x" to "minLength", "toolong" to "maxLength", "BAD" to "pattern", "!" to "minLength", "futurelong" to "maxLength", "FUTURE" to "pattern")
    if (checked) {
        for ((bad, rule) in invalid) {
            stringEnumBoundsReject(rule) { HolderDto(required = listOf(HolderDto.RequiredInner.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, codes = listOf(HolderDto.CodesInner.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, labels = mapOf("a" to HolderDto.LabelsValue.fromSerialName(bad))) }
        }
        stringEnumBoundsReject("minItems") { HolderDto(required = emptyList()) }
        stringEnumBoundsReject("maxItems") { HolderDto(required = value.required, codes = List(4) { HolderDto.CodesInner.ONE }) }
        stringEnumBoundsReject("maxProperties") { HolderDto(required = value.required, labels = mapOf("a" to HolderDto.LabelsValue.YES, "b" to HolderDto.LabelsValue.YES, "c" to HolderDto.LabelsValue.YES)) }
        stringEnumBoundsReject("minLength") { HolderDto(required = value.required, bareCodes = listOf(HolderDto.BareCodesInner.X)) }
        stringEnumBoundsReject("pattern") { HolderDto(required = value.required, bareLabels = mapOf("a" to HolderDto.BareLabelsValue.BAD)) }
        stringEnumBoundsReject("minLength") { PatchInputDto(required = null, codes = Patch.Value(listOf(PatchInputDto.CodesInner.X))) }
        stringEnumBoundsReject("minLength") { PatchInputDto(required = null, labels = Patch.Value(mapOf("a" to PatchInputDto.LabelsValue.Unrecognized("x")))) }
    } else {
        check(HolderDto(required = listOf(HolderDto.RequiredInner.X), codes = listOf(HolderDto.CodesInner.BAD)).codes?.single() == HolderDto.CodesInner.BAD)
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // The vocabulary codec remains open and independent of field bounds.
        check(codec.decodeFromString<HolderDto.CodesInner>("\"x\"") == HolderDto.CodesInner.X)
        check(codec.decodeFromString<HolderDto.CodesInner>("\"FUTURE\"") == HolderDto.CodesInner.Unrecognized("FUTURE"))
        check(codec.encodeToString(HolderDto.CodesInner.serializer(), HolderDto.CodesInner.X) == "\"x\"")
        val missing = codec.decodeFromString<HolderDto>("""{"required":["two"]}""")
        check(missing.codes == value.codes && missing.labels == value.labels)
        stringEnumBoundsReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"codes":null,"labels":null}""").let { it.codes == null && it.labels == null })
        check(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), future)) == future)
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"bareLabels":{"a":"hotdog","nil":null}}""").bareLabels?.get("a") == HolderDto.BareLabelsValue.HOTDOG)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("codes") == defaults && encoded.containsKey("labels") == defaults)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        val requiredMaps = RequiredMapsDto(labels = mapOf("a" to RequiredMapsDto.LabelsValue.YES), nullableLabels = null)
        check(codec.decodeFromJsonElement(RequiredMapsDto.serializer(), codec.encodeToJsonElement(RequiredMapsDto.serializer(), requiredMaps)) == requiredMaps)
        stringEnumBoundsReject { codec.decodeFromString<RequiredMapsDto>("{}") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.codes == Patch.Absent && absent.labels == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "codes" || it == "labels" })
        val update = PatchInputDto(required = listOf(PatchInputDto.RequiredInner.TWO), codes = Patch.Value(listOf(PatchInputDto.CodesInner.Unrecognized("future"))), labels = Patch.Value(mapOf("yes" to PatchInputDto.LabelsValue.YES, "nil" to null)))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val explicitNull = codec.decodeFromString<PatchInputDto>("""{"required":["two"],"codes":null,"labels":null}""")
        check(explicitNull.codes == Patch.Null && explicitNull.labels == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), explicitNull) as JsonObject)["codes"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        stringEnumBoundsReject { codec.decodeFromString<HolderDto>("""{"required":[null]}""") }
        stringEnumBoundsReject { codec.decodeFromString<HolderDto>("""{"required":["two"],"labels":{"bad":1}}""") }
        if (checked) {
            for ((bad, rule) in invalid) for (field in listOf("required", "codes", "labels")) {
                val payload = if (field == "labels") """{"required":["two"],"labels":{"a":"$bad"}}""" else if (field == "required") """{"required":["$bad"]}""" else """{"required":["two"],"codes":["$bad"]}"""
                stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>(payload) }
                stringEnumBoundsReject(rule) { codec.decodeFromJsonElement(HolderDto.serializer(), codec.parseToJsonElement(payload)) }
            }
            stringEnumBoundsReject("minItems") { PatchInputDto(required = null, codes = Patch.Value(emptyList())) }
        }
        stringEnumBoundsWire(codec, missing, checked, invalid)
        println("String enum collection bounds defaults=$defaults nulls=$nulls checked=$checked ctor-serde-Patch-HTTP-fragmented-SSE passed")
    }
}

private suspend fun stringEnumBoundsWire(codec: Json, value: HolderDto, checked: Boolean, invalid: List<Pair<String, String>>) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":["two"]}"""
    var failing = false
    var calls = 0
    var expectedHolder = value
    var expectedPatch = JsonObject(emptyMap())
    val engine = StringEnumBoundsEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), expectedHolder))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == expectedPatch)
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
            else -> error("Unexpected String enum bounds path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = StringEnumBounds(http, "http://example.test/", codec).defaults
        check(api.send(value).codes == value.codes)
        if (checked) {
            val beforeInvalid = calls
            for ((bad, rule) in invalid) {
                stringEnumBoundsReject(rule) { api.send(HolderDto(required = listOf(HolderDto.RequiredInner.fromSerialName(bad)))) }
            }
            check(calls == beforeInvalid)
        }
        for (patch in listOf(PatchInputDto(required = null), PatchInputDto(required = null, codes = Patch.Null, labels = Patch.Null), PatchInputDto(required = null, codes = Patch.Value(listOf(PatchInputDto.CodesInner.Unrecognized("future"))), labels = Patch.Value(mapOf("nil" to null))))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.codes == value.codes) }
        payload = """{"required":["future"],"codes":[null,"future"],"labels":{"new":"future"}}"""
        check(api.send(value).required.single() == HolderDto.RequiredInner.Unrecognized("future"))
        expectedHolder = codec.decodeFromString<HolderDto>(payload)
        check(api.send(expectedHolder) == expectedHolder)
        expectedHolder = value
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.labels?.get("new") == HolderDto.LabelsValue.Unrecognized("future"))
        val failures = mutableListOf<Pair<String, String?>>("""{"required":[null]}""" to null, """{"required":["two"],"labels":{"bad":1}}""" to null)
        if (checked) {
            for ((bad, rule) in invalid) {
                failures += """{"required":["$bad"]}""" to rule
                failures += """{"required":["two"],"labels":{"a":"$bad"}}""" to rule
            }
            failures += """{"required":[]}""" to "minItems"
            failures += """{"required":["two"],"codes":["one","one","one","one"]}""" to "maxItems"
        }
        for ((bad, rule) in failures) {
            payload = bad; failing = true
            stringEnumBoundsReject(rule) { api.send(value) }
            stringEnumBoundsReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":["two"]}"""; failing = false
            check(api.send(value).codes == value.codes)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.codes == value.codes)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
