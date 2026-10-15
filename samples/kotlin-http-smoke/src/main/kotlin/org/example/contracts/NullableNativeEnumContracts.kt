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
import org.example.nullablenativeenums.api.DefaultApi
import org.example.nullablenativeenums.api.models.ApplyPatchInputDto
import org.example.nullablenativeenums.api.models.WatchCodeEventDto
import org.example.nullablenativeenums.models.CodeDto
import org.example.nullablenativeenums.models.HolderDto
import org.example.nullablenativeenums.models.Patch
import org.example.nullablenativeenums.models.RecordDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullableNativeEnumEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun nullableNativeEnumBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableNativeEnumBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic nullable enum body")
}

suspend fun nullableNativeEnumContracts() {
    val independent: CodeDto = 3
    check(independent == 3)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<CodeDto>("null") == null)
        check(codec.decodeFromString<CodeDto>("3") == 3)
        val value = RecordDto(code = null, rows = listOf(null, 2, 4), flags = mapOf("x" to null), aliasedRows = listOf(null, 2))
        check(value.defaulted == null)
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(value)) == value)
        check(HolderDto(inherited = null, own = null, narrow = 2).narrow == 2)
        check(codec.decodeFromString<HolderDto>("{\"inherited\":null,\"own\":null,\"narrow\":2}").inherited == null)
        check(runCatching { codec.decodeFromString<HolderDto>("{\"narrow\":null}") }.isFailure)
        check(ApplyPatchInputDto().code === Patch.Absent)
        val clear = ApplyPatchInputDto(code = Patch.Null)
        val set = ApplyPatchInputDto(code = Patch.Value(2))
        check((codec.encodeToJsonElement(clear) as JsonObject)["code"] == JsonNull)
        check((codec.encodeToJsonElement(set) as JsonObject)["code"] == JsonPrimitive(2))
        for (invalid in listOf(1, 3, 6)) {
            check(runCatching { RecordDto(code = invalid) }.isFailure)
            check(runCatching { RecordDto(code = null, rows = listOf(invalid)) }.isFailure)
            check(runCatching { HolderDto(inherited = invalid, own = null, narrow = 2) }.isFailure)
            check(runCatching { HolderDto(inherited = null, own = invalid, narrow = 2) }.isFailure)
            check(runCatching { ApplyPatchInputDto(code = Patch.Value(invalid)) }.isFailure)
            check(runCatching { codec.decodeFromString<RecordDto>("{\"code\":$invalid}") }.isFailure)
        }
        var calls = 0
        var response = "2"
        val engine = NullableNativeEnumEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/read" -> {
                    check(request.url.parameters["code"] in listOf(null, "2"))
                    respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/rows" -> {
                    check(codec.parseToJsonElement(nullableNativeEnumBody(request.body)).toString() == "[null,2,4]")
                    respond("[$response]", headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/direct" -> {
                    check(nullableNativeEnumBody(request.body) in listOf("null", "2"))
                    respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/patch" -> respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/watch" -> respond("event: sample\ndata: $response\n\n", headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                else -> error("Unexpected synthetic nullable enum route")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(http, "https://example.test/", codec)
            check(api.readCode(2) == 2)
            check(api.applyCode(2) == 2 && api.applyPatch(set) == 2)
            check(api.applyRows(listOf(null, 2, 4)) == listOf(2))
            response = "null"
            check(api.readCode(null) == null)
            check(api.applyCode(null) == null && api.applyPatch(clear) == null)
            check(api.applyRows(listOf(null, 2, 4)) == listOf(null))
            check((api.watchCode().toList().single() as WatchCodeEventDto.Sample).data == null)
            for (invalid in listOf(1, 3, 6)) {
                val prior = calls
                check(runCatching { api.readCode(invalid) }.isFailure)
                check(runCatching { api.applyCode(invalid) }.isFailure)
                check(runCatching { api.applyRows(listOf(invalid)) }.isFailure)
                check(calls == prior)
                response = invalid.toString()
                check(runCatching { api.readCode(null) }.isFailure)
                check(runCatching { api.applyRows(listOf(null, 2, 4)) }.isFailure)
                check(runCatching { api.watchCode().toList() }.isFailure)
                check(calls == prior + 3)
            }
            response = "2"
            check(api.readCode(2) == 2)
        } finally { http.close() }
        println("NullableNativeEnum Json defaults=$defaults nulls=$nulls intrinsic-ownaliases-nearestfalse-Patch-nullbody-result-SSE passed")
    }
}
