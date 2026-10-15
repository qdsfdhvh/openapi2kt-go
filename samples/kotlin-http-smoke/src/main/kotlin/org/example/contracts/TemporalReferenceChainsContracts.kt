@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, kotlin.time.ExperimentalTime::class)
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
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.temporalchains.TemporalReferenceChainsApiClient
import org.example.temporalchains.api.models.PatchTemporalInputDto
import org.example.temporalchains.api.models.WatchTemporalEventDto
import org.example.temporalchains.models.AgainBridgeDto
import org.example.temporalchains.models.DayBridgeDto
import org.example.temporalchains.models.DayFalseDto
import org.example.temporalchains.models.HolderDto
import org.example.temporalchains.models.NativeBridgeDto
import org.example.temporalchains.models.Patch
import org.example.temporalchains.models.StampBridgeDto
import org.example.temporalchains.models.StampFalseBridgeDto
import org.example.temporalchains.models.TemporalChoiceDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class TemporalChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun temporalChainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> temporalChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic temporal body")
}

suspend fun temporalReferenceChainsContracts() = withTimeout(20000) {
    check(serializer<StampBridgeDto>().descriptor.isNullable)
    check(serializer<DayBridgeDto>().descriptor.isNullable)
    check(!serializer<StampFalseBridgeDto>().descriptor.isNullable)
    check(!serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<DayFalseDto>().descriptor.isNullable)
    check(!StampBridgeDto.serializer().descriptor.isNullable)
    check(!DayBridgeDto.serializer().descriptor.isNullable)
    val stamp: StampFalseBridgeDto = StampFalseBridgeDto.parse("2024-02-29T12:34:56Z")
    val native: NativeBridgeDto = NativeBridgeDto.parse("2024-02-29T12:34:56Z")
    val day: DayFalseDto = DayFalseDto.parse("2024-02-29")
    check(stamp == native && AgainBridgeDto.parse(stamp.toString()) == stamp)
    check(runCatching { DayBridgeDto.parse("2024-02-30") }.isFailure)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        for (value in listOf(null,stamp)) check(codec.decodeFromString<StampBridgeDto>(codec.encodeToString<StampBridgeDto>(value)) == value)
        for (value in listOf(null,day)) check(codec.decodeFromString<DayBridgeDto>(codec.encodeToString<DayBridgeDto>(value)) == value)
        check(codec.encodeToString<StampFalseBridgeDto>(stamp) == "\"2024-02-29T12:34:56Z\"")
        check(codec.encodeToString<DayFalseDto>(day) == "\"2024-02-29\"")
        check(runCatching { codec.decodeFromString<StampFalseBridgeDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<DayFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<StampBridgeDto>("123") }.isFailure)
        check(runCatching { codec.decodeFromString<DayBridgeDto>("\"2024-02-30\"") }.isFailure)
        val holder = HolderDto(null,stamp,day,native,listOf(null,stamp),mapOf("day" to day))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        val choice: TemporalChoiceDto = TemporalChoiceDto.Time(stamp)
        check(codec.decodeFromString<TemporalChoiceDto>(codec.encodeToString<TemporalChoiceDto>(choice)) == choice)
        val patches = listOf(PatchTemporalInputDto(requiredStamp = null,falseStamp = stamp),PatchTemporalInputDto(requiredStamp = null,falseStamp = stamp,stamp = Patch.Null,day = Patch.Null),PatchTemporalInputDto(requiredStamp = stamp,falseStamp = stamp,stamp = Patch.Value(stamp),day = Patch.Value(day)))
        check("stamp" !in codec.encodeToJsonElement(patches[0]).jsonObject)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["stamp"] == JsonNull)
        for (patch in patches) check(codec.decodeFromString<PatchTemporalInputDto>(codec.encodeToString(patch)) == patch)
        check(runCatching { codec.decodeFromString<PatchTemporalInputDto>("{\"requiredStamp\":null,\"falseStamp\":\"2024-02-29T12:34:56Z\",\"day\":\"2024-02-30\"}") }.isFailure)
        var expected = "null"
        var expectedPatch = codec.encodeToJsonElement(patches[0])
        var failure = ""
        var queryDay = true
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = TemporalChainStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/holder" -> { check(codec.decodeFromString<HolderDto>(temporalChainBody(request.body)) == holder); respond(codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type","application/json")) }
                    "/stamp", "/false", "/day", "/native", "/erasure-left", "/erasure-right" -> {
                        check(codec.parseToJsonElement(temporalChainBody(request.body)) == codec.parseToJsonElement(expected))
                        val result = when (failure) { "stamp" -> "\"invalid\""; "day" -> "\"2024-02-30\""; "narrow" -> "null"; else -> expected }
                        respond(result,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/query" -> { check(request.url.parameters["stamp"] == stamp.toString()); check(request.url.parameters["day"] == if (queryDay) day.toString() else null); respond("",HttpStatusCode.NoContent) }
                    "/patch" -> { check(request.method.value == "PATCH"); check(codec.parseToJsonElement(temporalChainBody(request.body)) == expectedPatch); respond("",HttpStatusCode.NoContent) }
                    "/watch" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val frames=when(failure) {
                            "stamp" -> "event: stamp\r\ndata: \"invalid\"\r\n\r\n"
                            "day" -> "event: day\r\ndata: \"2024-02-30\"\r\n\r\n"
                            "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                            else -> "event: stamp\r\ndata: null\r\n\r\nevent: narrow\r\ndata: ${codec.encodeToString<StampFalseBridgeDto>(stamp)}\r\n\r\nevent: day\r\ndata: ${codec.encodeToString<DayBridgeDto>(day)}\r\n\r\n"
                        }
                        writers.add(scope.launch {
                            try { for(byte in frames.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() }
                            catch(closed:ClosedByteChannelException) { if(!channel.isClosedForWrite) throw closed }
                        })
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic temporal route")
                }
            }
        })
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api=TemporalReferenceChainsApiClient(http,"https://example.test/",codec).default
            check(api.echoTemporalHolder(holder) == holder)
            check(api.echoStamp(null) == null && api.echoDay(null) == null)
            expected=codec.encodeToString<StampFalseBridgeDto>(stamp)
            check(api.echoFalseStamp(stamp) == stamp && api.echoNative(native) == native && api.echoTime(stamp) == stamp && api.echoTime2(native) == native)
            api.queryTemporal(stamp,day);queryDay=false;api.queryTemporal(stamp)
            failure="stamp";check(runCatching { api.echoStamp(stamp) }.isFailure);failure="narrow";check(runCatching { api.echoFalseStamp(stamp) }.isFailure)
            failure="";expected=codec.encodeToString<DayBridgeDto>(day);check(api.echoDay(day) == day)
            failure="day";check(runCatching { api.echoDay(day) }.isFailure);failure=""
            for(patch in patches) {expectedPatch=codec.encodeToJsonElement(patch);api.patchTemporal(patch)}
            val cold=api.watchTemporal();check(channels.isEmpty())
            repeat(2) {
                val events=cold.take(3).toList()
                check((events[0] as WatchTemporalEventDto.Stamp).data == null)
                check((events[1] as WatchTemporalEventDto.Narrow).data == stamp)
                check((events[2] as WatchTemporalEventDto.Day).data == day)
                check(channels.last().isClosedForWrite)
            }
            for(bad in listOf("stamp","day","narrow")) {failure=bad;check(runCatching { cold.take(1).toList() }.isFailure);check(channels.last().isClosedForWrite)}
            failure="";check((cold.take(1).toList().single() as WatchTemporalEventDto.Stamp).data == null);check(channels.last().isClosedForWrite)
            expected=codec.encodeToString<StampFalseBridgeDto>(stamp);check(api.echoFalseStamp(stamp) == stamp)
        } finally {http.close();scope.cancel();writers.forEach {it.join()} }
        println("TemporalReferenceChains Json defaults=$defaults nulls=$nulls canonical-ISO-null-false-Patch-query-erasure-HTTP-liveSSEclose-reuse-cold passed")
    }
}
