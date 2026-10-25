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
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import java.util.concurrent.CopyOnWriteArrayList

import org.example.defaultgraphs.ObjectDefaultGraphsApiClient
import org.example.defaultgraphs.models.HolderDto
import org.example.defaultgraphs.models.ConfigDto
import org.example.defaultgraphs.models.Patch
import org.example.defaultgraphs.api.models.PatchInputDto
import org.example.defaultgraphs.api.models.WatchEventDto
import kotlin.time.Instant

private fun graphDefaultBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> graphDefaultBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic graph default body")
}

private suspend fun graphDefaultExpect(checked: Boolean, keyword: String, action: suspend () -> Unit) {
    if (!checked) { action(); return }
    try { action(); error("Missing graph default rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(keyword) == true
        }) { "Wrong graph default failure: $cause" }
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class GraphDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}


suspend fun objectDefaultGraphsContracts(checked: Boolean) = withTimeout(30000) {
    val expectedTime = Instant.parse("2026-01-02T03:04:05Z")
    val first = HolderDto(token = "token")
    val second = HolderDto(token = "token")
    check(first.cfg != null && second.cfg != null && first.cfg !== second.cfg)
    check(first.target == null)
    check(first.cfg!!.rows == listOf(mapOf("a" to "first", "z" to "last", "nil" to null), null))
    check(first.cfg!!.rows !== second.cfg!!.rows && first.cfg!!.nodes !== second.cfg!!.nodes)
    check(first.cfg!!.nodes!!.single().getValue("one") !== second.cfg!!.nodes!!.single().getValue("one"))
    check(first.cfg!!.times!!.single().getValue("at") == expectedTime)
    check(first.cfg!!.modes!!.single().getValue("first").serialName == "a")
    check(first.cfg!!.nodes!!.single().getValue("one").name == "named")
    check(first.cfg!!.composed!!.single().getValue("one").count == 5)
    check(first.cfg!!.bridge == null && first.cfg!!.narrow == emptyList<Map<String, String?>?>())
    check(first.cfg!!.inlineRows == listOf(mapOf("a" to "inline")))
    check(ConfigDto.default().rows == first.cfg!!.rows)
    check(ConfigDto.default().rows !== ConfigDto.default().rows)
    check(PatchInputDto().cfg === Patch.Absent)
    val patches = listOf(PatchInputDto(), PatchInputDto(cfg = Patch.Null), PatchInputDto(cfg = Patch.Value(PatchInputDto.Cfg(rows = Patch.Value(listOf(mapOf("a" to "patch")))))))
    if (checked) graphDefaultExpect(true, "minLength") { HolderDto("token", cfg = HolderDto.Cfg(rows = listOf(mapOf("a" to "x")))) }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val encoded = codec.encodeToString(first)
        check(codec.decodeFromString<HolderDto>("""{"token":"token"}""") == first)
        check(codec.decodeFromString<HolderDto>(encoded) == first)
        check(codec.decodeFromString<HolderDto>("""{"token":"token","cfg":null}""").cfg == null)
        for (patch in patches) check(codec.encodeToJsonElement(codec.decodeFromString<PatchInputDto>(codec.encodeToString(patch))) == codec.encodeToJsonElement(patch))
        check(codec.encodeToJsonElement(patches[0]).jsonObject["cfg"] == null)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["cfg"] == kotlinx.serialization.json.JsonNull)
        val knownUnknown = codec.decodeFromString<HolderDto>("""{"token":"token","cfg":{"modes":[{"one":"future"}]}}""")
        check(knownUnknown.cfg!!.modes!!.single().getValue("one").serialName == "future")
        val invalidTime = """{"token":"token","cfg":{"times":[{"at":"invalid"}]}}"""
        val missingName = """{"token":"token","cfg":{"nodes":[{"one":{}}]}}"""
        val invalidBound = """{"token":"token","cfg":{"rows":[{"one":"x"}]}}"""
        check(runCatching { codec.decodeFromString<HolderDto>(invalidTime) }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>(missingName) }.isFailure)
        if (checked) graphDefaultExpect(true, "minLength") { codec.decodeFromString<HolderDto>(invalidBound) }
        var payload = encoded
        var expectedPatch = patches[0]
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val jobs = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = GraphDefaultEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(graphDefaultBody(request.body)) == codec.parseToJsonElement(encoded))
                    respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(graphDefaultBody(request.body)) == codec.encodeToJsonElement(expectedPatch))
                    respond("", HttpStatusCode.NoContent)
                }
                "/watch" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val eventPayload = payload
                    jobs += scope.launch {
                        for (byte in "event: sample\r\ndata: $eventPayload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        awaitCancellation()
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic graph default URL")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = ObjectDefaultGraphsApiClient(client, "https://example.test/", codec).default
            check(api.echo(first) == first)
            for (patch in patches) { expectedPatch = patch; api.patchPatch(patch) }
            val before = calls
            val cold = api.watch(); check(calls == before)
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == first)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            for (bad in listOf(invalidTime, missingName) + if (checked) listOf(invalidBound) else emptyList()) {
                payload = bad
                check(runCatching { api.echo(first) }.isFailure)
                check(runCatching { cold.take(1).toList() }.isFailure)
                withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            }
            payload = """{"token":"token"}"""
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == first)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            check(api.echo(first) == first)
        } finally { client.close(); scope.cancel(); jobs.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
        println("ObjectDefaultGraphs checked=$checked defaults=$defaults nulls=$nulls finite-alias-collections-defaults-fresh-factory-Patch-HTTP-liveSSE-close-cold-reuse passed")
    }
}
