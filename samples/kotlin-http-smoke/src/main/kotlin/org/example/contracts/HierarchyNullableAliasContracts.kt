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
import org.example.hierarchynullable.HierarchyNullableAliasApiClient
import org.example.hierarchynullable.api.models.AliasPatchInputDto
import org.example.hierarchynullable.api.models.AliasWatchEventDto
import org.example.hierarchynullable.models.AFirstDto
import org.example.hierarchynullable.models.AGroupDto
import org.example.hierarchynullable.models.ARootDto
import org.example.hierarchynullable.models.AReusedDto
import org.example.hierarchynullable.models.AReusedLeafDto
import org.example.hierarchynullable.models.AliasHolderDto
import org.example.hierarchynullable.models.BRootDto
import org.example.hierarchynullable.models.CRootDto
import org.example.hierarchynullable.models.DRootDto
import org.example.hierarchynullable.models.FirstDto
import org.example.hierarchynullable.models.GroupDtoMode
import org.example.hierarchynullable.models.GroupDtoTag
import org.example.hierarchynullable.models.Patch
import org.example.hierarchynullable.models.RangeDto
import org.example.hierarchynullable.models.ReusedFirstDto
import org.example.hierarchynullable.models.RootDto
import org.example.hierarchynullable.models.UnknownGroupDto
import org.example.hierarchynullable.models.UnknownRootDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class HierarchyNullableStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun hierarchyNullableBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> hierarchyNullableBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    is OutgoingContent.NoContent -> "null"
    else -> error("Unexpected synthetic nullable hierarchy body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun hierarchyNullableAliasContracts() = withTimeout(20000) {
    val first = AFirstDto("canonical-first")
    check(first == FirstDto("canonical-first"))
    check(!ARootDto.serializer().descriptor.isNullable)
    check(serializer<ARootDto>().descriptor.isNullable && serializer<BRootDto>().descriptor.isNullable)
    check(!serializer<CRootDto>().descriptor.isNullable && serializer<DRootDto>().descriptor.isNullable)
    check(serializer<AFirstDto>().descriptor.isNullable && serializer<AGroupDto>().descriptor.isNullable)
    check(serializer<AReusedDto>().descriptor.isNullable && serializer<AReusedLeafDto>().descriptor.isNullable)
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
        val engine = HierarchyNullableStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/nullable", "/narrow", "/optional" -> {
                        check(codec.parseToJsonElement(hierarchyNullableBody(request.body)) == expected)
                        val body = when (failure) {
                            "null" -> "null"
                            "known" -> """{"family":"group","tag":"one","mode":"fast"}"""
                            else -> expected.toString()
                        }
                        respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/aliases" -> {
                        check(codec.decodeFromString<AliasHolderDto>(hierarchyNullableBody(request.body)) == holder)
                        respond(codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/alias-patch" -> {
                        check(codec.parseToJsonElement(hierarchyNullableBody(request.body)) == patchExpected)
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
            val api = HierarchyNullableAliasApiClient(http, "http://example.test/", codec).default
            check(api.nullable(null) == null && api.optional() == null)
            expected = codec.encodeToJsonElement<ARootDto>(range)
            check(api.nullable(range) == range && api.narrow(range) == range && api.optional(range) == range)
            check(api.aliases(holder) == holder)
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
            println("HierarchyNullableAlias Json defaults=$defaults nulls=$nulls canonical-nullablecodec-nearest-false-Unknown-Patch-HTTP-null-SSEclose-reuse-cold passed")
        } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
    }
}
