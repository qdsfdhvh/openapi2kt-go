@file:OptIn(kotlin.uuid.ExperimentalUuidApi::class, kotlinx.serialization.ExperimentalSerializationApi::class, io.ktor.utils.io.InternalAPI::class)
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
import org.example.allofuuid.UUIDDefaultsClient
import org.example.allofuuid.api.models.PatchInputDto
import org.example.allofuuid.api.models.WatchEventDto
import org.example.allofuuid.models.FactoryDto
import org.example.allofuuid.models.HolderDto
import org.example.allofuuid.models.LowerDto
import org.example.allofuuid.models.NullableValuesDto
import org.example.allofuuid.models.Patch
import org.example.allofuuid.models.RecordDto
import org.example.allofuuid.models.WideDto
import org.example.allofuuid.models.`NarrowFactoryDto default`
import org.example.allofuuid.models.`NullableFactoryDto default`

private class AllOfUUIDEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun allOfUUIDBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> allOfUUIDBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected allOf String request content")
}

private fun allOfUUIDDefaults(value: HolderDto) {
    check(value.value?.value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002") && value.inline?.value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002"))
    check(value.cfg?.node?.value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002") && value.rows?.single()?.value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002"))
    check(value.maps?.getValue("k")?.value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000") && value.empty?.value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000"))
}

private suspend fun allOfUUIDFailure(block: suspend () -> Unit) {
    try { block() }
    catch (cause: CancellationException) { throw cause }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException })
        return
    }
    error("Expected retained String constraint failure")
}

suspend fun allOfUUIDDefaultsContracts() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto(required = RecordDto(kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000")))
        allOfUUIDDefaults(holder)
        check(LowerDto().value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174001") && RecordDto().value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000"))
        check(FactoryDto.default().value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002") && FactoryDto().value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000"))
        check(`NullableFactoryDto default`() == null && `NarrowFactoryDto default`().value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000"))
        check(NullableValuesDto.default().value == null)
        check(WideDto.default().p239 == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002") && WideDto().p239 == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000"))
        check(holder.rows !== HolderDto(required = RecordDto(kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174000"))).rows)
        allOfUUIDDefaults(codec.decodeFromString<HolderDto>("""{"required":{"value":"123e4567-e89b-12d3-a456-426614174000"}}"""))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.encodeToJsonElement(holder).jsonObject.containsKey("value") == defaults)
        check(runCatching { RecordDto(kotlin.uuid.Uuid.parse("00000000-0000-0000-0000-000000000000")) }.isFailure && runCatching { RecordDto(kotlin.uuid.Uuid.parse("11111111-1111-1111-1111-111111111111")) }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(PatchInputDto().value === Patch.Absent)
        allOfUUIDWire(codec,holder)
    }
}

private suspend fun allOfUUIDWire(codec: Json, holder: HolderDto) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var calls = 0
    var callbacks = 0
    var payload = """{"required":{"value":"123e4567-e89b-12d3-a456-426614174002"}}"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = AllOfUUIDEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(allOfUUIDBody(request.body)) == codec.encodeToJsonElement(holder))
                respond("""{"required":{"value":"123e4567-e89b-12d3-a456-426614174000"}}""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(allOfUUIDBody(request.body)) == expectedPatch)
                respond("",HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(allOfUUIDBody(request.body)) == codec.encodeToJsonElement(holder))
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
        val api = UUIDDefaultsClient(http,"http://example.test/",codec).strings
        allOfUUIDDefaults(api.send(holder) { callbacks++ })
        val before = calls
        val flow = api.watch(input = holder) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { flow.toList() }.single() as WatchEventDto.Record
        check(event.data.required.value == kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002")); allOfUUIDDefaults(event.data)
        for ((patch,expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(value = Patch.Null) to JsonObject(mapOf("value" to JsonNull)),
            PatchInputDto(value = Patch.Value(PatchInputDto.Value(kotlin.uuid.Uuid.parse("123e4567-e89b-12d3-a456-426614174002")))) to codec.parseToJsonElement("""{"value":{"value":"123e4567-e89b-12d3-a456-426614174002"}}""").jsonObject,
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        payload = """{"required":{"value":"00000000-0000-0000-0000-000000000000"}}"""
        allOfUUIDFailure { withTimeout(10000) { api.watch(input = holder) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        allOfUUIDDefaults(api.send(holder) { callbacks++ })
    } finally { http.close(); scope.cancel(); channels.forEach { it.close() } }
}
