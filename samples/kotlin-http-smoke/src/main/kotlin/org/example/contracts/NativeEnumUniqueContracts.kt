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
import org.example.nativeenumunique.NativeEnumUniqueApiClient
import org.example.nativeenumunique.api.models.PatchInputDto
import org.example.nativeenumunique.api.models.WatchEventDto
import org.example.nativeenumunique.models.HolderDto
import org.example.nativeenumunique.models.Patch
import org.example.nativeenumunique.models.`DefaultCodesDto default`
import java.util.concurrent.CopyOnWriteArrayList

private fun nativeEnumUniqueBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> nativeEnumUniqueBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic native enum unique body")
}

private suspend fun nativeEnumUniqueExpect(checked: Boolean, keyword: String = "uniqueItems", action: suspend () -> Unit) {
    if (!checked) { action(); return }
    try { action(); error("Missing native enum uniqueItems rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(keyword) == true
        }) { "Wrong native enum uniqueness failure: $cause" }
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NativeEnumUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

suspend fun nativeEnumUniqueContracts(checked: Boolean = true) = withTimeout(20000) {
    val flag = true
    val code = 1L
    val same = 1L
    val flags = listOf(true, null, false)
    val codes = listOf(Long.MIN_VALUE, null, Long.MAX_VALUE, 0L, 1L)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto(flags, codes, mapOf("codes" to codes), ints = listOf(Int.MIN_VALUE, 0, Int.MAX_VALUE, null), bounded = listOf(2, 4), borrowedCodes = listOf(0L, 1L))
        val encoded = codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(encoded) == holder)
        check(holder.defaultFlags == listOf(flag, null))
        check(holder.defaultCodes == listOf(code, null))
        check(holder.defaultNested == listOf(listOf(code, null)))
        check(`DefaultCodesDto default`() == listOf(code, null))
        HolderDto(emptyList(), emptyList(), emptyMap())
        for (duplicate in listOf(listOf(flag, flag), listOf(null, null))) {
            nativeEnumUniqueExpect(checked) { HolderDto(duplicate) }
            nativeEnumUniqueExpect(checked) { PatchInputDto(flags = Patch.Value(duplicate)) }
        }
        nativeEnumUniqueExpect(checked) { HolderDto(flags, listOf(code, same)) }
        nativeEnumUniqueExpect(checked) { HolderDto(flags, nested = mapOf("bad" to listOf(code, same))) }
        nativeEnumUniqueExpect(checked) { HolderDto(flags, defaultNested = listOf(listOf(code, same))) }
        val badCodes = "[0,-0]"
        val bad = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("codes" to codec.parseToJsonElement(badCodes))).toString()
        nativeEnumUniqueExpect(checked) { codec.decodeFromString<HolderDto>(bad) }
        nativeEnumUniqueExpect(checked) { HolderDto(flags, ints = listOf(0, 0)) }
        nativeEnumUniqueExpect(checked) { HolderDto(flags, bounded = listOf(2, 2)) }
        nativeEnumUniqueExpect(checked) { HolderDto(flags, borrowedCodes = listOf(1L, 1L)) }
        nativeEnumUniqueExpect(checked, "enum") { HolderDto(flags, codes = listOf(9L)) }
        for (invalid in listOf(1, 3, 6)) check(runCatching { HolderDto(flags, bounded = listOf(invalid)) }.isFailure == checked)
        check(holder.ints == listOf(Int.MIN_VALUE, 0, Int.MAX_VALUE, null))
        PatchInputDto(); PatchInputDto(flags = Patch.Null, codes = Patch.Null)

        var mode = 0
        var calls = 0
        var expectedCodes: List<Long?>? = codes
        var expectedPatch = PatchInputDto()
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val jobs = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = NativeEnumUniqueEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(nativeEnumUniqueBody(request.body)) == codec.parseToJsonElement(encoded))
                    respond(if (mode == 1) bad else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/codes" -> {
                    check(codec.parseToJsonElement(nativeEnumUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedCodes)))
                    respond(if (mode == 1) badCodes else codec.encodeToString(expectedCodes), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(nativeEnumUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedPatch)))
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
                else -> error("Unexpected synthetic native enum unique URL")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = NativeEnumUniqueApiClient(client, "https://example.test/", codec).default
            check(api.echo(holder) == holder)
            check(api.codes(codes) == codes)
            expectedCodes = null; check(api.codes(null) == null)
            expectedCodes = codes
            for (patch in listOf(PatchInputDto(), PatchInputDto(flags = Patch.Null), PatchInputDto(flags = Patch.Value(flags), codes = Patch.Value(codes)))) {
                expectedPatch = patch; api.patchPatch(patch)
            }
            check(codec.parseToJsonElement(codec.encodeToString(PatchInputDto(flags = Patch.Null))).jsonObject["flags"] == JsonNull)
            val before = calls
            var callback = false
            expectedCodes = listOf(code, same)
            nativeEnumUniqueExpect(checked) { api.codes(listOf(code, same)) { callback = true } }
            if (checked) check(!callback && calls == before)
            expectedCodes = codes
            mode = 1
            nativeEnumUniqueExpect(checked) { api.echo(holder) }
            nativeEnumUniqueExpect(checked) { api.codes(codes) }
            mode = 0
            val count = calls
            val cold = api.watch(); check(calls == count)
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 1
            nativeEnumUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 0
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            check(api.echo(holder) == holder)
        } finally { client.close(); scope.cancel(); jobs.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
        println("NativeEnumUnique checked=$checked defaults=$defaults nulls=$nulls native-equality-membership-bounds-extrema-null-defaults-alias-Patch-preflight-HTTP-liveSSEclose-cold-reuse passed")
    }
}
