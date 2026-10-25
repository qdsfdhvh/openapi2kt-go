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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import org.example.allofdefaultoverrides.DefaultOverridesClient
import org.example.allofdefaultoverrides.api.models.PatchInputDto
import org.example.allofdefaultoverrides.api.models.WatchEventDto
import org.example.allofdefaultoverrides.models.NumbersDto
import org.example.allofdefaultoverrides.models.StrictDto
import org.example.allofdefaultoverrides.models.FactoryDto
import org.example.allofdefaultoverrides.models.HolderDto
import org.example.allofdefaultoverrides.models.LowerDto
import org.example.allofdefaultoverrides.models.NullableValuesDto
import org.example.allofdefaultoverrides.models.Patch
import org.example.allofdefaultoverrides.models.RecordDto
import org.example.allofdefaultoverrides.models.WideDto
import org.example.allofdefaultoverrides.models.ExactLongDto
import org.example.allofdefaultoverrides.models.PlainStringDto
import org.example.allofdefaultoverrides.models.`NarrowFactoryDto default`
import org.example.allofdefaultoverrides.models.`NullableFactoryDto default`

private class AllOfDefaultOverridesEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun allOfDefaultOverridesBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> allOfDefaultOverridesBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected allOf numeric request content")
}

private fun allOfDefaultOverrides(value: HolderDto) {
    check(value.value?.value == 1 && value.inline?.value == 1)
    check(value.cfg?.node?.value == 1 && value.rows?.single()?.value == 1)
    check(value.maps?.getValue("k")?.value == 3 && value.empty?.value == 2)
}

private suspend fun allOfDefaultOverridesFailure(block: suspend () -> Unit) {
    try { block() }
    catch (cause: CancellationException) { throw cause }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException })
        return
    }
    error("Expected retained numeric constraint failure")
}

suspend fun allOfDefaultOverridesContracts() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val numbers = NumbersDto.default()
        check(numbers.intValue == Int.MIN_VALUE && numbers.longValue == Long.MIN_VALUE)
        check(numbers.floatValue == 16777216f && numbers.doubleValue == 9007199254740992.0)
        check(numbers.zero?.toBits() == (-0.0).toBits())
        check(codec.decodeFromString<NumbersDto>(codec.encodeToString(numbers)) == numbers)
        check(runCatching { NumbersDto(floatValue = 16777214f) }.isFailure)
        check(runCatching { NumbersDto(doubleValue = 9007199254740990.0) }.isFailure)
        check(StrictDto.default().value == 3 && StrictDto().value == 2)
        for (bad in listOf(0,4)) check(runCatching { StrictDto(bad) }.isFailure)
        val holder = HolderDto(required = RecordDto(id="required",value=2))
        allOfDefaultOverrides(holder)
        check(LowerDto(id="early").value == 0 && RecordDto(id="record").value == 2)
        check(FactoryDto.default().value == 1 && FactoryDto(id="ctor").value == 2)
        check(`NullableFactoryDto default`() == null && `NarrowFactoryDto default`().value == 3)
        check(NullableValuesDto.default().value == null)
        check(PlainStringDto.default().value=="winner")
        check(ExactLongDto.default().value==9007199254740999L)
        check(runCatching{ExactLongDto(value=9007199254740994L)}.isFailure)
        check(runCatching{WideDto(p239="x")}.isFailure)
        check(holder.value?.uuid==kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000") && holder.value?.label=="ok")
        check(WideDto.default().p239 == "ok" && WideDto().p239 == "ok")
        check(holder.rows !== HolderDto(required = RecordDto(id="required",value=2)).rows)
        allOfDefaultOverrides(codec.decodeFromString<HolderDto>("""{"required":{"id":"required","value":2}}"""))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.encodeToJsonElement(holder).jsonObject.containsKey("value") == defaults)
        check(runCatching { RecordDto(id="bad",value=0) }.isFailure && runCatching { RecordDto(id="bad",value=4) }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(PatchInputDto().value === Patch.Absent)
        allOfDefaultOverridesWire(codec,holder)
    }
}

private suspend fun allOfDefaultOverridesWire(codec: Json, holder: HolderDto) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var calls = 0
    var callbacks = 0
    var payload = """{"required":{"id":"required","value":2},"label":"雪🍃"}"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = AllOfDefaultOverridesEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(allOfDefaultOverridesBody(request.body)) == codec.encodeToJsonElement(holder))
                respond("""{"required":{"id":"required","value":2}}""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(allOfDefaultOverridesBody(request.body)) == expectedPatch)
                respond("",HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(allOfDefaultOverridesBody(request.body)) == codec.encodeToJsonElement(holder))
                val channel = ByteChannel(autoFlush = true); channels += channel
                val frame = "event: record\r\ndata: $payload\r\n\r\n"
                scope.launch { try { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)) } finally { channel.close() } }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            }
            else -> error("Unexpected allOf numeric request path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = DefaultOverridesClient(http,"http://example.test/",codec).strings
        allOfDefaultOverrides(api.send(holder) { callbacks++ })
        val before = calls
        val flow = api.watch(input = holder) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Record
        check(event.data.required.value == 2 && event.data.label == "雪🍃"); allOfDefaultOverrides(event.data)
        for ((patch,expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(value = Patch.Null) to JsonObject(mapOf("value" to JsonNull)),
            PatchInputDto(value = Patch.Value(PatchInputDto.Value(id="patch",value=3))) to JsonObject(mapOf("value" to codec.encodeToJsonElement(PatchInputDto.Value(id="patch",value=3)))),
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        payload = """{"required":{"id":"bad","value":4}}"""
        allOfDefaultOverridesFailure { withTimeout(10000) { api.watch(input = holder) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        allOfDefaultOverrides(api.send(holder) { callbacks++ })
    } finally { http.close(); scope.cancel(); channels.forEach { it.close() } }
}
