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
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.nativeenumchecks.api.DefaultApi
import org.example.nativeenumchecks.api.models.ApplyCheckedInputDto
import org.example.nativeenumchecks.api.models.WatchCheckedEventDto
import org.example.nativeenumchecks.models.HolderDto
import org.example.nativeenumchecks.models.MappedDto
import org.example.nativeenumchecks.models.Patch
import org.example.nativeenumchecks.models.RecordDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NativeEnumValidationEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun nativeEnumBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nativeEnumBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic native enum body")
}

suspend fun nativeEnumValidationContracts(checked: Boolean = false) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val payload = "{\"count\":1,\"wide\":-9223372036854775808,\"ratio\":2.5,\"flag\":true}"
        val value = codec.decodeFromString<RecordDto>(payload)
        check(value == RecordDto(1, Long.MIN_VALUE, 2.5f, true))
        check(value.optional == 2 && codec.decodeFromString<RecordDto>(codec.encodeToString(value)) == value)
        check(runCatching { RecordDto(3, Long.MIN_VALUE, 2.5f, true) }.isFailure == checked)
        check(runCatching { RecordDto(1, Long.MIN_VALUE, 2.5f, false) }.isFailure == checked)
        check(runCatching { RecordDto(1, Long.MIN_VALUE, Float.NaN, true) }.isFailure == checked)
        check(runCatching { codec.decodeFromString<RecordDto>(payload.replace("\"count\":1", "\"count\":3")) }.isFailure == checked)
        check(MappedDto(16777217f).value == 16777216f)
        val holder = codec.decodeFromString<HolderDto>("{\"rows\":[$payload],\"map\":{\"x\":$payload}}")
        check(holder.rows!!.single() == value && holder.map!!["x"] == value)
        val absent = codec.decodeFromString<ApplyCheckedInputDto>("{\"required\":1}")
        check(absent.patch === Patch.Absent)
        val clearing = ApplyCheckedInputDto(required = 1, patch = Patch.Null)
        check((codec.encodeToJsonElement(clearing) as JsonObject)["patch"] == JsonNull)
        val setting = ApplyCheckedInputDto(required = 1, patch = Patch.Value(2))
        check((codec.encodeToJsonElement(setting) as JsonObject)["patch"] == JsonPrimitive(2))
        check(runCatching { ApplyCheckedInputDto(required = 1, patch = Patch.Value(3)) }.isFailure == checked)
        var calls = 0
        var bad = false
        val engine = NativeEnumValidationEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            val outgoing = if (bad) payload.replace("\"count\":1", "\"count\":3") else payload
            when (request.url.encodedPath) {
                "/apply" -> { check(codec.parseToJsonElement(nativeEnumBody(request.body)) == codec.encodeToJsonElement(setting)); respond(outgoing, headers = headersOf(HttpHeaders.ContentType, "application/json")) }
                "/read" -> respond(outgoing, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/watch" -> respond("event: sample\ndata: $outgoing\n\n", headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                else -> error("Unexpected synthetic native enum route")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(http, "https://example.test/", codec)
            check(api.applyChecked(setting) == value && api.readChecked() == value)
            check((api.watchChecked().toList().single() as WatchCheckedEventDto.Sample).data == value)
            bad = true
            check(runCatching { api.readChecked() }.isFailure == checked)
            check(runCatching { api.watchChecked().toList() }.isFailure == checked)
            bad = false
            check(api.readChecked() == value && calls == 6)
        } finally { http.close() }
        println("NativeEnumValidation Json defaults=$defaults nulls=$nulls checked=$checked members-mapped-required-default-Patch-HTTP-SSE passed")
    }
}
