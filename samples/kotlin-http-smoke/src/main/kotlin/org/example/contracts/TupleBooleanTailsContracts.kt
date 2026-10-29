@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
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
import kotlinx.datetime.LocalDate
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import org.example.booleantails.api.DefaultApi
import org.example.booleantails.api.models.EventsEventDto
import org.example.booleantails.api.models.UpdateInputDto
import org.example.booleantails.models.ChoiceDto
import org.example.booleantails.models.ClosedDto
import org.example.booleantails.models.DatesDto
import org.example.booleantails.models.HolderDto
import org.example.booleantails.models.ItemDto
import org.example.booleantails.models.MaybeDto
import org.example.booleantails.models.MixedDto
import org.example.booleantails.models.OpenDto
import org.example.booleantails.models.Patch
import org.example.booleantails.models.`NumbersDto default`
import org.example.booleantails.models.`NumbersAliasDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BooleanTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers,
            response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers,
            response.version, body, response.callContext)
    }
}

private fun booleanTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> booleanTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic Boolean tuple body")
}

suspend fun tupleBooleanTailsContracts(checked: Boolean = false) = withTimeout(15000) {
    val closed: ClosedDto = listOf("a", "b", "extra")
    val open: OpenDto = listOf(JsonPrimitive(true), JsonNull, JsonPrimitive("tail"))
    val mixed: MixedDto = listOf(JsonNull, JsonPrimitive(false))
    val dates: DatesDto = listOf(LocalDate.parse("2026-10-27"))
    val maybe: MaybeDto = listOf("v", null)
    val value = HolderDto(closed = closed, open = open, mixed = mixed, dates = dates,
        maybe = maybe, enums = listOf(ChoiceDto.Unrecognized("unknown")), objects = listOf(ItemDto("item")))
    check(value.defaults == listOf(3))
    check(`NumbersDto default`() == listOf(1) && `NumbersAliasDto default`() == listOf(2))
    check(`NumbersDto default`() !== `NumbersDto default`())
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<ClosedDto>(codec.encodeToString(closed)) == closed)
        check(codec.decodeFromString<ClosedDto>("[]").isEmpty())
        check(codec.decodeFromString<ClosedDto>("[\"a\",\"b\",\"extra\"]") == closed) // Closed-tail length remains approximate.
        check(runCatching { codec.decodeFromString<ClosedDto>("[true]") }.isFailure)
        check(codec.decodeFromString<OpenDto>("[{},[],null,3,true]").size == 5) // Prefix positions remain approximate.
        check(codec.decodeFromString<MixedDto>("[null,false]") == mixed)
        check(codec.decodeFromString<MaybeDto>("null") == null)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = BooleanTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(booleanTupleBody(request.body)) == value)
                        respond(codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/open" -> {
                        check(codec.decodeFromString<List<JsonElement>>(booleanTupleBody(request.body)) == open)
                        respond(codec.encodeToString(open), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/closed" -> {
                        check(codec.decodeFromString<List<String>>(booleanTupleBody(request.body)) == closed)
                        respond(codec.encodeToString(closed), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        val body = codec.parseToJsonElement(booleanTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("rows" !in body)
                            1 -> check(body["rows"] == JsonNull)
                            2 -> check(body["rows"] == codec.parseToJsonElement("[\"a\",\"b\",\"extra\"]"))
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val data = if (bad) "{\"closed\":null}" else codec.encodeToString(value)
                        val frame = "event: message\r\ndata: $data\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic Boolean tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value && api.open(open) == open && api.closed(closed) == closed)
            if (checked) {
                val before = calls
                var callbacks = 0
                check(runCatching { api.open(emptyList()) { callbacks++ } }.isFailure)
                check(runCatching { api.open(listOf(JsonNull, JsonNull)) { callbacks++ } }.isFailure)
                val left = codec.parseToJsonElement("{\"a\":1,\"b\":2}")
                val right = codec.parseToJsonElement("{\"b\":2,\"a\":1}")
                check(runCatching { api.open(listOf(left, right)) { callbacks++ } }.isFailure)
                check(callbacks == 0 && calls == before)
                // Standalone aliases retain their inherent unchecked assignment
                // and codec behavior; only typed consumers enforce counts.
                check(codec.decodeFromString<OpenDto>("[]").isEmpty())
            }
            api.updateUpdate(UpdateInputDto(requiredRows = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(requiredRows = null, rows = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(requiredRows = closed, rows = Patch.Value(closed)))
            val flow = api.events()
            check(channels.isEmpty())
            check((flow.take(1).toList().single() as EventsEventDto.Message).data == value)
            check(channels.last().isClosedForRead)
            bad = true
            check(runCatching { flow.take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead)
            bad = false
            check(flow.take(1).toList().size == 1 && api.echo(value) == value)
        } finally {
            client.close()
            scope.cancel()
            writers.forEach { it.cancel() }
            channels.forEach { it.cancel(null) }
        }
    }
}
