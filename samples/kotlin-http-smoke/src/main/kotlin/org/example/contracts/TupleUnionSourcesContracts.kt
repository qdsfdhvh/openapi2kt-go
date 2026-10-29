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
import org.example.uniontuples.api.DefaultApi
import org.example.uniontuples.api.models.EventsEventDto
import org.example.uniontuples.api.models.UpdateInputDto
import org.example.uniontuples.api.models.StringSendInputDtoInner
import org.example.uniontuples.api.models.IntInlineRowsResponseDtoInner
import org.example.uniontuples.models.HolderDto
import org.example.uniontuples.models.RowsDto
import org.example.uniontuples.models.IntRowsDtoInner
import org.example.uniontuples.models.NumberRowsDtoInner
import org.example.uniontuples.models.NullableRowsDto
import org.example.uniontuples.models.ObjectRowsDto
import org.example.uniontuples.models.UnknownObjectRowsDtoInner
import org.example.uniontuples.models.EnumRowsDto
import org.example.uniontuples.models.RepeatedDto
import org.example.uniontuples.models.ClosedRowsDto
import org.example.uniontuples.models.`ClosedRowsDto default`
import org.example.uniontuples.models.Patch
import org.example.uniontuples.models.`RowsDto default`
import org.example.uniontuples.models.`AliasDto default`
import org.example.uniontuples.models.`RefRowsDto default`
import org.example.uniontuples.models.`ObjectRowsDto default`
import org.example.uniontuples.models.`RefObjectRowsDto default`
import org.example.uniontuples.models.`NullableRowsDto default`
import org.example.uniontuples.models.`FoldRowsDto default`
import org.example.uniontuples.models.`EnumRowsDto default`
import org.example.uniontuples.models.`TailDto default`
import org.example.uniontuples.models.`MixedDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class UnionTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun unionTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> unionTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic union tuple body")
}

suspend fun tupleUnionSourcesContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = HolderDto(required = listOf(HolderDto.StringRequiredInner("ready")))
        check(value.rows?.size == 4 && value.refs?.size == 1 && value.borrowed == null)
        check(value.rows !== HolderDto(required = value.required).rows)
        check(codec.encodeToString(`RowsDto default`()) == "[\"ready\",1,1.5,true]")
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`().isEmpty())
        check(codec.encodeToString(`RefRowsDto default`()) == "[\"ready\",1]")
        check(codec.parseToJsonElement(codec.encodeToString(`ObjectRowsDto default`())) == codec.parseToJsonElement("""[{"kind":"text","name":"ready"}]"""))
        check(codec.parseToJsonElement(codec.encodeToString(`RefObjectRowsDto default`())) == codec.parseToJsonElement("""[{"kind":"count","n":7}]"""))
        check(`NullableRowsDto default`().first() == null)
        check(`FoldRowsDto default`() == listOf(7))
        check(codec.parseToJsonElement(codec.encodeToString(`ClosedRowsDto default`())) == codec.parseToJsonElement("""[{"type":"Variant1","value":"ready"}]"""))
        check(codec.decodeFromString<ClosedRowsDto>(codec.encodeToString(`ClosedRowsDto default`())) == `ClosedRowsDto default`())
        check(`EnumRowsDto default`().single().serialName == "ready")
        check(codec.decodeFromString<EnumRowsDto>("[\"other\"]").single().serialName == "other")
        check(codec.decodeFromString<RowsDto>("[1]").single() is IntRowsDtoInner)
        check(codec.decodeFromString<RowsDto>("[1.0]").single() is NumberRowsDtoInner)
        check(codec.decodeFromString<NullableRowsDto>("[null]").single() == null)
        check(codec.decodeFromString<ObjectRowsDto>("""[{"kind":"unknown","data":1}]""").single() is UnknownObjectRowsDtoInner)
        check(codec.decodeFromString<RepeatedDto>("[1,\"ready\"]").size == 2)
        check(`TailDto default`() == listOf(7))
        check(`MixedDto default`() == codec.decodeFromString<List<JsonElement>>("""[{"data":1},null]"""))
        for (wrong in listOf("[{}]", "[null]", "[[]]")) {
            check(runCatching { codec.decodeFromString<RowsDto>(wrong) }.isFailure)
        }
        check(runCatching { codec.decodeFromString<HolderDto>("{}")} .isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<HolderDto>("""{"required":["ready"],"rows":null}""").rows == null)
        check(("rows" in codec.parseToJsonElement(codec.encodeToString(value)).jsonObject) == defaults)
        check(UpdateInputDto(required = null).optional == Patch.Absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = UnionTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(unionTupleBody(request.body)) == value)
                        respond(if (bad) """{"required":[{}]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> respond(if (bad) "[{}]" else "[9]", headers = headersOf("Content-Type", "application/json"))
                    "/send" -> {
                        check(codec.parseToJsonElement(unionTupleBody(request.body)) == codec.parseToJsonElement("[\"sent\"]"))
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/shape-a", "/shape-b" -> respond("""{"name":"hoisted"}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val data = codec.parseToJsonElement(unionTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in data)
                            1 -> check(data["optional"] == JsonNull)
                            2 -> check(data["optional"] == codec.parseToJsonElement("""[{"raw":"request projection"}]"""))
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (bad) """{"required":[{}]}""" else codec.encodeToString(value)
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
            check((api.inlineRows().single() as IntInlineRowsResponseDtoInner).value == 9L)
            api.send(listOf(StringSendInputDtoInner("sent")))
            check(api.getShapeA().name == "hoisted" && api.getShapeB().name == "hoisted")
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Value(listOf(codec.parseToJsonElement("""{"raw":"request projection"}""")))))
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
