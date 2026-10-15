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
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import org.example.hierarchyfields.HierarchyFieldsApiClient
import org.example.hierarchyfields.api.models.PatchInputDto
import org.example.hierarchyfields.api.models.WatchEventDto
import org.example.hierarchyfields.models.DeepFirstDto
import org.example.hierarchyfields.models.DeepGroupDto
import org.example.hierarchyfields.models.DeepRootDto
import org.example.hierarchyfields.models.DeepRootDtoFamily
import org.example.hierarchyfields.models.DeepGroupDtoStage
import org.example.hierarchyfields.models.DeepSecondDto
import org.example.hierarchyfields.models.FixedLeafDto
import org.example.hierarchyfields.models.FixedRootDto
import org.example.hierarchyfields.models.FixedRootDtoKind
import org.example.hierarchyfields.models.HolderDto
import org.example.hierarchyfields.models.MissingFirstDto
import org.example.hierarchyfields.models.MissingRootDto
import org.example.hierarchyfields.models.MissingRootDtoKind
import org.example.hierarchyfields.models.MissingSecondDto
import org.example.hierarchyfields.models.Patch
import org.example.hierarchyfields.models.UnknownDeepGroupDto
import org.example.hierarchyfields.models.UnknownDeepRootDto
import org.example.hierarchyfields.models.UnknownFixedRootDto
import org.example.hierarchyfields.models.UnknownMissingRootDto
import org.example.hierarchyfields.models.UnknownVaryGroupDto
import org.example.hierarchyfields.models.UnknownVaryRootDto
import org.example.hierarchyfields.models.VaryFirstDto
import org.example.hierarchyfields.models.VaryGroupDto
import org.example.hierarchyfields.models.VaryRootDto
import org.example.hierarchyfields.models.VaryRootDtoFamily
import org.example.hierarchyfields.models.VarySecondDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class HierarchyFieldsStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun hierarchyFieldsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> hierarchyFieldsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic hierarchy body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun hierarchyFieldsContracts() = withTimeout(20000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val fixed = FixedLeafDto("fixed-id")
        check(fixed.kind === FixedRootDtoKind.FIXED)
        check(codec.decodeFromJsonElement<FixedRootDto>(codec.encodeToJsonElement<FixedRootDto>(fixed)) == fixed)
        val first = MissingFirstDto("first-id")
        val second = MissingSecondDto(7)
        check(first.kind === MissingRootDtoKind.FIRST_A_B && second.kind === MissingRootDtoKind.SECOND)
        for (leaf in listOf<MissingRootDto>(first, second)) {
            val wire = codec.encodeToJsonElement<MissingRootDto>(leaf)
            check("kind" in wire.jsonObject)
            check(codec.decodeFromJsonElement<MissingRootDto>(wire) == leaf)
        }
        val absentTag = codec.parseToJsonElement("""{"id":"first-id"}""")
        check(codec.decodeFromJsonElement<MissingFirstDto>(absentTag) == first)
        check(codec.decodeFromJsonElement<MissingRootDto>(absentTag) is UnknownMissingRootDto)
        check(codec.decodeFromString<MissingRootDto>("""{"kind":"alias-only","id":"ignored"}""") is UnknownMissingRootDto)
        val deepFirst = DeepFirstDto("deep-id")
        val deepSecond = DeepSecondDto(9)
        check(deepFirst.family === DeepRootDtoFamily.GROUP && deepFirst.stage === DeepGroupDtoStage.FIRST)
        for (leaf in listOf<DeepRootDto>(deepFirst, deepSecond)) {
            val wire = codec.encodeToJsonElement<DeepRootDto>(leaf)
            check(codec.decodeFromJsonElement<DeepRootDto>(wire) == leaf)
            check(codec.decodeFromJsonElement<DeepGroupDto>(wire) == leaf)
        }
        val varying = VaryFirstDto(VaryRootDtoFamily.GROUP, "vary-id")
        val alternate = VaryFirstDto(VaryRootDtoFamily.ALTERNATE, "vary-alternate")
        val alternateWire = codec.encodeToJsonElement<VaryGroupDto>(alternate)
        check(codec.decodeFromJsonElement<VaryGroupDto>(alternateWire) == alternate)
        check(codec.decodeFromJsonElement<VaryRootDto>(alternateWire) is UnknownVaryRootDto)
        val varySecond = VarySecondDto(11)
        check(varySecond.family === VaryRootDtoFamily.GROUP)
        for (leaf in listOf<VaryRootDto>(varying, varySecond)) {
            val wire = codec.encodeToJsonElement<VaryRootDto>(leaf)
            check(codec.decodeFromJsonElement<VaryRootDto>(wire) == leaf)
            check(codec.decodeFromJsonElement<VaryGroupDto>(wire) == leaf)
        }
        for (raw in listOf("{}", """{"kind":"future","extra":[true,null]}""", """{"kind":null}""")) {
            val element = codec.parseToJsonElement(raw)
            val unknown = codec.decodeFromJsonElement<FixedRootDto>(element) as UnknownFixedRootDto
            check(unknown.kind == FixedRootDtoKind.Unrecognized(""))
            check(codec.encodeToJsonElement<FixedRootDto>(unknown) == element)
        }
        val deepRaw = codec.parseToJsonElement("""{"family":"group","stage":"future","extra":[1,null]}""")
        val deepUnknown = codec.decodeFromJsonElement<DeepRootDto>(deepRaw) as UnknownDeepGroupDto
        check(deepUnknown.family == DeepRootDtoFamily.Unrecognized("") && deepUnknown.stage == DeepGroupDtoStage.Unrecognized(""))
        check(codec.encodeToJsonElement<DeepRootDto>(deepUnknown) == deepRaw)
        check(codec.encodeToJsonElement<DeepGroupDto>(deepUnknown) == deepRaw)
        check(codec.decodeFromString<DeepRootDto>("""{"family":"future"}""") is UnknownDeepRootDto)
        val varyRaw = codec.parseToJsonElement("""{"family":"group","kind":"future","extra":true}""")
        val varyUnknown = codec.decodeFromJsonElement<VaryRootDto>(varyRaw) as UnknownVaryGroupDto
        check(codec.encodeToJsonElement<VaryRootDto>(varyUnknown) == varyRaw)
        check(codec.encodeToJsonElement<VaryGroupDto>(varyUnknown) == varyRaw)
        val requiredFamily = codec.parseToJsonElement("""{"family":"group","kind":"first","id":"bad"}""")
        check(runCatching { codec.decodeFromJsonElement<VaryGroupDto>(JsonObject(requiredFamily.jsonObject - "family")) }.isFailure)
        check(runCatching { codec.decodeFromString<VaryGroupDto>("""{"family":null,"kind":"first","id":"bad"}""") }.isFailure)
        for (bad in listOf("[]", """{"kind":[]}""", """{"kind":"first/a~b"}""")) {
            check(runCatching { codec.decodeFromString<MissingRootDto>(bad) }.isFailure)
        }
        val known = HolderDto(fixed, first, deepFirst, varying, listOf(first, second))
        val unknownHolder = HolderDto(UnknownFixedRootDto(codec.parseToJsonElement("{}")), UnknownMissingRootDto(absentTag), deepUnknown, varyUnknown, listOf(second))
        for (value in listOf(known, unknownHolder)) check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)

        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var expected = known
        var patchMode = 0
        var bad = false
        val engine = HierarchyFieldsStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(hierarchyFieldsBody(request.body)) == expected)
                        respond(codec.encodeToString(expected), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        val wire = codec.parseToJsonElement(hierarchyFieldsBody(request.body)).jsonObject
                        check(wire["requiredNullable"] == if (nulls) JsonNull else null)
                        when (patchMode) {
                            0 -> check("value" !in wire)
                            1 -> check(wire["value"] == JsonNull)
                            2 -> check(codec.decodeFromJsonElement<MissingRootDto>(wire.getValue("value")) == first)
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
                    else -> error("Unexpected synthetic hierarchy URL")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val client = HierarchyFieldsApiClient(http, "http://example.test/", codec)
            for (value in listOf(known, unknownHolder)) { expected = value; check(client.hierarchy.echo(value) == value) }
            client.default.patchPatch(PatchInputDto(requiredNullable = null))
            patchMode = 1; client.default.patchPatch(PatchInputDto(requiredNullable = null, value = Patch.Null))
            patchMode = 2; client.default.patchPatch(PatchInputDto(requiredNullable = null, value = Patch.Value(first)))
            val flow = client.default.watch(); check(channels.isEmpty())
            for (value in listOf(known, unknownHolder)) {
                expected = value
                check((flow.take(1).toList().single() as WatchEventDto.Sample).data == value)
                check(channels.last().isClosedForRead)
            }
            bad = true; check(runCatching { flow.take(1).toList() }.isFailure); check(channels.last().isClosedForRead)
            bad = false; check(flow.take(1).toList().size == 1 && client.hierarchy.echo(expected) == expected)
            println("HierarchyFields Json defaults=$defaults nulls=$nulls fixed-missing-ancestor-typed-Unknown-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
    }
}
