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
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.enumowners.EnumNominalOwnersApiClient
import org.example.enumowners.api.models.PatchEnumInputDto
import org.example.enumowners.api.models.WatchEnumEventDto
import org.example.enumowners.models.BorrowOwnerDto
import org.example.enumowners.models.ChoiceDto
import org.example.enumowners.models.EnumConsumerChoiceDto
import org.example.enumowners.models.HolderDto
import org.example.enumowners.models.MapStateDto
import org.example.enumowners.models.NamedBridgeDto
import org.example.enumowners.models.NamedStateDto
import org.example.enumowners.models.NestedBridgeDto
import org.example.enumowners.models.NestedStateDto
import org.example.enumowners.models.NestedStateFalseDto
import org.example.enumowners.models.OwnerDto
import org.example.enumowners.models.Patch
import org.example.enumowners.models.RootMapDtoValue
import org.example.enumowners.models.RootMapStateDto
import org.example.enumowners.models.RootRowsDtoInner
import org.example.enumowners.models.RootRowStateDto
import org.example.enumowners.models.RowStateDto
import org.example.enumowners.models.VariantStateDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class EnumOwnerStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun enumOwnerBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> enumOwnerBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic enum body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun enumNominalOwnersContracts() = withTimeout(20000) {
    check(NestedStateDto.entries == OwnerDto.State.entries)
    check(NestedStateDto.fromSerialName("new") === OwnerDto.State.NEW)
    check(NestedStateFalseDto.fromSerialName("future") == OwnerDto.State.Unrecognized("future"))
    check(RowStateDto.entries == OwnerDto.RowsInner.entries)
    check(MapStateDto.entries == OwnerDto.EntriesValue.entries)
    check(VariantStateDto.entries == ChoiceDto.A.State.entries)
    check(RootRowStateDto.entries == RootRowsDtoInner.entries)
    check(RootMapStateDto.entries == RootMapDtoValue.entries)
    check(NamedBridgeDto.entries == NamedStateDto.entries)
    check(serializer<NestedStateDto>().descriptor.isNullable)
    check(serializer<NestedBridgeDto>().descriptor.isNullable)
    check(!serializer<NestedStateFalseDto>().descriptor.isNullable)
    check(!serializer<NamedBridgeDto>().descriptor.isNullable)
    check(!NestedStateDto.serializer().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val known = OwnerDto.State.NEW
        val future = OwnerDto.State.Unrecognized("future-雪")
        for (value in listOf(null, known, future)) {
            check(codec.decodeFromString<NestedBridgeDto>(codec.encodeToString<NestedBridgeDto>(value)) == value)
            check(codec.decodeFromString(OwnerDto.State.serializer().nullable, codec.encodeToString<NestedStateDto>(value)) == value)
        }
        check(runCatching { codec.decodeFromString<NestedStateFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<NestedStateDto>("123") }.isFailure)
        check(runCatching { codec.decodeFromString<NamedBridgeDto>("null") }.isFailure)
        val holder = HolderDto(future, known, ChoiceDto.A.State.Unrecognized("future"), NamedStateDto.ON)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        val union: EnumConsumerChoiceDto = EnumConsumerChoiceDto.A(future)
        check(codec.decodeFromString<EnumConsumerChoiceDto>(codec.encodeToString(union)) == union)
        val owner = OwnerDto(future, listOf(OwnerDto.RowsInner.LEFT, OwnerDto.RowsInner.Unrecognized("future")), mapOf("known" to OwnerDto.EntriesValue.YES))
        check(codec.decodeFromString<OwnerDto>(codec.encodeToString(owner)) == owner)
        val borrowed = BorrowOwnerDto(BorrowOwnerDto.State.Unrecognized("future"))
        check(codec.decodeFromString<BorrowOwnerDto>(codec.encodeToString(borrowed)) == borrowed)
        check(codec.decodeFromString<VariantStateDto>("\"future\"") == ChoiceDto.A.State.Unrecognized("future"))
        check(codec.decodeFromString<RootRowStateDto>("\"future\"") == RootRowsDtoInner.Unrecognized("future"))
        check(codec.decodeFromString<RootMapStateDto>("\"future\"") == RootMapDtoValue.Unrecognized("future"))
        val absent = PatchEnumInputDto(required = null, narrow = known)
        val clear = PatchEnumInputDto(required = null, narrow = known, value = Patch.Null)
        val assigned = PatchEnumInputDto(required = future, narrow = known, value = Patch.Value(future))
        check("value" !in codec.encodeToJsonElement(absent).jsonObject)
        check(if (nulls) codec.encodeToJsonElement(absent).jsonObject["required"] == JsonNull else "required" !in codec.encodeToJsonElement(absent).jsonObject)
        check(codec.encodeToJsonElement(clear).jsonObject["value"] == JsonNull)
        for (patch in listOf(absent, clear, assigned)) check(codec.decodeFromString<PatchEnumInputDto>(codec.encodeToString(patch)) == patch)
        var expectedPatch = codec.encodeToJsonElement(absent)
        var expectedEnum = "null"
        var failure = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = EnumOwnerStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/enums" -> {
                        check(codec.decodeFromString<HolderDto>(enumOwnerBody(request.body)) == holder)
                        respond(if (failure == "holder") "{}" else codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/enum-nullable", "/enum-false" -> {
                        check(codec.parseToJsonElement(enumOwnerBody(request.body)) == codec.parseToJsonElement(expectedEnum))
                        respond(if (failure == "kind") "123" else expectedEnum, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/enum-patch" -> {
                        check(request.method.value == "PATCH")
                        check(codec.parseToJsonElement(enumOwnerBody(request.body)) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/enum-watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frames = when (failure) {
                            "kind" -> "event: nullable\r\ndata: 123\r\n\r\n"
                            "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                            else -> "event: nullable\r\ndata: null\r\n\r\nevent: narrow\r\ndata: ${codec.encodeToString<NestedStateFalseDto>(future)}\r\n\r\nevent: holder\r\ndata: ${codec.encodeToString(holder)}\r\n\r\n"
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
                    else -> error("Unexpected synthetic enum path")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = EnumNominalOwnersApiClient(http, "https://example.test/", codec).default
            check(api.echoEnumHolder(holder) == holder)
            failure = "holder"; check(runCatching { api.echoEnumHolder(holder) }.isFailure); failure = ""
            check(api.nullableEnum(null) == null)
            expectedEnum = codec.encodeToString<NestedStateFalseDto>(future)
            check(api.nullableEnum(future) == future && api.falseEnum(future) == future)
            failure = "kind"; check(runCatching { api.nullableEnum(future) }.isFailure); failure = ""
            for (patch in listOf(absent, clear, assigned)) { expectedPatch = codec.encodeToJsonElement(patch); api.patchEnum(patch) }
            val cold = api.watchEnum(); check(channels.isEmpty())
            repeat(2) {
                val events = cold.take(3).toList()
                check((events[0] as WatchEnumEventDto.Nullable).data == null)
                check((events[1] as WatchEnumEventDto.Narrow).data == future)
                check((events[2] as WatchEnumEventDto.Holder).data == holder)
                check(channels.last().isClosedForWrite)
            }
            for (bad in listOf("kind", "narrow")) {
                failure = bad; check(runCatching { cold.take(1).toList() }.isFailure)
                check(channels.last().isClosedForWrite)
            }
            failure = ""; check((cold.take(1).toList().single() as WatchEnumEventDto.Nullable).data == null)
            check(channels.last().isClosedForWrite)
            check(api.falseEnum(future) == future)
        } finally { http.close(); scope.cancel(); writers.forEach { it.join() } }
        println("EnumNominalOwners Json defaults=$defaults nulls=$nulls actual-retained-vocabulary-owner-unknown-null-false-Patch-HTTP-SSEclose-reuse-cold passed")
    }
}
