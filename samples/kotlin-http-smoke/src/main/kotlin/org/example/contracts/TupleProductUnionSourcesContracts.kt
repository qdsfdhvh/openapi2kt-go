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
import org.example.producttuples.api.DefaultApi
import org.example.producttuples.api.models.EventsEventDto
import org.example.producttuples.api.models.UpdateInputDto
import org.example.producttuples.api.models.SendInputDtoInner
import org.example.producttuples.api.models.InlineRowsResponseDtoInner
import org.example.producttuples.models.HolderDto
import org.example.producttuples.models.RowsDto
import org.example.producttuples.models.RowsDtoInner
import org.example.producttuples.models.NullableRowsDto
import org.example.producttuples.models.CarrierRowsDto
import org.example.producttuples.models.ViewRowsDto
import org.example.producttuples.models.ProductDto
import org.example.producttuples.models.InternalRowsDto
import org.example.producttuples.models.OwnerDto
import org.example.producttuples.models.`InternalRowsDto default`
import org.example.producttuples.models.NullNullableProductDto
import org.example.producttuples.models.UnknownRowsDtoInner
import org.example.producttuples.models.RepeatedDto
import org.example.producttuples.models.Patch
import org.example.producttuples.models.`RowsDto default`
import org.example.producttuples.models.`AliasDto default`
import org.example.producttuples.models.`RefRowsDto default`
import org.example.producttuples.models.`NullableRowsDto default`
import org.example.producttuples.models.`CarrierRowsDto default`
import org.example.producttuples.models.`ViewRowsDto default`
import org.example.producttuples.models.`TailDto default`
import org.example.producttuples.models.`MixedDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ProductUnionTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun productUnionTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> productUnionTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic union tuple body")
}

suspend fun tupleProductUnionSourcesContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val wire = """[{"kind":"text","name":"ready","mode":"fast","fast":true,"common":"shared"}]"""
        val expectedWire = codec.parseToJsonElement(wire)
        val value = HolderDto(required = listOf(HolderDto.RequiredInner.Text("ready", true, "shared")))
        check(value.rows?.size == 1 && value.refs?.size == 1 && value.borrowed == null)
        check(value.rows !== HolderDto(required = value.required).rows)
        check(codec.parseToJsonElement(codec.encodeToString(`RowsDto default`())) == expectedWire)
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`().isEmpty())
        val internal = `InternalRowsDto default`()
        check(internal.single() is OwnerDto.Choice.Count4)
        check(codec.decodeFromString<InternalRowsDto>(codec.encodeToString(internal)) == internal)
        check(codec.parseToJsonElement(codec.encodeToString(internal)) == codec.parseToJsonElement("""[{"kind":"count","n":7,"mode":"slow","slow":false,"common":"internal"}]"""))
        check(codec.parseToJsonElement(codec.encodeToString(`RefRowsDto default`())) == expectedWire)
        check(`NullableRowsDto default`().first() == null)
        check(`CarrierRowsDto default`().first() === NullNullableProductDto)
        check(`ViewRowsDto default`().first() == null)
        check(codec.decodeFromString<CarrierRowsDto>("[null]").single() === NullNullableProductDto)
        check(codec.decodeFromString<ViewRowsDto>("[null]").single() == null)
        check(codec.decodeFromString<NullableRowsDto>("[null]").single() == null)
        val variants = listOf<RowsDtoInner>(RowsDtoInner.Text("one", true, "shared"), RowsDtoInner.Text2("two", false, "shared"), RowsDtoInner.Count(3, true, "shared"), RowsDtoInner.Count4(4, false, "shared"))
        for (variant in variants) check(codec.decodeFromString<RowsDto>(codec.encodeToString<List<RowsDtoInner>>(listOf(variant))).single() == variant)
        val unknownWire = """[{"kind":"future","mode":"future","common":"shared","data":[null,1]}]"""
        val unknown = codec.decodeFromString<RowsDto>(unknownWire)
        check(unknown.single() is UnknownRowsDtoInner)
        check(codec.parseToJsonElement(codec.encodeToString(unknown)) == codec.parseToJsonElement(unknownWire))
        check(codec.decodeFromString<RepeatedDto>(wire).single() is ProductDto.Text)
        check(`TailDto default`() == listOf(7))
        check(`MixedDto default`() == codec.decodeFromString<List<JsonElement>>("""[{"data":1},null]"""))
        for (wrong in listOf("[1]", "[null]", "[[]]", """[{"kind":"text","mode":"fast","name":"missing"}]""")) {
            check(runCatching { codec.decodeFromString<RowsDto>(wrong) }.isFailure)
        }
        check(runCatching { codec.decodeFromString<HolderDto>("{}")} .isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(("rows" in codec.parseToJsonElement(codec.encodeToString(value)).jsonObject) == defaults)
        check(UpdateInputDto(required = null).optional == Patch.Absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = ProductUnionTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(productUnionTupleBody(request.body)) == value)
                        respond(if (bad) """{"required":[1]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> respond(if (bad) "[1]" else wire, headers = headersOf("Content-Type", "application/json"))
                    "/send" -> {
                        check(codec.parseToJsonElement(productUnionTupleBody(request.body)) == expectedWire)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/shape-a", "/shape-b" -> respond("""{"name":"hoisted"}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val data = codec.parseToJsonElement(productUnionTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in data)
                            1 -> check(data["optional"] == JsonNull)
                            2 -> check(data["optional"] == expectedWire)
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (bad) """{"required":[1]}""" else codec.encodeToString(value)
                        val frame = "event: message\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic union tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            check(api.inlineRows().single() is InlineRowsResponseDtoInner.Text)
            api.send(listOf(SendInputDtoInner.Text("ready", true, "shared")))
            check(api.getShapeA().name == "hoisted" && api.getShapeB().name == "hoisted")
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Value(listOf(UpdateInputDto.OptionalInner.Text("ready", true, "shared")))))
            val event = api.events().take(1).toList().single()
            check(event is EventsEventDto.Message && event.data == value)
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
