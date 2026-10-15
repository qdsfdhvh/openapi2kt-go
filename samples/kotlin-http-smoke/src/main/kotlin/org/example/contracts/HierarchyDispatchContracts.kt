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
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import org.example.hierarchydispatch.HierarchyDispatchApiClient
import org.example.hierarchydispatch.api.models.PatchInputDto
import org.example.hierarchydispatch.api.models.WatchEventDto
import org.example.hierarchydispatch.models.FirstDto
import org.example.hierarchydispatch.models.GroupDto
import org.example.hierarchydispatch.models.GroupDtoMode
import org.example.hierarchydispatch.models.GroupDtoTag
import org.example.hierarchydispatch.models.HolderDto
import org.example.hierarchydispatch.models.Patch
import org.example.hierarchydispatch.models.RangeDto
import org.example.hierarchydispatch.models.ReusedFirstDto
import org.example.hierarchydispatch.models.ReusedGroupDto
import org.example.hierarchydispatch.models.ReusedMissingFirstDto
import org.example.hierarchydispatch.models.ReusedMissingGroupDto
import org.example.hierarchydispatch.models.ReusedMissingRootDto
import org.example.hierarchydispatch.models.ReusedMissingRootDtoTag
import org.example.hierarchydispatch.models.ReusedMissingSecondDto
import org.example.hierarchydispatch.models.ReusedRootDto
import org.example.hierarchydispatch.models.ReusedRootDtoTag
import org.example.hierarchydispatch.models.ReusedSecondDto
import org.example.hierarchydispatch.models.RootDto
import org.example.hierarchydispatch.models.RootDtoFamily
import org.example.hierarchydispatch.models.SecondDto
import org.example.hierarchydispatch.models.UnknownGroupDto
import org.example.hierarchydispatch.models.UnknownReusedGroupDto
import org.example.hierarchydispatch.models.UnknownReusedMissingRootDto
import org.example.hierarchydispatch.models.UnknownReusedRootDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class HierarchyDispatchStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun hierarchyDispatchBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> hierarchyDispatchBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic hierarchy dispatch body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun hierarchyDispatchContracts() = withTimeout(20000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val first = FirstDto("first")
        val second = SecondDto("second")
        val range = RangeDto(GroupDtoTag.THREE, GroupDtoMode.SLOW, "range")
        for (leaf in listOf<RootDto>(first, second, range)) {
            val wire = codec.encodeToJsonElement<RootDto>(leaf)
            check(codec.decodeFromJsonElement<RootDto>(wire) == leaf)
            check(codec.decodeFromJsonElement<GroupDto>(wire) == leaf)
        }
        // Source declares Range first and an identical FirstDuplicate afterward.
        // Const-first and first identical claim must select First.
        check(codec.decodeFromString<GroupDto>("""{"family":"group","tag":"one","mode":"fast","id":"selected"}""") == FirstDto("selected"))
        check(runCatching { codec.decodeFromString<GroupDto>("""{"family":"group","tag":"one","mode":"fast","duplicate":"invalid-known"}""") }.isFailure)
        val oneSlow = codec.decodeFromString<GroupDto>("""{"family":"group","tag":"one","mode":"slow","id":"enum"}""") as RangeDto
        check(oneSlow.tag === GroupDtoTag.ONE && oneSlow.mode === GroupDtoMode.SLOW)
        for (raw in listOf("""{"family":"group","tag":"two","mode":"fast","extra":[true,null]}""", """{"family":"group","tag":"one"}""", """{"family":"group","tag":null,"mode":"slow"}""")) {
            val element = codec.parseToJsonElement(raw)
            val unknown = codec.decodeFromJsonElement<RootDto>(element) as UnknownGroupDto
            check(unknown.family == RootDtoFamily.Unrecognized(""))
            check(unknown.tag == GroupDtoTag.Unrecognized("") && unknown.mode == GroupDtoMode.Unrecognized(""))
            check(codec.encodeToJsonElement<RootDto>(unknown) == element)
            check(codec.encodeToJsonElement<GroupDto>(unknown) == element)
        }
        for (raw in listOf("[]", """{"family":"group","tag":[],"mode":"fast"}""")) {
            check(runCatching { codec.decodeFromString<RootDto>(raw) }.isFailure)
        }
        val reusedFirst = ReusedFirstDto("reused-first")
        val reusedSecond = ReusedSecondDto("reused-second")
        val ancestorType: ReusedRootDtoTag = reusedSecond.tag
        check(ancestorType === ReusedRootDtoTag.TWO)
        check(codec.decodeFromJsonElement<ReusedRootDto>(codec.encodeToJsonElement<ReusedRootDto>(reusedFirst)) == reusedFirst)
        val secondWire = codec.encodeToJsonElement<ReusedGroupDto>(reusedSecond)
        check(codec.decodeFromJsonElement<ReusedGroupDto>(secondWire) == reusedSecond)
        check(codec.decodeFromJsonElement<ReusedRootDto>(secondWire) is UnknownReusedRootDto)
        val innerRaw = codec.parseToJsonElement("""{"tag":"future","extra":[1,null]}""")
        val innerUnknown = codec.decodeFromJsonElement<ReusedGroupDto>(innerRaw) as UnknownReusedGroupDto
        check(innerUnknown.tag == ReusedRootDtoTag.Unrecognized(""))
        check(codec.encodeToJsonElement<ReusedRootDto>(innerUnknown) == innerRaw)
        val missingFirst = ReusedMissingFirstDto("missing-first")
        val missingSecond = ReusedMissingSecondDto("missing-second")
        check(missingFirst.tag === ReusedMissingRootDtoTag.ONE && missingSecond.tag === ReusedMissingRootDtoTag.TWO)
        val missingWire = codec.encodeToJsonElement<ReusedMissingGroupDto>(missingSecond)
        check(codec.decodeFromJsonElement<ReusedMissingGroupDto>(missingWire) == missingSecond)
        check(codec.decodeFromJsonElement<ReusedMissingRootDto>(missingWire) is UnknownReusedMissingRootDto)
        check(codec.decodeFromString<ReusedMissingRootDto>(codec.encodeToString<ReusedMissingRootDto>(missingFirst)) == missingFirst)
        val raw = codec.parseToJsonElement("""{"family":"group","tag":"future","mode":"future"}""")
        val known = HolderDto(range, reusedFirst, missingFirst, listOf(first, second, range))
        val unknownHolder = HolderDto(UnknownGroupDto(raw), UnknownReusedRootDto(innerRaw), UnknownReusedMissingRootDto(innerRaw), listOf(first))
        for (value in listOf(known, unknownHolder)) check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var expected = known
        var patchMode = 0
        var bad = false
        val engine = HierarchyDispatchStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(hierarchyDispatchBody(request.body)) == expected)
                        respond(codec.encodeToString(expected), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        val wire = codec.parseToJsonElement(hierarchyDispatchBody(request.body)).jsonObject
                        check(wire["requiredNullable"] == if (nulls) JsonNull else null)
                        when (patchMode) {
                            0 -> check("value" !in wire)
                            1 -> check(wire["value"] == JsonNull)
                            2 -> check(codec.decodeFromJsonElement<ReusedRootDto>(wire.getValue("value")) == reusedFirst)
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
                    else -> error("Unexpected synthetic dispatch URL")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = HierarchyDispatchApiClient(http, "http://example.test/", codec).default
            for (value in listOf(known, unknownHolder)) { expected = value; check(api.echo(value) == value) }
            api.patchPatch(PatchInputDto(requiredNullable = null))
            patchMode = 1; api.patchPatch(PatchInputDto(requiredNullable = null, value = Patch.Null))
            patchMode = 2; api.patchPatch(PatchInputDto(requiredNullable = null, value = Patch.Value(reusedFirst)))
            val flow = api.watch(); check(channels.isEmpty())
            for (value in listOf(known, unknownHolder)) {
                expected = value
                check((flow.take(1).toList().single() as WatchEventDto.Sample).data == value)
                check(channels.last().isClosedForRead)
            }
            bad = true; check(runCatching { flow.take(1).toList() }.isFailure); check(channels.last().isClosedForRead)
            bad = false; check(flow.take(1).toList().size == 1 && api.echo(expected) == expected)
            println("HierarchyDispatch Json defaults=$defaults nulls=$nulls compound-constfirst-enumOR-duplicate-reused-Unknown-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
    }
}
