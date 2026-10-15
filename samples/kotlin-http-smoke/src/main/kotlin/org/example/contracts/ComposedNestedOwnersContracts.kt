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
import org.example.composedowners.ComposedNestedOwnersApiClient
import org.example.composedowners.api.models.PatchComposedInputDto
import org.example.composedowners.api.models.WatchComposedEventDto
import org.example.composedowners.models.AnyChoiceDto
import org.example.composedowners.models.AnyNodeDto
import org.example.composedowners.models.BorrowComposedDto
import org.example.composedowners.models.ChoiceBranchDto
import org.example.composedowners.models.ChoiceDto
import org.example.composedowners.models.ChoiceNodeDto
import org.example.composedowners.models.ComposeHolderDto
import org.example.composedowners.models.ComposedDto
import org.example.composedowners.models.ComposedNodeDto
import org.example.composedowners.models.ComposedRowDto
import org.example.composedowners.models.ComposedValueDto
import org.example.composedowners.models.MaybeComposedNodeDto
import org.example.composedowners.models.MixedChoiceDto
import org.example.composedowners.models.MixedNodeDto
import org.example.composedowners.models.NarrowChoiceNodeDto
import org.example.composedowners.models.ObjectMixedChoiceDto
import org.example.composedowners.models.Patch
import org.example.composedowners.models.RootRowNodeDto
import org.example.composedowners.models.RootMapNodeDto
import org.example.composedowners.models.RootRowsDto
import org.example.composedowners.models.RootRowsDtoInner
import org.example.composedowners.models.RootMapDto
import org.example.composedowners.models.RootMapDtoValue
import org.example.composedowners.models.UnknownChoiceDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ComposedOwnerStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun composedOwnerBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> composedOwnerBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic composed body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun composedNestedOwnersContracts(checked: Boolean = false) = withTimeout(20000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val leaf = ChoiceNodeDto("choice")
        val node = ComposedNodeDto("composed", leaf)
        val back = ChoiceNodeDto("back", node)
        check(node == ComposedDto.Node("composed", ChoiceDto.First.Node("choice")))
        check(ChoiceBranchDto(back) == ChoiceDto.First(back))
        val row = ComposedRowDto("row", node)
        val entry = ComposedValueDto("entry", leaf)
        val mixed = MixedNodeDto("mixed", back)
        val any = AnyNodeDto("any")
        check(mixed == ObjectMixedChoiceDto.Node("mixed", back))
        check(any == AnyChoiceDto.A.Node("any"))
        val rootRow = RootRowNodeDto("root-row")
        val rootMap = RootMapNodeDto("root-map")
        check(rootRow == RootRowsDtoInner.R.Node("root-row"))
        check(rootMap == RootMapDtoValue.M.Node("root-map"))
        val rootRows: RootRowsDto = listOf(RootRowsDtoInner.R(rootRow))
        val rootValues: RootMapDto = mapOf("value" to RootMapDtoValue.M(rootMap))
        check(codec.decodeFromString<RootRowsDto>(codec.encodeToString(rootRows)) == rootRows)
        check(codec.decodeFromString<RootMapDto>(codec.encodeToString(rootValues)) == rootValues)
        check(serializer<MaybeComposedNodeDto>().descriptor.isNullable)
        check(!serializer<NarrowChoiceNodeDto>().descriptor.isNullable)
        check(!ComposedNodeDto.serializer().descriptor.isNullable)
        check(codec.decodeFromString<MaybeComposedNodeDto>("null") == null)
        check(runCatching { codec.decodeFromString<ComposedNodeDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<NarrowChoiceNodeDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<ComposedNodeDto>("{}") }.isFailure)
        check(codec.decodeFromString<ChoiceNodeDto>(codec.encodeToString(back)) == back)
        val holder = ComposeHolderDto(node, back, null, back, listOf(null, node), row, entry, mixed, any)
        check(codec.decodeFromString<ComposeHolderDto>(codec.encodeToString(holder)) == holder)
        val composed = ComposedDto(node, listOf(row), mapOf("entry" to entry), "label")
        check(codec.decodeFromString<ComposedDto>(codec.encodeToString(composed)) == composed)
        val borrowed = BorrowComposedDto(BorrowComposedDto.Node("borrowed", back), listOf(BorrowComposedDto.RowsInner("row", node)), mapOf("entry" to BorrowComposedDto.EntriesValue("entry", leaf)))
        check(codec.decodeFromString<BorrowComposedDto>(codec.encodeToString(borrowed)) == borrowed)
        val choice: ChoiceDto = ChoiceBranchDto(back)
        check(codec.decodeFromString<ChoiceDto>(codec.encodeToString(choice)) == choice)
        check(runCatching { codec.decodeFromString<ChoiceDto>("""{"kind":"first","node":{}}""") }.isFailure)
        val raw = codec.parseToJsonElement("""{"kind":"future","extra":[null,true]}""")
        check(codec.encodeToJsonElement<ChoiceDto>(codec.decodeFromJsonElement<ChoiceDto>(raw) as UnknownChoiceDto) == raw)
        val mixedValue: MixedChoiceDto = ObjectMixedChoiceDto("object", mixed)
        check(codec.decodeFromString<MixedChoiceDto>(codec.encodeToString(mixedValue)) == mixedValue)
        val anyValue: AnyChoiceDto = AnyChoiceDto.A(any)
        check(codec.decodeFromString<AnyChoiceDto>(codec.encodeToString(anyValue)) == anyValue)
        if (checked) {
            check(runCatching { ComposedNodeDto("") }.isFailure)
            check(runCatching { ChoiceNodeDto("") }.isFailure)
            check(runCatching { codec.decodeFromString<ComposedNodeDto>("""{"id":""}""") }.isFailure)
        }
        val absent = PatchComposedInputDto(narrow = back, required = null)
        val clear = PatchComposedInputDto(narrow = back, required = null, value = Patch.Null)
        val assigned = PatchComposedInputDto(narrow = back, required = node, value = Patch.Value(node))
        check("value" !in codec.encodeToJsonElement(absent).jsonObject)
        check(codec.encodeToJsonElement(clear).jsonObject["value"] == JsonNull)
        check(codec.decodeFromString<PatchComposedInputDto>(codec.encodeToString(assigned)) == assigned)
        var failure = ""
        var expectedPatch = codec.encodeToJsonElement(absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = ComposedOwnerStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/composed" -> {
                        check(codec.decodeFromString<ComposeHolderDto>(composedOwnerBody(request.body)) == holder)
                        respond(if (failure == "holder") "{}" else codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/compose-patch" -> {
                        check(request.method.value == "PATCH")
                        check(codec.parseToJsonElement(composedOwnerBody(request.body)) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/compose-watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frames = if (failure == "node") "event: composed\r\ndata: {}\r\n\r\n" else "event: nullable\r\ndata: null\r\n\r\nevent: composed\r\ndata: ${codec.encodeToString(node)}\r\n\r\nevent: holder\r\ndata: ${codec.encodeToString(holder)}\r\n\r\n"
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
                    else -> error("Unexpected synthetic composed path")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = ComposedNestedOwnersApiClient(http, "https://example.test/", codec).default
            check(api.echoComposed(holder) == holder)
            failure = "holder"
            check(runCatching { api.echoComposed(holder) }.isFailure)
            failure = ""
            api.patchComposed(absent)
            expectedPatch = codec.encodeToJsonElement(clear); api.patchComposed(clear)
            expectedPatch = codec.encodeToJsonElement(assigned); api.patchComposed(assigned)
            val cold = api.watchComposed(); check(channels.isEmpty())
            repeat(2) {
                val events = cold.take(3).toList()
                check((events[0] as WatchComposedEventDto.Nullable).data == null)
                check((events[1] as WatchComposedEventDto.Composed).data == node)
                check((events[2] as WatchComposedEventDto.Holder).data == holder)
                check(channels.last().isClosedForWrite)
            }
            failure = "node"
            check(runCatching { cold.take(1).toList() }.isFailure)
            check(channels.last().isClosedForWrite)
            failure = ""
            check((cold.take(1).toList().single() as WatchComposedEventDto.Nullable).data == null)
            check(channels.last().isClosedForWrite)
        } finally { http.close(); scope.cancel(); writers.forEach { it.join() } }
        println("ComposedNestedOwners Json defaults=$defaults nulls=$nulls actual-allOf-union-lexical-owner-recursion-peer-Patch-HTTP-SSEclose-reuse-cold passed")
    }
}
