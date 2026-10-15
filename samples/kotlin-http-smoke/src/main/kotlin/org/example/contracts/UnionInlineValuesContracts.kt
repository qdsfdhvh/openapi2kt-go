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
import io.ktor.utils.io.ClosedByteChannelException
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
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
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import org.example.inlinevalues.UnionInlineValuesApiClient
import org.example.inlinevalues.api.models.PatchInputDto
import org.example.inlinevalues.api.models.WatchEventDto
import org.example.inlinevalues.models.DirectDto
import org.example.inlinevalues.models.HolderDto
import org.example.inlinevalues.models.MappingDto
import org.example.inlinevalues.models.MismatchDto
import org.example.inlinevalues.models.NestedDto
import org.example.inlinevalues.models.NullableDto
import org.example.inlinevalues.models.Patch
import org.example.inlinevalues.models.ProductDto
import org.example.inlinevalues.models.RowsDto
import org.example.inlinevalues.models.UnknownDirectDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class InlineValueStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun inlineValueBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> inlineValueBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic inline value body")
}
private suspend fun inlineValueReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing inline value rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong inline value failure: $cause" }
    }
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun unionInlineValuesContracts(checked: Boolean = false) = withTimeout(30000) {
    fun holder(second: Boolean = false) = if (!second) HolderDto(
        direct = DirectDto.A(DirectDto.A.Value("first")),
        rows = RowsDto.A(listOf(RowsDto.A.ValueInner("row"))),
        mapping = MappingDto.A(mapOf("key" to MappingDto.A.ValueValue("mapped"))),
        nullable = NullableDto.A(null),
        mismatch = MismatchDto.A(MismatchDto.A.Value("different")),
        nested = NestedDto.A(NestedDto.A.Value.Id("inner")),
        product = ProductDto.A(ProductDto.A.Value("product")),
    ) else HolderDto(
        direct = DirectDto.B(DirectDto.B.Value("second")),
        rows = RowsDto.B(listOf(RowsDto.B.ValueInner("row"))),
        mapping = MappingDto.B(mapOf("key" to MappingDto.B.ValueValue("mapped"))),
        nullable = NullableDto.B(),
        mismatch = MismatchDto.B(MismatchDto.B.Value(true)),
        nested = NestedDto.B(NestedDto.B.Value.Ok(true)),
        product = ProductDto.B(ProductDto.B.Value("product")),
    )
    check((holder().direct as DirectDto.A).value.id == "first")
    check((holder(true).mismatch as MismatchDto.B).value.ok)
    if (checked) inlineValueReject("minLength") { DirectDto.A.Value("") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        for (value in listOf(holder(), holder(true))) {
            val wire = codec.encodeToJsonElement(value)
            check(codec.encodeToJsonElement(codec.decodeFromJsonElement<HolderDto>(wire)) == wire)
        }
        val raw = codec.parseToJsonElement("""{"kind":"future","value":{"id":"raw"},"extra":[true,null]}""")
        val unknown = codec.decodeFromJsonElement<DirectDto>(raw) as UnknownDirectDto
        check(unknown.rawElement == raw && codec.encodeToJsonElement<DirectDto>(unknown) == raw)
        inlineValueReject("value") { codec.decodeFromString<DirectDto>("""{"kind":"a"}""") }
        inlineValueReject("null") { codec.decodeFromString<DirectDto>("""{"kind":"a","value":null}""") }
        inlineValueReject("id") { codec.decodeFromString<DirectDto>("""{"kind":"a","value":{}}""") }
        if (checked) inlineValueReject("minLength") { codec.decodeFromString<DirectDto>("""{"kind":"a","value":{"id":""}}""") }
        val absent = PatchInputDto()
        val clear = PatchInputDto(value = Patch.Null, choice = Patch.Null)
        val assigned = PatchInputDto(value = Patch.Value(PatchInputDto.Value("patch")), choice = Patch.Value(PatchInputDto.Choice.A(PatchInputDto.Choice.A.Value("choice"))))
        for (value in listOf(absent, clear, assigned)) {
            val wire = codec.encodeToJsonElement(value)
            check(codec.encodeToJsonElement(codec.decodeFromJsonElement<PatchInputDto>(wire)) == wire)
        }
        check("value" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject).getValue("value") == JsonNull)
        var expected = holder()
        var expectedPatch = absent
        var mode = 0
        var calls = 0
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        fun responseWire(): String {
            val base = codec.encodeToJsonElement(expected) as JsonObject
            if (mode == 1) return JsonObject(base + ("direct" to codec.parseToJsonElement("""{"kind":"a","value":null}"""))).toString()
            if (mode == 2) return JsonObject(base + ("direct" to codec.parseToJsonElement("""{"kind":"a","value":{"id":""}}"""))).toString()
            if (mode == 3) return "{"
            return base.toString()
        }
        val engine = InlineValueStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(inlineValueBody(request.body)) == codec.encodeToJsonElement(expected))
                    respond(responseWire(), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(inlineValueBody(request.body)) == codec.encodeToJsonElement(expectedPatch))
                    respond("", HttpStatusCode.NoContent)
                }
                "/watch" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val frame = "event: sample\r\ndata: ${responseWire()}\r\n\r\n"
                    val keepOpen = endless || mode != 0
                    scope.launch { try {
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic inline value URL")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = UnionInlineValuesApiClient(http, "https://example.test/", codec).default
            for (value in listOf(holder(), holder(true))) {
                expected = value
                check(codec.encodeToJsonElement(api.echo(value)) == codec.encodeToJsonElement(value))
            }
            for (value in listOf(absent, clear, assigned)) { expectedPatch = value; api.patchPatch(value) }
            mode = 1
            inlineValueReject("null") { api.echo(expected) }
            inlineValueReject("null") { api.watch().toList() }
            check(channels.last().isClosedForWrite)
            if (checked) {
                mode = 2
                inlineValueReject("minLength") { api.echo(expected) }
                inlineValueReject("minLength") { api.watch().toList() }
                check(channels.last().isClosedForWrite)
            }
            mode = 3
            inlineValueReject("EOF") { api.watch().toList() }
            check(channels.last().isClosedForWrite)
            mode = 0; endless = true
            val stream = api.watch()
            val before = calls
            check(calls == before)
            repeat(2) {
                val event = stream.take(1).toList().single() as WatchEventDto.Sample
                check(codec.encodeToJsonElement(event.data) == codec.encodeToJsonElement(expected))
                while (!channels.last().isClosedForWrite) yield()
            }
            check(calls == before + 2)
            check(codec.encodeToJsonElement(api.echo(expected)) == codec.encodeToJsonElement(expected))
        } finally { http.close(); scope.cancel() }
        println("UnionInlineValues Json defaults=$defaults nulls=$nulls lexicalOwners-noCommonIdentity-originalChecks-Patch-HTTP-SSEclose-cancel-cold-reuse passed")
    }
}
