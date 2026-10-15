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
import org.example.nativeenumconsumers.api.DefaultApi
import org.example.nativeenumconsumers.api.models.ApplyPatchInputDto
import org.example.nativeenumconsumers.api.models.WatchCodeEventDto
import org.example.nativeenumconsumers.models.AliasDto
import org.example.nativeenumconsumers.models.Patch
import org.example.nativeenumconsumers.models.RecordDto
import org.example.nativeenumconsumers.models.RowsDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NativeEnumConsumerEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun nativeEnumConsumerBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nativeEnumConsumerBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic enum consumer body")
}

suspend fun nativeEnumConsumerContracts(checked: Boolean = false) {
    val uncheckedAlias: AliasDto = 3
    val uncheckedRows: RowsDto = listOf(3)
    check(uncheckedAlias == 3 && uncheckedRows.single() == 3)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = RecordDto(code = 1, rows = listOf(1, null, 2), flags = mapOf("x" to true), aliasedRows = listOf(2))
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(value)) == value)
        check(runCatching { RecordDto(code = uncheckedAlias) }.isFailure == checked)
        check(runCatching { RecordDto(code = 1, rows = listOf(3)) }.isFailure == checked)
        check(runCatching { RecordDto(code = 1, flags = mapOf("x" to false)) }.isFailure == checked)
        check(runCatching { RecordDto(code = 1, aliasedRows = uncheckedRows) }.isFailure == checked)
        check(runCatching { codec.decodeFromString<RecordDto>("{\"code\":3}") }.isFailure == checked)
        check(ApplyPatchInputDto().code === Patch.Absent)
        val clear = ApplyPatchInputDto(code = Patch.Null)
        val set = ApplyPatchInputDto(code = Patch.Value(2))
        check((codec.encodeToJsonElement(clear) as JsonObject)["code"] == JsonNull)
        check((codec.encodeToJsonElement(set) as JsonObject)["code"] == JsonPrimitive(2))
        check(runCatching { ApplyPatchInputDto(code = Patch.Value(3)) }.isFailure == checked)
        var calls = 0
        var bad = false
        val engine = NativeEnumConsumerEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            val response = if (bad) "3" else "2"
            when (request.url.encodedPath) {
                "/read" -> {
                    check(request.url.parameters["code"] in listOf("1", "3"))
                    respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/rows" -> {
                    check(codec.parseToJsonElement(nativeEnumConsumerBody(request.body)).toString() in listOf("[1,2]", "[3]"))
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
            check(api.readCode(1, listOf(1, null, 2)) == 2)
            check(api.applyRows(listOf(1, 2)) == listOf(2))
            check(api.applyCode(1) == 2 && api.applyPatch(set) == 2)
            check((api.watchCode().toList().single() as WatchCodeEventDto.Sample).data == 2)
            val before = calls
            check(runCatching { api.readCode(3) }.isFailure == checked)
            check(runCatching { api.readCode(1, listOf(3)) }.isFailure == checked)
            check(runCatching { api.applyRows(uncheckedRows) }.isFailure == checked)
            check(runCatching { api.applyCode(3) }.isFailure == checked)
            check(calls == before + if (checked) 0 else 4)
            bad = true
            check(runCatching { api.readCode(1) }.isFailure == checked)
            check(runCatching { api.applyRows(listOf(1, 2)) }.isFailure == checked)
            check(runCatching { api.watchCode().toList() }.isFailure == checked)
            bad = false
            check(api.readCode(1) == 2)
        } finally { http.close() }
        println("NativeEnumConsumers Json defaults=$defaults nulls=$nulls checked=$checked aliases-list-map-query-body-response-SSE-Patch passed")
    }
}
