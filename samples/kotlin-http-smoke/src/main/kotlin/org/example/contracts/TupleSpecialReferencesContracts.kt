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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.jsonObject
import org.example.specialtuples.api.DefaultApi
import org.example.specialtuples.api.models.EventsEventDto
import org.example.specialtuples.api.models.UpdateInputDto
import org.example.specialtuples.models.HolderDto
import org.example.specialtuples.models.SamplesDto
import org.example.specialtuples.models.RowsDto
import org.example.specialtuples.models.BytesRowsDto
import org.example.specialtuples.models.BinaryRowsDto
import org.example.specialtuples.models.TokenDto
import org.example.specialtuples.models.CountDto
import org.example.specialtuples.models.MoneyRowsDto
import org.example.specialtuples.models.NativeLongRowsDto
import org.example.specialtuples.models.NativeBytesRowsDto
import org.example.specialtuples.models.NativeTextRowsDto
import org.example.specialtuples.models.DateNativeRowsDto
import org.example.specialtuples.models.RawRowsDto
import org.example.specialtuples.models.OpenRowsDto
import org.example.specialtuples.models.NarrowRowsDto
import org.example.specialtuples.models.NullableRowsDto
import org.example.specialtuples.models.Patch
import org.example.specialtuples.models.`RowsDto default`
import org.example.specialtuples.models.`AliasDto default`
import org.example.specialtuples.models.`NullableRowsDto default`
import org.example.specialtuples.models.`NarrowRowsDto default`
import org.example.specialtuples.models.`BytesRowsDto default`
import org.example.specialtuples.models.`BinaryRowsDto default`
import org.example.specialtuples.models.`TokenRowsDto default`
import org.example.specialtuples.models.`CountRowsDto default`
import org.example.specialtuples.models.`MoneyRowsDto default`
import org.example.specialtuples.models.`NullableMoneyRowsDto default`
import org.example.specialtuples.models.`IndependentBytesDto default`
import org.example.specialtuples.models.`NativeLongRowsDto default`
import org.example.specialtuples.models.`NativeBytesRowsDto default`
import org.example.specialtuples.models.`NativeTextRowsDto default`
import org.example.specialtuples.models.`DateNativeRowsDto default`
import org.example.specialtuples.models.`RawRowsDto default`
import org.example.specialtuples.models.`OpenRowsDto default`
import org.example.specialtuples.models.`InlineObjectsDto default`
import org.example.specialtuples.models.`TailDto default`
import org.example.specialtuples.models.`MixedDto default`
import org.example.specialtuples.api.models.SampleEventsEventDto
import org.example.scalarvalues.Money
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonObject
import kotlin.time.Instant

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class SpecialTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun specialTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> specialTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic cover tuple body")
}

suspend fun tupleSpecialReferencesContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val wire = "[9223372036854775807]"
        val expectedWire = codec.parseToJsonElement(wire)
        val first = Long.MAX_VALUE
        val value = HolderDto(required = listOf(first), direct = Long.MAX_VALUE, label = "值")
        check(value.rows == `RowsDto default`() && value.borrowed == null)
        check(value.rows !== HolderDto(required = value.required).rows)
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`().isEmpty())
        check(codec.decodeFromString<RowsDto>(codec.encodeToString(`RowsDto default`())) == `RowsDto default`())
        check(`NullableRowsDto default`().first() == null && `NarrowRowsDto default`() == listOf(7L))
        check(codec.decodeFromString<NullableRowsDto>("[null]").single() == null)
        check(runCatching { codec.decodeFromString<NarrowRowsDto>("[null]") }.isFailure)
        val holderWire = codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(holderWire["direct"] == JsonPrimitive(Long.MAX_VALUE.toString()))
        check(holderWire["required"] == expectedWire)
        val bytes = byteArrayOf(-128, 0, 127)
        check(`BytesRowsDto default`().first().contentEquals(bytes))
        check(`IndependentBytesDto default`().single().isEmpty())
        check(codec.decodeFromString<BytesRowsDto>(codec.encodeToString(`BytesRowsDto default`())).first().contentEquals(bytes))
        check(codec.decodeFromString<BinaryRowsDto>(codec.encodeToString(`BinaryRowsDto default`())).first().contentEquals(byteArrayOf(1)))
        check(`TokenRowsDto default`().single() == TokenDto("ready"))
        check(`CountRowsDto default`().last() == CountDto(Long.MAX_VALUE))
        if (checked) check(runCatching { TokenDto("x") }.isFailure) else check(TokenDto("x").rawValue == "x")
        check(`MoneyRowsDto default`().isEmpty() && `NullableMoneyRowsDto default`().single() == null)
        val provided = listOf(Money("12.50"))
        check(codec.encodeToString<MoneyRowsDto>(provided) == """["12.50"]""")
        check(codec.decodeFromString<MoneyRowsDto>(codec.encodeToString<MoneyRowsDto>(provided)) == provided)
        check(`NativeLongRowsDto default`() == listOf(1L) && `NativeTextRowsDto default`() == listOf("ready"))
        check(codec.decodeFromString<NativeLongRowsDto>("[9223372036854775807]").single() == Long.MAX_VALUE)
        check(`NativeBytesRowsDto default`().single().contentEquals(bytes))
        check(codec.decodeFromString<NativeBytesRowsDto>(codec.encodeToString(`NativeBytesRowsDto default`())).single().contentEquals(bytes))
        check(codec.decodeFromString<NativeTextRowsDto>("""["ready"]""").single() == "ready")
        check(`DateNativeRowsDto default`().single() == Instant.parse("2026-01-02T03:04:05Z"))
        check(codec.decodeFromString<DateNativeRowsDto>(codec.encodeToString(`DateNativeRowsDto default`())) == `DateNativeRowsDto default`())
        check(`RawRowsDto default`().first() === JsonNull)
        check(codec.decodeFromString<RawRowsDto>(codec.encodeToString(`RawRowsDto default`())) == `RawRowsDto default`())
        check(codec.decodeFromString<OpenRowsDto>(codec.encodeToString(`OpenRowsDto default`())) == `OpenRowsDto default`())
        check(`InlineObjectsDto default`().single().containsKey("description"))
        check(runCatching { codec.decodeFromString<OpenRowsDto>("[1]") }.isFailure)
        val samples = SamplesDto()
        if (checked) {
            check(runCatching { SamplesDto(`open` = listOf(JsonObject(emptyMap()))) }.isFailure)
            check(runCatching { codec.decodeFromString<SamplesDto>("""{"open":[{}]}""") }.isFailure)
        } else {
            check(checkNotNull(SamplesDto(`open` = listOf(JsonObject(emptyMap()))).`open`).single().isEmpty())
        }
        check(checkNotNull(samples.kotlin).single() == TokenDto("ready") && checkNotNull(samples.kotlinx).isEmpty())
        check(checkNotNull(samples.bytes).first().contentEquals(bytes) && checkNotNull(samples.token).single() == TokenDto("ready"))
        check(codec.parseToJsonElement(codec.encodeToString(codec.decodeFromString<SamplesDto>(codec.encodeToString(samples)))) == codec.parseToJsonElement(codec.encodeToString(samples)))
        check(`TailDto default`() == listOf(7))
        check(`MixedDto default`() == codec.decodeFromString<List<JsonElement>>("""[{"data":1},null]"""))
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(("rows" in holderWire) == defaults)
        check(UpdateInputDto(required = null).optional == Patch.Absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = SpecialTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(specialTupleBody(request.body)) == value)
                        respond(if (bad) "{}" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> respond(if (bad) "[" else wire, headers = headersOf("Content-Type", "application/json"))
                    "/send" -> {
                        check(codec.parseToJsonElement(specialTupleBody(request.body)) == expectedWire)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/shape-a", "/shape-b" -> respond("""{"name":"hoisted"}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val data = codec.parseToJsonElement(specialTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in data)
                            1 -> check(data["optional"] == JsonNull)
                            2 -> check(data["optional"] == expectedWire)
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/samples" -> {
                        check(codec.parseToJsonElement(specialTupleBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(samples)))
                        respond(if (bad) """{"bytes":["base64"]}""" else codec.encodeToString(samples), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/events", "/sample-events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (request.url.encodedPath == "/sample-events") { if (bad) """{"bytes":["base64"]}""" else codec.encodeToString(samples) } else { if (bad) "{}" else codec.encodeToString(value) }
                        val frame = "event: message\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic cover tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            check(codec.parseToJsonElement(codec.encodeToString(api.echoSamples(samples))) == codec.parseToJsonElement(codec.encodeToString(samples)))
            check(api.inlineRows().single() == first)
            api.send(listOf(first))
            check(api.getShapeA().name == "hoisted" && api.getShapeB().name == "hoisted")
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Value(listOf(first))))
            val event = api.events().take(1).toList().single()
            check(event is EventsEventDto.Message && event.data == value)
            check(channels.last().isClosedForRead)
            val sampleEvent = api.sampleEvents().take(1).toList().single()
            check(sampleEvent is SampleEventsEventDto.Message && checkNotNull(sampleEvent.data.bytes).first().contentEquals(bytes))
            check(channels.last().isClosedForRead)
            if (checked) {
                val before = calls
                check(runCatching { api.echo(HolderDto(required = emptyList())) }.isFailure)
                check(calls == before)
            } else {
                check(codec.decodeFromString<HolderDto>("""{"required":[]}""").required.isEmpty())
            }
            bad = true
            check(runCatching { api.echo(value) }.isFailure)
            check(runCatching { api.inlineRows() }.isFailure)
            check(runCatching { api.echoSamples(samples) }.isFailure)
            check(runCatching { api.sampleEvents().take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead)
            check(runCatching { api.events().take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead)
            bad = false
            check(api.echo(value) == value && api.events().take(1).toList().size == 1)
        } finally {
            client.close()
            writers.forEach { it.cancel() }
            channels.forEach { it.cancel(null) }
            scope.cancel()
        }
    }
}
