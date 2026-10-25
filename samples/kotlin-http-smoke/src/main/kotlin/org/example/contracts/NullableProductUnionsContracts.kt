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
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.nullableproducts.NullableProductUnionsApiClient
import org.example.nullableproducts.api.models.PatchInputDto
import org.example.nullableproducts.api.models.WatchEventDto
import org.example.nullableproducts.models.JointADto
import org.example.nullableproducts.models.JointPairDto
import org.example.nullableproducts.models.JointPairAliasDto
import org.example.nullableproducts.models.JointPairFalseDto
import org.example.nullableproducts.models.JointNarrowDto
import org.example.nullableproducts.models.JointHolderDto
import org.example.nullableproducts.models.JointProductTTDto
import org.example.nullableproducts.models.JointProductTFDto
import org.example.nullableproducts.models.JointProductFTDto
import org.example.nullableproducts.models.JointProductFFDto
import org.example.nullableproducts.models.JointProductAliasDto
import org.example.nullableproducts.models.JointProductMaybeDto
import org.example.nullableproducts.models.JointProductFalseDto
import org.example.nullableproducts.models.JointNullBranchDto
import org.example.nullableproducts.models.NullJointNullBranchDto
import org.example.nullableproducts.models.JointNestedAliasDto
import org.example.nullableproducts.models.JointNestedFalseDto
import org.example.nullableproducts.models.NullJointProductTTDto
import org.example.nullableproducts.models.UnknownJointProductTTDto
import org.example.nullableproducts.api.models.JointWatchEventDto
import org.example.nullableproducts.models.BoxDto
import org.example.nullableproducts.models.HolderDto
import org.example.nullableproducts.models.MaybeDto
import org.example.nullableproducts.models.NestedAliasDto
import org.example.nullableproducts.models.NestedFalseDto
import org.example.nullableproducts.models.NullProductDto
import org.example.nullableproducts.models.Patch
import org.example.nullableproducts.models.ProductDto
import org.example.nullableproducts.models.UnknownProductDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullableProductEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun nullableProductBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableProductBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic nullable product body")
}
private suspend fun nullableProductReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing nullable product rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong nullable product failure: $cause" }
    }
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun nullableProductUnionsContracts(checked: Boolean = false) = withTimeout(30000) {
    check(!serializer<ProductDto>().descriptor.isNullable)
    check(serializer<MaybeDto>().descriptor.isNullable && !MaybeDto.serializer().descriptor.isNullable)
    check(serializer<NestedAliasDto>().descriptor.isNullable && !serializer<NestedFalseDto>().descriptor.isNullable)
    val values = listOf<ProductDto>(ProductDto.A("known"), ProductDto.B(true), NullProductDto)
    if (checked) nullableProductReject("minLength") { ProductDto.A("") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        nullableIntersectionContracts(codec, checked)
        check(codec.decodeFromString<ProductDto>("null") === NullProductDto)
        check(codec.decodeFromString<MaybeDto>("null") == null)
        check(codec.decodeFromString<NestedAliasDto>("null") == null)
        nullableProductReject("object") { codec.decodeFromString<NestedFalseDto>("null") }
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
        nullableProductReject("id") { codec.decodeFromString<ProductDto>("""{"kind":"a","stamp":"shared"}""") }
        if (checked) nullableProductReject("minLength") { codec.decodeFromString<ProductDto>("""{"kind":"a","id":"","stamp":"shared"}""") }
        val patches = listOf(PatchInputDto(), PatchInputDto(value = Patch.Null), PatchInputDto(value = Patch.Value(PatchInputDto.Value.A("patch"))))
        for (value in patches) check(codec.encodeToJsonElement(codec.decodeFromString<PatchInputDto>(codec.encodeToString(value))) == codec.encodeToJsonElement(value))
        var expected: ProductDto = values[0]
        var expectedPatch = patches[0]
        var mode = 0
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        fun payload() = when (mode) { 1 -> "{"; 2 -> """{"kind":"a","id":"","stamp":"shared"}"""; else -> codec.encodeToString(expected) }
        val engine = NullableProductEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/echo" -> {
                    check(codec.parseToJsonElement(nullableProductBody(request.body)) == codec.encodeToJsonElement(expected))
                    respond(payload(), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    check(codec.parseToJsonElement(nullableProductBody(request.body)) == codec.encodeToJsonElement(expectedPatch))
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
            val api = NullableProductUnionsApiClient(http, "https://example.test/", codec).default
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
            nullableProductReject("JSON") { api.echo(expected) }
            nullableProductReject("JSON") { cold.toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            if (checked) {
                mode = 2
                nullableProductReject("minLength") { api.echo(expected) }
                nullableProductReject("minLength") { cold.toList() }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; expected = NullProductDto
            check((cold.take(1).toList().single() as WatchEventDto.Sample).data === NullProductDto)
            check(api.echo(expected) === NullProductDto)
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
        check(codec.encodeToJsonElement<ProductDto>(NullProductDto) == JsonNull)
        println("NullableProductUnions Json defaults=$defaults nulls=$nulls actualProduct-nullCarrier-inlineNullable-originalSource-Patch-HTTP-SSEclose-cancel-cold-reuse passed")
    }
}


private suspend fun nullableIntersectionContracts(codec: Json, checked: Boolean) {
    check(!JointPairDto.serializer().descriptor.isNullable)
    check(serializer<JointPairAliasDto>().descriptor.isNullable)
    check(!serializer<JointPairFalseDto>().descriptor.isNullable)
    check(serializer<JointNestedAliasDto>().descriptor.isNullable)
    check(!serializer<JointNestedFalseDto>().descriptor.isNullable)
    check(codec.decodeFromString<JointPairAliasDto>("null") == null)
    check(codec.decodeFromString<JointNestedAliasDto>("null") == null)
    nullableProductReject("null") { codec.decodeFromString<JointPairFalseDto>("null") }
    nullableProductReject("object") { codec.decodeFromString<JointNestedFalseDto>("null") }
    check(codec.decodeFromString<JointProductTTDto>("null") === NullJointProductTTDto)
    check(codec.decodeFromString<JointProductAliasDto>("null") === NullJointProductTTDto)
    check(codec.decodeFromString<JointProductMaybeDto>("null") == null)
    check(codec.decodeFromString<JointProductFalseDto>("null") === NullJointProductTTDto)
    check(codec.decodeFromString<JointNullBranchDto>("null") === NullJointNullBranchDto)
    nullableProductReject("object") { codec.decodeFromString<JointProductTFDto>("null") }
    nullableProductReject("object") { codec.decodeFromString<JointProductFTDto>("null") }
    nullableProductReject("object") { codec.decodeFromString<JointProductFFDto>("null") }
    val pair = JointPairDto(left = "left", self = JointADto("leaf"), right = "right")
    val narrow = JointNarrowDto(left = "left", right = "right")
    check(codec.decodeFromString<JointPairDto>(codec.encodeToString(pair)) == pair)
    check(JointADto.serializer().descriptor.serialName == "org.example.nullableproducts.models.JointADto")
    val productJson = """{"kind":"a","id":"known","label":"label"}"""
    val product = codec.decodeFromString<JointProductTTDto>(productJson)
    check(product is JointProductTTDto.A && product.id == "known" && product.label == "label")
    val unknownJson = codec.parseToJsonElement("""{"kind":"future","extra":[null,true]}""")
    val unknown = codec.decodeFromJsonElement<JointProductTTDto>(unknownJson)
    check(unknown is UnknownJointProductTTDto && codec.encodeToJsonElement(unknown) == unknownJson)
    val holderJson = """{"pair":{"left":"left","right":"right"},"narrow":{"left":"left","right":"right"},"product":{"kind":"a","id":"known","label":"label"},"mixed":{"kind":"a","id":"known","label":"label"},"inlineProduct":null,"inlineMixed":{"kind":"a","id":"known","label":"label"},"rawValues":[{"x":1},{"x":2}]}"""
    val holder = codec.decodeFromString<JointHolderDto>(holderJson)
    check(holder.pair != null && holder.inlineProduct == null)
    check(codec.encodeToJsonElement(codec.decodeFromString<JointHolderDto>(codec.encodeToString(holder))) == codec.encodeToJsonElement(holder))
    val holderObject = codec.parseToJsonElement(holderJson).jsonObject
    val badNull = JsonObject(holderObject + ("narrow" to JsonNull)).toString()
    val badBound = JsonObject(holderObject + ("pair" to codec.parseToJsonElement("""{"left":"x","right":"right"}"""))).toString()
    val badRaw = JsonObject(holderObject + ("rawValues" to codec.parseToJsonElement("""[{"x":1},{"x":1.0}]"""))).toString()
    nullableProductReject("null") { codec.decodeFromString<JointHolderDto>(badNull) }
    if (checked) {
        nullableProductReject("minLength") { JointPairDto(left = "x", right = "right") }
        nullableProductReject("minLength") { codec.decodeFromString<JointHolderDto>(badBound) }
        nullableProductReject("uniqueItems") { codec.decodeFromString<JointHolderDto>(badRaw) }
    }
    val patches = listOf(
        PatchInputDto(),
        PatchInputDto(pair = Patch.Null, jointProduct = Patch.Null),
        PatchInputDto(pair = Patch.Value(pair), jointProduct = Patch.Value(product))
    )
    for (patch in patches) {
        check(codec.encodeToJsonElement(codec.decodeFromString<PatchInputDto>(codec.encodeToString(patch))) == codec.encodeToJsonElement(patch))
    }
    check(codec.encodeToJsonElement(patches[0]).jsonObject["pair"] == null)
    check(codec.encodeToJsonElement(patches[1]).jsonObject["pair"] == JsonNull)
    var expectedPair: JointPairDto? = pair
    var expectedProduct: JointProductTTDto = product
    var expectedPatch = patches[0]
    var mode = 0
    var calls = 0
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    fun payload() = when (mode) { 1 -> badNull; 2 -> badBound; 3 -> badRaw; else -> codec.encodeToString(holder) }
    val engine = NullableProductEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/joint-pair" -> {
                check(codec.parseToJsonElement(nullableProductBody(request.body)) == codec.encodeToJsonElement(expectedPair))
                respond(codec.encodeToString(expectedPair), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/joint-narrow" -> {
                check(codec.parseToJsonElement(nullableProductBody(request.body)) == codec.encodeToJsonElement(narrow))
                respond(if (mode == 1) "null" else codec.encodeToString(narrow), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/joint-product" -> {
                check(codec.parseToJsonElement(nullableProductBody(request.body)) == codec.encodeToJsonElement(expectedProduct))
                respond(codec.encodeToString(expectedProduct), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/joint-echo" -> {
                check(codec.parseToJsonElement(nullableProductBody(request.body)) == codec.encodeToJsonElement(holder))
                respond(payload(), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nullableProductBody(request.body)) == codec.encodeToJsonElement(expectedPatch))
                respond("", HttpStatusCode.NoContent)
            }
            "/joint-watch" -> {
                val channel = ByteChannel(autoFlush = true); channels += channel
                scope.launch { try {
                    for (byte in "event: sample\r\ndata: ${payload()}\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    awaitCancellation()
                } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected synthetic intersection URL")
        }
    } })
    val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
    try {
        val api = NullableProductUnionsApiClient(http, "https://example.test/", codec).default
        check(api.jointPair(expectedPair) == expectedPair)
        expectedPair = null; check(api.jointPair(null) == null); expectedPair = pair
        check(api.jointNarrow(narrow) == narrow)
        for (value in listOf(product, NullJointProductTTDto, unknown)) { expectedProduct = value; check(codec.encodeToJsonElement(api.jointProduct(value)) == codec.encodeToJsonElement(value)) }
        check(api.jointEcho(holder) == holder)
        for (patch in patches) { expectedPatch = patch; api.patchPatch(patch) }
        val before = calls
        val cold = api.jointWatch(); check(calls == before)
        check((cold.take(1).toList().single() as JointWatchEventDto.Sample).data == holder)
        withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
        mode = 1
        nullableProductReject("null") { api.jointNarrow(narrow) }
        nullableProductReject("null") { api.jointEcho(holder) }
        nullableProductReject("null") { cold.toList() }
        withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
        if (checked) {
            for ((bad, message) in listOf(2 to "minLength", 3 to "uniqueItems")) {
                mode = bad
                nullableProductReject(message) { api.jointEcho(holder) }
                nullableProductReject(message) { cold.toList() }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
        }
        mode = 0
        check((cold.take(1).toList().single() as JointWatchEventDto.Sample).data == holder)
        check(api.jointEcho(holder) == holder)
    } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
    println("Nullable intersections AND-OR-alias-physical-codec-recursion-raw-equality-Patch-HTTP-liveSSE passed checked=$checked")
}
