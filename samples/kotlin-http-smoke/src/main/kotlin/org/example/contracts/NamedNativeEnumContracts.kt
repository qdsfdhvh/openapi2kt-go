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
import org.example.namednativeenums.api.DefaultApi
import org.example.namednativeenums.api.models.ApplyPatchInputDto
import org.example.namednativeenums.api.models.WatchCodeEventDto
import org.example.namednativeenums.models.AliasDto
import org.example.namednativeenums.models.Patch
import org.example.namednativeenums.models.RecordDto
import org.example.namednativeenums.models.RowsDto
import org.example.namednativeenums.models.ViewsDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NamedNativeEnumEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun namedNativeEnumBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> namedNativeEnumBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic enum consumer body")
}

suspend fun namedNativeEnumContracts() {
    val uncheckedAlias: AliasDto = 3
    val uncheckedRows: RowsDto = listOf(3)
    check(uncheckedAlias == 3 && uncheckedRows.single() == 3)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = RecordDto(code = 2, rows = listOf(2, null, 4), flags = mapOf("x" to true), aliasedRows = listOf(2))
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(value)) == value)
        check(value.defaulted == 2)
        check(ViewsDto(maybe = null, value = 2).maybe == null)
        check(ViewsDto(maybe = 4, value = 2).maybe == 4)
        check(runCatching { ViewsDto(maybe = 1, value = 2) }.isFailure)
        check(runCatching { codec.decodeFromString<ViewsDto>("{\"maybe\":null,\"value\":null}") }.isFailure)
        check(codec.decodeFromString<RecordDto>("{\"code\":2}").defaulted == 2)
        check(RecordDto(code = 2, floating = 2.0f, longValue = 4L).longValue == 4L)
        for (invalid in listOf(1, 3, 6)) {
            check(runCatching { RecordDto(code = invalid) }.isFailure)
            check(runCatching { RecordDto(code = 2, defaulted = invalid) }.isFailure)
            check(runCatching { RecordDto(code = 2, longValue = invalid.toLong()) }.isFailure)
            check(runCatching { codec.decodeFromString<RecordDto>("{\"code\":$invalid}") }.isFailure)
            check(runCatching { ApplyPatchInputDto(code = Patch.Value(invalid)) }.isFailure)
        }
        for (invalid in listOf(1.0f, 3.0f, 4.0f, Float.NaN, Float.POSITIVE_INFINITY)) {
            check(runCatching { RecordDto(code = 2, floating = invalid) }.isFailure)
        }
        check(runCatching { RecordDto(code = uncheckedAlias) }.isFailure)
        check(runCatching { RecordDto(code = 2, rows = listOf(3)) }.isFailure)
        check(runCatching { RecordDto(code = 2, flags = mapOf("x" to false)) }.isFailure)
        check(runCatching { RecordDto(code = 2, aliasedRows = uncheckedRows) }.isFailure)
        check(runCatching { codec.decodeFromString<RecordDto>("{\"code\":3}") }.isFailure)
        check(ApplyPatchInputDto().code === Patch.Absent)
        val clear = ApplyPatchInputDto(code = Patch.Null)
        val set = ApplyPatchInputDto(code = Patch.Value(2))
        check((codec.encodeToJsonElement(clear) as JsonObject)["code"] == JsonNull)
        check((codec.encodeToJsonElement(set) as JsonObject)["code"] == JsonPrimitive(2))
        check(runCatching { ApplyPatchInputDto(code = Patch.Value(3)) }.isFailure)
        var calls = 0
        var response = "2"
        val engine = NamedNativeEnumEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/read" -> {
                    check(request.url.parameters["code"] == "2")
                    respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/rows" -> {
                    check(codec.parseToJsonElement(namedNativeEnumBody(request.body)).toString() == "[2,4]")
                    respond("[$response]", headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/patch" -> respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/direct" -> respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/watch" -> respond("event: sample\ndata: $response\n\n", headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                else -> error("Unexpected synthetic enum consumer route")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(http, "https://example.test/", codec)
            check(api.readCode(2, listOf(2, null, 4)) == 2)
            check(api.applyRows(listOf(2, 4)) == listOf(2))
            check(api.applyCode(2) == 2 && api.applyPatch(set) == 2)
            check((api.watchCode().toList().single() as WatchCodeEventDto.Sample).data == 2)
            val before = calls
            check(runCatching { api.readCode(3) }.isFailure)
            check(runCatching { api.readCode(2, listOf(3)) }.isFailure)
            check(runCatching { api.applyRows(uncheckedRows) }.isFailure)
            check(runCatching { api.applyCode(3) }.isFailure)
            check(calls == before)
            for (invalid in listOf(1, 3, 6)) {
                val prior = calls
                check(runCatching { api.readCode(invalid) }.isFailure)
                check(runCatching { api.applyCode(invalid) }.isFailure)
                check(runCatching { api.applyRows(listOf(invalid)) }.isFailure)
                check(calls == prior)
                response = invalid.toString()
                check(runCatching { api.readCode(2) }.isFailure)
                check(runCatching { api.applyRows(listOf(2, 4)) }.isFailure)
                check(runCatching { api.watchCode().toList() }.isFailure)
                check(calls == prior + 3)
            }
            response = "2"
            check(api.readCode(2) == 2)
        } finally { http.close() }
        println("NamedNativeEnum Json defaults=$defaults nulls=$nulls original-catalog-vocabulary-export-rootbridge-list-map-Patch-HTTP-SSE passed")
    }
}
