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
import kotlinx.serialization.json.jsonObject
import org.example.unionallof.UnionAllOfApiClient
import org.example.unionallof.api.models.PatchInputDto
import org.example.unionallof.api.models.WatchEventDto
import org.example.unionallof.models.ComposedDto
import org.example.unionallof.models.HolderDto
import org.example.unionallof.models.Patch
import org.example.unionallof.models.ProductDto
import org.example.unionallof.models.UnknownComposedDto
import org.example.unionallof.models.UnknownProductDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class UnionAllOfStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun unionAllOfBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> unionAllOfBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic union/allOf body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun unionAllOfContracts() = withTimeout(20000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val note = ComposedDto.Note(id = "n", revision = 1, body = "hello")
        val count = ComposedDto.Count(id = "c", revision = 2, count = 7)
        for (value in listOf(note, count)) {
            check(codec.decodeFromString<ComposedDto>(codec.encodeToString<ComposedDto>(value)) == value)
            check(codec.parseToJsonElement(codec.encodeToString<ComposedDto>(value)).jsonObject["kind"] != null)
        }
        val products = listOf<ProductDto>(ProductDto.Note("fast", true), ProductDto.Note2("slow", false), ProductDto.Count(3, true), ProductDto.Count4(4, false))
        for (value in products) check(codec.decodeFromString<ProductDto>(codec.encodeToString<ProductDto>(value)) == value)
        val unknownWire = """{"kind":"future","mode":"future","extra":[true,null]}"""
        val unknown = codec.decodeFromString<ComposedDto>(unknownWire) as UnknownComposedDto
        check(codec.parseToJsonElement(codec.encodeToString<ComposedDto>(unknown)) == codec.parseToJsonElement(unknownWire))
        val unknownProduct = codec.decodeFromString<ProductDto>(unknownWire) as UnknownProductDto
        check(codec.parseToJsonElement(codec.encodeToString<ProductDto>(unknownProduct)) == codec.parseToJsonElement(unknownWire))
        check(runCatching { codec.decodeFromString<ComposedDto>("""{"kind":"note","revision":1,"body":"missing"}""") }.isFailure)
        val values = products.map { HolderDto(value = note, inline = HolderDto.Inline.Count("i", 3, 9), rows = listOf(note, count, unknown), product = it) }
        for (value in values) check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var expected = values.first()
        var patchMode = 0
        var bad = false
        val engine = UnionAllOfStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(unionAllOfBody(request.body)) == expected)
                        respond(codec.encodeToString(expected), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        val wire = codec.parseToJsonElement(unionAllOfBody(request.body)).jsonObject
                        check(wire["requiredNullable"] == if (nulls) JsonNull else null)
                        when (patchMode) {
                            0 -> check("value" !in wire)
                            1 -> check(wire["value"] == JsonNull)
                            2 -> check(codec.decodeFromJsonElement<ComposedDto>(wire.getValue("value")) == note)
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val data = if (bad) "{}" else codec.encodeToString(expected)
                        scope.launch {
                            for (byte in "event: sample\r\ndata: $data\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic union/allOf URL")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = UnionAllOfApiClient(http, "http://example.test/", codec).default
            for (value in values) { expected = value; check(api.echo(value) == value) }
            api.patchPatch(PatchInputDto(requiredNullable = null))
            patchMode = 1; api.patchPatch(PatchInputDto(requiredNullable = null, value = Patch.Null))
            patchMode = 2; api.patchPatch(PatchInputDto(requiredNullable = null, value = Patch.Value(note)))
            val flow = api.watch(); check(channels.isEmpty())
            for (value in values) {
                expected = value; check((flow.take(1).toList().single() as WatchEventDto.Sample).data == value)
                check(channels.last().isClosedForRead)
            }
            bad = true; check(runCatching { flow.take(1).toList() }.isFailure); check(channels.last().isClosedForRead)
            bad = false; check(flow.take(1).toList().size == 1 && api.echo(expected) == expected)
            println("UnionAllOf Json defaults=$defaults nulls=$nulls borrowed-shapes-four-products-known-Unknown-required-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
    }
}
