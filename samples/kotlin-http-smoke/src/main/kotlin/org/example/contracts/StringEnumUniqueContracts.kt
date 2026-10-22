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
import org.example.enumunique.StringEnumUniqueApiClient
import org.example.enumunique.api.models.PatchInputDto
import org.example.enumunique.api.models.WatchEventDto
import org.example.enumunique.models.HolderDto
import org.example.enumunique.models.HolderDto.CodesInner as Code
import org.example.enumunique.models.HolderDto.NestedInnerInner as NestedCode
import org.example.enumunique.models.HolderDto.BoundedInner as BoundedCode
import org.example.enumunique.models.RowsDtoInner as RowCode
import org.example.enumunique.api.models.PatchInputDto.CodesInner as PatchCode
import org.example.enumunique.models.Patch
import org.example.enumunique.models.StateDto as RefCode
import org.example.enumunique.models.OwnerDto.State as InternalCode
import java.util.concurrent.CopyOnWriteArrayList

private fun stringEnumUniqueBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> stringEnumUniqueBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic String enum unique body")
}

private suspend fun stringEnumUniqueExpect(checked: Boolean, action: suspend () -> Unit) {
    if (!checked) { action(); return }
    try { action(); error("Missing String enum uniqueItems rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains("uniqueItems") == true
        }) { "Wrong String enum uniqueness failure: $cause" }
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StringEnumUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

suspend fun stringEnumUniqueContracts(checked: Boolean = true) = withTimeout(20000) {
    val codes = listOf(Code.READY, Code.Unrecognized("future"), Code.NULL, null, Code.Unrecognized("é"), Code.Unrecognized("e\u0301"))
    val rows = listOf(RowCode.READY, RowCode.Unrecognized("future"), RowCode.NULL, null)
    val refCodes = listOf(RefCode.READY, RefCode.Unrecognized("future"), RefCode.NULL, null)
    val internalCodes = listOf(InternalCode.READY, InternalCode.Unrecognized("future"), InternalCode.NULL, null)
    val patchValues = listOf(PatchCode.READY, PatchCode.Unrecognized("future"), null)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = HolderDto(codes = codes, rows = rows, bounded = listOf(BoundedCode.OK, BoundedCode.Unrecognized("future")), refCodes = refCodes, refNarrow = refCodes.filterNotNull(), internalCodes = internalCodes, internalNarrow = refCodes.filterNotNull(), bridgeCodes = refCodes)
        val encoded = codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(encoded) == holder)
        check(HolderDto().codes == listOf(Code.READY, null))
        check(HolderDto().nested == listOf(listOf(NestedCode.READY, null)))
        check(HolderDto().refRows == listOf(RefCode.READY, null))
        check(HolderDto().refNested == listOf(mapOf("entry" to listOf(InternalCode.READY, null))))
        check(HolderDto().refNested !== HolderDto().refNested)
        val refDuplicate = listOf(RefCode.READY, RefCode.Unrecognized("ready"))
        val internalDuplicate = listOf(InternalCode.READY, InternalCode.Unrecognized("ready"))
        stringEnumUniqueExpect(checked) { HolderDto(refCodes = refDuplicate) }
        stringEnumUniqueExpect(checked) { HolderDto(refNarrow = refDuplicate) }
        stringEnumUniqueExpect(checked) { HolderDto(internalCodes = internalDuplicate) }
        stringEnumUniqueExpect(checked) { HolderDto(internalNarrow = refDuplicate) }
        stringEnumUniqueExpect(checked) { HolderDto(bridgeCodes = refDuplicate) }
        stringEnumUniqueExpect(checked) { HolderDto(refRows = refDuplicate) }
        stringEnumUniqueExpect(checked) { HolderDto(refNested = listOf(mapOf("entry" to internalDuplicate))) }
        stringEnumUniqueExpect(checked) { HolderDto(refCodes = listOf(null, null)) }
        stringEnumUniqueExpect(checked) { PatchInputDto(refCodes = Patch.Value(refDuplicate)) }
        HolderDto(refRows = null, refCodes = null)
        PatchInputDto(refCodes = Patch.Null)
        check(Code.Unrecognized("ready") != Code.READY)
        for (duplicate in listOf(listOf(Code.READY, Code.Unrecognized("ready")), listOf(Code.Unrecognized("future"), Code.Unrecognized("future")), listOf(null, null))) {
            stringEnumUniqueExpect(checked) { HolderDto(codes = duplicate) }
        }
        stringEnumUniqueExpect(checked) { HolderDto(rows = listOf(RowCode.READY, RowCode.Unrecognized("ready"))) }
        stringEnumUniqueExpect(checked) { HolderDto(nested = listOf(listOf(NestedCode.READY, NestedCode.Unrecognized("ready")))) }
        stringEnumUniqueExpect(checked) { PatchInputDto(codes = Patch.Value(listOf(PatchCode.READY, PatchCode.Unrecognized("ready")))) }
        if (checked) check(runCatching { HolderDto(bounded = listOf(BoundedCode.X)) }.isFailure)
        val badRows = "[\"ready\",\"\\u0072eady\"]"
        val bad = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("codes" to codec.parseToJsonElement(badRows))).toString()
        stringEnumUniqueExpect(checked) { codec.decodeFromString<HolderDto>(bad) }
        val badRef = JsonObject(codec.parseToJsonElement(encoded).jsonObject + ("refCodes" to codec.parseToJsonElement(badRows))).toString()
        stringEnumUniqueExpect(checked) { codec.decodeFromString<HolderDto>(badRef) }
        for (key in listOf("refNarrow", "internalCodes", "internalNarrow", "bridgeCodes", "refRows")) {
            val invalid = JsonObject(codec.parseToJsonElement(encoded).jsonObject + (key to codec.parseToJsonElement(badRows))).toString()
            stringEnumUniqueExpect(checked) { codec.decodeFromString<HolderDto>(invalid) }
        }
        check(runCatching { codec.decodeFromString<HolderDto>("{\"refNarrow\":[null]}") }.isFailure)
        PatchInputDto(); PatchInputDto(codes = Patch.Null)

        var mode = 0
        var calls = 0
        var expectedRefs: List<RefCode?>? = refCodes
        var expectedRows: List<RowCode?> = rows
        var expectedPatch = PatchInputDto()
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val jobs = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StringEnumUniqueEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(stringEnumUniqueBody(request.body)) == codec.parseToJsonElement(encoded))
                    respond(if (mode == 1) bad else encoded, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/rows" -> {
                    check(codec.parseToJsonElement(stringEnumUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedRows)))
                    respond(if (mode == 1) badRows else codec.encodeToString(expectedRows), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/refrows" -> {
                    check(codec.parseToJsonElement(stringEnumUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedRefs)))
                    respond(if (mode == 1) badRows else codec.encodeToString(expectedRefs), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(stringEnumUniqueBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(expectedPatch)))
                    respond("", HttpStatusCode.NoContent)
                }
                "/watch" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val payload = when (mode) { 1 -> bad; 2 -> badRef; else -> encoded }
                    jobs += scope.launch {
                        for (byte in "event: sample\r\ndata: $payload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        awaitCancellation()
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic String enum unique URL")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = StringEnumUniqueApiClient(client, "https://example.test/", codec).default
            check(api.echo(holder) == holder)
            check(api.rows(rows) == rows)
            check(api.refrows(refCodes) == refCodes)
            expectedRefs = null
            check(api.refrows(null) == null)
            expectedRefs = refCodes
            expectedRows = rows
            for (patch in listOf(PatchInputDto(), PatchInputDto(codes = Patch.Null), PatchInputDto(codes = Patch.Value(patchValues)), PatchInputDto(refCodes = Patch.Value(refCodes)), PatchInputDto(refCodes = Patch.Null))) {
                expectedPatch = patch; api.patchPatch(patch)
            }
            check(codec.parseToJsonElement(codec.encodeToString(PatchInputDto(codes = Patch.Null))).jsonObject["codes"] == JsonNull)
            val before = calls
            var callback = false
            expectedRows = listOf(RowCode.READY, RowCode.Unrecognized("ready"))
            stringEnumUniqueExpect(checked) { api.rows(expectedRows) { callback = true } }
            if (checked) check(!callback && calls == before)
            expectedRows = rows
            val beforeRefs = calls
            callback = false
            expectedRefs = refDuplicate
            stringEnumUniqueExpect(checked) { api.refrows(refDuplicate) { callback = true } }
            if (checked) check(!callback && calls == beforeRefs)
            expectedRefs = refCodes
            mode = 1
            stringEnumUniqueExpect(checked) { api.echo(holder) }
            stringEnumUniqueExpect(checked) { api.rows(rows) }
            stringEnumUniqueExpect(checked) { api.refrows(refCodes) }
            mode = 0
            val count = calls
            val cold = api.watch(); check(calls == count)
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 1
            stringEnumUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 2
            stringEnumUniqueExpect(checked) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForRead) yield() }
            mode = 0
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data == holder)
            check(api.echo(holder) == holder)
        } finally { client.close(); scope.cancel(); jobs.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
        println("StringEnumUnique checked=$checked defaults=$defaults nulls=$nulls serialName-equality-open-null-defaults-named-internal-ref-narrowing-Patch-preflight-HTTP-liveSSEclose-cold-reuse passed")
    }
}
