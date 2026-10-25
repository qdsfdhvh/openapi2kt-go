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
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import org.example.unformatted.UnformattedFactoryClient
import org.example.unformatted.api.models.PatchInputDto
import org.example.unformatted.api.models.WatchEventDto
import org.example.unformatted.models.HolderDto
import org.example.unformatted.models.Patch
import org.example.unformatted.models.`CodeDto default`
import org.example.unformatted.models.`NegativeZeroDto default`
import org.example.unformatted.models.`RatioDto default`
import org.example.unformatted.models.`RoundedDto default`
import org.example.unformatted.models.`VocabularyDto default`
import org.example.unformatted.models.`WrappedCodeDto default`
import org.example.unformatted.models.`WrappedRatioDto default`

private class UnformattedNativeEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun unformattedNativeBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> unformattedNativeBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected native factory request content")
}

private suspend fun unformattedNativeFailure(block: suspend () -> Unit) {
    try { block() }
    catch (cause: CancellationException) { throw cause }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException || it is IllegalArgumentException })
        return
    }
    error("Expected invalid native consumer value")
}

suspend fun unformattedNativeFactoriesContracts(checked: Boolean = true) {
    check(`CodeDto default`() == Int.MIN_VALUE && `WrappedCodeDto default`() == Int.MAX_VALUE)
    check(`RatioDto default`() == 1.5 && `WrappedRatioDto default`() == 2.5)
    check(`RoundedDto default`() == 9007199254740992.0)
    check(`NegativeZeroDto default`().toRawBits() == Long.MIN_VALUE)
    check(`VocabularyDto default`() == 239)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val empty = HolderDto(requiredCode = Int.MIN_VALUE,label = "empty")
        check(empty.code == null && empty.ratio == null && empty.rounded == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCode":-2147483648,"label":"empty"}""").code == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"label":"missing"}""") }.isFailure)
        val holder = HolderDto(requiredCode = `CodeDto default`(),label = "request",code = `CodeDto default`(),ratio = `RatioDto default`(),rounded = `RoundedDto default`())
        check(codec.encodeToJsonElement(holder).jsonObject.containsKey("code"))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        if (checked) check(runCatching { HolderDto(requiredCode = 0,label = "bad") }.isFailure)
        else check(HolderDto(requiredCode = 0,label = "open").requiredCode == 0)
        unformattedNativeWire(codec,holder,checked)
    }
}

private suspend fun unformattedNativeWire(codec: Json,holder: HolderDto,checked: Boolean) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var calls = 0
    var callbacks = 0
    var payload = """{"requiredCode":-2147483648,"label":"雪🍃"}"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = UnformattedNativeEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(unformattedNativeBody(request.body)) == codec.encodeToJsonElement(holder))
                respond("""{"requiredCode":-2147483648,"label":"response"}""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(unformattedNativeBody(request.body)) == expectedPatch)
                respond("",HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(unformattedNativeBody(request.body)) == codec.encodeToJsonElement(holder))
                val channel = ByteChannel(autoFlush = true); channels += channel
                val frame = "event: value\r\ndata: $payload\r\n\r\n"
                scope.launch { try { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)) } finally { channel.close() } }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            }
            else -> error("Unexpected unformatted native path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = UnformattedFactoryClient(http,"http://example.test/",codec).nativeValues
        check(api.send(holder) { callbacks++ }.let { it.label == "response" && it.code == null && it.ratio == null })
        val before = calls
        val flow = api.watch(input = holder) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Value
        check(event.data.label == "雪🍃" && event.data.requiredCode == Int.MIN_VALUE && event.data.code == null)
        for ((patch,expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(code = Patch.Null) to JsonObject(mapOf("code" to JsonNull)),
            PatchInputDto(code = Patch.Value(Int.MAX_VALUE)) to JsonObject(mapOf("code" to JsonPrimitive(Int.MAX_VALUE))),
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        if (checked) {
            val prior = calls; val priorCallbacks = callbacks
            check(runCatching { api.send(HolderDto(requiredCode = 0,label = "bad")) { callbacks++ } }.isFailure)
            check(calls == prior && callbacks == priorCallbacks)
            payload = """{"requiredCode":0,"label":"bad"}"""
            unformattedNativeFailure { withTimeout(10000) { api.watch(input = holder) { callbacks++ }.toList() } }
            check(channels.last().isClosedForRead)
        }
        payload = "{}"
        unformattedNativeFailure { withTimeout(10000) { api.watch(input = holder) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        check(api.send(holder) { callbacks++ }.code == null)
    } finally { http.close(); scope.cancel(); channels.forEach { it.close() } }
}
