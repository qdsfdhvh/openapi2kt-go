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
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.ClosedByteChannelException
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
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
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.productterminals.ComposedUnionTerminalsApiClient
import org.example.productterminals.api.models.PatchProductInputDto
import org.example.productterminals.api.models.WatchProductEventDto
import org.example.productterminals.models.AConsumerDto
import org.example.productterminals.models.BoxDto
import org.example.productterminals.models.HolderDto
import org.example.productterminals.models.MapAliasDto
import org.example.productterminals.models.MapDtoValue
import org.example.productterminals.models.NarrowBridgeDto
import org.example.productterminals.models.NestedFalseDto
import org.example.productterminals.models.NestedMaybeDto
import org.example.productterminals.models.Patch
import org.example.productterminals.models.ProductDto
import org.example.productterminals.models.ProductFalseDto
import org.example.productterminals.models.ProductMaybeDto
import org.example.productterminals.models.RootBridgeDto
import org.example.productterminals.models.RowAliasDto
import org.example.productterminals.models.RowsDtoInner
import org.example.productterminals.models.UnknownProductDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ProductTerminalStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun productTerminalBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> productTerminalBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic product body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun composedUnionTerminalsContracts() = withTimeout(20000) {
    check(serializer<ProductMaybeDto>().descriptor.isNullable)
    check(serializer<NestedMaybeDto>().descriptor.isNullable)
    check(!serializer<ProductFalseDto>().descriptor.isNullable)
    check(!serializer<NestedFalseDto>().descriptor.isNullable)
    check(!serializer<NarrowBridgeDto>().descriptor.isNullable)
    check(!ProductMaybeDto.serializer().descriptor.isNullable)
    check(!NestedMaybeDto.serializer().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val values = listOf<ProductDto>(ProductDto.Note("note", true, "1"), ProductDto.Note2("note", false, "2"), ProductDto.Count(3, true, "3"), ProductDto.Count4(4, false, "4"))
        for (value in values) {
            val root: RootBridgeDto = value
            val maybe: ProductMaybeDto = value
            val narrow: ProductFalseDto = value
            check(codec.decodeFromString<RootBridgeDto>(codec.encodeToString<RootBridgeDto>(root)) == value)
            check(codec.decodeFromString<ProductMaybeDto>(codec.encodeToString<ProductMaybeDto>(maybe)) == value)
            check(codec.decodeFromString<ProductFalseDto>(codec.encodeToString<ProductFalseDto>(narrow)) == value)
            val wire = codec.encodeToJsonElement<ProductDto>(value).jsonObject
            check("kind" in wire && "mode" in wire)
        }
        check(codec.decodeFromString<ProductMaybeDto>("null") == null)
        check(codec.decodeFromString<NestedMaybeDto>("null") == null)
        check(runCatching { codec.decodeFromString<ProductFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<NestedFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<RootBridgeDto>("""{"kind":"note","mode":"fast","fast":true,"id":"missing-body"}""") }.isFailure)
        val raw = codec.parseToJsonElement("""{"kind":"future","mode":"future","extra":[true,null]}""")
        val unknown = codec.decodeFromJsonElement<ProductMaybeDto>(raw) as UnknownProductDto
        check(codec.encodeToJsonElement<ProductMaybeDto>(unknown) == raw)
        check(codec.encodeToJsonElement<ProductMaybeDto>(null) == JsonNull)
        val nested: NestedFalseDto = BoxDto.Choice.Note("nested", true, "n")
        val row: RowAliasDto = RowsDtoInner.Count(5, false, "r")
        val map: MapAliasDto = MapDtoValue.Note2("map", true, "m")
        val first: AConsumerDto = AConsumerDto.Left(values[0])
        check(codec.decodeFromString<AConsumerDto>(codec.encodeToString<AConsumerDto>(first)) == first)
        val holder = HolderDto(values[0], null, values[1], listOf(row), mapOf("value" to map))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<BoxDto>(codec.encodeToString(BoxDto(nested))) == BoxDto(nested))
        val patches = listOf(PatchProductInputDto(), PatchProductInputDto(value = Patch.Null), PatchProductInputDto(value = Patch.Value(nested), falseView = nested))
        check("value" !in codec.encodeToJsonElement(patches[0]).jsonObject)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["value"] == JsonNull)
        for (patch in patches) check(codec.decodeFromString<PatchProductInputDto>(codec.encodeToString(patch)) == patch)
        var expected: ProductDto = values[0]
        var expectedPatch = codec.encodeToJsonElement(patches[0])
        var bad = false
        var streamNull = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = ProductTerminalStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/product" -> {
                        check(codec.decodeFromString<ProductDto>(productTerminalBody(request.body)) == expected)
                        respond(if (bad) "null" else codec.encodeToString<ProductDto>(expected), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(request.method.value == "PATCH")
                        check(codec.parseToJsonElement(productTerminalBody(request.body)) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val payload = if (bad) "[" else if (streamNull) "null" else codec.encodeToString<ProductDto>(expected)
                        writers.add(scope.launch {
                            try {
                                for (byte in "event: product\r\ndata: $payload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                awaitCancellation()
                            } catch (closed: ClosedByteChannelException) {
                                if (!channel.isClosedForWrite) throw closed
                            }
                        })
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic product path")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = ComposedUnionTerminalsApiClient(http, "https://example.test/", codec).default
            for (value in values + unknown) { expected = value; check(api.echoProduct(value) == value) }
            for (patch in patches) { expectedPatch = codec.encodeToJsonElement(patch); api.patchProduct(patch) }
            val cold = api.watchProduct(); check(channels.isEmpty())
            for (value in values + unknown) {
                expected = value
                check((cold.take(1).toList().single() as WatchProductEventDto.Product).data == value)
                check(channels.last().isClosedForWrite)
            }
            streamNull = true
            check((cold.take(1).toList().single() as WatchProductEventDto.Product).data == null)
            check(channels.last().isClosedForWrite)
            streamNull = false; bad = true
            check(runCatching { cold.take(1).toList() }.isFailure)
            check(channels.last().isClosedForWrite)
            check(runCatching { api.echoProduct(expected) }.isFailure)
            bad = false
            check((cold.take(1).toList().single() as WatchProductEventDto.Product).data == expected)
            check(channels.last().isClosedForWrite && api.echoProduct(expected) == expected)
        } finally { http.close(); scope.cancel(); writers.forEach { it.join() } }
        println("ComposedUnionTerminals Json defaults=$defaults nulls=$nulls actual-product-known-Unknown-null-false-Patch-HTTP-liveSSEclose-reuse-cold passed")
    }
}
