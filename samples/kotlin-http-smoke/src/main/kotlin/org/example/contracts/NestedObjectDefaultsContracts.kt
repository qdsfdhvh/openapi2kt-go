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
import org.example.nestedobject.NestedObject
import org.example.nestedobject.api.models.PatchHolderInputDto
import org.example.nestedobject.api.models.WatchHoldersEventDto
import org.example.nestedobject.models.HolderDto
import org.example.nestedobject.models.Patch
import org.example.nestedobject.models.PatchInputDto
import org.example.nestedobject.models.RequiredNullableCfgDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NestedObjectEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nestedObjectReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun nestedObjectBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nestedObjectBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected object default request body $body")
}

private fun nestedObjectCfg(value: HolderDto.Cfg?): Map<String, Any?> = linkedMapOf(
    "name" to value?.name, "age" to value?.age, "status" to value?.status,
    "code" to value?.code, "when" to value?.`when`, "tags" to value?.tags,
    "counts" to value?.counts, "child" to value?.child, "childFlag" to value?.child?.flag,
    "leaf" to value?.leaf?.z, "note" to value?.note, "composed" to value?.composed, "linked" to value?.linked,
    "viaAlias" to value?.viaAlias, "aliasNull" to value?.aliasNull,
    "owned" to value?.owned, "ownedAlias" to value?.ownedAlias, "ownedRow" to value?.ownedRow, "ownedVariant" to value?.ownedVariant,
)

private fun nestedObjectHolderFields(value: HolderDto): Map<String, Any?> = linkedMapOf(
    "cfg" to nestedObjectCfg(value.cfg),
    "requiredCfg" to value.requiredCfg.id,
    "allOfCfg" to value.allOfCfg,
    "allOfNull" to value.allOfNull,
    "rows" to value.rows,
)

private fun nestedObjectFresh(left: Any?, right: Any?) {
    check(left == right)
    when (left) {
        is List<*> -> { check(left !== right); right as List<*>; left.indices.forEach { nestedObjectFresh(left[it], right[it]) } }
        is Map<*, *> -> { check(left !== right); right as Map<*, *>; left.keys.forEach { nestedObjectFresh(left[it], right[it]) } }
    }
}

suspend fun nestedObjectDefaultsContracts(checked: Boolean) {
    val value = HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x"))
    check(nestedObjectCfg(value.cfg) == nestedObjectCfg(HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x")).cfg))
    check(value.cfg?.name == "outer" && value.cfg?.age == 9)
    check(value.cfg?.status == HolderDto.Cfg.Status.ON && value.cfg?.code == 2)
    check(value.cfg?.`when`?.toString() == "2024-01-02")
    check(value.cfg?.tags == listOf("a", "b") && value.cfg?.counts == mapOf("k" to 1))
    check(value.cfg?.child?.flag == false && value.cfg?.note == null)
    check(value.cfg?.leaf?.z == "deep")
    check(value.cfg?.composed?.label == "outer member" && value.cfg?.composed?.count == 3 && value.cfg?.composed?.child?.flag == false)
    check(value.cfg?.linked?.label == "outer linked" && value.cfg?.linked?.count == 2 && value.cfg?.linked?.child?.flag == true)
    check(value.cfg?.viaAlias?.label == "outer alias" && value.cfg?.viaAlias?.count == 2 && value.cfg?.viaAlias?.child?.flag == true)
    check(value.cfg?.aliasNull == null)
    check(value.cfg?.owned?.label == "outer owned" && value.cfg?.owned?.count == 2 && value.cfg?.owned?.child?.flag == true)
    check(value.cfg?.ownedAlias == null)
    check(value.cfg?.ownedRow?.label == "outer row" && value.cfg?.ownedRow?.count == 2)
    check(value.cfg?.ownedVariant?.label == "outer variant" && value.cfg?.ownedVariant?.count == 2)
    check(value.requiredCfg.id == "x")
    check(value.allOfCfg?.z == "merged" && value.allOfCfg?.count == 5 && value.allOfCfg?.meta?.flag == true)
    check(value.allOfNull == null)
    val other = HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x"))
    check(value.allOfCfg == other.allOfCfg && value.allOfCfg !== other.allOfCfg)
    check(value.allOfCfg?.meta !== other.allOfCfg?.meta)
    check(value.cfg?.composed == other.cfg?.composed && value.cfg?.composed !== other.cfg?.composed)
    check(value.cfg?.composed?.child !== other.cfg?.composed?.child)
    check(value.cfg?.linked == other.cfg?.linked && value.cfg?.linked !== other.cfg?.linked)
    check(value.cfg?.linked?.child !== other.cfg?.linked?.child)
    check(value.cfg?.viaAlias == other.cfg?.viaAlias && value.cfg?.viaAlias !== other.cfg?.viaAlias)
    check(value.cfg?.viaAlias?.child !== other.cfg?.viaAlias?.child)
    check(value.cfg?.owned == other.cfg?.owned && value.cfg?.owned !== other.cfg?.owned)
    check(value.cfg?.owned?.child !== other.cfg?.owned?.child)
    check(value.cfg?.ownedRow == other.cfg?.ownedRow && value.cfg?.ownedRow !== other.cfg?.ownedRow)
    check(value.cfg?.ownedVariant == other.cfg?.ownedVariant && value.cfg?.ownedVariant !== other.cfg?.ownedVariant)
    // A required object default keeps its constructor argument and no initializer.
    check(!HolderDto.serializer().descriptor.isElementOptional(1))
    // Fresh nested trees per instance, not one shared mutable default.
    nestedObjectFresh(nestedObjectCfg(HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x")).cfg), nestedObjectCfg(value.cfg))
    // Nested property own defaults apply when the outer default omits them.
    val fresh = HolderDto.Cfg(name = "n")
    check(fresh.age == 7 && fresh.child?.flag == true && fresh.status == null && fresh.code == null)
    check(fresh.composed?.label == "member" && fresh.composed?.count == 2 && fresh.composed?.child?.flag == true)
    check(fresh.linked?.label == "linked member" && fresh.linked?.count == 2 && fresh.linked?.child?.flag == true)
    check(fresh.viaAlias?.label == "alias member" && fresh.viaAlias?.count == 2 && fresh.viaAlias?.child?.flag == true)
    check(fresh.aliasNull == null)
    check(fresh.owned?.label == "owned member" && fresh.owned?.count == 2 && fresh.owned?.child?.flag == true)
    check(fresh.ownedAlias == null && fresh.ownedRow?.label == "row member" && fresh.ownedVariant?.label == "variant member")
    if (checked) {
        nestedObjectReject("enum") { HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x"), cfg = HolderDto.Cfg(name = "n", code = 9)) }
    } else {
        check(HolderDto.Cfg(name = "n", code = 9).code == 9)
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // requiredCfg is required: a missing key fails regardless of Json settings.
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        val missing = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"}}""")
        check(nestedObjectHolderFields(missing) == nestedObjectHolderFields(value))
        val overridden = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"y"},"cfg":{"name":"z"}}""")
        check(overridden.requiredCfg.id == "y" && overridden.cfg?.name == "z" && overridden.cfg?.age == 7 && overridden.cfg?.child?.flag == true)
        check(overridden.cfg?.composed?.label == "member" && overridden.cfg?.composed?.count == 2)
        check(overridden.cfg?.linked?.label == "linked member" && overridden.cfg?.linked?.count == 2)
        check(overridden.cfg?.viaAlias?.label == "alias member" && overridden.cfg?.viaAlias?.count == 2 && overridden.cfg?.aliasNull == null)
        check(overridden.cfg?.owned?.label == "owned member" && overridden.cfg?.ownedRow?.label == "row member" && overridden.cfg?.ownedVariant?.label == "variant member")
        val ownedOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","owned":{"label":"replacement"},"ownedAlias":{"label":"set"},"ownedRow":{"label":"row"},"ownedVariant":{"label":"variant"}}}""")
        check(ownedOverride.cfg?.owned?.label == "replacement" && ownedOverride.cfg?.owned?.count == 2 && ownedOverride.cfg?.owned?.child?.flag == true)
        check(ownedOverride.cfg?.ownedAlias?.label == "set" && ownedOverride.cfg?.ownedAlias?.child?.flag == true)
        check(ownedOverride.cfg?.ownedRow?.label == "row" && ownedOverride.cfg?.ownedVariant?.label == "variant")
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","ownedAlias":null}}""").cfg?.ownedAlias == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","owned":{}}}""") }.isFailure)
        val aliasOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaAlias":{"label":"replacement"},"aliasNull":{"label":"set"}}}""")
        check(aliasOverride.cfg?.viaAlias?.label == "replacement" && aliasOverride.cfg?.viaAlias?.child?.flag == true)
        check(aliasOverride.cfg?.aliasNull?.label == "set" && aliasOverride.cfg?.aliasNull?.count == 2)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaAlias":{}}}""") }.isFailure)
        val linkedOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","linked":{"label":"replacement"}}}""")
        check(linkedOverride.cfg?.linked?.label == "replacement" && linkedOverride.cfg?.linked?.count == 2 && linkedOverride.cfg?.linked?.child?.flag == true)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","linked":null}}""").cfg?.linked == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","linked":{}}}""") }.isFailure)
        val memberOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","composed":{"label":"replacement"}}}""")
        check(memberOverride.cfg?.composed?.label == "replacement" && memberOverride.cfg?.composed?.count == 2 && memberOverride.cfg?.composed?.child?.flag == true)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","composed":null}}""").cfg?.composed == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","composed":{}}}""") }.isFailure)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":null}""").cfg == null)
        val mergedOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"allOfCfg":{"z":"replacement","count":8},"allOfNull":{"a":"set","b":1}}""")
        check(mergedOverride.allOfCfg?.z == "replacement" && mergedOverride.allOfCfg?.count == 8 && mergedOverride.allOfCfg?.meta?.flag == true)
        check(mergedOverride.allOfNull?.a == "set" && mergedOverride.allOfNull?.b == 1)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"allOfCfg":null}""").allOfCfg == null)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), missing))) == nestedObjectHolderFields(missing))
        // Own defaults on the referenced component remain ordinary nullable fields.
        val referenced = codec.decodeFromString<PatchInputDto>("{}")
        check(referenced.cfg?.a == "x" && referenced.cfgNullable == null)
        // Inline request object defaults keep complete validation but Patch.Absent.
        val absent = codec.decodeFromString<PatchHolderInputDto>("""{"required":null}""")
        check(absent.cfg == Patch.Absent && absent.allOfCfg == Patch.Absent && absent.aliasCfg == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), absent) as JsonObject).keys.none { it == "cfg" || it == "allOfCfg" || it == "aliasCfg" })
        val cleared = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"cfg":null,"allOfCfg":null,"aliasCfg":null}""")
        check(cleared.cfg == Patch.Null && cleared.allOfCfg == Patch.Null && cleared.aliasCfg == Patch.Null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["cfg"] === JsonNull)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["allOfCfg"] === JsonNull)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["aliasCfg"] === JsonNull)
        val aliasUpdate = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"member":{"label":"update"},"owned":{"label":"owned update"}}}""").aliasCfg
        check(aliasUpdate is Patch.Value && aliasUpdate.value.member?.label == "update" && aliasUpdate.value.member?.count == 2)
        check(aliasUpdate.value.owned?.label == "owned update" && aliasUpdate.value.owned?.child?.flag == true)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"member":{}}}""") }.isFailure)
        val mergedUpdate = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"allOfCfg":{"a":"u","b":3}}""")
        check(mergedUpdate.allOfCfg == Patch.Value(PatchHolderInputDto.AllOfCfg(a = "u", b = 3)))
        val update = PatchHolderInputDto(required = null, cfg = Patch.Value(PatchHolderInputDto.Cfg(a = "u")))
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), update) as JsonObject)["cfg"] == codec.parseToJsonElement("""{"a":"u"}"""))
        // Required nullable object keeps a required argument; explicit null is null.
        check(codec.decodeFromString<RequiredNullableCfgDto>("""{"value":null}""").value == null)
        check(codec.decodeFromString<RequiredNullableCfgDto>("""{"value":{"n":3}}""").value?.n == 3)
        val nullWire = codec.encodeToJsonElement(RequiredNullableCfgDto.serializer(), RequiredNullableCfgDto(value = null)) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableCfgDto.serializer(), nullWire).value == null)
        nestedObjectWire(codec, value, checked)
    }
}

private suspend fun nestedObjectWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val framesWritten = CopyOnWriteArrayList<CompletableDeferred<Unit>>()
    var payload = codec.encodeToString(HolderDto.serializer(), value)
    var failing = false
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NestedObjectEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nestedObjectBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nestedObjectBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/required" -> {
                check(codec.parseToJsonElement(nestedObjectBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(RequiredNullableCfgDto.serializer(), RequiredNullableCfgDto(value = null))))
                respond(codec.encodeToString(RequiredNullableCfgDto.serializer(), RequiredNullableCfgDto(value = null)), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
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
            else -> error("Unexpected nested object default path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = NestedObject(http, "http://example.test/", codec).defaults
        check(nestedObjectHolderFields(api.sendHolder(value) { callbacks++ }) == nestedObjectHolderFields(value))
        expectedPatch = codec.encodeToJsonElement(PatchHolderInputDto.serializer(), PatchHolderInputDto(required = null)) as JsonObject
        api.patchHolder(PatchHolderInputDto(required = null)) { callbacks++ }
        check(api.sendRequired(RequiredNullableCfgDto(value = null)) { callbacks++ }.value == null)
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watchHolders { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        val event = (withTimeout(10000) { stream.toList() }.single() as WatchHoldersEventDto.Holder).data
        check(nestedObjectHolderFields(event) == nestedObjectHolderFields(value))
        if (checked) {
            payload = """{"requiredCfg":{"id":"x"},"cfg":{"name":"n","code":9}}"""
            failing = true
            val beforeFailure = calls
            nestedObjectReject("enum") { api.sendHolder(value) { callbacks++ } }
            check(calls == beforeFailure + 1 && callbacks == calls)
            nestedObjectReject("enum") { withTimeout(10000) { api.watchHolders { callbacks++ }.toList() } }
            check(calls == beforeFailure + 2 && callbacks == calls)
            withTimeout(10000) { framesWritten.last().await() }
            check(channels.last().isClosedForRead) { "Failed full-frame still-open stream was not closed" }
            payload = codec.encodeToString(HolderDto.serializer(), value)
            failing = false
            check(nestedObjectHolderFields(api.sendHolder(value) { callbacks++ }) == nestedObjectHolderFields(value))
            val recovered = (withTimeout(10000) { api.watchHolders { callbacks++ }.toList() }.single() as WatchHoldersEventDto.Holder).data
            check(nestedObjectHolderFields(recovered) == nestedObjectHolderFields(value))
        } else {
            payload = """{"requiredCfg":{"id":"x"},"cfg":{"name":"n","code":9}}"""
            val unknown = api.sendHolder(value) { callbacks++ }
            check(unknown.cfg?.code == 9)
            payload = codec.encodeToString(HolderDto.serializer(), value)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
    println("Nested object defaults checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
}
