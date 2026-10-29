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
import kotlinx.serialization.json.jsonObject
import org.example.consttuples.api.DefaultApi
import org.example.consttuples.api.models.EventsEventDto
import org.example.consttuples.api.models.UpdateInputDto

import org.example.consttuples.models.HolderDto
import org.example.consttuples.models.RowsDto
import org.example.consttuples.models.NullableRowsDto
import org.example.consttuples.models.Patch

import org.example.consttuples.models.`RowsDto default`
import org.example.consttuples.models.`AliasDto default`
import org.example.consttuples.models.`RefRowsDto default`
import org.example.consttuples.models.`NullableRowsDto default`
import org.example.consttuples.models.`TailDto default`
import org.example.consttuples.models.`MixedDto default`
import org.example.consttuples.models.`MaybeDto default`
import org.example.consttuples.models.`NumbersDto default`
import org.example.consttuples.models.`FlagsDto default`
import org.example.consttuples.models.`LongsDto default`
import org.example.consttuples.models.`FloatsDto default`
import org.example.consttuples.models.`DoublesDto default`
import org.example.consttuples.models.`UuidsDto default`
import org.example.consttuples.models.`DatesDto default`
import org.example.consttuples.models.`InstantsDto default`
import org.example.consttuples.models.`BigintsDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ConstTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun constTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> constTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline DTO tuple body")
}

suspend fun tupleScalarConstContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = HolderDto(required = listOf("other"), nested = listOf(mapOf("key" to listOf("other"))))
        check(value.rows == listOf("fixed") && value.refs == listOf("fixed") && value.borrowed == null)
        check(value.rows !== HolderDto(required = value.required).rows)
        check(`RowsDto default`() == listOf("fixed", "fixed"))
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`().isEmpty())
        check(`RefRowsDto default`() == listOf("fixed"))
        check(`NullableRowsDto default`() == listOf(null, "fixed"))
        check(`NumbersDto default`() == listOf(7) && `FlagsDto default`() == listOf(true))
        check(`LongsDto default`() == listOf(7L) && `BigintsDto default`() == listOf(7L))
        check(`FloatsDto default`() == listOf(1.5f) && `DoublesDto default`() == listOf(1.5))
        check(`UuidsDto default`().single().toString() == "123e4567-e89b-12d3-a456-426614174000")
        check(`DatesDto default`().single().toString() == "2026-01-01")
        check(`InstantsDto default`().single().toString() == "2026-01-01T00:00:00Z")
        check(`TailDto default`() == listOf(7) && `MaybeDto default`() == null)
        check(`MixedDto default`() == codec.decodeFromString<List<JsonElement>>("""[{"${'$'}ref":"data"},null]"""))
        check(codec.decodeFromString<RowsDto>("[]").isEmpty())
        check(codec.decodeFromString<RowsDto>("[\"other\"]").single() == "other")
        check(codec.decodeFromString<NullableRowsDto>("[null]").single() == null)
        for (wrong in listOf("[{}]", "[[]]")) {
            check(runCatching { codec.decodeFromString<RowsDto>(wrong) }.isFailure)
        }
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<HolderDto>("""{"required":["other"],"rows":null}""").rows == null)
        check(("rows" in codec.parseToJsonElement(codec.encodeToString(value)).jsonObject) == defaults)
        check(UpdateInputDto(required = null).optional == Patch.Absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = ConstTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(constTupleBody(request.body)) == value)
                        respond(if (bad) """{"required":[{}]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> respond(if (bad) "[{}]" else "[\"other\"]", headers = headersOf("Content-Type", "application/json"))
                    "/send" -> {
                        check(codec.decodeFromString<List<Int>>(constTupleBody(request.body)) == listOf(8))
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/shape-a", "/shape-b" -> respond("""{"name":"hoisted"}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val data = codec.parseToJsonElement(constTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in data)
                            1 -> check(data["optional"] == JsonNull)
                            2 -> check(data["optional"] == codec.parseToJsonElement("[\"other\"]"))
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
                    else -> error("Unexpected synthetic const tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            check(api.inlineRows() == listOf("other"))
            api.send(listOf(8))
            check(api.getShapeA().name == "hoisted" && api.getShapeB().name == "hoisted")
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Value(listOf("other"))))
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
