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

import kotlinx.serialization.serializer
import kotlinx.serialization.json.JsonObject
import org.example.tupleallof.TupleAllOfClient
import org.example.tupleallof.api.models.PatchRowsInputDto
import org.example.tupleallof.api.models.WatchEventDto
import org.example.tupleallof.models.RowsDto
import org.example.tupleallof.models.MaybeRowsDto
import org.example.tupleallof.models.MixedRowsDto
import org.example.tupleallof.models.TailRowsDto
import org.example.tupleallof.models.ScalarTailDto
import org.example.tupleallof.models.InternalRowsDto
import org.example.tupleallof.models.NodeRowsDto
import org.example.tupleallof.models.ItemDto
import org.example.tupleallof.models.HolderDto
import org.example.tupleallof.models.OwnerDto
import org.example.tupleallof.models.NodeDto
import org.example.tupleallof.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class TupleAllOfStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun tupleAllOfBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> tupleAllOfBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic DTO tuple body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun tupleAllOfReferencesContracts(checked: Boolean = false) = withTimeout(15000) {
    val first = ItemDto(value = "valid")
    check(first.count == 7 && first.mode?.serialName == "ready" && first.child?.label == "child")
    check(first.child !== ItemDto(value = "valid").child)
    val rows: RowsDto = listOf(first,ItemDto(value = "future",mode = ItemDto.Mode.Unrecognized("future")))
    val maybe: MaybeRowsDto = listOf(null,first)
    val mixed: MixedRowsDto = listOf(JsonPrimitive(3),JsonNull,JsonObject(mapOf("outside" to JsonPrimitive(true))))
    val tail: TailRowsDto = rows
    val internal: InternalRowsDto = listOf(OwnerDto.Node(label = "internal"))
    val nodes: NodeRowsDto = listOf(NodeDto(value = "root",next = NodeDto(value = "leaf")))
    val value = HolderDto(rows,maybe,mixed,tail,internal,nodes)
    check(serializer<RowsDto>().descriptor.getElementDescriptor(0).serialName == ItemDto.serializer().descriptor.serialName)
    check(serializer<MaybeRowsDto>().descriptor.getElementDescriptor(0).isNullable)
    check(ItemDto.Mode.Serializer.descriptor.serialName.endsWith(".models.ItemDto.Mode"))
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        check(codec.decodeFromString<RowsDto>(codec.encodeToString(rows)) == rows)
        check(codec.decodeFromString<RowsDto>("[{\"value\":\"valid\"}]").single() == first)
        val duplicateDefaults = codec.decodeFromString<RowsDto>("[{\"value\":\"valid\"},{\"value\":\"valid\"}]")
        check(duplicateDefaults[0].child !== duplicateDefaults[1].child)
        check(codec.decodeFromString<MaybeRowsDto>(codec.encodeToString(maybe)) == maybe)
        check(codec.decodeFromString<MaybeRowsDto>("[null]").single() == null)
        check(codec.decodeFromString<MixedRowsDto>(codec.encodeToString(mixed)) == mixed)
        check(codec.decodeFromString<TailRowsDto>(codec.encodeToString(tail)) == tail)
        check(codec.decodeFromString<ScalarTailDto>("[3]") == listOf(3))
        check(codec.decodeFromString<InternalRowsDto>(codec.encodeToString(internal)) == internal)
        check(codec.decodeFromString<NodeRowsDto>(codec.encodeToString(nodes)) == nodes)
        check(codec.decodeFromString<RowsDto>("[]").isEmpty()) // standalone list codecs do not enforce tuple length/counts
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        for (invalid in listOf("[null]","[3]","[{}]","[{\"value\":null}]")) check(runCatching { codec.decodeFromString<RowsDto>(invalid) }.isFailure)
        val extra = "[{\"value\":\"valid\",\"count\":7,\"mode\":\"ready\",\"child\":{\"label\":\"child\"},\"extra\":true}]"
        check(codec.decodeFromString<RowsDto>(extra).single().value == "valid") // allOf roots do not gain ordinary root counts
        if (checked) {
            check(runCatching { ItemDto(value = "x") }.isFailure)
            check(runCatching { ItemDto(value = "valid",count = -1) }.isFailure)
            check(runCatching { ItemDto.Child(label = "x") }.isFailure)
            check(runCatching { codec.decodeFromString<RowsDto>("[{\"value\":\"x\"}]") }.isFailure)
            check(runCatching { HolderDto(emptyList(),maybe,mixed,tail,internal,nodes) }.isFailure)
        }
        val scope = CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var patchMode = 0
        var bad = false
        var cancelled = false
        var requests = 0
        val cancellation = CancellationException("synthetic DTO tuple cancellation")
        val engine = TupleAllOfStreamEngine(MockEngineConfig().apply {
            addHandler { req ->
                requests++
                if (cancelled) throw cancellation
                when(req.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(tupleAllOfBody(req.body)) == value)
                        respond(codec.encodeToString(value),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/rows" -> {
                        check(codec.decodeFromString<RowsDto>(tupleAllOfBody(req.body)) == rows)
                        respond(if (bad) "[{\"value\":null}]" else codec.encodeToString(rows),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {
                        val body = codec.parseToJsonElement(tupleAllOfBody(req.body)).jsonObject
                        when(patchMode) {
                            0 -> check("values" !in body)
                            1 -> check(body["values"] == JsonNull)
                            2 -> check(body["values"] == codec.parseToJsonElement(codec.encodeToString(rows)))
                        }
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush = true);channels.add(channel)
                        val data = if (bad) "[{\"value\":null}]" else codec.encodeToString(rows)
                        val frame = "event: sample\r\ndata: $data\r\n\r\n"
                        writers.add(scope.launch { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() })
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic DTO tuple URL")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = TupleAllOfClient(http,"http://example.test/",codec).default
            check(api.echo(value) == value && api.rows(rows) == rows)
            bad = true;check(runCatching { api.rows(rows) }.isFailure);bad = false
            if (checked) {
                val count = requests
                check(runCatching { api.rows(emptyList()) }.isFailure && requests == count)
            }
            api.patchRows(PatchRowsInputDto(requiredNullable = null))
            patchMode = 1;api.patchRows(PatchRowsInputDto(values = Patch.Null,requiredNullable = null))
            patchMode = 2;api.patchRows(PatchRowsInputDto(values = Patch.Value(rows),requiredNullable = rows))
            val flow = api.watch();check(channels.isEmpty())
            check((flow.take(1).toList().single() as WatchEventDto.Sample).data == rows)
            check(channels.last().isClosedForRead)
            bad = true;check(runCatching { flow.take(1).toList() }.isFailure);check(channels.last().isClosedForRead);bad = false
            check(flow.take(1).toList().size == 1)
            cancelled = true;check(runCatching { api.rows(rows) }.exceptionOrNull() === cancellation);cancelled = false
            check(api.rows(rows) == rows)
            println("TupleAllOfReferences defaults=$defaults nulls=$nulls checked=$checked source-owner-defaults-checks-Patch-HTTP-liveSSE-cancel-reuse passed")
        } finally { http.close();scope.cancel();writers.forEach { it.cancel() };channels.forEach { it.cancel(null) } }
    }
}
