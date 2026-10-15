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
import kotlinx.serialization.serializer
import org.example.refstringdefaults.models.NullableModesDto
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
import org.example.refstringdefaults.RefStringDefaults
import org.example.refstringdefaults.api.models.PatchInputDto
import org.example.refstringdefaults.api.models.WatchEventDto
import org.example.refstringdefaults.models.HolderDto
import org.example.refstringdefaults.models.InlinesDtoInner
import org.example.refstringdefaults.models.ModeDto
import org.example.refstringdefaults.models.NullableRowsDtoInner
import org.example.refstringdefaults.models.NullableStatusDto
import org.example.refstringdefaults.models.Patch
import org.example.refstringdefaults.models.RequiredNullableDto
import org.example.refstringdefaults.models.StatusDto
import org.example.refstringdefaults.models.TypelessDtoInner

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedStringEnumCollectionDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun referencedStringEnumCollectionDefaultReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun referencedStringEnumCollectionDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedStringEnumCollectionDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedStringEnumCollectionFieldDefaultsContracts(checked: Boolean) {
    val value = HolderDto(required = listOf(listOf(StatusDto.ON)), requiredModes = listOf(NullableStatusDto.OFF), kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow")
    check(value.rows == listOf(listOf(NullableRowsDtoInner.ON, null, NullableRowsDtoInner.OFF), null, emptyList<NullableRowsDtoInner?>()))
    check(value.listMaps == listOf(mapOf("z" to null, "a/~" to NullableStatusDto.OFF, "\$ref" to NullableStatusDto.ON), null, emptyMap()))
    check(value.mapLists == mapOf("z" to null, "b" to emptyList(), "a" to listOf(NullableStatusDto.ON, null, NullableStatusDto.OFF)))
    check(value.mapMaps == mapOf("z" to emptyMap(), "a" to mapOf("z" to null, "a" to NullableStatusDto.ON)))
    check(value.inline == listOf(InlinesDtoInner.ON, InlinesDtoInner.A_Z, InlinesDtoInner.OFF))
    check(value.typeless == listOf(TypelessDtoInner.WIRE, TypelessDtoInner.RAW))
    check(value.modes == listOf(ModeDto.RAW, ModeDto.WIRE))
    check(value.borrowed == listOf(StatusDto.OFF))
    check(value.empty == emptyList<NullableStatusDto?>() && value.nil == null)
    check(HolderDto.Nested().value == listOf(listOf(StatusDto.ON), listOf(StatusDto.OFF, StatusDto.ON)))
    check(serializer<org.example.refstringdefaults.models.InlinesDto>().descriptor.serialName == "kotlin.collections.ArrayList")
    check(serializer<org.example.refstringdefaults.models.MapMapsDto>().descriptor.serialName == "kotlin.collections.LinkedHashMap")
    check(serializer<org.example.refstringdefaults.models.ModesDto>().descriptor.serialName == "kotlin.collections.ArrayList")
    val descriptor = HolderDto.serializer().descriptor
    val requiredIndex = descriptor.getElementIndex("required")
    check(!descriptor.isElementOptional(requiredIndex))
    check(descriptor.getElementDescriptor(requiredIndex).getElementDescriptor(0).getElementDescriptor(0).serialName == "org.example.refstringdefaults.models.StatusDto")
    // Fresh evaluation identity: equal trees are never shared between instances.
    val second = HolderDto(required = listOf(listOf(StatusDto.ON)), requiredModes = listOf(NullableStatusDto.OFF))
    check(value.rows !== second.rows && value.rows?.first() !== second.rows?.first())
    check(value.mapMaps !== second.mapMaps && value.mapMaps?.get("a") !== second.mapMaps?.get("a"))
    check(value.inline !== second.inline && value.typeless !== second.typeless)
    check(value.empty !== second.empty && value.borrowed !== second.borrowed)
    check(value.rows == second.rows && value.inline == second.inline)
    // Supplied values replace defaults and keep the open runtime vocabulary.
    check(RequiredNullableDto(value = null).value == null)
    if (checked) {
        referencedStringEnumCollectionDefaultReject("maxItems") { HolderDto(required = value.required, requiredModes = value.requiredModes, rows = List(4) { null }) }
        referencedStringEnumCollectionDefaultReject("minItems") { RequiredNullableDto(value = emptyList()) }
    }
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"]}""").inline == value.inline)
        check(codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"],"rows":null,"nil":null}""").let { it.rows == null && it.nil == null })
        check(codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"],"modes":["wire","gone"]}""").modes?.last() is ModeDto.Unrecognized)
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire) == requiredNull)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val missing = codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"]}""")
        check(missing.listMaps == value.listMaps && missing.mapLists == value.mapLists && missing.mapMaps == value.mapMaps)
        referencedStringEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"],"inline":null,"typeless":null}""").let { it.inline == null && it.typeless == null })
        check(codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"],"typeless":["gone"]}""").typeless?.single() is TypelessDtoInner.Unrecognized)
        check(codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"],"rows":[["wat"]]}""").rows?.single()?.single() is NullableRowsDtoInner.Unrecognized)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("rows") == defaults && encoded.containsKey("listMaps") == defaults && encoded.containsKey("mapLists") == defaults)
        check(encoded.containsKey("inline") == defaults && encoded.containsKey("typeless") == defaults && encoded.containsKey("modes") == defaults)
        check(encoded.containsKey("empty") == defaults && encoded.containsKey("borrowed") == defaults)
        if (defaults) {
            check(encoded["rows"] == codec.parseToJsonElement("""[["on",null,"off"],null,[]]"""))
            check(encoded["listMaps"] == codec.parseToJsonElement("""[{"${'$'}ref":"on","a/~":"off","z":null},null,{}]"""))
            check(encoded["mapLists"] == codec.parseToJsonElement("""{"a":["on",null,"off"],"b":[],"z":null}"""))
            check(encoded["mapMaps"] == codec.parseToJsonElement("""{"a":{"a":"on","z":null},"z":{}}"""))
            check(encoded["inline"] == codec.parseToJsonElement("""["on","a~z","off"]"""))
            check(encoded["borrowed"] == codec.parseToJsonElement("""["off"]"""))
        }
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        referencedStringEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[null],"requiredModes":["off"]}""") }
        check(codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":[null]}""").requiredModes == listOf(null))
        referencedStringEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[[false]],"requiredModes":["off"]}""") }
        referencedStringEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"],"listMaps":[{"bad":false}]}""") }
        referencedStringEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[["on"]],"requiredModes":["off"],"mapMaps":{"a":{"a":1}}}""") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent && absent.labels == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "rows" || it == "labels" })
        val update = PatchInputDto(required = listOf(listOf(StatusDto.OFF)), rows = Patch.Value(listOf(listOf(StatusDto.ON))), labels = Patch.Value(mapOf("a" to listOf(ModeDto.RAW))))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":null,"labels":null}""")
        check(cleared.rows == Patch.Null && cleared.labels == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["rows"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        referencedStringEnumCollectionDefaultWire(codec, missing, checked)
        println("Referenced string enum collection defaults Json defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun referencedStringEnumCollectionDefaultWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":[["on"]],"requiredModes":["off"]}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = ReferencedStringEnumCollectionDefaultEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(referencedStringEnumCollectionDefaultBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(referencedStringEnumCollectionDefaultBody(request.body)) == expectedPatch)
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
        val api = RefStringDefaults(http, "http://example.test/", codec).defaults
        var callbacks = 0
        val beforeCall = calls
        check(api.send(value, requestBuilder = { callbacks++ }).rows == value.rows)
        check(callbacks == 1 && calls == beforeCall + 1)
        for (patch in listOf(
            PatchInputDto(required = listOf(listOf(StatusDto.ON))),
            PatchInputDto(required = null, rows = Patch.Null, labels = Patch.Null),
            PatchInputDto(required = listOf(listOf(StatusDto.OFF)), rows = Patch.Value(listOf(listOf(StatusDto.ON))), labels = Patch.Value(mapOf("a" to listOf(ModeDto.RAW))))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.rows == value.rows) }
        val invalid = mutableListOf<Pair<String, String?>>(
            """{"required":[null],"requiredModes":["off"]}""" to null,
            """{"required":[[false]],"requiredModes":["off"]}""" to null,
            """{"required":[["on"]],"requiredModes":["off"],"listMaps":[{"bad":false}]}""" to null,
            "{" to null)
        invalid += listOf(
            """{"required":[["on"]],"requiredModes":["off"],"mapMaps":{"a":{"a":1}}}""" to null,
            """{"required":[["on"]],"requiredModes":["off"],"typeless":[true]}""" to null)
        if (checked) invalid += listOf(
            """{"required":[[]],"requiredModes":["off"]}""" to "minItems",
            """{"required":[["on"]],"requiredModes":[]}""" to "minItems",
            """{"required":[["on"]],"requiredModes":["off"],"rows":[[],[],[],[]]}""" to "maxItems")
        for ((bad, rule) in invalid) {
            payload = bad
            failing = true
            val beforeFailure = calls
            val beforeCallbacks = callbacks
            referencedStringEnumCollectionDefaultReject(rule) { api.send(value, requestBuilder = { callbacks++ }) }
            check(calls == beforeFailure + 1 && callbacks == beforeCallbacks + 1)
            referencedStringEnumCollectionDefaultReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":[["on"]],"requiredModes":["off"]}"""
            failing = false
            check(api.send(value).mapMaps == value.mapMaps)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.mapMaps == value.mapMaps)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}
