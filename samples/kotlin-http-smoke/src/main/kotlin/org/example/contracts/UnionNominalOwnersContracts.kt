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
import org.example.unionowners.UnionNominalOwnersApiClient
import org.example.unionowners.api.models.PatchUnionInputDto
import org.example.unionowners.api.models.WatchUnionEventDto
import org.example.unionowners.models.CarrierFalseDto
import org.example.unionowners.models.BorrowUnionBoxDto
import org.example.unionowners.models.CarrierUnionDto
import org.example.unionowners.models.ChoiceDto
import org.example.unionowners.models.NativeInnerDto
import org.example.unionowners.models.NativeInnerFalseDto
import org.example.unionowners.models.NullableCarrierDto
import org.example.unionowners.models.NullableUnionDto
import org.example.unionowners.models.NullableUnionInheritDto
import org.example.unionowners.models.NullableUnionFalseDto
import org.example.unionowners.models.NullableUnionAgainDto
import org.example.unionowners.models.NullCarrierUnionDto
import org.example.unionowners.models.ObjectMixedChoiceDto
import org.example.unionowners.models.Patch
import org.example.unionowners.models.StringCarrierUnionDto
import org.example.unionowners.models.RootRowUnionDto
import org.example.unionowners.models.RootRowUnionMaybeDto
import org.example.unionowners.models.RootMapUnionDto
import org.example.unionowners.models.RootRowsDtoInner
import org.example.unionowners.models.RootMapDtoValue
import org.example.unionowners.models.UnionBoxDto
import org.example.unionowners.models.UnionHolderDto
import org.example.unionowners.models.UnionInnerDto
import org.example.unionowners.models.UnionInnerFalseDto
import org.example.unionowners.models.UnionRootBridgeDto
import org.example.unionowners.models.UnionNativeBridgeDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class UnionOwnerStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun unionOwnerBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> unionOwnerBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic union body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun unionNominalOwnersContracts(checked: Boolean = false) = withTimeout(20000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val inner: UnionInnerFalseDto = UnionBoxDto.Choice.A("recursive", UnionBoxDto.Choice.B("child"))
        val native: NativeInnerFalseDto = UnionBoxDto.StringNative("native")
        val root: UnionRootBridgeDto = ChoiceDto.Other("root")
        val rootNative: UnionNativeBridgeDto = ObjectMixedChoiceDto("root-native", ObjectMixedChoiceDto.Node("node"))
        check(serializer<UnionInnerDto>().descriptor.isNullable)
        check(!serializer<UnionInnerFalseDto>().descriptor.isNullable)
        check(serializer<NativeInnerDto>().descriptor.isNullable)
        check(!serializer<NativeInnerFalseDto>().descriptor.isNullable)
        check(serializer<NullableUnionDto>().descriptor.isNullable)
        check(serializer<NullableUnionInheritDto>().descriptor.isNullable)
        check(!serializer<NullableUnionFalseDto>().descriptor.isNullable)
        check(serializer<NullableUnionAgainDto>().descriptor.isNullable)
        check(!UnionInnerDto.serializer().descriptor.isNullable)
        check(codec.decodeFromString<UnionInnerDto>("null") == null)
        check(codec.decodeFromString<NativeInnerDto>("null") == null)
        check(runCatching { codec.decodeFromString<UnionInnerFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<NativeInnerFalseDto>("null") }.isFailure)
        check(codec.decodeFromString<NullableUnionDto>("null") == null)
        check(runCatching { codec.decodeFromString<NullableUnionFalseDto>("null") }.isFailure)
        check(codec.decodeFromString<NullableUnionDto>(codec.encodeToString<NullableUnionDto>(root)) == root)
        check(codec.decodeFromString<UnionInnerFalseDto>(codec.encodeToString(inner)) == inner)
        check(codec.decodeFromString<NativeInnerFalseDto>(codec.encodeToString(native)) == native)
        val borrowed = BorrowUnionBoxDto(BorrowUnionBoxDto.Choice.A("borrowed", inner), BorrowUnionBoxDto.BooleanNative(true))
        check(codec.decodeFromString<BorrowUnionBoxDto>(codec.encodeToString(borrowed)) == borrowed)
        val row: RootRowUnionDto = RootRowsDtoInner.R(RootRowsDtoInner.R.Node("row"))
        val map: RootMapUnionDto = RootMapDtoValue.M(RootMapDtoValue.M.Node("map"))
        check(codec.decodeFromString<RootRowUnionDto>(codec.encodeToString(row)) == row)
        check(codec.decodeFromString<RootMapUnionDto>(codec.encodeToString(map)) == map)
        check(codec.decodeFromString<RootRowUnionMaybeDto>("null") == null)
        val holder = UnionHolderDto(null, inner, null, native, root, rootNative)
        check(codec.decodeFromString<UnionHolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<CarrierUnionDto>("null") == NullCarrierUnionDto)
        check(codec.decodeFromString<CarrierFalseDto>("null") == NullCarrierUnionDto)
        check(codec.decodeFromString<NullableCarrierDto>("null") == null)
        check(codec.decodeFromString(CarrierUnionDto.serializer(), "null") == NullCarrierUnionDto)
        check(!serializer<CarrierFalseDto>().descriptor.isNullable)
        check(serializer<NullableCarrierDto>().descriptor.isNullable)
        check(codec.encodeToString<NullableCarrierDto>(NullCarrierUnionDto) == "null")
        check(codec.decodeFromString<NullableCarrierDto>(codec.encodeToString<NullableCarrierDto>(NullCarrierUnionDto)) == null)
        check(codec.decodeFromString<CarrierFalseDto>(codec.encodeToString<CarrierFalseDto>(NullCarrierUnionDto)) == NullCarrierUnionDto)
        check(runCatching { codec.decodeFromString<UnionInnerFalseDto>("""{"kind":"a"}""") }.isFailure)
        if (checked) {
            check(runCatching { UnionBoxDto.Choice.A("") }.isFailure)
            check(runCatching { codec.decodeFromString<UnionInnerFalseDto>("""{"kind":"a","id":""}""") }.isFailure)
        }
        val absent = PatchUnionInputDto(required = null, narrow = inner)
        val clear = PatchUnionInputDto(required = null, narrow = inner, inner = Patch.Null, native = Patch.Null)
        val assigned = PatchUnionInputDto(required = root, narrow = inner, inner = Patch.Value(inner), native = Patch.Value(native))
        check("inner" !in codec.encodeToJsonElement(absent).jsonObject)
        check(codec.encodeToJsonElement(clear).jsonObject["inner"] == JsonNull)
        check(codec.encodeToJsonElement(clear).jsonObject["native"] == JsonNull)
        check(codec.decodeFromString<PatchUnionInputDto>(codec.encodeToString(assigned)) == assigned)
        var expectedPatch = codec.encodeToJsonElement(absent)
        var expectedCarrier = "null"
        var failure = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = UnionOwnerStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/unions" -> {
                        check(codec.decodeFromString<UnionHolderDto>(unionOwnerBody(request.body)) == holder)
                        respond(if (failure == "holder") "{}" else codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/union-nullable", "/union-false" -> {
                        check(codec.parseToJsonElement(unionOwnerBody(request.body)) == codec.parseToJsonElement(expectedCarrier))
                        respond(expectedCarrier, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/union-patch" -> {
                        check(request.method.value == "PATCH")
                        check(codec.parseToJsonElement(unionOwnerBody(request.body)) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/union-watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frames = when (failure) {
                            "inner" -> "event: inner\r\ndata: {\"kind\":\"a\"}\r\n\r\n"
                            "null-inner" -> "event: inner\r\ndata: null\r\n\r\n"
                            else -> "event: nullable\r\ndata: null\r\n\r\nevent: carrier\r\ndata: null\r\n\r\nevent: inner\r\ndata: ${codec.encodeToString(inner)}\r\n\r\nevent: holder\r\ndata: ${codec.encodeToString(holder)}\r\n\r\n"
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
                    else -> error("Unexpected synthetic union path")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = UnionNominalOwnersApiClient(http, "https://example.test/", codec).default
            check(api.echoUnion(holder) == holder)
            failure = "holder"; check(runCatching { api.echoUnion(holder) }.isFailure)
            failure = ""
            check(api.nullableUnion(null) == null)
            check(api.falseUnion(NullCarrierUnionDto) == NullCarrierUnionDto)
            val carrier = StringCarrierUnionDto("known")
            expectedCarrier = codec.encodeToString<CarrierUnionDto>(carrier)
            check(api.nullableUnion(carrier) == carrier && api.falseUnion(carrier) == carrier)
            api.patchUnion(absent)
            expectedPatch = codec.encodeToJsonElement(clear); api.patchUnion(clear)
            expectedPatch = codec.encodeToJsonElement(assigned); api.patchUnion(assigned)
            val cold = api.watchUnion(); check(channels.isEmpty())
            repeat(2) {
                val events = cold.take(4).toList()
                check((events[0] as WatchUnionEventDto.Nullable).data == null)
                check((events[1] as WatchUnionEventDto.Carrier).data == NullCarrierUnionDto)
                check((events[2] as WatchUnionEventDto.Inner).data == inner)
                check((events[3] as WatchUnionEventDto.Holder).data == holder)
                check(channels.last().isClosedForWrite)
            }
            for (bad in listOf("inner", "null-inner")) {
                failure = bad; check(runCatching { cold.take(1).toList() }.isFailure)
                check(channels.last().isClosedForWrite)
            }
            failure = ""; check((cold.take(1).toList().single() as WatchUnionEventDto.Nullable).data == null)
            check(channels.last().isClosedForWrite)
        } finally { http.close(); scope.cancel(); writers.forEach { it.join() } }
        println("UnionNominalOwners Json defaults=$defaults nulls=$nulls actual-retained-union-owner-nullcarrier-nullable-false-recursion-Patch-HTTP-SSEclose-reuse-cold passed")
    }
}
