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
import org.example.tupleenums.TupleEnumsApiClient
import org.example.tupleenums.api.models.PatchRowsInputDto
import org.example.tupleenums.api.models.WatchEventDto
import org.example.tupleenums.models.ChoiceDto
import org.example.tupleenums.models.HolderDto
import org.example.tupleenums.models.InternalDto
import org.example.tupleenums.models.MaybeChoiceDto
import org.example.tupleenums.models.MaybeRowsDto
import org.example.tupleenums.models.MixedDto
import org.example.tupleenums.models.OwnerDto
import org.example.tupleenums.models.Patch
import org.example.tupleenums.models.RowsDto
import org.example.tupleenums.models.ScalarTailDto
import org.example.tupleenums.models.TailDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class TupleEnumStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun tupleEnumBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> tupleEnumBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic enum tuple body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun tupleEnumReferencesContracts(checked: Boolean = false) = withTimeout(15000) {
    val rows: RowsDto = listOf(ChoiceDto.READY,ChoiceDto.Unrecognized("future"))
    val maybe: MaybeRowsDto = listOf(null,MaybeChoiceDto.fromSerialName("null"),MaybeChoiceDto.Unrecognized("future"))
    val mixed: MixedDto = listOf(JsonPrimitive(3),JsonNull,JsonPrimitive(true))
    val tail: TailDto = rows
    val internal: InternalDto = listOf(OwnerDto.Mode.ON,OwnerDto.Mode.Unrecognized("future"))
    val value = HolderDto(rows,maybe,mixed,tail,internal)
    check(ChoiceDto.Serializer.descriptor.serialName.endsWith(".models.ChoiceDto"))
    check(OwnerDto.Mode.Serializer.descriptor.serialName.endsWith(".models.OwnerDto.Mode"))
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<RowsDto>("[\"ready\",\"future\"]") == rows)
        check(codec.encodeToString(rows) == "[\"ready\",\"future\"]")
        check(codec.decodeFromString<MaybeRowsDto>(codec.encodeToString(maybe)) == maybe)
        check(codec.encodeToString(maybe) == "[null,\"null\",\"future\"]")
        check(codec.decodeFromString<MixedDto>("[3,null,true]") == mixed) // no positional enum membership
        check(codec.decodeFromString<TailDto>("[\"ready\",\"future\"]") == tail) // items wins
        check(codec.decodeFromString<ScalarTailDto>("[3]") == listOf(3)) // prefix enum still validates at generation
        check(codec.decodeFromString<InternalDto>(codec.encodeToString(internal)) == internal)
        check(codec.decodeFromString<RowsDto>("[]").isEmpty()) // no tuple length enforcement on alias codecs
        for (invalid in listOf("[null]","[3]","[\"ready\",true]")) {
            check(runCatching { codec.decodeFromString<RowsDto>(invalid) }.isFailure)
        }
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        if (checked) {
            check(runCatching { HolderDto(emptyList(),maybe,mixed,tail,internal) }.isFailure)
            val invalid = codec.parseToJsonElement(codec.encodeToString(value)).jsonObject.toMutableMap()
            invalid["rows"] = codec.parseToJsonElement("[]")
            check(runCatching { codec.decodeFromString<HolderDto>(kotlinx.serialization.json.JsonObject(invalid).toString()) }.isFailure)
        }
        val scope = CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var patchMode = 0
        var badResponse = false
        var badEvent = false
        var cancelled = false
        var requests = 0
        val cancellation = CancellationException("synthetic enum tuple cancellation")
        val engine = TupleEnumStreamEngine(MockEngineConfig().apply {
            addHandler { req ->
                requests++
                if (cancelled) throw cancellation
                when(req.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(tupleEnumBody(req.body)) == value)
                        respond(codec.encodeToString(value),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/rows" -> {
                        check(codec.decodeFromString<RowsDto>(tupleEnumBody(req.body)) == rows)
                        respond(if (badResponse) "[3]" else codec.encodeToString(rows),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {
                        val body = codec.parseToJsonElement(tupleEnumBody(req.body)).jsonObject
                        when(patchMode) {
                            0 -> check("values" !in body)
                            1 -> check(body["values"] == JsonNull)
                            2 -> check(body["values"] == codec.parseToJsonElement(codec.encodeToString(rows)))
                        }
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val data = if (badEvent) "[3]" else codec.encodeToString(rows)
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
            val api = TupleEnumsApiClient(http,"http://example.test/",codec).default
            check(api.echo(value) == value && api.rows(rows) == rows)
            badResponse = true; check(runCatching { api.rows(rows) }.isFailure); badResponse = false
            if (checked) {
                val count = requests
                check(runCatching { api.rows(emptyList()) }.isFailure && requests == count)
            }
            api.patchRows(PatchRowsInputDto(requiredNullable = null))
            patchMode = 1; api.patchRows(PatchRowsInputDto(values = Patch.Null,requiredNullable = null))
            patchMode = 2; api.patchRows(PatchRowsInputDto(values = Patch.Value(rows),requiredNullable = rows))
            val flow = api.watch(); check(channels.isEmpty())
            check((flow.take(1).toList().single() as WatchEventDto.Sample).data == rows)
            check(channels.last().isClosedForRead)
            badEvent = true; check(runCatching { flow.take(1).toList() }.isFailure); check(channels.last().isClosedForRead)
            badEvent = false; check(flow.take(1).toList().size == 1)
            cancelled = true; check(runCatching { api.rows(rows) }.exceptionOrNull() === cancellation); cancelled = false
            check(api.rows(rows) == rows)
            println("TupleEnumReferences defaults=$defaults nulls=$nulls checked=$checked owner-openenum-null-Patch-HTTP-liveSSE-cancel-reuse passed")
        } finally { http.close(); scope.cancel(); writers.forEach { it.cancel() }; channels.forEach { it.cancel(null) } }
    }
}
