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
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.uniondef.UnionDef
import org.example.uniondef.api.models.PatchClosedInputDto
import org.example.uniondef.api.models.WatchHoldersEventDto
import org.example.uniondef.models.ClosedDto
import org.example.uniondef.models.ClosedNullableDto
import org.example.uniondef.models.ComposedUnionDto
import org.example.uniondef.models.HolderDto
import org.example.uniondef.models.Patch
import org.example.uniondef.models.RefObjUnionDto
import org.example.uniondef.models.StringRefUnionDto
import org.example.uniondef.models.RequiredCfgDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class UnionFieldDefaultsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun unionFieldDefaultsReject(block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure" }
}

private fun unionFieldDefaultsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> unionFieldDefaultsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected union default request body $body")
}

private fun unionFieldDefaultsFields(value: HolderDto): Map<String, Any?> = linkedMapOf(
    "str" to value.str, "i" to value.i, "n" to value.n, "b" to value.b,
    "maybe" to value.maybe, "requiredUnion" to value.requiredUnion,
    "pinEnum" to value.pinEnum, "pinConst" to value.pinConst,
    "rows" to value.rows, "rowMap" to value.rowMap, "refPin" to value.refPin,
    "objUnion" to value.objUnion, "constUnion" to value.constUnion, "presUnion" to value.presUnion,
    "refUnion" to value.refUnion, "refObjUnion" to value.refObjUnion,
    "composedInline" to value.composedInline, "composedRef" to value.composedRef,
    "closed" to value.closed, "closedNullable" to value.closedNullable, "closedNil" to value.closedNil,
)

suspend fun unionFieldDefaultsContracts(checked: Boolean) {
    val value = HolderDto(requiredUnion = HolderDto.StringRequiredUnion("r"))
    check(value.str is HolderDto.StringStr && (value.str as HolderDto.StringStr).value == "x")
    check(value.i is HolderDto.IntI && (value.i as HolderDto.IntI).value == 3L)
    check(value.n is HolderDto.NumberN && (value.n as HolderDto.NumberN).value == 1.5)
    check(value.b is HolderDto.BooleanB && (value.b as HolderDto.BooleanB).value)
    check(value.maybe == null && (value.requiredUnion as HolderDto.StringRequiredUnion).value == "r")
    check(value.pinEnum is HolderDto.StringPinEnum && (value.pinEnum as HolderDto.StringPinEnum).value == "a")
    check(value.pinConst is HolderDto.StringPinConst && (value.pinConst as HolderDto.StringPinConst).value == "fix")
    check(value.rows?.get(0) is HolderDto.StringRowsInner && (value.rows?.get(0) as HolderDto.StringRowsInner).value == "x")
    check(value.rows?.get(1) is HolderDto.IntRowsInner && (value.rows?.get(1) as HolderDto.IntRowsInner).value == 1L)
    check(value.rowMap?.get("k") is HolderDto.StringRowMapValue && (value.rowMap?.get("k") as HolderDto.StringRowMapValue).value == "x")
    check(value.refPin is HolderDto.StringRefPin && (value.refPin as HolderDto.StringRefPin).value == "s1")
    check(value.objUnion is HolderDto.ObjUnion.A && (value.objUnion as HolderDto.ObjUnion.A).a == "x")
    check(value.constUnion is HolderDto.ConstUnion.A && (value.constUnion as HolderDto.ConstUnion.A).a == "x")
    check(value.presUnion is HolderDto.PresUnion.Variant1 && (value.presUnion as HolderDto.PresUnion.Variant1).a == "x")
    check(value.refUnion is StringRefUnionDto && (value.refUnion as StringRefUnionDto).value == "x")
    check(value.refObjUnion is RefObjUnionDto.A && (value.refObjUnion as RefObjUnionDto.A).a == "x")
    check(value.composedInline is HolderDto.ComposedInline.Note && (value.composedInline as HolderDto.ComposedInline.Note).body == "x")
    check(value.composedRef is ComposedUnionDto.Count4 && (value.composedRef as ComposedUnionDto.Count4).count == 7)
    check(!HolderDto.serializer().descriptor.isElementOptional(5))
    val closed = value.closed as ClosedDto.Variant1
    check(closed.value == "closed" && closed.age == 7 && closed.meta?.flag == true)
    check((value.closedNullable as ClosedNullableDto.Variant2).value == 8 && value.closedNil == null)
    val fresh = HolderDto(requiredUnion = HolderDto.StringRequiredUnion("r")).closed as ClosedDto.Variant1
    check(fresh !== closed && fresh.meta !== closed.meta)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) for ((discriminator, arrays) in listOf("type" to false, "_kind" to false, "type" to true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; classDiscriminator = discriminator; useArrayPolymorphism = arrays }
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        val missing = codec.decodeFromString<HolderDto>("""{"requiredUnion":"r"}""")
        check(unionFieldDefaultsFields(missing) == unionFieldDefaultsFields(value))
        // Schema annotations use Json.Default; the resulting typed value honors
        // this caller's discriminator or array polymorphism on every use.
        val closedWire = codec.encodeToJsonElement(ClosedDto.serializer(), closed)
        if (arrays) check((closedWire as JsonArray).first() == JsonPrimitive("Variant1"))
        else check((closedWire as JsonObject)[discriminator] == JsonPrimitive("Variant1"))
        check(codec.decodeFromJsonElement(ClosedDto.serializer(), closedWire) == closed)
        val replacement = codec.encodeToJsonElement(ClosedDto.serializer(), ClosedDto.Variant2(value = 6))
        val changed = codec.decodeFromJsonElement(HolderDto.serializer(), JsonObject(mapOf("requiredUnion" to JsonPrimitive("r"), "closed" to replacement)))
        check((changed.closed as ClosedDto.Variant2).value == 6)
        check(codec.decodeFromString<HolderDto>("""{"requiredUnion":"r","closed":null}""").closed == null)
        val unknownClosed = if (arrays) JsonArray(listOf(JsonPrimitive("future"), JsonObject(emptyMap())))
            else JsonObject(mapOf(discriminator to JsonPrimitive("future")))
        unionFieldDefaultsReject { codec.decodeFromJsonElement(ClosedDto.serializer(), unknownClosed) }
        val absent = codec.decodeFromString<PatchClosedInputDto>("{}")
        check(absent.closed == Patch.Absent && (codec.encodeToJsonElement(PatchClosedInputDto.serializer(), absent) as JsonObject).isEmpty())
        val cleared = codec.decodeFromString<PatchClosedInputDto>("""{"closed":null}""")
        check(cleared.closed == Patch.Null && (codec.encodeToJsonElement(PatchClosedInputDto.serializer(), cleared) as JsonObject)["closed"] === JsonNull)
        val update = codec.decodeFromJsonElement(PatchClosedInputDto.serializer(), JsonObject(mapOf("closed" to replacement)))
        check((update.closed as Patch.Value<*>).value is ClosedDto.Variant2)
        val overridden = codec.decodeFromString<HolderDto>("""{"requiredUnion":"z","str":1,"i":"s","n":2.5,"b":false}""")
        check(overridden.requiredUnion is HolderDto.StringRequiredUnion && (overridden.requiredUnion as HolderDto.StringRequiredUnion).value == "z")
        check(overridden.str is HolderDto.IntStr && (overridden.str as HolderDto.IntStr).value == 1L)
        check(overridden.i is HolderDto.StringI && (overridden.i as HolderDto.StringI).value == "s")
        check(overridden.n is HolderDto.NumberN && (overridden.n as HolderDto.NumberN).value == 2.5)
        check(overridden.b is HolderDto.BooleanB && !(overridden.b as HolderDto.BooleanB).value)
        check(codec.decodeFromString<HolderDto>("""{"requiredUnion":"r","maybe":null}""").maybe == null)
        check(unionFieldDefaultsFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), missing))) == unionFieldDefaultsFields(missing))
        // A union member that matches no branch fails decoding at runtime.
        unionFieldDefaultsReject { codec.decodeFromString<HolderDto>("""{"requiredUnion":"r","i":[]}""") }
        val required = RequiredCfgDto(u = RequiredCfgDto.StringU("v"))
        check(codec.decodeFromString(RequiredCfgDto.serializer(), codec.encodeToString(RequiredCfgDto.serializer(), required)) == required)
        unionFieldDefaultsWire(codec, value, checked)
    }
}

private suspend fun unionFieldDefaultsWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val framesWritten = CopyOnWriteArrayList<CompletableDeferred<Unit>>()
    var payload = codec.encodeToString(HolderDto.serializer(), value)
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = UnionFieldDefaultsEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(unionFieldDefaultsBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/closed" -> {
                check(codec.parseToJsonElement(unionFieldDefaultsBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/required" -> {
                val body = codec.encodeToString(RequiredCfgDto.serializer(), RequiredCfgDto(u = RequiredCfgDto.StringU("v")))
                check(codec.parseToJsonElement(unionFieldDefaultsBody(request.body)) == codec.parseToJsonElement(body))
                respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                val written = CompletableDeferred<Unit>()
                channels += channel
                framesWritten += written
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    written.complete(Unit)
                } catch (cause: Throwable) { written.completeExceptionally(cause); throw cause } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected union default path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = UnionDef(http, "http://example.test/", codec).defaults
        check(unionFieldDefaultsFields(api.sendHolder(value) { callbacks++ }) == unionFieldDefaultsFields(value))
        check(api.sendRequired(RequiredCfgDto(u = RequiredCfgDto.StringU("v"))) { callbacks++ }.u is RequiredCfgDto.StringU)
        for (patch in listOf(PatchClosedInputDto(), PatchClosedInputDto(closed = Patch.Null), PatchClosedInputDto(closed = Patch.Value(ClosedDto.Variant2(value = 5))))) {
            expectedPatch = codec.encodeToJsonElement(PatchClosedInputDto.serializer(), patch) as JsonObject
            api.patchClosed(patch) { callbacks++ }
        }
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watchHolders { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        val event = (withTimeout(10000) { stream.toList() }.single() as WatchHoldersEventDto.Holder).data
        check(unionFieldDefaultsFields(event) == unionFieldDefaultsFields(value))
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
    println("Union field defaults checked=$checked constructor-serde-HTTP-liveSSE passed")
}
