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
import org.example.collectiontuples.api.DefaultApi
import org.example.collectiontuples.api.models.EventsEventDto
import org.example.collectiontuples.api.models.UpdateInputDto
import org.example.collectiontuples.models.HolderDto
import org.example.collectiontuples.models.RowsDto
import org.example.collectiontuples.models.EnumsDto
import org.example.collectiontuples.models.Patch
import org.example.collectiontuples.models.`RowsDto default`
import org.example.collectiontuples.models.`AliasDto default`
import org.example.collectiontuples.models.`MapsDto default`
import org.example.collectiontuples.models.`NullableRowsDto default`
import org.example.collectiontuples.models.`RefRowsDto default`
import org.example.collectiontuples.models.`EnumsDto default`
import org.example.collectiontuples.models.`ObjectsDto default`
import org.example.collectiontuples.models.`TailDto default`
import org.example.collectiontuples.models.`MixedDto default`
import org.example.collectiontuples.models.`MaybeDto default`
import org.example.collectiontuples.models.`NumbersDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class CollectionTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun collectionTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> collectionTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline DTO tuple body")
}

suspend fun tupleCollectionSourcesContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = HolderDto(required = listOf(listOf(3)), nested = listOf(mapOf("key" to listOf(listOf(4)))))
        check(value.rows == listOf(listOf(1)) && value.refs == listOf(listOf(2)) && value.borrowed == null)
        check(value.objects!!.single().single().name == "field")
        check(value.objects!!.single().single().state?.serialName == "ready")
        check(value.objects!!.single().single().child?.label == "inner")
        val another = HolderDto(required = value.required)
        check(value.rows !== another.rows && value.rows!!.single() !== another.rows!!.single())
        check(value.objects!!.single().single().child !== another.objects!!.single().single().child)
        check(`RowsDto default`() == listOf(listOf(1,2), emptyList<Int>()))
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`() == listOf(listOf(3)))
        check(`MapsDto default`() == listOf(mapOf("default" to 7)))
        check(`NullableRowsDto default`() == listOf(null, listOf(null, 1)))
        check(`RefRowsDto default`() == listOf(listOf(4)) && `NumbersDto default`() == listOf(9))
        check(`EnumsDto default`().single().single().serialName == "ready")
        check(`ObjectsDto default`().single().single().name == "root")
        check(`TailDto default`() == listOf(mapOf("x" to 1)))
        check(`MixedDto default`() == codec.decodeFromString<List<JsonElement>>("""[null,{"items":"data"}]"""))
        check(`MaybeDto default`() == null)
        val future = codec.decodeFromString<EnumsDto>("""[["future"]]""")
        check(future.single().single().serialName == "future")
        check(codec.encodeToString(future) == """[["future"]]""")
        check(codec.decodeFromString<RowsDto>("[]").isEmpty())
        check(runCatching { codec.decodeFromString<RowsDto>("[[null]]") }.isFailure)
        check(runCatching { codec.decodeFromString<RowsDto>("[{}]") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<HolderDto>("""{"required":[[3]],"rows":null}""").rows == null)
        check(("rows" in codec.parseToJsonElement(codec.encodeToString(value)).jsonObject) == defaults)
        check(UpdateInputDto(required = null).optional == Patch.Absent)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = CollectionTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(collectionTupleBody(request.body)) == value)
                        respond(if (bad) """{"required":[[null]]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> respond(if (bad) "[{}]" else "[[1,2],[]]", headers = headersOf("Content-Type", "application/json"))
                    "/send" -> {
                        check(codec.decodeFromString<List<Map<String,Int>>>(collectionTupleBody(request.body)) == listOf(mapOf("default" to 7)))
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/shape-a", "/shape-b" -> respond("""{"name":"hoisted"}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val body = codec.parseToJsonElement(collectionTupleBody(request.body)).jsonObject
                        when(patchMode) {
                            0 -> check("optional" !in body)
                            1 -> check(body["optional"] == JsonNull)
                            2 -> check(body["optional"] == codec.parseToJsonElement("[[5]]"))
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (bad) """{"required":[[null]]}""" else codec.encodeToString(value)
                        val frame = "event: message\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic collection tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client,"https://example.test/",codec)
            check(api.echo(value) == value)
            check(api.inlineRows() == listOf(listOf(1,2),emptyList<Int>()))
            api.send(listOf(mapOf("default" to 7)))
            check(api.shapeA().name == "hoisted" && api.shapeB().name == "hoisted")
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Value(listOf(listOf(5)))))
            val event = api.events().take(1).toList().single()
            check(event is EventsEventDto.Message && event.data == value)
            check(channels.last().isClosedForRead)
            if (checked) {
                val before = calls
                check(runCatching { api.echo(HolderDto(required = listOf(emptyList()))) }.isFailure)
                check(runCatching { api.echo(HolderDto(required = listOf(listOf(-1)))) }.isFailure)
                check(calls == before)
            } else {
                check(codec.decodeFromString<HolderDto>("""{"required":[[]]}""").required == listOf(emptyList<Int>()))
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
