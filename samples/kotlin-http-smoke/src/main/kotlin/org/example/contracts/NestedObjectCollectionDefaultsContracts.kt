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
import kotlinx.coroutines.CompletableDeferred
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
import org.example.colobject.ColObject
import org.example.colobject.api.models.PatchHolderInputDto
import org.example.colobject.api.models.WatchHoldersEventDto
import org.example.colobject.models.HolderDto
import org.example.colobject.models.Patch
import org.example.colobject.models.RequiredCfgDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NestedObjectCollectionEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nestedObjectCollectionReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun nestedObjectCollectionBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nestedObjectCollectionBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected object collection request body $body")
}

private fun nestedObjectCollectionRows(value: HolderDto): Map<String, Any?> = linkedMapOf(
    "rows" to value.rows?.map { listOf(it.name, it.age, it.kind, it.code) },
    "maps" to value.maps?.mapValues { it.value.v },
    "mixed" to value.mixed?.map { entry -> entry.mapValues { it.value.z } },
    "nilRows" to value.nilRows?.map { it.n },
    "requiredRows" to value.requiredRows.map { it.k },
    "scalars" to value.scalars,
)

suspend fun nestedObjectCollectionDefaultsContracts(checked: Boolean) {
    val requiredRows = listOf(HolderDto.RequiredRowsInner(k = "v"))
    val value = HolderDto(requiredRows = requiredRows)
    check(value.rows?.map { it.name } == listOf("a", "b"))
    check(value.rows?.get(0)?.age == 7 && value.rows?.get(1)?.age == 3)
    check(value.rows?.get(0)?.kind == HolderDto.RowsInner.Kind.A && value.rows?.get(1)?.code == 2)
    check(value.maps == mapOf("k" to HolderDto.MapsValue(v = 1)))
    check(value.mixed == listOf(mapOf("a" to HolderDto.MixedInnerValue(z = "x"))))
    check(value.nilRows == null && value.scalars == listOf("a", "b"))
    check(!HolderDto.serializer().descriptor.isElementOptional(4))
    check(HolderDto(requiredRows = requiredRows).rows !== value.rows)
    if (checked) {
        nestedObjectCollectionReject("enum") { HolderDto(requiredRows = requiredRows, rows = listOf(HolderDto.RowsInner(name = "n", code = 9))) }
    } else {
        check(HolderDto(requiredRows = requiredRows, rows = listOf(HolderDto.RowsInner(name = "n", code = 9))).rows?.single()?.code == 9)
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        val missing = codec.decodeFromString<HolderDto>("""{"requiredRows":[{"k":"v"}]}""")
        check(nestedObjectCollectionRows(missing) == nestedObjectCollectionRows(value))
        val overridden = codec.decodeFromString<HolderDto>("""{"requiredRows":[{"k":"v"}],"rows":[{"name":"z"}],"maps":{}}""")
        check(overridden.rows?.size == 1 && overridden.rows?.get(0)?.name == "z" && overridden.rows?.get(0)?.age == 7 && overridden.maps == emptyMap<String, HolderDto.MapsValue>())
        check(codec.decodeFromString<HolderDto>("""{"requiredRows":[{"k":"v"}],"nilRows":null}""").nilRows == null)
        check(nestedObjectCollectionRows(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), missing))) == nestedObjectCollectionRows(missing))
        val absent = codec.decodeFromString<PatchHolderInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), absent) as JsonObject).keys.none { it == "rows" })
        val cleared = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"rows":null}""")
        check(cleared.rows == Patch.Null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["rows"] === JsonNull)
        val update = PatchHolderInputDto(required = null, rows = Patch.Value(listOf(PatchHolderInputDto.RowsInner(x = "u"))))
        check(((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), update) as JsonObject)["rows"] as List<*>) == listOf(codec.parseToJsonElement("""{"x":"u"}""")))
        check(codec.decodeFromString<RequiredCfgDto>("""{"rows":[{"r":1},{}]}""").rows.map { it.r } == listOf(1, null))
        nestedObjectCollectionWire(codec, value, checked)
    }
}

private suspend fun nestedObjectCollectionWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val framesWritten = CopyOnWriteArrayList<CompletableDeferred<Unit>>()
    val requiredValue = RequiredCfgDto(rows = listOf(RequiredCfgDto.RowsInner(r = 1)))
    var payload = codec.encodeToString(HolderDto.serializer(), value)
    var failing = false
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NestedObjectCollectionEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nestedObjectCollectionBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nestedObjectCollectionBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/required" -> {
                val body = codec.encodeToString(RequiredCfgDto.serializer(), requiredValue)
                check(codec.parseToJsonElement(nestedObjectCollectionBody(request.body)) == codec.parseToJsonElement(body))
                respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                val written = CompletableDeferred<Unit>()
                channels += channel
                framesWritten += written
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    written.complete(Unit)
                    if (holdOpen) awaitCancellation()
                } catch (cause: Throwable) { written.completeExceptionally(cause); throw cause } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected nested object collection path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ColObject(http, "http://example.test/", codec).defaults
        check(nestedObjectCollectionRows(api.sendHolder(value) { callbacks++ }) == nestedObjectCollectionRows(value))
        expectedPatch = codec.encodeToJsonElement(PatchHolderInputDto.serializer(), PatchHolderInputDto(required = null)) as JsonObject
        api.patchHolder(PatchHolderInputDto(required = null)) { callbacks++ }
        check(api.sendRequired(requiredValue) { callbacks++ }.rows.size == 1)
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watchHolders { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        val event = (withTimeout(10000) { stream.toList() }.single() as WatchHoldersEventDto.Holder).data
        check(nestedObjectCollectionRows(event) == nestedObjectCollectionRows(value))
        if (checked) {
            payload = """{"requiredRows":[{"k":"v"}],"rows":[{"name":"n","code":9}]}"""
            failing = true
            val beforeFailure = calls
            nestedObjectCollectionReject("enum") { api.sendHolder(value) { callbacks++ } }
            check(calls == beforeFailure + 1 && callbacks == calls)
            nestedObjectCollectionReject("enum") { withTimeout(10000) { api.watchHolders { callbacks++ }.toList() } }
            check(calls == beforeFailure + 2 && callbacks == calls)
            withTimeout(10000) { framesWritten.last().await() }
            check(channels.last().isClosedForRead) { "Failed full-frame still-open stream was not closed" }
            payload = codec.encodeToString(HolderDto.serializer(), value)
            failing = false
            check(nestedObjectCollectionRows(api.sendHolder(value) { callbacks++ }) == nestedObjectCollectionRows(value))
            val recovered = (withTimeout(10000) { api.watchHolders { callbacks++ }.toList() }.single() as WatchHoldersEventDto.Holder).data
            check(nestedObjectCollectionRows(recovered) == nestedObjectCollectionRows(value))
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
    println("Nested object collection defaults checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
}
