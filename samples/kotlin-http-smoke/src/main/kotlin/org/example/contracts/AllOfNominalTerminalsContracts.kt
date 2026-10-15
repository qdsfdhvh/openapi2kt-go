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
import org.example.allofterminals.AllOfNominalTerminalsApiClient
import org.example.allofterminals.api.models.PatchAllOfInputDto
import org.example.allofterminals.api.models.WatchAllOfEventDto
import org.example.allofterminals.models.AllChoiceDto
import org.example.allofterminals.models.AllContainerDto
import org.example.allofterminals.models.AllFalseDto
import org.example.allofterminals.models.AllHolderDto
import org.example.allofterminals.models.AllMaybeDto
import org.example.allofterminals.models.AllModelDto
import org.example.allofterminals.models.AllNestedDto
import org.example.allofterminals.models.AllNestedBridgeDto
import org.example.allofterminals.models.AllRootBridgeDto
import org.example.allofterminals.models.AllVariantDto
import org.example.allofterminals.models.IntrinsicAllContainerDto
import org.example.allofterminals.models.IntrinsicAllNodeDto
import org.example.allofterminals.models.IntrinsicAllFalseDto
import org.example.allofterminals.models.InheritedAllContainerDto
import org.example.allofterminals.models.InheritedAllNodeDto
import org.example.allofterminals.models.InheritedAllFalseDto
import org.example.allofterminals.models.Patch
import org.example.allofterminals.models.UnknownAllChoiceDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class AllOfTerminalStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun allOfTerminalBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> allOfTerminalBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic allOf terminal body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun allOfNominalTerminalsContracts(checked: Boolean = false) = withTimeout(20000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val terminal = AllRootBridgeDto("named")
        val node = AllNestedDto("nested", peer = terminal)
        val recursive = AllRootBridgeDto("recursive", nested = node, peer = node)
        val recursiveNode = AllNestedDto("recursive-node", recursive, node)
        check(node == AllContainerDto.Node("nested", AllModelDto("named")))
        check(recursive == AllModelDto("recursive", node, node))
        check(codec.decodeFromString<AllRootBridgeDto>(codec.encodeToString(recursive)) == recursive)
        check(codec.decodeFromString<AllNestedDto>(codec.encodeToString(recursiveNode)) == recursiveNode)
        check(serializer<AllMaybeDto>().descriptor.isNullable)
        check(serializer<AllNestedBridgeDto>().descriptor.isNullable)
        check(!serializer<AllFalseDto>().descriptor.isNullable)
        check(!AllNestedDto.serializer().descriptor.isNullable)
        check(codec.decodeFromString<AllMaybeDto>("null") == null)
        check(runCatching { codec.decodeFromString<AllFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<AllRootBridgeDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<AllNestedDto>("{}") }.isFailure)
        check(serializer<IntrinsicAllNodeDto>().descriptor.isNullable)
        check(!serializer<IntrinsicAllFalseDto>().descriptor.isNullable)
        check(!IntrinsicAllContainerDto.Node.serializer().descriptor.isNullable)
        check(codec.decodeFromString<IntrinsicAllNodeDto>("null") == null)
        check(runCatching { codec.decodeFromString<IntrinsicAllFalseDto>("null") }.isFailure)
        val intrinsic = IntrinsicAllNodeDto("intrinsic", IntrinsicAllNodeDto("next"))
        check(codec.decodeFromString<IntrinsicAllNodeDto>(codec.encodeToString<IntrinsicAllNodeDto>(intrinsic)) == intrinsic)
        check(serializer<InheritedAllNodeDto>().descriptor.isNullable)
        check(!serializer<InheritedAllFalseDto>().descriptor.isNullable)
        check(!InheritedAllContainerDto.Node.serializer().descriptor.isNullable)
        check(codec.decodeFromString<InheritedAllNodeDto>("null") == null)
        check(runCatching { codec.decodeFromString<InheritedAllFalseDto>("null") }.isFailure)
        val inherited = InheritedAllNodeDto("inherited")
        check(codec.decodeFromString<InheritedAllNodeDto>(codec.encodeToString<InheritedAllNodeDto>(inherited)) == inherited)
        val holder = AllHolderDto(recursive, recursiveNode, null)
        check(codec.decodeFromString<AllHolderDto>(codec.encodeToString(holder)) == holder)
        val variant = AllVariantDto(node, "variant")
        check(variant == AllChoiceDto.All(node, "variant"))
        check(codec.decodeFromString<AllVariantDto>(codec.encodeToString(variant)) == variant)
        check(codec.decodeFromString<AllChoiceDto>(codec.encodeToString<AllChoiceDto>(variant)) == variant)
        check(runCatching { codec.decodeFromString<AllChoiceDto>("""{"kind":"all","node":{}}""") }.isFailure)
        val raw = codec.parseToJsonElement("""{"kind":"future","extra":null}""")
        check(codec.encodeToJsonElement<AllChoiceDto>(codec.decodeFromJsonElement<AllChoiceDto>(raw) as UnknownAllChoiceDto) == raw)
        if (checked) {
            check(runCatching { AllNestedDto("") }.isFailure)
            check(runCatching { AllRootBridgeDto("") }.isFailure)
            check(runCatching { codec.decodeFromString<AllNestedDto>("""{"id":""}""") }.isFailure)
        }
        val absent = PatchAllOfInputDto(required = null, narrow = node)
        val clear = PatchAllOfInputDto(required = null, narrow = node, value = Patch.Null)
        val assigned = PatchAllOfInputDto(required = node, narrow = node, value = Patch.Value(node))
        check("value" !in codec.encodeToJsonElement(absent).jsonObject)
        check(codec.encodeToJsonElement(clear).jsonObject["value"] == JsonNull)
        check(codec.decodeFromString<PatchAllOfInputDto>(codec.encodeToString(assigned)) == assigned)
        var expectedPatch = codec.encodeToJsonElement(absent)
        var failure = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = AllOfTerminalStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/allof" -> {
                        check(codec.decodeFromString<AllHolderDto>(allOfTerminalBody(request.body)) == holder)
                        respond(if (failure == "holder") "{}" else codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/allof-patch" -> {
                        check(request.method.value == "PATCH")
                        check(codec.parseToJsonElement(allOfTerminalBody(request.body)) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/allof-watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frames = if (failure == "node") "event: nested\r\ndata: {}\r\n\r\n" else "event: nullable\r\ndata: null\r\n\r\nevent: nested\r\ndata: ${codec.encodeToString(node)}\r\n\r\nevent: named\r\ndata: ${codec.encodeToString(recursive)}\r\n\r\n"
                        writers.add(scope.launch {
                            try {
                                for (byte in frames.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                awaitCancellation()
                            } catch (closed: ClosedByteChannelException) {
                                if (!channel.isClosedForWrite) throw closed
                            }
                        })
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic allOf terminal path")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = AllOfNominalTerminalsApiClient(http, "https://example.test/", codec).default
            check(api.echoAllOf(holder) == holder)
            failure = "holder"; check(runCatching { api.echoAllOf(holder) }.isFailure)
            failure = ""; api.patchAllOf(absent)
            expectedPatch = codec.encodeToJsonElement(clear); api.patchAllOf(clear)
            expectedPatch = codec.encodeToJsonElement(assigned); api.patchAllOf(assigned)
            val cold = api.watchAllOf(); check(channels.isEmpty())
            repeat(2) {
                val events = cold.take(3).toList()
                check((events[0] as WatchAllOfEventDto.Nullable).data == null)
                check((events[1] as WatchAllOfEventDto.Nested).data == node)
                check((events[2] as WatchAllOfEventDto.Named).data == recursive)
                check(channels.last().isClosedForWrite)
            }
            failure = "node"; check(runCatching { cold.take(1).toList() }.isFailure)
            check(channels.last().isClosedForWrite)
            failure = ""; check((cold.take(1).toList().single() as WatchAllOfEventDto.Nullable).data == null)
            check(channels.last().isClosedForWrite)
        } finally { http.close(); scope.cancel(); writers.forEach { it.join() } }
        println("AllOfNominalTerminals Json defaults=$defaults nulls=$nulls actual-allOf-node-named-original-owner-recursion-null-false-Patch-HTTP-SSEclose-reuse-cold passed")
    }
}
