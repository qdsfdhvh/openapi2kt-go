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
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.wrapperchains.ValueClassReferenceChainsApiClient
import org.example.wrapperchains.api.models.PatchWrapperInputDto
import org.example.wrapperchains.api.models.WatchWrapperEventDto
import org.example.wrapperchains.models.AgainCountBridgeDto
import org.example.wrapperchains.models.CountBridgeDto
import org.example.wrapperchains.models.CountBridgeFalseDto
import org.example.wrapperchains.models.CountBridgeTrueDto
import org.example.wrapperchains.models.CountDto
import org.example.wrapperchains.models.HolderDto
import org.example.wrapperchains.models.NameBridgeDto
import org.example.wrapperchains.models.NameDto
import org.example.wrapperchains.models.Patch
import org.example.wrapperchains.models.WrapperChoiceDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class WrapperChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun wrapperChainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> wrapperChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic wrapper body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun valueClassReferenceChainsContracts() = withTimeout(20000) {
    check(serializer<CountBridgeDto>().descriptor.isNullable)
    check(serializer<CountBridgeTrueDto>().descriptor.isNullable)
    check(!serializer<CountBridgeFalseDto>().descriptor.isNullable)
    check(!serializer<AgainCountBridgeDto>().descriptor.isNullable)
    check(serializer<NameBridgeDto>().descriptor.isNullable)
    check(!CountBridgeDto.serializer().descriptor.isNullable)
    val count: CountBridgeFalseDto = CountBridgeFalseDto(3L)
    val named: NameBridgeDto = NameBridgeDto("雪")
    check(count == CountDto(3L) && count.rawValue == 3L)
    check(named == NameDto("雪"))
    check(runCatching { CountBridgeDto(0L) }.isFailure)
    check(runCatching { NameBridgeDto("") }.isFailure)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        for (value in listOf(null, count, CountDto(Long.MAX_VALUE))) check(codec.decodeFromString<CountBridgeDto>(codec.encodeToString<CountBridgeDto>(value)) == value)
        check(codec.decodeFromString<CountBridgeFalseDto>("3").rawValue == 3L)
        check(codec.decodeFromString<NameBridgeDto>(codec.encodeToString<NameBridgeDto>(named)) == named)
        check(runCatching { codec.decodeFromString<CountBridgeFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<CountBridgeDto>("0") }.isFailure)
        check(runCatching { codec.decodeFromString<NameBridgeDto>("\"\"") }.isFailure)
        val holder = HolderDto(null, count, named, listOf(null, count), mapOf("count" to count))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        val choice: WrapperChoiceDto = WrapperChoiceDto.A(count)
        check(codec.decodeFromString<WrapperChoiceDto>(codec.encodeToString(choice)) == choice)
        val absent = PatchWrapperInputDto(required = null, narrow = count)
        val clear = PatchWrapperInputDto(required = null, narrow = count, value = Patch.Null)
        val assigned = PatchWrapperInputDto(required = count, narrow = count, value = Patch.Value(count))
        check("value" !in codec.encodeToJsonElement(absent).jsonObject)
        check(if (nulls) codec.encodeToJsonElement(absent).jsonObject["required"] == JsonNull else "required" !in codec.encodeToJsonElement(absent).jsonObject)
        check(codec.encodeToJsonElement(clear).jsonObject["value"] == JsonNull)
        for (patch in listOf(absent, clear, assigned)) check(codec.decodeFromString<PatchWrapperInputDto>(codec.encodeToString(patch)) == patch)
        check(runCatching { codec.decodeFromString<PatchWrapperInputDto>("{\"required\":null,\"narrow\":3,\"value\":0}") }.isFailure)
        var expectedPatch = codec.encodeToJsonElement(absent)
        var expectedValue = "null"
        var failure = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = WrapperChainStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/wrappers" -> {
                        check(codec.decodeFromString<HolderDto>(wrapperChainBody(request.body)) == holder)
                        respond(if (failure == "holder") "{}" else codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/wrapper-nullable", "/wrapper-false", "/wrapper-name" -> {
                        check(codec.parseToJsonElement(wrapperChainBody(request.body)) == codec.parseToJsonElement(expectedValue))
                        respond(if (failure == "bounds") "0" else expectedValue, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/wrapper-query" -> {
                        check(request.url.parameters["count"] == "3")
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/wrapper-patch" -> {
                        check(request.method.value == "PATCH")
                        check(codec.parseToJsonElement(wrapperChainBody(request.body)) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/wrapper-watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frames = when (failure) {
                            "bounds" -> "event: nullable\r\ndata: 0\r\n\r\n"
                            "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                            else -> "event: nullable\r\ndata: null\r\n\r\nevent: narrow\r\ndata: 3\r\n\r\nevent: holder\r\ndata: ${codec.encodeToString(holder)}\r\n\r\n"
                        }
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
                    else -> error("Unexpected synthetic wrapper path")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = ValueClassReferenceChainsApiClient(http, "https://example.test/", codec).default
            check(api.echoWrappers(holder) == holder)
            api.queryWrapper(count)
            failure = "holder"; check(runCatching { api.echoWrappers(holder) }.isFailure); failure = ""
            check(api.nullableWrapper(null) == null)
            expectedValue = "3"; check(api.nullableWrapper(count) == count && api.falseWrapper(count) == count)
            failure = "bounds"; check(runCatching { api.nullableWrapper(count) }.isFailure); failure = ""
            expectedValue = codec.encodeToString<NameBridgeDto>(named); check(api.nameWrapper(named) == named)
            for (patch in listOf(absent, clear, assigned)) { expectedPatch = codec.encodeToJsonElement(patch); api.patchWrapper(patch) }
            val cold = api.watchWrapper(); check(channels.isEmpty())
            repeat(2) {
                val events = cold.take(3).toList()
                check((events[0] as WatchWrapperEventDto.Nullable).data == null)
                check((events[1] as WatchWrapperEventDto.Narrow).data == count)
                check((events[2] as WatchWrapperEventDto.Holder).data == holder)
                check(channels.last().isClosedForWrite)
            }
            for (bad in listOf("bounds", "narrow")) {
                failure = bad; check(runCatching { cold.take(1).toList() }.isFailure)
                check(channels.last().isClosedForWrite)
            }
            failure = ""; check((cold.take(1).toList().single() as WatchWrapperEventDto.Nullable).data == null)
            check(channels.last().isClosedForWrite)
            expectedValue = "3"; check(api.falseWrapper(count) == count)
        } finally { http.close(); scope.cancel(); writers.forEach { it.join() } }
        println("ValueClassReferenceChains Json defaults=$defaults nulls=$nulls canonical-wrapper-rawValue-null-false-bounds-Patch-HTTP-SSEclose-reuse-cold passed")
    }
}
