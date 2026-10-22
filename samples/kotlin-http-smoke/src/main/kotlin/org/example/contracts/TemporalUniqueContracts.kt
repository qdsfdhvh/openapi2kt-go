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
import kotlinx.datetime.LocalDate
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.jsonObject
import org.example.temporalunique.TemporalUniqueApiClient
import org.example.temporalunique.api.models.PatchInputDto
import org.example.temporalunique.api.models.WatchEventDto
import org.example.temporalunique.models.HolderDto
import org.example.temporalunique.models.Patch
import org.example.temporalunique.models.`DefaultMomentsDto default`
import java.util.concurrent.CopyOnWriteArrayList
import kotlin.time.Instant

private fun temporalUniqueBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> temporalUniqueBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic temporal unique body")
}

private suspend fun temporalUniqueExpect(checked: Boolean, action: suspend () -> Unit) {
    if (!checked) { action(); return }
    try { action(); error("Missing temporal uniqueItems rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains("uniqueItems") == true
        }) { "Wrong temporal uniqueness failure: $cause" }
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class TemporalUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

suspend fun temporalUniqueContracts(checked: Boolean = true) = withTimeout(20000) {
    val day = LocalDate.parse("2024-02-29")
    val instant = Instant.parse("2026-01-02T03:04:05Z")
    val same = Instant.parse("2026-01-02T05:04:05.000000000+02:00")
    val dates = listOf(day, null, LocalDate.parse("2024-03-01"))
    val moments = listOf(instant, null, Instant.parse("2026-01-02T03:04:05.000000001Z"))
    check(instant == same)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto(dates, moments, mapOf("times" to moments))
        val encoded = codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(encoded) == holder)
        check(holder.defaultDates == listOf(day, null))
        check(holder.defaultMoments == listOf(instant, null))
        check(holder.defaultNested == listOf(listOf(instant, null)))
        check(`DefaultMomentsDto default`() == listOf(instant, null))
        HolderDto(emptyList(), emptyList(), emptyMap())
        for (duplicate in listOf(listOf(day, day), listOf(null, null))) {
            temporalUniqueExpect(checked) { HolderDto(duplicate) }
            temporalUniqueExpect(checked) { PatchInputDto(dates = Patch.Value(duplicate)) }
        }
        temporalUniqueExpect(checked) { HolderDto(dates, listOf(instant, same)) }
        temporalUniqueExpect(checked) { HolderDto(dates, nested = mapOf("bad" to listOf(instant, same))) }
        temporalUniqueExpect(checked) { HolderDto(dates, defaultNested = listOf(listOf(instant, same))) }
        val badMoments = "[\"2026-01-02T03:04:05Z\",\"2026-01-02T05:04:05.000+02:00\"]"
        val bad = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("moments" to codec.parseToJsonElement(badMoments))).toString()
        temporalUniqueExpect(checked) { codec.decodeFromString<HolderDto>(bad) }
        PatchInputDto(); PatchInputDto(dates = Patch.Null, moments = Patch.Null)

        var mode = 0
        var calls = 0
        var expectedMoments: List<Instant?>? = moments
        var expectedPatch = PatchInputDto()
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val jobs = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = TemporalUniqueEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(temporalUniqueBody(request.body)) == codec.parseToJsonElement(encoded))
                    respond(if (mode == 1) bad else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/moments" -> {
                    check(codec.parseToJsonElement(temporalUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedMoments)))
                    respond(if (mode == 1) badMoments else codec.encodeToString(expectedMoments), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(temporalUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedPatch)))
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
                else -> error("Unexpected synthetic temporal unique URL")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = TemporalUniqueApiClient(client, "https://example.test/", codec).default
            check(api.echo(holder) == holder)
            check(api.moments(moments) == moments)
            expectedMoments = null; check(api.moments(null) == null)
            expectedMoments = moments
            for (patch in listOf(PatchInputDto(), PatchInputDto(dates = Patch.Null), PatchInputDto(dates = Patch.Value(dates), moments = Patch.Value(moments)))) {
                expectedPatch = patch; api.patchPatch(patch)
            }
            check(codec.parseToJsonElement(codec.encodeToString(PatchInputDto(dates = Patch.Null))).jsonObject["dates"] == JsonNull)
            val before = calls
            var callback = false
            expectedMoments = listOf(instant, same)
            temporalUniqueExpect(checked) { api.moments(listOf(instant, same)) { callback = true } }
            if (checked) check(!callback && calls == before)
            expectedMoments = moments
            mode = 1
            temporalUniqueExpect(checked) { api.echo(holder) }
            temporalUniqueExpect(checked) { api.moments(moments) }
            mode = 0
            val count = calls
            val cold = api.watch(); check(calls == count)
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 1
            temporalUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 0
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            check(api.echo(holder) == holder)
        } finally { client.close(); scope.cancel(); jobs.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
        println("TemporalUnique checked=$checked defaults=$defaults nulls=$nulls ISO-equality-null-defaults-alias-Patch-preflight-HTTP-liveSSEclose-cold-reuse passed")
    }
}
