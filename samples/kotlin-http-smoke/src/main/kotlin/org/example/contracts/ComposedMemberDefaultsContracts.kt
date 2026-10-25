@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, io.ktor.utils.io.InternalAPI::class)
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
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.double
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.example.composedmember.ComposedMemberClient
import org.example.composedmember.api.models.PatchInputDto
import org.example.composedmember.api.models.WatchEventDto
import org.example.composedmember.models.HolderDto
import org.example.composedmember.models.LegacyDto
import org.example.composedmember.models.LegacyHolderDto
import org.example.composedmember.models.LossyDto
import org.example.composedmember.models.LossyHolderDto
import org.example.composedmember.models.Patch
import org.example.composedmember.models.StableDto
import org.example.composedmember.models.VocabularyDto
import org.example.composedmember.models.VocabularyHolderDto
import org.example.composedmember.models.WideDto
import org.example.composedmember.models.WideHolderDto

private class ComposedMemberEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun composedMemberBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> composedMemberBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected composed member body $body")
}

private fun composedMemberValue(value: StableDto?): StableDto.A {
    check(value is StableDto.A) { "Default must retain its original selected constructor" }
    check(value.flag == true && value.i32 == Int.MAX_VALUE && value.i64 == Long.MAX_VALUE)
    check(value.fp32 == 2.5f && value.fp64 == 1.0 && value.state?.serialName == "first")
    return value
}

private suspend fun composedMemberFailure(block: suspend () -> Unit) {
    try {
        block()
    } catch (cause: CancellationException) {
        throw cause
    } catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException })
        return
    }
    error("Expected a serialization failure")
}

suspend fun composedMemberDefaultsContracts(checked: Boolean = true) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto()
        val value = composedMemberValue(holder.value)
        check(value.mode?.serialName == "on")
        val own = StableDto.A()
        check(own.mode?.serialName == "off" && own.flag == false)
        check(own.i32 == Int.MIN_VALUE && own.i64 == Long.MIN_VALUE)
        check(own.fp32 == 1.0f && own.fp64 == 2.5 && own.state == null)
        check(holder.value !== HolderDto().value)
        check(composedMemberValue(codec.decodeFromString<HolderDto>("{}").value).mode?.serialName == "on")
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        if (checked) check(runCatching { StableDto.A(i32 = 0) }.isFailure)
        else check(StableDto.A(i32 = 0).i32 == 0)

        val legacy = LegacyHolderDto().value
        check(legacy is LegacyDto.A && legacy.fp32 == 16777216f)
        val manual = LegacyHolderDto(value = LegacyDto.A(fp32 = 16777216f))
        // Fixing the old Unknown default also fixes default equality/presence.
        check(codec.encodeToJsonElement(manual).jsonObject.containsKey("value") == defaults)
        val lossy = LossyHolderDto().value
        check(lossy is LossyDto.A && lossy.fp32 == 16777216f && lossy.fp64 == 9007199254740992.0)
        val full = Json(codec) { encodeDefaults = true }
        val lossyWire = full.encodeToString<LossyDto>(lossy)
        val tree = full.parseToJsonElement(lossyWire).jsonObject
        check(tree.getValue("fp32").jsonPrimitive.double == 16777216.0)
        check(tree.getValue("fp64").jsonPrimitive.double == 9007199254740992.0)
        // Runtime pins still compare JSON text; native rounding is unchanged.
        check(full.decodeFromString<LossyDto>(lossyWire)::class.simpleName?.startsWith("Unknown") == true)

        val wide = WideHolderDto().value
        check(wide is WideDto.A && wide.v0 == 1 && wide.v237 == 1)
        check(WideDto.A().v237 == 0)
        val vocabulary = VocabularyHolderDto().value
        check(vocabulary is VocabularyDto.A && vocabulary.number == 0)
        check(VocabularyDto.A().number == 239)
        check(PatchInputDto().value === Patch.Absent)
        check(codec.decodeFromString<PatchInputDto>("{}").value === Patch.Absent)
        composedMemberWire(codec)
    }
}

private suspend fun composedMemberWire(codec: Json) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val value = HolderDto(value = StableDto.A(mode = StableDto.A.Mode.fromSerialName("other"), flag = true, i32 = Int.MAX_VALUE, i64 = Long.MAX_VALUE, fp32 = 2.5f, fp64 = 1.0, state = StableDto.A.State.fromSerialName("first")), label = "request-label")
    var calls = 0
    var callbacks = 0
    var eventPayload = """{"label":"雪🍃"}"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = ComposedMemberEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(composedMemberBody(request.body)) == codec.encodeToJsonElement(value))
                respond("""{"label":"response-label"}""", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(composedMemberBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(composedMemberBody(request.body)) == codec.encodeToJsonElement(value))
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val frame = "event: member\r\ndata: $eventPayload\r\n\r\n"
                scope.launch {
                    try { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)) }
                    finally { channel.close() }
                }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected composed member path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ComposedMemberClient(http, "http://example.test/", codec).members
        val response = api.send(value) { callbacks++ }
        check(response.label == "response-label")
        composedMemberValue(response.value)
        val before = calls
        val cold = api.watch(input = value) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { cold.toList() }.single() as WatchEventDto.Member
        check(event.data.label == "雪🍃")
        composedMemberValue(event.data.value)
        for ((patch, expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(value = Patch.Null) to JsonObject(mapOf("value" to JsonNull)),
            PatchInputDto(value = Patch.Value(value.value!!)) to JsonObject(mapOf("value" to codec.encodeToJsonElement<StableDto>(value.value!!))),
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        eventPayload = """{"label":{"bad":true}}"""
        composedMemberFailure { withTimeout(10000) { api.watch(input = value) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        composedMemberValue(api.send(value) { callbacks++ }.value)
    } finally {
        http.close()
        scope.cancel()
        channels.forEach { it.close() }
    }
}
