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
import org.example.allofstring.StringDefaultsClient
import org.example.allofstring.api.models.PatchInputDto
import org.example.allofstring.api.models.WatchEventDto
import org.example.allofstring.models.FactoryDto
import org.example.allofstring.models.HolderDto
import org.example.allofstring.models.LowerDto
import org.example.allofstring.models.NullableValuesDto
import org.example.allofstring.models.Patch
import org.example.allofstring.models.RecordDto
import org.example.allofstring.models.WideDto
import org.example.allofstring.models.`NarrowFactoryDto default`
import org.example.allofstring.models.`NullableFactoryDto default`

private class AllOfStringEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun allOfStringBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> allOfStringBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected allOf String request content")
}

private fun allOfStringDefaults(value: HolderDto) {
    check(value.value?.value == "abc" && value.inline?.value == "abc")
    check(value.cfg?.node?.value == "abc" && value.rows?.single()?.value == "abc")
    check(value.maps?.getValue("k")?.value == "ab" && value.empty?.value == "ab")
}

private suspend fun allOfStringFailure(block: suspend () -> Unit) {
    try { block() }
    catch (cause: CancellationException) { throw cause }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException })
        return
    }
    error("Expected retained String constraint failure")
}

suspend fun allOfStringDefaultsContracts() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto(required = RecordDto("ab"))
        allOfStringDefaults(holder)
        check(LowerDto().value == "longer" && RecordDto().value == "ab")
        check(FactoryDto.default().value == "abc" && FactoryDto().value == "ab")
        check(`NullableFactoryDto default`() == null && `NarrowFactoryDto default`().value == "ab")
        check(NullableValuesDto.default().value == null)
        check(WideDto.default().p239 == "abc" && WideDto().p239 == "ab")
        check(holder.rows !== HolderDto(required = RecordDto("ab")).rows)
        allOfStringDefaults(codec.decodeFromString<HolderDto>("""{"required":{"value":"ab"}}"""))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.encodeToJsonElement(holder).jsonObject.containsKey("value") == defaults)
        check(runCatching { RecordDto("a") }.isFailure && runCatching { RecordDto("abcd") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(PatchInputDto().value === Patch.Absent)
        allOfStringWire(codec,holder)
    }
}

private suspend fun allOfStringWire(codec: Json, holder: HolderDto) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var calls = 0
    var callbacks = 0
    var payload = """{"required":{"value":"雪🍃"}}"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = AllOfStringEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(allOfStringBody(request.body)) == codec.encodeToJsonElement(holder))
                respond("""{"required":{"value":"ab"}}""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(allOfStringBody(request.body)) == expectedPatch)
                respond("",HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(allOfStringBody(request.body)) == codec.encodeToJsonElement(holder))
                val channel = ByteChannel(autoFlush = true); channels += channel
                val frame = "event: record\r\ndata: $payload\r\n\r\n"
                scope.launch { try { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)) } finally { channel.close() } }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            }
            else -> error("Unexpected allOf String request path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = StringDefaultsClient(http,"http://example.test/",codec).strings
        allOfStringDefaults(api.send(holder) { callbacks++ })
        val before = calls
        val flow = api.watch(input = holder) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Record
        check(event.data.required.value == "雪🍃"); allOfStringDefaults(event.data)
        for ((patch,expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(value = Patch.Null) to JsonObject(mapOf("value" to JsonNull)),
            PatchInputDto(value = Patch.Value(PatchInputDto.Value("abc"))) to codec.parseToJsonElement("""{"value":{"value":"abc"}}""").jsonObject,
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        payload = """{"required":{"value":"a"}}"""
        allOfStringFailure { withTimeout(10000) { api.watch(input = holder) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        allOfStringDefaults(api.send(holder) { callbacks++ })
    } finally { http.close(); scope.cancel(); channels.forEach { it.close() } }
}
