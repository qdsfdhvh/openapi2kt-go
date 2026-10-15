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
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.hierarchychains.HierarchyReferenceChainsApiClient
import org.example.hierarchychains.api.models.AliasPatchInputDto
import org.example.hierarchychains.api.models.AliasWatchEventDto
import org.example.hierarchychains.api.models.IntrinsicWatchEventDto
import org.example.hierarchychains.models.AFirstDto
import org.example.hierarchychains.models.AGroupDto
import org.example.hierarchychains.models.ARootDto
import org.example.hierarchychains.models.AReusedDto
import org.example.hierarchychains.models.AReusedLeafDto
import org.example.hierarchychains.models.AliasHolderDto
import org.example.hierarchychains.models.BRootDto
import org.example.hierarchychains.models.CRootDto
import org.example.hierarchychains.models.DRootDto
import org.example.hierarchychains.models.FirstDto
import org.example.hierarchychains.models.ERootDto
import org.example.hierarchychains.models.FRootDto
import org.example.hierarchychains.models.GRootDto
import org.example.hierarchychains.models.EFirstDto
import org.example.hierarchychains.models.FFirstDto
import org.example.hierarchychains.models.EGroupDto
import org.example.hierarchychains.models.FGroupDto
import org.example.hierarchychains.models.GroupDto
import org.example.hierarchychains.models.IntrinsicHolderDto
import org.example.hierarchychains.models.GroupDtoMode
import org.example.hierarchychains.models.GroupDtoTag
import org.example.hierarchychains.models.Patch
import org.example.hierarchychains.models.RangeDto
import org.example.hierarchychains.models.ReusedFirstDto
import org.example.hierarchychains.models.RootDto
import org.example.hierarchychains.models.UnknownGroupDto
import org.example.hierarchychains.models.UnknownRootDto
import org.example.hierarchychains.models.ParentBridgeDto
import org.example.hierarchychains.models.ParentBridgeChainDto
import org.example.hierarchychains.models.ParentFalseDto
import org.example.hierarchychains.models.ParentReNullDto
import org.example.hierarchychains.models.LeafBridgeDto
import org.example.hierarchychains.models.LeafFalseDto
import org.example.hierarchychains.models.GroupBridgeDto
import org.example.hierarchychains.models.ChainHolderDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class HierarchyChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun hierarchyChainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> hierarchyChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    is OutgoingContent.NoContent -> "null"
    else -> error("Unexpected synthetic nullable hierarchy body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun hierarchyReferenceChainsContracts() = withTimeout(20000) {
    val first = AFirstDto("canonical-first")
    check(first == FirstDto("canonical-first"))
    check(!ARootDto.serializer().descriptor.isNullable)
    check(serializer<ARootDto>().descriptor.isNullable && serializer<BRootDto>().descriptor.isNullable)
    check(!serializer<CRootDto>().descriptor.isNullable && serializer<DRootDto>().descriptor.isNullable)
    check(serializer<AFirstDto>().descriptor.isNullable && serializer<AGroupDto>().descriptor.isNullable)
    check(serializer<AReusedDto>().descriptor.isNullable && serializer<AReusedLeafDto>().descriptor.isNullable)
    check(serializer<ERootDto>().descriptor.isNullable && serializer<GRootDto>().descriptor.isNullable)
    check(!serializer<FRootDto>().descriptor.isNullable)
    check(serializer<EFirstDto>().descriptor.isNullable && !serializer<FFirstDto>().descriptor.isNullable)
    check(serializer<EGroupDto>().descriptor.isNullable && !serializer<FGroupDto>().descriptor.isNullable)
    check(!RootDto.serializer().descriptor.isNullable && !FirstDto.serializer().descriptor.isNullable)
    check(serializer<ParentBridgeDto>().descriptor.isNullable && serializer<ParentBridgeChainDto>().descriptor.isNullable)
    check(!serializer<ParentFalseDto>().descriptor.isNullable && serializer<ParentReNullDto>().descriptor.isNullable)
    check(serializer<LeafBridgeDto>().descriptor.isNullable && !serializer<LeafFalseDto>().descriptor.isNullable)
    check(serializer<GroupBridgeDto>().descriptor.isNullable && !ParentBridgeDto.serializer().descriptor.isNullable)
    check(LeafBridgeDto("chain-constructor") == FirstDto("chain-constructor"))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<ARootDto>("null") == null)
        check(codec.encodeToString<ARootDto>(null) == "null")
        check(codec.decodeFromString<DRootDto>("null") == null)
        check(codec.decodeFromString(RootDto.serializer().nullable, "null") == null)
        check(runCatching { codec.decodeFromString<CRootDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString(ARootDto.serializer(), "null") }.isFailure)
        val range = RangeDto(GroupDtoTag.THREE, GroupDtoMode.SLOW, "nullable-range")
        check(codec.decodeFromString<ARootDto>(codec.encodeToString<ARootDto>(range)) == range)
        check(codec.decodeFromString<AFirstDto>(codec.encodeToString<AFirstDto>(first)) == first)
        check(codec.decodeFromString<ERootDto>("null") == null)
        check(codec.decodeFromString<EFirstDto>("null") == null)
        check(codec.decodeFromString<EGroupDto>("null") == null)
        check(runCatching { codec.decodeFromString<FRootDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<FFirstDto>("null") }.isFailure)
        check(codec.decodeFromString<ParentBridgeChainDto>("null") == null)
        check(runCatching { codec.decodeFromString<ParentFalseDto>("null") }.isFailure)
        val recursive = FirstDto("recursive", ancestor = range, peer = first)
        check(codec.decodeFromString<LeafBridgeDto>(codec.encodeToString<LeafBridgeDto>(recursive)) == recursive)
        val chainHolder = ChainHolderDto(null, range, null, first, null, listOf(null, recursive, range), mapOf("missing" to null, "known" to recursive))
        check(codec.decodeFromString<ChainHolderDto>(codec.encodeToString(chainHolder)) == chainHolder)
        val intrinsicHolder = IntrinsicHolderDto(null, range, null, range, null, first, listOf(null, first, range))
        check(codec.decodeFromString<IntrinsicHolderDto>(codec.encodeToString(intrinsicHolder)) == intrinsicHolder)
        val reused = AReusedLeafDto("canonical-reused")
        check(reused == ReusedFirstDto("canonical-reused"))
        val unknownRaw = codec.parseToJsonElement("""{"family":"group","tag":"future","mode":"future","extra":[null,true]}""")
        val unknown = codec.decodeFromJsonElement<ARootDto>(unknownRaw) as UnknownGroupDto
        check(codec.encodeToJsonElement<ARootDto>(unknown) == unknownRaw)
        val rootUnknown = UnknownRootDto(codec.parseToJsonElement("""{"family":"future"}"""))
        val holder = AliasHolderDto(null, range, null, null, null, null, listOf(null, range, unknown, rootUnknown))
        check(codec.decodeFromString<AliasHolderDto>(codec.encodeToString(holder)) == holder)
        check(runCatching { codec.decodeFromString<AliasHolderDto>("""{"narrowed":null,"rows":[]}""") }.isFailure)
        val patch = AliasPatchInputDto(requiredNull = null, requiredFalse = range)
        val absent = codec.encodeToJsonElement(patch).jsonObject
        check("value" !in absent && "leaf" !in absent)
        check(absent["requiredNull"] == if (nulls) JsonNull else null)
        val clear = AliasPatchInputDto(requiredNull = null, requiredFalse = range, value = Patch.Null, leaf = Patch.Null)
        check(codec.encodeToJsonElement(clear).jsonObject["value"] == JsonNull)
        check(codec.decodeFromString<AliasPatchInputDto>(codec.encodeToString(clear)) == clear)
        val value = AliasPatchInputDto(requiredNull = range, requiredFalse = range, value = Patch.Value(unknown), leaf = Patch.Value(first))
        check(codec.decodeFromString<AliasPatchInputDto>(codec.encodeToString(value)) == value)
        check(runCatching { codec.decodeFromString<ARootDto>("""{"family":"group","tag":"one","mode":"fast"}""") }.isFailure)

        var expected: JsonElement = JsonNull
        var patchExpected: JsonElement = absent
        var failure = ""
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val engine = HierarchyChainStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/nullable", "/narrow", "/optional", "/intrinsic", "/intrinsic-false", "/intrinsic-leaf" -> {
                        check(codec.parseToJsonElement(hierarchyChainBody(request.body)) == expected)
                        val body = when (failure) {
                            "null" -> "null"
                            "known" -> """{"family":"group","tag":"one","mode":"fast"}"""
                            else -> expected.toString()
                        }
                        respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/aliases" -> {
                        check(codec.decodeFromString<AliasHolderDto>(hierarchyChainBody(request.body)) == holder)
                        respond(codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/chain-holder" -> {
                        check(codec.decodeFromString<ChainHolderDto>(hierarchyChainBody(request.body)) == chainHolder)
                        respond(codec.encodeToString(chainHolder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/intrinsic-holder" -> {
                        check(codec.decodeFromString<IntrinsicHolderDto>(hierarchyChainBody(request.body)) == intrinsicHolder)
                        respond(codec.encodeToString(intrinsicHolder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/intrinsic-watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frames = when(failure) {
                            "null" -> "event: narrow\r\ndata: null\r\n\r\n"
                            "known" -> "event: root\r\ndata: {\"family\":\"group\",\"tag\":\"one\",\"mode\":\"fast\"}\r\n\r\n"
                            else -> "event: root\r\ndata: null\r\n\r\nevent: leaf\r\ndata: null\r\n\r\nevent: holder\r\ndata: ${codec.encodeToString(intrinsicHolder)}\r\n\r\n"
                        }
                        scope.launch {
                            for(byte in frames.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    "/alias-patch" -> {
                        check(codec.parseToJsonElement(hierarchyChainBody(request.body)) == patchExpected)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/alias-watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frames = when (failure) {
                            "null" -> "event: narrow\r\ndata: null\r\n\r\n"
                            "known" -> "event: nullable\r\ndata: {\"family\":\"group\",\"tag\":\"one\",\"mode\":\"fast\"}\r\n\r\n"
                            else -> "event: nullable\r\ndata: null\r\n\r\nevent: nullable\r\ndata: ${codec.encodeToString<ARootDto>(range)}\r\n\r\nevent: aliases\r\ndata: ${codec.encodeToString(holder)}\r\n\r\n"
                        }
                        scope.launch {
                            for (byte in frames.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic nullable alias route")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = HierarchyReferenceChainsApiClient(http, "http://example.test/", codec).default
            check(api.nullable(null) == null && api.optional() == null && api.intrinsic(null) == null && api.intrinsicLeaf(null) == null)
            expected = codec.encodeToJsonElement<ARootDto>(range)
            check(api.nullable(range) == range && api.narrow(range) == range && api.optional(range) == range)
            check(api.aliases(holder) == holder && api.intrinsic(range) == range && api.intrinsicFalse(range) == range)
            check(api.intrinsicHolder(intrinsicHolder) == intrinsicHolder && api.chainHolder(chainHolder) == chainHolder)
            for (input in listOf(patch, clear, value)) { patchExpected = codec.encodeToJsonElement(input); api.aliasPatch(input) }
            failure = "null"; check(runCatching { api.narrow(range) }.isFailure)
            failure = "known"; check(runCatching { api.nullable(range) }.isFailure)
            failure = ""
            val flow = api.aliasWatch(); check(channels.isEmpty())
            val events = flow.take(3).toList()
            check((events[0] as AliasWatchEventDto.Nullable).data == null)
            check((events[1] as AliasWatchEventDto.Nullable).data == range)
            check((events[2] as AliasWatchEventDto.Aliases).data == holder)
            check(channels.last().isClosedForRead)
            for (bad in listOf("null", "known")) {
                failure = bad; check(runCatching { flow.take(1).toList() }.isFailure)
                check(channels.last().isClosedForRead)
            }
            failure = ""; check(flow.take(3).toList().size == 3 && api.narrow(range) == range)
            val intrinsicFlow = api.intrinsicWatch()
            val intrinsicEvents = intrinsicFlow.take(3).toList()
            check((intrinsicEvents[0] as IntrinsicWatchEventDto.Root).data == null)
            check((intrinsicEvents[1] as IntrinsicWatchEventDto.Leaf).data == null)
            check((intrinsicEvents[2] as IntrinsicWatchEventDto.Holder).data == intrinsicHolder)
            check(channels.last().isClosedForRead)
            for(bad in listOf("null","known")) {
                failure = bad; check(runCatching { intrinsicFlow.take(1).toList() }.isFailure)
                check(channels.last().isClosedForRead)
            }
            failure = ""; check(intrinsicFlow.take(3).toList().size == 3 && api.intrinsicFalse(range) == range)
            println("HierarchyReferenceChains Json defaults=$defaults nulls=$nulls named-parent-leaf-chain-canonical-recursive-false-Unknown-Patch-HTTP-null-SSEclose-reuse-cold passed")
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
    }
}
