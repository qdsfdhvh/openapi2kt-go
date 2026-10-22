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
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.yield
import kotlin.uuid.Uuid
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.jsonObject
import org.example.uuidunique.UUIDUniqueApiClient
import org.example.uuidunique.api.models.PatchInputDto
import org.example.uuidunique.api.models.WatchEventDto
import org.example.uuidunique.models.HolderDto
import org.example.uuidunique.models.Patch
import org.example.uuidunique.models.`DefaultValuesDto default`
import java.util.concurrent.CopyOnWriteArrayList

private fun uuidUniqueBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> uuidUniqueBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic UUID unique body")
}

private suspend fun uuidUniqueExpect(checked: Boolean, action: suspend () -> Unit) {
    if (!checked) { action(); return }
    try { action(); error("Missing UUID uniqueItems rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains("uniqueItems") == true
        }) { "Wrong UUID uniqueness failure: $cause" }
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class UUIDUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

suspend fun uuidUniqueContracts(checked: Boolean = true) = withTimeout(20000) {
    val directId = Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
    val id = Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
    val same = Uuid.parse("550E8400E29B41D4A716446655440000")
    val directValues = listOf(directId, null, Uuid.parse("550e8400-e29b-41d4-a716-446655440001"))
    val values = listOf(id, null, Uuid.parse("550e8400-e29b-41d4-a716-446655440001"))
    check(id == same)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto(directValues, values, mapOf("ids" to values))
        val encoded = codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(encoded) == holder)
        val bridged = HolderDto(directValues, bridged = listOf(id, Uuid.NIL))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(bridged)) == bridged)
        check(holder.defaultDirectValues == listOf(directId, null))
        check(holder.defaultValues == listOf(id, null))
        check(holder.defaultNested == listOf(listOf(id, null)))
        check(`DefaultValuesDto default`() == listOf(id, null))
        HolderDto(emptyList(), emptyList(), emptyMap())
        for (duplicate in listOf(listOf(directId, directId), listOf(null, null))) {
            uuidUniqueExpect(checked) { HolderDto(duplicate) }
            uuidUniqueExpect(checked) { PatchInputDto(directValues = Patch.Value(duplicate)) }
        }
        uuidUniqueExpect(checked) { HolderDto(directValues, listOf(id, same)) }
        uuidUniqueExpect(checked) { HolderDto(directValues, bridged = listOf(id, same)) }
        uuidUniqueExpect(checked) { HolderDto(directValues, nested = mapOf("bad" to listOf(id, same))) }
        uuidUniqueExpect(checked) { HolderDto(directValues, defaultNested = listOf(listOf(id, same))) }
        val badValues = "[\"550e8400-e29b-41d4-a716-446655440000\",\"550E8400E29B41D4A716446655440000\"]"
        val bad = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("values" to codec.parseToJsonElement(badValues))).toString()
        uuidUniqueExpect(checked) { codec.decodeFromString<HolderDto>(bad) }
        val badBridge = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("bridged" to codec.parseToJsonElement(badValues))).toString()
        uuidUniqueExpect(checked) { codec.decodeFromString<HolderDto>(badBridge) }
        PatchInputDto(); PatchInputDto(directValues = Patch.Null, values = Patch.Null)

        var mode = 0
        var calls = 0
        var expectedValues: List<Uuid?>? = values
        var expectedPatch = PatchInputDto()
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val jobs = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = UUIDUniqueEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(uuidUniqueBody(request.body)) == codec.parseToJsonElement(encoded))
                    respond(if (mode == 1) bad else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/values" -> {
                    check(codec.parseToJsonElement(uuidUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedValues)))
                    respond(if (mode == 1) badValues else codec.encodeToString(expectedValues), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(uuidUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedPatch)))
                    respond("", HttpStatusCode.NoContent)
                }
                "/watch" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val payload = if (mode == 1) bad else encoded
                    jobs += scope.launch {
                        for (byte in "event: sample\r\ndata: $payload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        awaitCancellation()
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic UUID unique URL")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = UUIDUniqueApiClient(client, "https://example.test/", codec).default
            check(api.echo(holder) == holder)
            check(api.values(values) == values)
            expectedValues = null; check(api.values(null) == null)
            expectedValues = values
            for (patch in listOf(PatchInputDto(), PatchInputDto(directValues = Patch.Null), PatchInputDto(directValues = Patch.Value(directValues), values = Patch.Value(values)))) {
                expectedPatch = patch; api.patchPatch(patch)
            }
            check(codec.parseToJsonElement(codec.encodeToString(PatchInputDto(directValues = Patch.Null))).jsonObject["directValues"] == JsonNull)
            val before = calls
            var callback = false
            expectedValues = listOf(id, same)
            uuidUniqueExpect(checked) { api.values(listOf(id, same)) { callback = true } }
            if (checked) check(!callback && calls == before)
            expectedValues = values
            mode = 1
            uuidUniqueExpect(checked) { api.echo(holder) }
            uuidUniqueExpect(checked) { api.values(values) }
            mode = 0
            val count = calls
            val cold = api.watch(); check(calls == count)
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 1
            uuidUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 0
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            check(api.echo(holder) == holder)
        } finally { client.close(); scope.cancel(); jobs.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
        println("UUIDUnique checked=$checked defaults=$defaults nulls=$nulls UUID-value-equality-null-defaults-alias-Patch-preflight-HTTP-liveSSEclose-cold-reuse passed")
    }
}
