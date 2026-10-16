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
import org.example.refcol.RefCol
import org.example.refcol.api.models.PatchHolderInputDto
import org.example.refcol.api.models.WatchHoldersEventDto
import org.example.refcol.models.AllOfItemDto
import org.example.refcol.models.AllOfOwnerDto
import org.example.refcol.models.AllOfRequiredDto
import org.example.refcol.models.HolderDto
import org.example.refcol.models.ItemDto
import org.example.refcol.models.Patch
import org.example.refcol.models.RequiredCfgDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedObjectCollectionEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun referencedObjectCollectionReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun referencedObjectCollectionBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedObjectCollectionBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected referenced object collection request body $body")
}

private fun referencedObjectCollectionItems(items: List<ItemDto>?): List<List<Any?>>? = items?.map { listOf(it.name, it.age, it.kind, it.code) }

private fun referencedObjectCollectionFields(value: HolderDto): Map<String, Any?> = linkedMapOf(
    "inlineRows" to referencedObjectCollectionItems(value.inlineRows),
    "inlineMap" to referencedObjectCollectionItems(value.inlineMap?.values?.toList()),
    "namedRows" to referencedObjectCollectionItems(value.namedRows),
    "namedMap" to referencedObjectCollectionItems(value.namedMap?.values?.toList()),
    "requiredRows" to referencedObjectCollectionItems(value.requiredRows),
    "nilRows" to referencedObjectCollectionItems(value.nilRows),
    "allOfRows" to value.allOfRows?.map { listOf(it.name, it.age, it.code, it.child?.note) },
    "allOfAlias" to value.allOfAlias?.map { listOf(it.name, it.age) },
    "allOfNested" to value.allOfNested?.map { row -> row.mapValues { (_, item) -> item?.let { listOf(it.name, it.child?.note) } } },
    "allOfNullable" to value.allOfNullable?.map { it?.name },
    "allOfNil" to value.allOfNil?.map { it.name },
)

suspend fun referencedObjectCollectionDefaultsContracts(checked: Boolean) {
    val requiredRows = listOf(ItemDto(name = "r"))
    val value = HolderDto(requiredRows = requiredRows)
    check(value.inlineRows?.map { it.name } == listOf("a", "b"))
    check(value.inlineRows?.get(0)?.age == 7 && value.inlineRows?.get(1)?.age == 3)
    check(value.inlineRows?.get(0)?.kind == ItemDto.Kind.A && value.inlineRows?.get(1)?.code == 2)
    check(value.inlineMap == mapOf("k" to ItemDto(name = "c")))
    check(value.namedRows == listOf(ItemDto(name = "d")) && value.namedMap == mapOf("m" to ItemDto(name = "e")))
    check(value.nilRows == null)
    val all: AllOfItemDto = value.allOfRows!!.single()
    val internal: AllOfOwnerDto.Node = value.allOfNested!!.single().getValue("k")!!
    check(all.name == "all" && all.age == 7 && all.child?.note == "inner")
    check(internal.name == "internal" && internal.child?.note == "inner")
    check(value.allOfAlias?.single()?.name == "alias" && value.allOfNested?.single()?.get("nil") == null)
    check(value.allOfNullable?.map { it?.name } == listOf(null, "nullable") && value.allOfNil == null)
    val fresh = HolderDto(requiredRows = requiredRows)
    check(fresh.allOfRows !== value.allOfRows && fresh.allOfRows?.single() !== all && fresh.allOfRows?.single()?.child !== all.child)
    check(fresh.allOfNested?.single()?.get("k")?.child !== internal.child)
    check(!AllOfRequiredDto.serializer().descriptor.isElementOptional(0))
    check(!HolderDto.serializer().descriptor.isElementOptional(4))
    check(HolderDto(requiredRows = requiredRows).inlineRows !== value.inlineRows)
    if (checked) {
        referencedObjectCollectionReject("enum") { AllOfItemDto(name = "bad", code = 9) }
        referencedObjectCollectionReject("minItems") { HolderDto(requiredRows = requiredRows, allOfRows = emptyList()) }
        referencedObjectCollectionReject("enum") { HolderDto(requiredRows = requiredRows, inlineRows = listOf(ItemDto(name = "n", code = 9))) }
    } else {
        check(HolderDto(requiredRows = requiredRows, inlineRows = listOf(ItemDto(name = "n", code = 9))).inlineRows?.single()?.code == 9)
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<AllOfRequiredDto>("{}") }.isFailure)
        check(codec.decodeFromString<AllOfRequiredDto>("""{"rows":[{"name":"supplied"}]}""").rows.single().age == 7)
        val missing = codec.decodeFromString<HolderDto>("""{"requiredRows":[{"name":"r"}]}""")
        check(referencedObjectCollectionFields(missing) == referencedObjectCollectionFields(value))
        val overridden = codec.decodeFromString<HolderDto>("""{"requiredRows":[{"name":"r"}],"inlineRows":[{"name":"z"}],"namedRows":[]}""")
        check(overridden.inlineRows?.size == 1 && overridden.inlineRows?.get(0)?.name == "z" && overridden.inlineRows?.get(0)?.age == 7 && overridden.namedRows == emptyList<ItemDto>())
        val suppliedAllOf = codec.decodeFromString<HolderDto>("""{"requiredRows":[{"name":"r"}],"allOfRows":[{"name":"wire"}],"allOfNil":null}""")
        check(suppliedAllOf.allOfRows?.single()?.name == "wire" && suppliedAllOf.allOfRows?.single()?.age == 7 && suppliedAllOf.allOfRows?.single()?.child == null && suppliedAllOf.allOfNil == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredRows":[{"name":"r"}],"nilRows":null}""").nilRows == null)
        check(referencedObjectCollectionFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), missing))) == referencedObjectCollectionFields(missing))
        val absent = codec.decodeFromString<PatchHolderInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent && absent.allOfRows == Patch.Absent)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), absent) as JsonObject).keys.none { it == "rows" || it == "allOfRows" })
        val cleared = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"rows":null,"allOfRows":null}""")
        check(cleared.rows == Patch.Null && cleared.allOfRows == Patch.Null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["rows"] === JsonNull)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["allOfRows"] === JsonNull)
        val update = PatchHolderInputDto(required = null, rows = Patch.Value(listOf(ItemDto(name = "u"))), allOfRows = Patch.Value(listOf(AllOfItemDto(name = "update"))))
        check(((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), update) as JsonObject)["allOfRows"] as List<*>).size == 1)
        check(((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), update) as JsonObject)["rows"] as List<*>).size == 1)
        check(codec.decodeFromString<RequiredCfgDto>("""{"rows":[{"name":"a"},{"name":"b"}]}""").rows.map { it.name } == listOf("a", "b"))
        referencedObjectCollectionWire(codec, value, checked)
    }
}

private suspend fun referencedObjectCollectionWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val framesWritten = CopyOnWriteArrayList<CompletableDeferred<Unit>>()
    val requiredValue = RequiredCfgDto(rows = listOf(ItemDto(name = "r")))
    var payload = codec.encodeToString(HolderDto.serializer(), value)
    var failing = false
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = ReferencedObjectCollectionEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(referencedObjectCollectionBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(referencedObjectCollectionBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/required" -> {
                val body = codec.encodeToString(RequiredCfgDto.serializer(), requiredValue)
                check(codec.parseToJsonElement(referencedObjectCollectionBody(request.body)) == codec.parseToJsonElement(body))
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
            else -> error("Unexpected referenced object collection path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = RefCol(http, "http://example.test/", codec).defaults
        check(referencedObjectCollectionFields(api.sendHolder(value) { callbacks++ }) == referencedObjectCollectionFields(value))
        expectedPatch = codec.encodeToJsonElement(PatchHolderInputDto.serializer(), PatchHolderInputDto(required = null)) as JsonObject
        api.patchHolder(PatchHolderInputDto(required = null)) { callbacks++ }
        for (change in listOf(Patch.Null, Patch.Value(listOf(AllOfItemDto(name = "update"))))) {
            val patch = PatchHolderInputDto(required = null, allOfRows = change)
            expectedPatch = codec.encodeToJsonElement(PatchHolderInputDto.serializer(), patch) as JsonObject
            api.patchHolder(patch) { callbacks++ }
        }
        check(api.sendRequired(requiredValue) { callbacks++ }.rows.size == 1)
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watchHolders { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        val event = (withTimeout(10000) { stream.toList() }.single() as WatchHoldersEventDto.Holder).data
        check(referencedObjectCollectionFields(event) == referencedObjectCollectionFields(value))
        if (checked) {
            payload = """{"requiredRows":[{"name":"r"}],"inlineRows":[{"name":"n","code":9}]}"""
            failing = true
            val beforeFailure = calls
            referencedObjectCollectionReject("enum") { api.sendHolder(value) { callbacks++ } }
            check(calls == beforeFailure + 1 && callbacks == calls)
            referencedObjectCollectionReject("enum") { withTimeout(10000) { api.watchHolders { callbacks++ }.toList() } }
            check(calls == beforeFailure + 2 && callbacks == calls)
            withTimeout(10000) { framesWritten.last().await() }
            check(channels.last().isClosedForRead) { "Failed full-frame still-open stream was not closed" }
            payload = """{"requiredRows":[{"name":"r"}],"allOfRows":[{"name":"n","code":9}]}"""
            referencedObjectCollectionReject("enum") { api.sendHolder(value) { callbacks++ } }
            referencedObjectCollectionReject("enum") { withTimeout(10000) { api.watchHolders { callbacks++ }.toList() } }
            withTimeout(10000) { framesWritten.last().await() }
            check(channels.last().isClosedForRead)
            payload = codec.encodeToString(HolderDto.serializer(), value)
            failing = false
            check(referencedObjectCollectionFields(api.sendHolder(value) { callbacks++ }) == referencedObjectCollectionFields(value))
            val recovered = (withTimeout(10000) { api.watchHolders { callbacks++ }.toList() }.single() as WatchHoldersEventDto.Holder).data
            check(referencedObjectCollectionFields(recovered) == referencedObjectCollectionFields(value))
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
    println("Referenced object collection defaults checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
}
