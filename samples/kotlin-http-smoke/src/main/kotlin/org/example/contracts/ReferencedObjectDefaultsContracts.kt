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
import org.example.refobject.RefObject
import org.example.refobject.api.models.PatchHolderInputDto
import org.example.refobject.api.models.WatchHoldersEventDto
import org.example.refobject.models.AllOfValueDto
import org.example.refobject.models.HolderDto
import org.example.refobject.models.InnerDto
import org.example.refobject.models.Patch
import org.example.refobject.models.RequiredAllOfCfgDto
import org.example.refobject.models.RequiredCfgDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedObjectEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun referencedObjectReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun referencedObjectBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedObjectBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected referenced object request body $body")
}

private fun referencedObjectInner(value: InnerDto?): Map<String, Any?> = linkedMapOf(
    "name" to value?.name, "age" to value?.age, "status" to value?.status,
    "code" to value?.code, "when" to value?.`when`, "tags" to value?.tags,
    "counts" to value?.counts, "childFlag" to value?.child?.flag, "note" to value?.note,
)

private fun referencedAllOfFields(value: AllOfValueDto?): Map<String, Any?> = linkedMapOf(
    "name" to value?.name, "age" to value?.age, "n" to value?.n, "childFlag" to value?.child?.flag,
)

private fun referencedObjectHolderFields(value: HolderDto): Map<String, Any?> = linkedMapOf(
    "cfg" to referencedObjectInner(value.cfg),
    "aliasCfg" to referencedObjectInner(value.aliasCfg),
    "requiredCfg" to referencedObjectInner(value.requiredCfg),
    "nilCfg" to referencedObjectInner(value.nilCfg),
    "allOfCfg" to referencedAllOfFields(value.allOfCfg),
    "aliasAllOfCfg" to referencedAllOfFields(value.aliasAllOfCfg),
    "bridgedAllOfCfg" to referencedAllOfFields(value.bridgedAllOfCfg),
    "allOfNil" to value.allOfNil?.name,
)

suspend fun referencedObjectDefaultsContracts(checked: Boolean) {
    val requiredCfg = InnerDto(name = "required")
    val value = HolderDto(requiredCfg = requiredCfg)
    check(value.cfg?.name == "outer" && value.cfg?.age == 9)
    check(value.cfg?.status == InnerDto.Status.ON && value.cfg?.code == 2)
    check(value.cfg?.`when`?.toString() == "2024-01-02")
    check(value.cfg?.tags == listOf("a", "b") && value.cfg?.counts == mapOf("k" to 1))
    check(value.cfg?.child?.flag == false && value.cfg?.note == null)
    check(value.aliasCfg?.name == "alias" && value.aliasCfg?.age == 7 && value.aliasCfg?.child?.flag == true)
    check(value.requiredCfg.name == "required" && value.nilCfg == null)
    check(!HolderDto.serializer().descriptor.isElementOptional(2))
    check(!RequiredAllOfCfgDto.serializer().descriptor.isElementOptional(0))
    val composed: AllOfValueDto = checkNotNull(value.allOfCfg)
    check(composed.name == "composed" && composed.n == 2 && composed.age == 7 && composed.child?.flag == true)
    check(value.aliasAllOfCfg?.name == "alias-composed" && value.aliasAllOfCfg?.n == 3 && value.allOfNil == null)
    check(value.bridgedAllOfCfg?.name == "bridge-composed" && value.bridgedAllOfCfg?.n == 4 && value.bridgedAllOfCfg?.child?.flag == true)
    check(HolderDto(requiredCfg = requiredCfg).allOfCfg !== composed && composed !== value.aliasAllOfCfg)
    check(HolderDto(requiredCfg = requiredCfg).allOfCfg?.child !== composed.child)
    // The referenced and alias fields share one decoded type, not a shared instance.
    check(value.cfg !== value.aliasCfg && HolderDto(requiredCfg = requiredCfg).cfg !== value.cfg)
    if (checked) {
        referencedObjectReject("enum") { HolderDto(requiredCfg = requiredCfg, cfg = InnerDto(name = "n", code = 9)) }
        referencedObjectReject("enum") { AllOfValueDto(name = "bad", n = 0) }
    } else {
        check(AllOfValueDto(name = "unchecked", n = 0).n == 0)
        check(HolderDto(requiredCfg = requiredCfg, cfg = InnerDto(name = "n", code = 9)).cfg?.code == 9)
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        val missing = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"name":"required"}}""")
        check(referencedObjectHolderFields(missing) == referencedObjectHolderFields(value))
        // A supplied value replaces the default; omitted nested keys use their own defaults.
        val overridden = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"name":"r"},"cfg":{"name":"z"}}""")
        check(overridden.cfg?.name == "z" && overridden.cfg?.age == 7 && overridden.cfg?.child?.flag == true)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"name":"r"},"cfg":null}""").cfg == null)
        val composedOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"name":"r"},"allOfCfg":{"name":"replacement","n":8},"aliasAllOfCfg":null}""")
        check(composedOverride.allOfCfg?.name == "replacement" && composedOverride.allOfCfg?.n == 8 && composedOverride.allOfCfg?.child?.flag == true)
        check(composedOverride.aliasAllOfCfg == null)
        check(codec.decodeFromString<RequiredAllOfCfgDto>("""{"value":null}""").value == null)
        check(codec.decodeFromString<RequiredAllOfCfgDto>("""{"value":{"name":"r","n":1}}""").value?.n == 1)
        check(referencedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), missing))) == referencedObjectHolderFields(missing))
        // Inline request object referenced defaults keep complete validation but Patch.Absent.
        val patchSource = """{"required":null}"""
        val absent = codec.decodeFromString<PatchHolderInputDto>(patchSource)
        check(absent.cfg == Patch.Absent && absent.allOfCfg == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), absent) as JsonObject).keys.none { it == "cfg" || it == "allOfCfg" })
        val cleared = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"cfg":null,"allOfCfg":null}""")
        check(cleared.cfg == Patch.Null && cleared.allOfCfg == Patch.Null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["allOfCfg"] === JsonNull)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["cfg"] === JsonNull)
        val update = PatchHolderInputDto(required = null, cfg = Patch.Value(InnerDto(name = "u")), allOfCfg = Patch.Value(AllOfValueDto(name = "u", n = 5)))
        val updateWire = codec.encodeToJsonElement(PatchHolderInputDto.serializer(), update) as JsonObject
        check((updateWire["allOfCfg"] as JsonObject)["n"] == codec.parseToJsonElement("5"))
        check((updateWire["cfg"] as JsonObject)["name"] == codec.parseToJsonElement("\"u\""))
        // Required nullable referenced object keeps a required argument.
        check(codec.decodeFromString<RequiredCfgDto>("""{"value":null}""").value == null)
        check(codec.decodeFromString<RequiredCfgDto>("""{"value":{"name":"v"}}""").value?.name == "v")
        val nullWire = codec.encodeToJsonElement(RequiredCfgDto.serializer(), RequiredCfgDto(value = null)) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredCfgDto.serializer(), nullWire).value == null)
        referencedObjectWire(codec, value, checked)
    }
}

private suspend fun referencedObjectWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val framesWritten = CopyOnWriteArrayList<CompletableDeferred<Unit>>()
    var payload = codec.encodeToString(HolderDto.serializer(), value)
    var failing = false
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = ReferencedObjectEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(referencedObjectBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(referencedObjectBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/required" -> {
                check(codec.parseToJsonElement(referencedObjectBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(RequiredCfgDto.serializer(), RequiredCfgDto(value = null))))
                respond(codec.encodeToString(RequiredCfgDto.serializer(), RequiredCfgDto(value = null)), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
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
            else -> error("Unexpected referenced object path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = RefObject(http, "http://example.test/", codec).defaults
        check(referencedObjectHolderFields(api.sendHolder(value) { callbacks++ }) == referencedObjectHolderFields(value))
        expectedPatch = codec.encodeToJsonElement(PatchHolderInputDto.serializer(), PatchHolderInputDto(required = null)) as JsonObject
        api.patchHolder(PatchHolderInputDto(required = null)) { callbacks++ }
        check(api.sendRequired(RequiredCfgDto(value = null)) { callbacks++ }.value == null)
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watchHolders { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        val event = (withTimeout(10000) { stream.toList() }.single() as WatchHoldersEventDto.Holder).data
        check(referencedObjectHolderFields(event) == referencedObjectHolderFields(value))
        if (checked) for ((rule, invalidPayload) in listOf(
            "enum" to """{"requiredCfg":{"name":"r"},"cfg":{"name":"n","code":9}}""",
            "enum" to """{"requiredCfg":{"name":"r"},"allOfCfg":{"name":"n","n":0}}""",
        )) {
            payload = invalidPayload
            failing = true
            val beforeFailure = calls
            referencedObjectReject(rule) { api.sendHolder(value) { callbacks++ } }
            check(calls == beforeFailure + 1 && callbacks == calls)
            referencedObjectReject(rule) { withTimeout(10000) { api.watchHolders { callbacks++ }.toList() } }
            check(calls == beforeFailure + 2 && callbacks == calls)
            withTimeout(10000) { framesWritten.last().await() }
            check(channels.last().isClosedForRead) { "Failed full-frame still-open stream was not closed" }
            payload = codec.encodeToString(HolderDto.serializer(), value)
            failing = false
            check(referencedObjectHolderFields(api.sendHolder(value) { callbacks++ }) == referencedObjectHolderFields(value))
            val recovered = (withTimeout(10000) { api.watchHolders { callbacks++ }.toList() }.single() as WatchHoldersEventDto.Holder).data
            check(referencedObjectHolderFields(recovered) == referencedObjectHolderFields(value))
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
    println("Referenced object defaults checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
}
