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
import org.example.commonnominals.UnionCommonNominalsApiClient
import org.example.commonnominals.api.models.PatchInputDto
import org.example.commonnominals.api.models.WatchEventDto
import org.example.commonnominals.models.BooleanNativeDto
import org.example.commonnominals.models.DirectDto
import org.example.commonnominals.models.FirstDto
import org.example.commonnominals.models.HierarchyChoiceDto
import org.example.commonnominals.models.HolderDto
import org.example.commonnominals.models.MismatchDto
import org.example.commonnominals.models.NativeChoiceDto
import org.example.commonnominals.models.NativeDto
import org.example.commonnominals.models.NullNativeDto
import org.example.commonnominals.models.NullableChoiceDto
import org.example.commonnominals.models.ObjectChoiceDto
import org.example.commonnominals.models.Patch
import org.example.commonnominals.models.ProductDto
import org.example.commonnominals.models.RecordDto
import org.example.commonnominals.models.StringNativeDto
import org.example.commonnominals.models.UnknownDirectDto
import org.example.commonnominals.models.UnknownNullableChoiceDto
import org.example.commonnominals.models.UnknownValueDto
import org.example.commonnominals.models.ValueDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class CommonNominalStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun commonNominalBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> commonNominalBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic common nominal body")
}
private suspend fun commonNominalReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing common nominal rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong common nominal failure: $cause" }
    }
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun unionCommonNominalsContracts(checked: Boolean = false) = withTimeout(30000) {
    val record = RecordDto("record")
    val direct: DirectDto = DirectDto.A(record)
    val objectChoice: ObjectChoiceDto = ObjectChoiceDto.B(ValueDto.Id("value"))
    val hierarchy: HierarchyChoiceDto = HierarchyChoiceDto.A(FirstDto("first"))
    val product: ProductDto = ProductDto.B(record)
    check(direct.value.id == "record" && (objectChoice.value as ValueDto.Id).id == "value")
    check((hierarchy.value as FirstDto).id == "first" && product.value.id == "record")
    fun holder(native: NativeDto = StringNativeDto("native"), value: ValueDto = ValueDto.Id("value")) = HolderDto(
        direct = direct, objectChoice = ObjectChoiceDto.B(value), nativeChoice = NativeChoiceDto.A(native),
        nullableChoice = NullableChoiceDto.B(null), hierarchyChoice = hierarchy,
        mismatch = MismatchDto.A(record), product = product,
    )
    if (checked) commonNominalReject("minLength") { RecordDto("") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        for (native in listOf(StringNativeDto("native"), BooleanNativeDto(true), NullNativeDto)) {
            val value = holder(native)
            check(value.nativeChoice.value === native)
            val wire = codec.encodeToJsonElement(value)
            check(codec.encodeToJsonElement(codec.decodeFromJsonElement<HolderDto>(wire)) == wire)
        }
        val decodedNull = codec.decodeFromString<NativeChoiceDto>("""{"kind":"a","value":null}""")
        check(decodedNull.value === NullNativeDto)
        val innerWire = codec.parseToJsonElement("""{"kind":"future","extra":[true,null]}""")
        val innerUnknown = codec.decodeFromJsonElement<ValueDto>(innerWire) as UnknownValueDto
        val model = holder(value = innerUnknown)
        check((model.objectChoice.value as UnknownValueDto).rawElement == innerWire)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(model))) == codec.encodeToJsonElement(model))
        val outerWire = codec.parseToJsonElement("""{"kind":"future","value":{"id":"raw"},"extra":true}""")
        val outer = codec.decodeFromJsonElement<DirectDto>(outerWire) as UnknownDirectDto
        check(outer.rawElement == outerWire && codec.encodeToJsonElement<DirectDto>(outer) == outerWire)
        try { outer.value; error("Missing common nominal Unknown access rejection") }
        catch (cause: IllegalStateException) { check(cause.message!!.contains("not available")) }
        val nullableUnknown = codec.decodeFromJsonElement<NullableChoiceDto>(outerWire) as UnknownNullableChoiceDto
        check(nullableUnknown.value == null && nullableUnknown.rawElement == outerWire)
        commonNominalReject("value") { codec.decodeFromString<DirectDto>("""{"kind":"a"}""") }
        commonNominalReject("null") { codec.decodeFromString<DirectDto>("""{"kind":"a","value":null}""") }
        commonNominalReject("id") { codec.decodeFromString<DirectDto>("""{"kind":"a","value":{}}""") }
        if (checked) commonNominalReject("minLength") { codec.decodeFromString<DirectDto>("""{"kind":"a","value":{"id":""}}""") }
        val absent = PatchInputDto(requiredNullable = null)
        val clear = PatchInputDto(requiredNullable = null, value = Patch.Null, choice = Patch.Null)
        val assigned = PatchInputDto(requiredNullable = null, value = Patch.Value(record), choice = Patch.Value(ValueDto.Id("value")))
        for (input in listOf(absent, clear, assigned)) {
            val wire = codec.encodeToJsonElement(input)
            check(codec.encodeToJsonElement(codec.decodeFromJsonElement<PatchInputDto>(wire)) == wire)
        }
        check("value" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject).getValue("value") == JsonNull)
        var expected = model
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
        val engine = CommonNominalStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(commonNominalBody(request.body)) == codec.encodeToJsonElement(expected))
                    respond(responseWire(), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(commonNominalBody(request.body)) == codec.encodeToJsonElement(expectedPatch))
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
                else -> error("Unexpected synthetic common nominal URL")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = UnionCommonNominalsApiClient(http, "https://example.test/", codec).default
            for (value in listOf(holder(), holder(NullNativeDto), model)) {
                expected = value
                check(codec.encodeToJsonElement(api.echo(value)) == codec.encodeToJsonElement(value))
            }
            mode = 1; commonNominalReject("null") { api.echo(expected) }
            if (checked) { mode = 2; commonNominalReject("minLength") { api.echo(expected) } }
            mode = 3; commonNominalReject("JSON") { api.echo(expected) }
            mode = 0
            for (input in listOf(absent, clear, assigned)) { expectedPatch = input; api.patchPatch(input) }
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val event = withTimeout(3000) { cold.toList().single() } as WatchEventDto.Sample
                check(codec.encodeToJsonElement(event.data) == codec.encodeToJsonElement(expected))
            }
            for ((bad, message) in listOf(1 to "null", 3 to "JSON") + if (checked) listOf(2 to "minLength") else emptyList()) {
                mode = bad
                withTimeout(3000) { commonNominalReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; endless = true
            withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false
            check(withTimeout(3000) { cold.toList() }.size == 1)
            check(codec.encodeToJsonElement(api.echo(expected)) == codec.encodeToJsonElement(expected))
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
        println("UnionCommonNominals Json defaults=$defaults nulls=$nulls actualDTO-union-hierarchy-codecs-nullUnknown-originalAuthority-Patch-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}
