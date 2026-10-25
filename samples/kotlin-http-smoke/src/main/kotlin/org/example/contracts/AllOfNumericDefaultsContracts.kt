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
import org.example.allofnumeric.NumericDefaultsClient
import org.example.allofnumeric.api.models.PatchInputDto
import org.example.allofnumeric.api.models.WatchEventDto
import org.example.allofnumeric.models.NumbersDto
import org.example.allofnumeric.models.StrictDto
import org.example.allofnumeric.models.FactoryDto
import org.example.allofnumeric.models.HolderDto
import org.example.allofnumeric.models.LowerDto
import org.example.allofnumeric.models.NullableValuesDto
import org.example.allofnumeric.models.Patch
import org.example.allofnumeric.models.RecordDto
import org.example.allofnumeric.models.WideDto
import org.example.allofnumeric.models.`NarrowFactoryDto default`
import org.example.allofnumeric.models.`NullableFactoryDto default`

private class AllOfNumericEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun allOfNumericBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> allOfNumericBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected allOf numeric request content")
}

private fun allOfNumericDefaults(value: HolderDto) {
    check(value.value?.value == 0 && value.inline?.value == 0)
    check(value.cfg?.node?.value == 0 && value.rows?.single()?.value == 0)
    check(value.maps?.getValue("k")?.value == 5 && value.empty?.value == 5)
}

private suspend fun allOfNumericFailure(block: suspend () -> Unit) {
    try { block() }
    catch (cause: CancellationException) { throw cause }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException })
        return
    }
    error("Expected retained numeric constraint failure")
}

suspend fun allOfNumericDefaultsContracts() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val numbers = NumbersDto.default()
        check(numbers.intValue == Int.MIN_VALUE && numbers.longValue == Long.MIN_VALUE)
        check(numbers.floatValue == 16777216f && numbers.doubleValue == 9007199254740992.0)
        check(numbers.zero?.toBits() == (-0.0).toBits())
        check(codec.decodeFromString<NumbersDto>(codec.encodeToString(numbers)) == numbers)
        check(runCatching { NumbersDto(floatValue = 16777214f) }.isFailure)
        check(runCatching { NumbersDto(doubleValue = 9007199254740990.0) }.isFailure)
        check(StrictDto.default().value == 3 && StrictDto().value == 0)
        for (bad in listOf(-9,9,1)) check(runCatching { StrictDto(bad) }.isFailure)
        val holder = HolderDto(required = RecordDto(5))
        allOfNumericDefaults(holder)
        check(LowerDto().value == -5 && RecordDto().value == 5)
        check(FactoryDto.default().value == 0 && FactoryDto().value == 5)
        check(`NullableFactoryDto default`() == null && `NarrowFactoryDto default`().value == 5)
        check(NullableValuesDto.default().value == null)
        check(WideDto.default().p239 == 0 && WideDto().p239 == 5)
        check(holder.rows !== HolderDto(required = RecordDto(5)).rows)
        allOfNumericDefaults(codec.decodeFromString<HolderDto>("""{"required":{"value":5}}"""))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.encodeToJsonElement(holder).jsonObject.containsKey("value") == defaults)
        check(runCatching { RecordDto(-11) }.isFailure && runCatching { RecordDto(11) }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(PatchInputDto().value === Patch.Absent)
        allOfNumericWire(codec,holder)
    }
}

private suspend fun allOfNumericWire(codec: Json, holder: HolderDto) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var calls = 0
    var callbacks = 0
    var payload = """{"required":{"value":5},"label":"雪🍃"}"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = AllOfNumericEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(allOfNumericBody(request.body)) == codec.encodeToJsonElement(holder))
                respond("""{"required":{"value":5}}""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(allOfNumericBody(request.body)) == expectedPatch)
                respond("",HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(allOfNumericBody(request.body)) == codec.encodeToJsonElement(holder))
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
        val api = NumericDefaultsClient(http,"http://example.test/",codec).strings
        allOfNumericDefaults(api.send(holder) { callbacks++ })
        val before = calls
        val flow = api.watch(input = holder) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Record
        check(event.data.required.value == 5 && event.data.label == "雪🍃"); allOfNumericDefaults(event.data)
        for ((patch,expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(value = Patch.Null) to JsonObject(mapOf("value" to JsonNull)),
            PatchInputDto(value = Patch.Value(PatchInputDto.Value(0))) to codec.parseToJsonElement("""{"value":{"value":0}}""").jsonObject,
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        payload = """{"required":{"value":-11}}"""
        allOfNumericFailure { withTimeout(10000) { api.watch(input = holder) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        allOfNumericDefaults(api.send(holder) { callbacks++ })
    } finally { http.close(); scope.cancel(); channels.forEach { it.close() } }
}
