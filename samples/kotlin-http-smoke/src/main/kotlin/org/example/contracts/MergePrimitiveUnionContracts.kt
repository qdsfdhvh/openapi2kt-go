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
import org.example.unionmerge.UnionMergeApiClient
import org.example.unionmerge.api.models.WatchEventDto
import org.example.unionmerge.models.ADto
import org.example.unionmerge.models.BooleanADto
import org.example.unionmerge.models.BooleanZDto
import org.example.unionmerge.models.HolderDto
import org.example.unionmerge.models.IntADto
import org.example.unionmerge.models.IntZDto
import org.example.unionmerge.models.NullableADto
import org.example.unionmerge.models.NullableZDto
import org.example.unionmerge.models.NullNullableADto
import org.example.unionmerge.models.NullNullableZDto
import org.example.unionmerge.models.NumberADto
import org.example.unionmerge.models.NumberZDto
import org.example.unionmerge.models.StringADto
import org.example.unionmerge.models.StringZDto
import org.example.unionmerge.models.ZDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class MergeUnionStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun mergeUnionBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> mergeUnionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic union merge body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun mergePrimitiveUnionContracts() = withTimeout(15000) {
    val oldString: ZDto = StringZDto("future")
    val canonical: ADto = oldString
    check(canonical == StringADto("future"))
    check(StringZDto::class == StringADto::class)
    check(StringZDto.serializer().descriptor.serialName == "org.example.unionmerge.models.StringADto")
    check(ZDto.serializer().descriptor.serialName == ADto.serializer().descriptor.serialName)
    val oldNull: NullNullableZDto = NullNullableADto
    check(oldNull === NullNullableZDto && oldNull === NullNullableADto)
    check(NullableZDto.serializer().descriptor.serialName == NullableADto.serializer().descriptor.serialName)
    val values: List<ZDto> = listOf(oldString,IntZDto(Long.MIN_VALUE),NumberZDto(3.25),BooleanZDto(false))
    val expected: List<ADto> = listOf(StringADto("future"),IntADto(Long.MIN_VALUE),NumberADto(3.25),BooleanADto(false))
    check(values == expected)
    val value = HolderDto(canonical,IntZDto(Long.MAX_VALUE),oldNull,values)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        for ((index,item) in values.withIndex()) {
            val encoded = codec.encodeToString<ZDto>(item)
            check(codec.decodeFromString<ZDto>(encoded) == expected[index])
            check(codec.decodeFromString<ADto>(encoded) == expected[index])
        }
        check(codec.decodeFromString<NullableZDto>("null") === NullNullableADto)
        check(codec.encodeToString<NullableZDto>(oldNull) == "null")
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        for (invalid in listOf("{}","[]","null")) check(runCatching { codec.decodeFromString<ZDto>(invalid) }.isFailure)
        check(runCatching { codec.decodeFromString<NullNullableZDto>("3") }.isFailure)
        val scope = CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var bad = false
        var cancelled = false
        val cancellation = CancellationException("synthetic union merge cancellation")
        val paths = mutableListOf<String>()
        val engine = MergeUnionStreamEngine(MockEngineConfig().apply {
            addHandler { req ->
                if (cancelled) throw cancellation
                paths.add(req.url.encodedPath)
                when(req.url.encodedPath) {
                    "/a", "/b" -> {
                        check(codec.decodeFromString<ADto>(mergeUnionBody(req.body)) == canonical)
                        respond(if (bad) "[]" else codec.encodeToString<ADto>(canonical),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(mergeUnionBody(req.body)) == value)
                        respond(codec.encodeToString(value),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/natural" -> respond("",HttpStatusCode.NoContent)
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush = true);channels.add(channel)
                        val data = if (bad) "[]" else codec.encodeToString<ZDto>(oldString)
                        val frame = "event: sample\r\ndata: $data\r\n\r\n"
                        writers.add(scope.launch { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() })
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic union merge URL")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = UnionMergeApiClient(http,"http://example.test/",codec).default
            check(api.submit(canonical) == canonical && api.submit3(oldString) == canonical)
            api.submit2(); check(paths.take(3) == listOf("/a","/b","/natural"))
            check(api.echo(value) == value)
            bad = true; check(runCatching { api.submit3(oldString) }.isFailure);bad = false
            val flow = api.watch();check(channels.isEmpty())
            check((flow.take(1).toList().single() as WatchEventDto.Sample).data == canonical)
            check(channels.last().isClosedForRead)
            bad = true;check(runCatching { flow.take(1).toList() }.isFailure);check(channels.last().isClosedForRead);bad = false
            check(flow.take(1).toList().size == 1)
            cancelled = true;check(runCatching { api.submit3(oldString) }.exceptionOrNull() === cancellation);cancelled = false
            check(api.submit3(oldString) == canonical)
            println("MergePrimitiveUnion defaults=$defaults nulls=$nulls root-wrapper-null-alias-codec-HTTP-overload-SSE-cancel-reuse passed")
        } finally { http.close(); scope.cancel(); writers.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
    }
}
