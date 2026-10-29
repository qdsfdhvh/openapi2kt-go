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
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import org.example.structuredtuples.api.DefaultApi
import org.example.structuredtuples.api.models.EventsEventDto
import org.example.structuredtuples.api.models.UpdateInputDto

import org.example.structuredtuples.models.HolderDto
import org.example.structuredtuples.models.RowsDto
import org.example.structuredtuples.models.NullableRowsDto
import org.example.structuredtuples.models.Patch


import org.example.structuredtuples.models.`RowsDto default`
import org.example.structuredtuples.models.`AliasDto default`
import org.example.structuredtuples.models.`RefRowsDto default`
import org.example.structuredtuples.models.`ArrayRowsDto default`
import org.example.structuredtuples.models.`NullableRowsDto default`
import org.example.structuredtuples.models.`TailDto default`
import org.example.structuredtuples.models.`MixedDto default`
import org.example.structuredtuples.models.`MaybeDto default`
import org.example.structuredtuples.models.`MetadataRowsDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StructuredTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun structuredTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> structuredTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline DTO tuple body")
}

suspend fun tupleStructuredConstContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val alternate = JsonObject(mapOf("other" to JsonPrimitive(2)))
        val fixed = JsonObject(mapOf("fixed" to JsonPrimitive(1)))
        val value = HolderDto(required = listOf(alternate), nested = listOf(mapOf("key" to listOf(alternate))))
        check(value.rows == listOf(fixed) && value.refs == listOf(fixed) && value.borrowed == null)
        check(value.rows !== HolderDto(required = value.required).rows)
        check(`RowsDto default`() == listOf(fixed))
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`().isEmpty())
        check(`RefRowsDto default`() == listOf(fixed))
        check(`NullableRowsDto default`() == listOf(null, fixed))
        check(`ArrayRowsDto default`() == listOf(codec.parseToJsonElement("[1,2]")))
        check(`MetadataRowsDto default`().single()["allOf"] == codec.parseToJsonElement("""[{"${'$'}ref":"#/data-only"}]"""))
        check(`MetadataRowsDto default`() !== `MetadataRowsDto default`())
        check(`TailDto default`() == listOf(7) && `MaybeDto default`() == null)
        check(`MixedDto default`() == codec.decodeFromString<List<JsonElement>>("""[{"${'$'}ref":"data"},null]"""))
        check(codec.decodeFromString<RowsDto>("[]").isEmpty())
        check(codec.decodeFromString<RowsDto>("""[{"other":2}]""").single() == alternate)
        check(codec.decodeFromString<NullableRowsDto>("[null]").single() == null)
        for (wrong in listOf("[0]", "[null]", "[[]]", "[\"object\"]")) {
            check(runCatching { codec.decodeFromString<RowsDto>(wrong) }.isFailure)
        }
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<HolderDto>("""{"required":[{"other":2}],"rows":null}""").rows == null)
        check(("rows" in codec.parseToJsonElement(codec.encodeToString(value)).jsonObject) == defaults)
        check(UpdateInputDto(required = null).optional == Patch.Absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = StructuredTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(structuredTupleBody(request.body)) == value)
                        respond(if (bad) """{"required":[1]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> respond(if (bad) "[{}]" else "[[9]]", headers = headersOf("Content-Type", "application/json"))
                    "/send" -> {
                        check(codec.decodeFromString<List<JsonObject>>(structuredTupleBody(request.body)) == listOf(alternate))
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/shape-a", "/shape-b" -> respond("""{"name":"hoisted"}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val data = codec.parseToJsonElement(structuredTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in data)
                            1 -> check(data["optional"] == JsonNull)
                            2 -> check(data["optional"] == codec.parseToJsonElement("""[{"other":2}]"""))
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
                    else -> error("Unexpected synthetic structured tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            check(api.inlineRows() == listOf(JsonArray(listOf(JsonPrimitive(9)))))
            api.send(listOf(alternate))
            check(api.getShapeA().name == "hoisted" && api.getShapeB().name == "hoisted")
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Value(listOf(alternate))))
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
