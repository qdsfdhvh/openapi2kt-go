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
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CancellationException
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
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import org.example.tuplenative.TupleNativeClient
import org.example.tuplenative.api.models.PatchRowsInputDto
import org.example.tuplenative.api.models.WatchEventDto
import org.example.tuplenative.models.HolderDto
import org.example.tuplenative.models.Patch
import org.example.tuplenative.models.RowsDto
import org.example.tuplenative.models.NullableRowsDto
import org.example.tuplenative.models.NarrowRowsDto
import org.example.tuplenative.models.LongRowsDto
import org.example.tuplenative.models.FloatRowsDto
import org.example.tuplenative.models.DoubleRowsDto
import org.example.tuplenative.models.BoolRowsDto
import org.example.tuplenative.models.InternalRowsDto
import org.example.tuplenative.models.MixedRowsDto
import org.example.tuplenative.models.DifferentRowsDto
import org.example.tuplenative.models.TailRowsDto
import org.example.tuplenative.models.ScalarTailRowsDto
import org.example.tuplenative.models.WideDto
import org.example.tuplenative.models.`CodeDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class TupleNativeEnumStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun tupleNativeEnumBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> tupleNativeEnumBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic enum tuple body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun tupleNativeEnumReferencesContracts(checked: Boolean = true) = withTimeout(15000) {
    val rows: RowsDto = listOf(1,2)
    val maybe: NullableRowsDto = listOf(null,1,2)
    val narrow: NarrowRowsDto = rows
    val longs: LongRowsDto = listOf(9007199254740993L,9007199254740996L)
    val floats: FloatRowsDto = listOf(0.1f,-0.0f)
    val doubles: DoubleRowsDto = listOf(0.1,-0.0,2.5)
    val flags: BoolRowsDto = listOf(true,false)
    val internal: InternalRowsDto = listOf(3,4)
    val mixed: MixedRowsDto = listOf(JsonPrimitive(99),JsonNull,JsonPrimitive(true))
    val different: DifferentRowsDto = mixed
    val tail: TailRowsDto = rows
    val value = HolderDto(rows,maybe,longs,floats,doubles,flags,internal,mixed,tail)
    check(`CodeDto default`() == 1)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<RowsDto>("[1,2]") == rows)
        check(codec.encodeToString(rows) == "[1,2]")
        check(codec.decodeFromString<NullableRowsDto>(codec.encodeToString(maybe)) == maybe)
        check(codec.decodeFromString<NarrowRowsDto>(codec.encodeToString(narrow)) == narrow)
        check(codec.decodeFromString<LongRowsDto>(codec.encodeToString(longs)) == longs)
        check(codec.encodeToString(longs) == "[9007199254740993,9007199254740996]")
        check(codec.decodeFromString<FloatRowsDto>(codec.encodeToString(floats)) == floats)
        check(codec.decodeFromString<DoubleRowsDto>(codec.encodeToString(doubles)) == doubles)
        check(codec.decodeFromString<BoolRowsDto>(codec.encodeToString(flags)) == flags)
        check(codec.decodeFromString<InternalRowsDto>(codec.encodeToString(internal)) == internal)
        check(codec.decodeFromString<MixedRowsDto>("[99,null,true]") == mixed)
        check(codec.decodeFromString<DifferentRowsDto>("[99,null,true]") == different)
        check(codec.decodeFromString<TailRowsDto>("[1,2]") == tail)
        check(codec.decodeFromString<ScalarTailRowsDto>("[\"tail\"]") == listOf("tail"))
        check(codec.decodeFromString<RowsDto>("[]").isEmpty()) // tuple length remains approximate
        check(codec.decodeFromString<RowsDto>("[99]") == listOf(99)) // standalone scalar alias codec has no constructor barrier
        for (invalid in listOf("[null]","[true]","[\"bad\"]")) {
            check(runCatching { codec.decodeFromString<RowsDto>(invalid) }.isFailure)
        }
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        for (invalid in listOf(
            { HolderDto(rows = listOf(9)) },
            { HolderDto(rows, nullableRows = listOf(null,9)) },
            { HolderDto(rows, longs = listOf(9007199254740994L)) },
            { HolderDto(rows, floats = listOf(Float.NaN)) },
            { HolderDto(rows, doubles = listOf(Double.POSITIVE_INFINITY)) },
            { HolderDto(rows, internals = listOf(9)) },
            { HolderDto(rows, tail = listOf(9)) }
        )) check(runCatching(invalid).isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("{\"rows\":[9]}") }.isFailure)
        val wideFields = (0 until 240).associate { "v" + it.toString().padStart(3,'0') to codec.parseToJsonElement("[1,2]") }
        val wide = codec.decodeFromString<WideDto>(kotlinx.serialization.json.JsonObject(wideFields).toString())
        check(wide.v000 == rows && wide.v239 == rows)
        val invalidWide = wideFields.toMutableMap(); invalidWide["v239"] = codec.parseToJsonElement("[9]")
        check(runCatching { codec.decodeFromString<WideDto>(kotlinx.serialization.json.JsonObject(invalidWide).toString()) }.isFailure)
        val scope = CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var patchMode = 0
        var badResponse = false
        var badEvent = false
        var cancelled = false
        var requests = 0
        val cancellation = CancellationException("synthetic enum tuple cancellation")
        val engine = TupleNativeEnumStreamEngine(MockEngineConfig().apply {
            addHandler { req ->
                requests++
                if (cancelled) throw cancellation
                when(req.url.encodedPath) {
                    "/send" -> {
                        check(codec.decodeFromString<HolderDto>(tupleNativeEnumBody(req.body)) == value)
                        respond(if (badResponse) "[9]" else codec.encodeToString(rows),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {
                        val body = codec.parseToJsonElement(tupleNativeEnumBody(req.body)).jsonObject
                        when(patchMode) {
                            0 -> check("rows" !in body)
                            1 -> check(body["rows"] == JsonNull)
                            2 -> check(body["rows"] == codec.parseToJsonElement(codec.encodeToString(rows)))
                        }
                        respond(codec.encodeToString(rows),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val data = if (badEvent) "[9]" else codec.encodeToString(rows)
                        val frame = "event: sample\r\ndata: $data\r\n\r\n"
                        writers.add(scope.launch { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() })
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic enum tuple URL")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = TupleNativeClient(http,"http://example.test/",codec).tuples
            check(api.send(value) == rows)
            badResponse = true; check(runCatching { api.send(value) }.isFailure); badResponse = false
            if (checked) {
                val count = requests
                check(runCatching { api.send(HolderDto(rows = listOf(9))) }.isFailure && requests == count)
            }
            api.patchRows(PatchRowsInputDto(requiredNullable = null))
            patchMode = 1; api.patchRows(PatchRowsInputDto(rows = Patch.Null,requiredNullable = null))
            patchMode = 2; api.patchRows(PatchRowsInputDto(rows = Patch.Value(rows),requiredNullable = rows))
            val flow = api.watch(); check(channels.isEmpty())
            check((flow.take(1).toList().single() as WatchEventDto.Sample).data == rows)
            check(channels.last().isClosedForRead)
            badEvent = true; check(runCatching { flow.take(1).toList() }.isFailure); check(channels.last().isClosedForRead)
            badEvent = false; check(flow.take(1).toList().size == 1)
            cancelled = true; check(runCatching { api.send(value) }.exceptionOrNull() === cancellation); cancelled = false
            check(api.send(value) == rows)
            println("TupleNativeEnumReferences defaults=$defaults nulls=$nulls checked=$checked native5-exactLong-floating-alias-null-width240-Patch-HTTP-liveSSE-cancel-reuse passed")
        } finally { http.close(); scope.cancel(); writers.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
    }
}
