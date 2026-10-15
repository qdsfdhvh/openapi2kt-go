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
import io.ktor.http.HttpHeaders
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.jsonunique.api.DefaultApi
import org.example.jsonunique.api.models.ApplyUniqueInputDto
import org.example.jsonunique.api.models.WatchUniqueEventDto
import org.example.jsonunique.models.HolderDto
import org.example.jsonunique.models.Patch
import org.example.jsonunique.models.RecordDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class JSONUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun jsonUniqueReject(label: String, action: () -> Unit) {
    try { action(); error("Missing JSON unique rejection: $label") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { (it is IllegalArgumentException || it is SerializationException) && it.message?.contains("uniqueItems") == true }) { "Wrong JSON uniqueness failure: $cause" }
    }
}
public suspend fun jsonUniqueContracts() {
    val parsed = Json
    val distinct = parsed.parseToJsonElement("[1,\"1\",true,\"true\",null,{},[],[1,2],[2,1],{\"a\":1},{\"b\":1}]") as JsonArray
    val duplicates = listOf("[1,1.0]", "[0,-0.0]", "[1e0,0.100e1]", "[null,null]", "[\"a\",\"\\u0061\"]", "[{\"a\":1,\"b\":2},{\"b\":2.0,\"a\":1e0}]", "[[1,2],[1.0,2e0]]")
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val record = RecordDto(distinct)
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(record)) == record)
        RecordDto(listOf(JsonPrimitive("é"), JsonPrimitive("e\u0301")))
        for (input in duplicates) {
            val values = codec.parseToJsonElement(input) as JsonArray
            jsonUniqueReject("constructor") { RecordDto(values) }
            jsonUniqueReject("nested") { RecordDto(emptyList(), nested = listOf(values)) }
            jsonUniqueReject("decoder") { codec.decodeFromString<RecordDto>("{\"values\":$input}") }
            jsonUniqueReject("Patch.Value") { ApplyUniqueInputDto(emptyList(), Patch.Value(values)) }
            jsonUniqueReject("transparent alias") { HolderDto(rows = values) }
        }
        val objectPair = listOf(JsonObject(mapOf("a" to JsonPrimitive(1), "b" to JsonPrimitive(2))), JsonObject(mapOf("b" to JsonPrimitive(2), "a" to JsonPrimitive(1))))
        jsonUniqueReject("JsonObject") { RecordDto(emptyList(), objects = objectPair) }
        ApplyUniqueInputDto(emptyList(), Patch.Absent)
        ApplyUniqueInputDto(emptyList(), Patch.Null)
        var deep: JsonElement = JsonPrimitive(1)
        repeat(48) { deep = JsonArray(listOf(deep)) }
        RecordDto(listOf(deep))
        deep = JsonArray(listOf(deep))
        jsonUniqueReject("depth") { RecordDto(listOf(deep)) }
        RecordDto((0 until 16384).map { JsonPrimitive(it) })
        jsonUniqueReject("work") { RecordDto((0 until 16385).map { JsonPrimitive(it) }) }
        jsonUniqueReject("exponent") { RecordDto(listOf(codec.parseToJsonElement("1e513"))) }
        jsonUniqueReject("precision") { RecordDto(listOf(codec.parseToJsonElement("1".repeat(257)))) }
        var bad = false
        var calls = 0
        val engine = JSONUniqueEngine(MockEngineConfig().apply { addHandler { outgoing ->
            calls++
            val body = if (bad) "{\"values\":[1,1.0]}" else "{\"values\":$distinct}"
            when (outgoing.url.encodedPath) {
                "/unique" -> respond(body, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/watch" -> respond("event: sample\ndata: $body\n\n", headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                else -> error("Unexpected synthetic JSON unique route")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.applyUnique(ApplyUniqueInputDto(emptyList())).values == distinct)
            check((api.watchUnique().toList().single() as WatchUniqueEventDto.Sample).data!!.values == distinct)
            bad = true
            try { api.applyUnique(ApplyUniqueInputDto(emptyList())); error("Missing HTTP uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            try { api.watchUnique().toList(); error("Missing SSE uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            check(calls == 4)
        } finally { client.close() }
        println("JSONUnique Json defaults=$defaults nulls=$nulls recursive-numeric-objectkeys-null-Patch-limits-HTTP-SSE passed")
    }
}
