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
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.jsonObject
import org.example.floatingunique.FloatingUniqueApiClient
import org.example.floatingunique.api.models.PatchInputDto
import org.example.floatingunique.api.models.WatchEventDto
import org.example.floatingunique.models.HolderDto
import org.example.floatingunique.models.Patch
import org.example.floatingunique.models.`DefaultDoublesDto default`
import java.util.concurrent.CopyOnWriteArrayList

private fun floatingUniqueBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> floatingUniqueBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic floating unique body")
}

private suspend fun floatingUniqueExpect(checked: Boolean, keyword: String = "uniqueItems", action: suspend () -> Unit) {
    if (!checked) { action(); return }
    try { action(); error("Missing floating uniqueItems rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(keyword) == true
        }) { "Wrong floating uniqueness failure: $cause" }
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class FloatingUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

suspend fun floatingUniqueContracts(checked: Boolean = true) = withTimeout(20000) {
    val floatValue = 1f
    val value = 1.0
    val same = 1.0
    val floats = listOf(-Float.MAX_VALUE, null, Float.MIN_VALUE, Float.MAX_VALUE, 0f, 1f)
    val doubles = listOf(-Double.MAX_VALUE, null, Double.MIN_VALUE, Double.MAX_VALUE, 0.0, 1.0)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto(floats, doubles, mapOf("doubles" to doubles), enumFloats = listOf(0f, 1f, 2f, null), enumDoubles = listOf(2.0, 4.0), borrowedDoubles = listOf(0.0, 1.0))
        val encoded = codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(encoded) == holder)
        check(holder.defaultFloats == listOf(floatValue, null))
        check(holder.defaultDoubles == listOf(value, null))
        check(holder.defaultNested == listOf(listOf(value, null)))
        check(`DefaultDoublesDto default`() == listOf(value, null))
        HolderDto(emptyList(), emptyList(), emptyMap())
        for (duplicate in listOf(listOf(floatValue, floatValue), listOf(null, null))) {
            floatingUniqueExpect(checked) { HolderDto(duplicate) }
            floatingUniqueExpect(checked) { PatchInputDto(floats = Patch.Value(duplicate)) }
        }
        floatingUniqueExpect(checked) { HolderDto(floats, listOf(value, same)) }
        floatingUniqueExpect(checked) { HolderDto(floats, nested = mapOf("bad" to listOf(value, same))) }
        floatingUniqueExpect(checked) { HolderDto(floats, defaultNested = listOf(listOf(value, same))) }
        val badDoubles = "[0,-0]"
        val bad = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("doubles" to codec.parseToJsonElement(badDoubles))).toString()
        floatingUniqueExpect(checked) { codec.decodeFromString<HolderDto>(bad) }
        floatingUniqueExpect(checked) { HolderDto(listOf(0f, -0f)) }
        floatingUniqueExpect(checked) { HolderDto(floats, doubles = listOf(0.0, -0.0)) }
        floatingUniqueExpect(checked) { HolderDto(floats, enumFloats = listOf(1f, 1.000000001f)) }
        floatingUniqueExpect(checked) { HolderDto(floats, enumDoubles = listOf(2.0, 2.0)) }
        floatingUniqueExpect(checked) { HolderDto(floats, borrowedDoubles = listOf(1.0, 1.0)) }
        floatingUniqueExpect(checked, "enum") { HolderDto(floats, enumFloats = listOf(9f)) }
        for (invalid in listOf(1.0, 6.0)) check(runCatching { HolderDto(floats, enumDoubles = listOf(invalid)) }.isFailure == checked)
        HolderDto(floats, enumDoubles = listOf(3.0))
        for (nonfinite in listOf(Float.NaN, Float.POSITIVE_INFINITY, Float.NEGATIVE_INFINITY)) {
            floatingUniqueExpect(checked) { HolderDto(listOf(nonfinite)) }
        }
        for (nonfinite in listOf(Double.NaN, Double.POSITIVE_INFINITY, Double.NEGATIVE_INFINITY)) {
            floatingUniqueExpect(checked) { HolderDto(floats, doubles = listOf(nonfinite)) }
        }
        val permissive = Json(codec) { allowSpecialFloatingPointValues = true }
        for (literal in listOf("NaN", "Infinity", "-Infinity")) {
            floatingUniqueExpect(checked) { permissive.decodeFromString<HolderDto>("{\"floats\":[$literal]}") }
        }
        floatingUniqueExpect(checked) { codec.decodeFromString<HolderDto>("{\"floats\":[1,1.000000001]}") }
        HolderDto(listOf(1f, Float.fromBits(1f.toBits() + 1)), doubles = listOf(1.0, Double.fromBits(1.0.toBits() + 1)))
        PatchInputDto(); PatchInputDto(floats = Patch.Null, doubles = Patch.Null)

        var mode = 0
        var calls = 0
        var expectedDoubles: List<Double?>? = doubles
        var expectedPatch = PatchInputDto()
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val jobs = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = FloatingUniqueEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(floatingUniqueBody(request.body)) == codec.parseToJsonElement(encoded))
                    respond(if (mode == 1) bad else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/doubles" -> {
                    check(codec.parseToJsonElement(floatingUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedDoubles)))
                    respond(if (mode == 1) badDoubles else codec.encodeToString(expectedDoubles), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(floatingUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedPatch)))
                    respond("", HttpStatusCode.NoContent)
                }
                "/watch" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val payload = when (mode) { 1 -> bad; 2 -> badDoubles; 3 -> codec.encodeToString(doubles); else -> encoded }
                    val event = if (mode >= 2) "values" else "sample"
                    jobs += scope.launch {
                        for (byte in "event: $event\r\ndata: $payload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        awaitCancellation()
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic floating unique URL")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = FloatingUniqueApiClient(client, "https://example.test/", codec).default
            check(api.echo(holder) == holder)
            check(api.doubles(doubles) == doubles)
            expectedDoubles = null; check(api.doubles(null) == null)
            expectedDoubles = doubles
            for (patch in listOf(PatchInputDto(), PatchInputDto(floats = Patch.Null), PatchInputDto(floats = Patch.Value(floats), doubles = Patch.Value(doubles)))) {
                expectedPatch = patch; api.patchPatch(patch)
            }
            check(codec.parseToJsonElement(codec.encodeToString(PatchInputDto(floats = Patch.Null))).jsonObject["floats"] == JsonNull)
            val before = calls
            var callback = false
            expectedDoubles = listOf(value, same)
            floatingUniqueExpect(checked) { api.doubles(listOf(value, same)) { callback = true } }
            if (checked) check(!callback && calls == before)
            expectedDoubles = doubles
            if (checked) {
                val beforeFinite = calls
                callback = false
                floatingUniqueExpect(true) { api.doubles(listOf(Double.NaN)) { callback = true } }
                check(!callback && calls == beforeFinite)
            }
            mode = 1
            floatingUniqueExpect(checked) { api.echo(holder) }
            floatingUniqueExpect(checked) { api.doubles(doubles) }
            mode = 0
            val count = calls
            val cold = api.watch(); check(calls == count)
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 1
            floatingUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 2
            floatingUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 3
            check((cold.take(1).toList().single() as WatchEventDto.Values).data == doubles)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 0
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            check(api.echo(holder) == holder)
        } finally { client.close(); scope.cancel(); jobs.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
        println("FloatingUnique checked=$checked defaults=$defaults nulls=$nulls mapped-floating-equality-rounding-signedzero-finite-membership-bounds-null-defaults-alias-Patch-preflight-HTTP-liveSSEclose-cold-reuse passed")
    }
}
