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
import io.ktor.http.content.TextContent
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
import org.example.jsonunique.models.CoveredHolderDto
import org.example.jsonunique.models.ComposedHolderDto
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
        val referenced = RecordDto(emptyList(), referenced = distinct, raw = listOf(null, JsonPrimitive(1)))
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(referenced)) == referenced)
        val covered = RecordDto(emptyList(), covered = distinct, selfCovered = distinct, coveredMap = mapOf("items" to listOf(null, JsonPrimitive(1))))
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(covered)) == covered)
        val composed = RecordDto(emptyList(), composed = distinct, composedMap = mapOf("items" to listOf(null, JsonPrimitive(1))))
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(composed)) == composed)
        jsonUniqueReject("composed null and JsonNull") { ComposedHolderDto(rows = listOf(null, JsonNull)) }
        jsonUniqueReject("Kotlin null and JsonNull") { RecordDto(emptyList(), raw = listOf(null, JsonNull)) }
        jsonUniqueReject("nullable cover null and JsonNull") { CoveredHolderDto(rows = listOf(null, JsonNull)) }
        RecordDto(listOf(JsonPrimitive("é"), JsonPrimitive("e\u0301")))
        for (input in duplicates) {
            val values = codec.parseToJsonElement(input) as JsonArray
            jsonUniqueReject("constructor") { RecordDto(values) }
            jsonUniqueReject("reference bridge") { RecordDto(emptyList(), referenced = values) }
            jsonUniqueReject("nullable raw reference") { RecordDto(emptyList(), raw = values) }
            jsonUniqueReject("mutual cover bridge") { RecordDto(emptyList(), covered = values) }
            jsonUniqueReject("self cover") { RecordDto(emptyList(), selfCovered = values) }
            jsonUniqueReject("nested cover map") { RecordDto(emptyList(), coveredMap = mapOf("items" to values)) }
            jsonUniqueReject("cover collection alias") { CoveredHolderDto(rows = values) }
            jsonUniqueReject("cover Patch.Value") { ApplyUniqueInputDto(emptyList(), covered = Patch.Value(values)) }
            jsonUniqueReject("composed bridge") { RecordDto(emptyList(), composed = values) }
            jsonUniqueReject("composed collection alias") { ComposedHolderDto(rows = values) }
            jsonUniqueReject("composed map") { RecordDto(emptyList(), composedMap = mapOf("items" to values)) }
            jsonUniqueReject("composed Patch.Value") { ApplyUniqueInputDto(emptyList(), composed = Patch.Value(values)) }
            jsonUniqueReject("nested") { RecordDto(emptyList(), nested = listOf(values)) }
            jsonUniqueReject("decoder") { codec.decodeFromString<RecordDto>("{\"values\":$input}") }
            jsonUniqueReject("Patch.Value") { ApplyUniqueInputDto(emptyList(), Patch.Value(values)) }
            jsonUniqueReject("transparent alias") { HolderDto(rows = values) }
        }
        val objectPair = listOf(JsonObject(mapOf("a" to JsonPrimitive(1), "b" to JsonPrimitive(2))), JsonObject(mapOf("b" to JsonPrimitive(2), "a" to JsonPrimitive(1))))
        jsonUniqueReject("JsonObject") { RecordDto(emptyList(), objects = objectPair) }
        ApplyUniqueInputDto(emptyList(), Patch.Absent)
        ApplyUniqueInputDto(emptyList(), Patch.Null)
        ApplyUniqueInputDto(emptyList(), covered = Patch.Null)
        ApplyUniqueInputDto(emptyList(), composed = Patch.Null)
        var deep: JsonElement = JsonPrimitive(1)
        repeat(48) { deep = JsonArray(listOf(deep)) }
        RecordDto(listOf(deep))
        deep = JsonArray(listOf(deep))
        jsonUniqueReject("depth") { RecordDto(listOf(deep)) }
        RecordDto((0 until 16384).map { JsonPrimitive(it) })
        jsonUniqueReject("work") { RecordDto((0 until 16385).map { JsonPrimitive(it) }) }
        jsonUniqueReject("exponent") { RecordDto(listOf(codec.parseToJsonElement("1e513"))) }
        jsonUniqueReject("precision") { RecordDto(listOf(codec.parseToJsonElement("1".repeat(257)))) }
        jsonUniqueReject("cover depth") { RecordDto(emptyList(), covered = listOf(deep)) }
        jsonUniqueReject("cover exponent") { RecordDto(emptyList(), covered = listOf(codec.parseToJsonElement("1e513"))) }
        jsonUniqueReject("composed depth") { RecordDto(emptyList(), composed = listOf(deep)) }
        var bad = false
        var badReference = false
        var badCover = false
        var badRoot = false
        var badComposed = false
        var calls = 0
        val engine = JSONUniqueEngine(MockEngineConfig().apply { addHandler { outgoing ->
            calls++
            val body = when {
                badComposed -> "{\"values\":[],\"composed\":[{\"a\":1},{\"a\":1.0}]}"
                badCover -> "{\"values\":[],\"covered\":[{\"a\":1},{\"a\":1.0}]}"
                badReference -> "{\"values\":[],\"referenced\":[{\"a\":1},{\"a\":1.0}]}"
                bad -> "{\"values\":[1,1.0]}"
                else -> "{\"values\":$distinct,\"referenced\":$distinct,\"covered\":$distinct,\"composed\":$distinct}"
            }
            when (outgoing.url.encodedPath) {
                "/unique" -> respond(body, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/watch" -> respond("event: sample\ndata: $body\n\n", headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                "/cover", "/composed" -> {
                    check(codec.parseToJsonElement((outgoing.body as TextContent).text) == distinct)
                    respond(if (badRoot) "[1,1.0]" else distinct.toString(), headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
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
            bad = false
            badReference = true
            try { api.applyUnique(ApplyUniqueInputDto(emptyList())); error("Missing referenced HTTP uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            try { api.watchUnique().toList(); error("Missing referenced SSE uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            badReference = false
            check(api.applyUnique(ApplyUniqueInputDto(emptyList())).referenced == distinct)
            badCover = true
            try { api.applyUnique(ApplyUniqueInputDto(emptyList())); error("Missing cover HTTP uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            try { api.watchUnique().toList(); error("Missing cover SSE uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            badCover = false
            val expectedCover = distinct.map { if (it == JsonNull) null else it }
            check(api.applyCovered(distinct) == expectedCover)
            val before = calls
            var callback = false
            try { api.applyCovered(listOf(JsonPrimitive(1), codec.parseToJsonElement("1.0"))) { callback = true }; error("Missing root cover preflight failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            check(!callback && calls == before)
            badRoot = true
            try { api.applyCovered(distinct); error("Missing root cover postflight failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            badRoot = false
            check(api.applyCovered(distinct) == expectedCover)
            check(calls == 12)
            badComposed = true
            try { api.applyUnique(ApplyUniqueInputDto(emptyList())); error("Missing composed HTTP uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            try { api.watchUnique().toList(); error("Missing composed SSE uniqueness failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            badComposed = false
            check(api.applyComposed(distinct) == expectedCover)
            val composedCalls = calls
            callback = false
            try { api.applyComposed(listOf(JsonPrimitive(1), codec.parseToJsonElement("1.0"))) { callback = true }; error("Missing composed preflight failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            check(!callback && calls == composedCalls)
            badRoot = true
            try { api.applyComposed(distinct); error("Missing composed postflight failure") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it.message?.contains("uniqueItems") == true }) }
            badRoot = false
            check(api.applyComposed(distinct) == expectedCover)
            check(calls == 17)
        } finally { client.close() }
        println("JSONUnique Json defaults=$defaults nulls=$nulls recursive-numeric-objectkeys-null-Patch-limits-HTTP-SSE passed")
    }
}
