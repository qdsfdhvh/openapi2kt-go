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
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.nullablerefmembers.NullableObjectReferenceMembersApiClient
import org.example.nullablerefmembers.api.models.PatchInputDto
import org.example.nullablerefmembers.api.models.WatchEventDto
import org.example.nullablerefmembers.models.BoxDto
import org.example.nullablerefmembers.models.HolderDto
import org.example.nullablerefmembers.models.MaybeDto
import org.example.nullablerefmembers.models.NestedAliasDto
import org.example.nullablerefmembers.models.NestedFalseDto
import org.example.nullablerefmembers.models.NullProductDto
import org.example.nullablerefmembers.models.Patch
import org.example.nullablerefmembers.models.ProductDto
import org.example.nullablerefmembers.models.UnknownProductDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullableReferenceMemberEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun nullableMemberBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableMemberBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic nullable product body")
}
private suspend fun nullableMemberReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing nullable product rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong nullable product failure: $cause" }
    }
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun nullableObjectReferenceMembersContracts(checked: Boolean = false) = withTimeout(30000) {
    check(!serializer<ProductDto>().descriptor.isNullable)
    check(serializer<MaybeDto>().descriptor.isNullable && !MaybeDto.serializer().descriptor.isNullable)
    check(serializer<NestedAliasDto>().descriptor.isNullable && !serializer<NestedFalseDto>().descriptor.isNullable)
    val values = listOf<ProductDto>(ProductDto.OriginalProductDto("known"), ProductDto.B(true), NullProductDto)
    if (checked) nullableMemberReject("minLength") { ProductDto.OriginalProductDto("") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<ProductDto>("null") === NullProductDto)
        check(codec.decodeFromString<MaybeDto>("null") == null)
        check(codec.decodeFromString<NestedAliasDto>("null") == null)
        nullableMemberReject("object") { codec.decodeFromString<NestedFalseDto>("null") }
        check(codec.decodeFromString<MaybeDto>(codec.encodeToString<MaybeDto>(NullProductDto)) == null)
        val raw = codec.parseToJsonElement("""{"kind":"future","extra":[true,null]}""")
        val unknown = codec.decodeFromJsonElement<ProductDto>(raw) as UnknownProductDto
        check(codec.encodeToJsonElement<ProductDto>(unknown) == raw)
        for (value in values + unknown) {
            check(codec.encodeToJsonElement(codec.decodeFromString<ProductDto>(codec.encodeToString(value))) == codec.encodeToJsonElement(value))
            val holder = HolderDto(value, null, value, BoxDto(null))
            check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))) == codec.encodeToJsonElement(holder))
        }
        val nested: NestedFalseDto = BoxDto.Value.A("nested")
        check(codec.encodeToJsonElement(codec.decodeFromString<BoxDto>(codec.encodeToString(BoxDto(nested)))) == codec.encodeToJsonElement(BoxDto(nested)))
        nullableMemberReject("id") { codec.decodeFromString<ProductDto>("""{"kind":"a","stamp":"shared"}""") }
        if (checked) nullableMemberReject("minLength") { codec.decodeFromString<ProductDto>("""{"kind":"a","id":"","stamp":"shared"}""") }
        val patches = listOf(PatchInputDto(), PatchInputDto(value = Patch.Null), PatchInputDto(value = Patch.Value(PatchInputDto.Value.A("patch"))))
        for (value in patches) check(codec.encodeToJsonElement(codec.decodeFromString<PatchInputDto>(codec.encodeToString(value))) == codec.encodeToJsonElement(value))
        var expected: ProductDto = values[0]
        var expectedPatch = patches[0]
        var mode = 0
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        fun payload() = when (mode) { 1 -> "{"; 2 -> """{"kind":"a","id":"","stamp":"shared"}"""; else -> codec.encodeToString(expected) }
        val engine = NullableReferenceMemberEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(nullableMemberBody(request.body)) == codec.encodeToJsonElement(expected))
                    respond(payload(), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(nullableMemberBody(request.body)) == codec.encodeToJsonElement(expectedPatch))
                    respond("", HttpStatusCode.NoContent)
                }
                "/watch" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    scope.launch { try {
                        for (byte in "event: sample\r\ndata: ${payload()}\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected synthetic nullable product URL")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = NullableObjectReferenceMembersApiClient(http, "https://example.test/", codec).default
            for (value in values + unknown) { expected = value; check(codec.encodeToJsonElement(api.echo(value)) == codec.encodeToJsonElement(value)) }
            for (patch in patches) { expectedPatch = patch; api.patchPatch(patch) }
            val before = calls; val cold = api.watch(); check(calls == before)
            for (value in values + unknown) {
                expected = value
                val event = cold.take(1).toList().single() as WatchEventDto.Sample
                check(codec.encodeToJsonElement(event.data) == codec.encodeToJsonElement(value))
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 1
            nullableMemberReject("JSON") { api.echo(expected) }
            nullableMemberReject("JSON") { cold.toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            if (checked) {
                mode = 2
                nullableMemberReject("minLength") { api.echo(expected) }
                nullableMemberReject("minLength") { cold.toList() }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; expected = NullProductDto
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data === NullProductDto)
            check(api.echo(expected) === NullProductDto)
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
        check(codec.encodeToJsonElement<ProductDto>(NullProductDto) == JsonNull)
        println("NullableObjectReferenceMembers Json defaults=$defaults nulls=$nulls actualReferenceMembers-nullCarrier-inlineNullable-originalSource-Patch-HTTP-SSEclose-cancel-cold-reuse passed")
    }
}
