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
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import org.example.commondictionaries.UnionCommonDictionariesApiClient
import org.example.commondictionaries.api.models.PatchInputDto
import org.example.commondictionaries.api.models.WatchEventDto
import org.example.commondictionaries.models.CheckedDto
import org.example.commondictionaries.models.DirectDto
import org.example.commondictionaries.models.HolderDto
import org.example.commondictionaries.models.MixedDto
import org.example.commondictionaries.models.NamedDto
import org.example.commondictionaries.models.NullableValuesDto
import org.example.commondictionaries.models.OptionalDto
import org.example.commondictionaries.models.Patch
import org.example.commondictionaries.models.ProductDto
import org.example.commondictionaries.models.RawDto
import org.example.commondictionaries.models.RecordDto
import org.example.commondictionaries.models.UnknownDirectDto
import org.example.commondictionaries.models.UnknownOptionalDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class CommonDictionaryStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun commonDictionaryBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> commonDictionaryBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic common dictionary body")
}
private suspend fun commonDictionaryReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing common dictionary rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong common dictionary failure: $cause" }
    }
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun unionCommonDictionariesContracts(checked: Boolean = false) = withTimeout(30000) {
    val mapping = mapOf("key" to "good")
    val direct: DirectDto = DirectDto.A(mapping)
    val named: NamedDto = NamedDto.B(mapOf("key" to RecordDto("record")))
    val raw: RawDto = RawDto.A(JsonObject(mapOf("nil" to JsonNull, "flag" to JsonPrimitive(true))))
    val product: ProductDto = ProductDto.B(mapping)
    check(direct.value == mapping && named.value.getValue("key").id == "record")
    check(raw.value.getValue("nil") == JsonNull && product.value == mapping && product.stamp == "shared")
    fun holder(variant: CheckedDto = CheckedDto.A(mapping)) = HolderDto(
        direct = direct, named = named, optional = OptionalDto.B(),
        nullableValues = NullableValuesDto.B(mapOf("key" to null)), raw = raw,
        mixed = MixedDto.B(mapOf("key" to 3)), checked = variant, product = product,
    )
    if (checked) {
        commonDictionaryReject("minProperties") { CheckedDto.A(emptyMap()) }
        commonDictionaryReject("minLength") { CheckedDto.A(mapOf("key" to "x")) }
        commonDictionaryReject("minLength") { CheckedDto.B(mapOf("key" to "ok")) }
        commonDictionaryReject("maxProperties") { CheckedDto.B(mapOf("a" to "good", "b" to "good", "c" to "good")) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        for (model in listOf(holder(), holder(CheckedDto.B(mapping)))) {
            val wire = codec.encodeToJsonElement(model)
            check(codec.encodeToJsonElement(codec.decodeFromJsonElement<HolderDto>(wire)) == wire)
        }
        check(codec.decodeFromString<NamedDto>("""{"kind":"a","value":{"key":{"id":"record"}}}""").value.getValue("key").id == "record")
        check(codec.decodeFromString<NullableValuesDto>("""{"kind":"a","value":{"key":null}}""").value.getValue("key") == null)
        check(codec.decodeFromString<OptionalDto>("""{"kind":"b"}""").value == null)
        commonDictionaryReject("value") { codec.decodeFromString<DirectDto>("""{"kind":"a"}""") }
        commonDictionaryReject("JSON") { codec.decodeFromString<DirectDto>("""{"kind":"a","value":[]}""") }
        val unknownWire = codec.parseToJsonElement("""{"kind":"future","value":{"key":"unused"},"extra":[true,null]}""")
        val unknown = codec.decodeFromJsonElement<DirectDto>(unknownWire) as UnknownDirectDto
        check(unknown.rawElement == unknownWire && codec.encodeToJsonElement<DirectDto>(unknown) == unknownWire)
        try { unknown.value; error("Missing Unknown common property rejection") }
        catch (cause: IllegalStateException) { check(cause.message!!.contains("not available")) }
        val unknownOptional = codec.decodeFromJsonElement<OptionalDto>(unknownWire) as UnknownOptionalDto
        check(unknownOptional.value == null && unknownOptional.rawElement == unknownWire)
        if (checked) {
            commonDictionaryReject("minProperties") { codec.decodeFromString<CheckedDto>("""{"kind":"a","value":{}}""") }
            commonDictionaryReject("minLength") { codec.decodeFromString<CheckedDto>("""{"kind":"b","value":{"key":"ok"}}""") }
        }
        val absent = PatchInputDto(requiredNullable = null)
        val clear = PatchInputDto(requiredNullable = null, value = Patch.Null, map = Patch.Null)
        val assigned = PatchInputDto(requiredNullable = null, value = Patch.Value(direct), map = Patch.Value(mapping))
        for (input in listOf(absent, clear, assigned)) {
            val wire = codec.encodeToJsonElement(input)
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
            if (mode == 1) return JsonObject(base + ("direct" to codec.parseToJsonElement("""{"kind":"a","value":[]}"""))).toString()
            if (mode == 2) return JsonObject(base + ("checked" to codec.parseToJsonElement("""{"kind":"a","value":{}}"""))).toString()
            return base.toString()
        }
        val engine = CommonDictionaryStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(commonDictionaryBody(request.body)) == codec.encodeToJsonElement(expected))
                    respond(responseWire(), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(commonDictionaryBody(request.body)) == codec.encodeToJsonElement(expectedPatch))
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
                else -> error("Unexpected synthetic common dictionary URL")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = UnionCommonDictionariesApiClient(http, "https://example.test/", codec).default
            for (value in listOf(holder(), holder(CheckedDto.B(mapping)))) {
                expected = value
                check(codec.encodeToJsonElement(api.echo(value)) == codec.encodeToJsonElement(value))
            }
            mode = 1; commonDictionaryReject("JSON") { api.echo(expected) }
            if (checked) { mode = 2; commonDictionaryReject("minProperties") { api.echo(expected) } }
            mode = 0
            for (input in listOf(absent, clear, assigned)) { expectedPatch = input; api.patchPatch(input) }
            val before = calls
            val cold = api.watch(); check(calls == before)
            repeat(2) {
                val event = withTimeout(3000) { cold.toList().single() } as WatchEventDto.Sample
                check(codec.encodeToJsonElement(event.data) == codec.encodeToJsonElement(expected))
            }
            for ((bad, message) in listOf(1 to "JSON") + if (checked) listOf(2 to "minProperties") else emptyList()) {
                mode = bad
                withTimeout(3000) { commonDictionaryReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; endless = true
            withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false
            check(withTimeout(3000) { cold.toList() }.size == 1)
            check(codec.encodeToJsonElement(api.echo(expected)) == codec.encodeToJsonElement(expected))
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
        println("UnionCommonDictionaries Json defaults=$defaults nulls=$nulls commonMap-JsonObject-original-nominalCodec-variantChecks-mismatch-Unknown-Patch-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}
