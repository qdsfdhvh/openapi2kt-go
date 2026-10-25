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
import org.example.tupleroot.TupleRootClient
import org.example.tupleroot.api.models.PatchInputDto
import org.example.tupleroot.api.models.WatchEventDto
import org.example.tupleroot.models.HolderDto
import org.example.tupleroot.models.Patch
import org.example.tupleroot.models.`RowsDto default`
import org.example.tupleroot.models.`LongsDto default`
import org.example.tupleroot.models.`FloatsDto default`
import org.example.tupleroot.models.`DatesDto default`
import org.example.tupleroot.models.`TimesDto default`
import org.example.tupleroot.models.`IdsDto default`
import org.example.tupleroot.models.`FlagsDto default`
import org.example.tupleroot.models.`EmptyDto default`
import org.example.tupleroot.models.`WrappedDto default`
import org.example.tupleroot.models.`NullableRowsDto default`
import org.example.tupleroot.models.`NarrowDto default`
import org.example.tupleroot.models.`TailDto default`
import kotlin.time.Instant
import kotlin.uuid.Uuid
import kotlinx.datetime.LocalDate

private class TupleRootEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun tupleRootBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> tupleRootBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected tuple root request content")
}

private fun tupleRootDefaults(value: HolderDto) {
    check(value.required == listOf(1,2))
    check(value.rows == null)
}

private suspend fun tupleRootFailure(block: suspend () -> Unit) {
    try { block() }
    catch (cause: CancellationException) { throw cause }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException || it is kotlinx.serialization.SerializationException })
        return
    }
    error("Expected tuple codec constraint failure")
}

suspend fun tupleRootDefaultsContracts() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(`RowsDto default`() == listOf(1,2))
        check(`RowsDto default`() !== `RowsDto default`())
        check(`WrappedDto default`() == listOf(3) && `NarrowDto default`() == listOf(4))
        check(`NullableRowsDto default`() == null && `FlagsDto default`() == null)
        check(`EmptyDto default`() == emptyList<Int>() && `TailDto default`() == listOf(5))
        check(`LongsDto default`() == listOf(Long.MIN_VALUE,Long.MAX_VALUE))
        check(`FloatsDto default`()[0].toBits() == (-0.0f).toBits())
        check(`DatesDto default`() == listOf(LocalDate.parse("2024-02-29")))
        check(`TimesDto default`() == listOf(Instant.parse("2024-02-29T10:04:56.123456789Z")))
        check(`IdsDto default`() == listOf(Uuid.parse("00000000-0000-0000-0000-000000000001")))
        val holder=HolderDto(required=`RowsDto default`())
        tupleRootDefaults(holder)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<HolderDto>("""{"required":[1,2]}""").rows == null)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"required":[true]}""") }.isFailure)
        check(PatchInputDto().rows === Patch.Absent)
        tupleRootWire(codec,holder)
    }
}

private suspend fun tupleRootWire(codec: Json, holder: HolderDto) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var calls = 0
    var callbacks = 0
    var payload = """{"required":[1,2],"label":"雪🍃"}"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = TupleRootEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(tupleRootBody(request.body)) == codec.encodeToJsonElement(holder))
                respond("""{"required":[1,2]}""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(tupleRootBody(request.body)) == expectedPatch)
                respond("",HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(tupleRootBody(request.body)) == codec.encodeToJsonElement(holder))
                val channel = ByteChannel(autoFlush = true); channels += channel
                val frame = "event: record\r\ndata: $payload\r\n\r\n"
                scope.launch { try { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)) } finally { channel.close() } }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            }
            else -> error("Unexpected tuple root request path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = TupleRootClient(http,"http://example.test/",codec).tuples
        tupleRootDefaults(api.send(holder) { callbacks++ })
        val before = calls
        val flow = api.watch(input = holder) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Record
        check(event.data.label == "雪🍃"); tupleRootDefaults(event.data)
        for ((patch,expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(rows = Patch.Null) to JsonObject(mapOf("rows" to JsonNull)),
            PatchInputDto(rows = Patch.Value(listOf(3))) to codec.parseToJsonElement("""{"rows":[3]}""").jsonObject,
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        payload = """{"required":[true]}"""
        tupleRootFailure { withTimeout(10000) { api.watch(input = holder) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        tupleRootDefaults(api.send(holder) { callbacks++ })
    } finally { http.close(); scope.cancel(); channels.forEach { it.close() } }
}
