@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.ResponseException
import io.ktor.client.plugins.api.createClientPlugin
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlin.coroutines.cancellation.CancellationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import org.example.inlineerrors.api.DefaultApi
import org.example.inlineerrors.api.errorBody
import org.example.inlineerrors.api.models.AlphaError400Dto
import org.example.inlineerrors.api.models.AlphaError4XXDto
import org.example.inlineerrors.api.models.AlphaError5XXDto
import org.example.inlineerrors.api.models.AlphaErrorDefaultDto
import org.example.inlineerrors.api.models.BorrowedError400Dto

private class InlineErrorCancelDecode

public suspend fun inlineErrorContracts(checked: Boolean = false) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val cancellation = CancellationException("synthetic inline error decode cancellation")
        val cancelPlugin = createClientPlugin("SyntheticInlineErrorCancelDecode") {
            transformResponseBody { _, _, type ->
                if (type.type == InlineErrorCancelDecode::class) throw cancellation
                null
            }
        }
        var mode = 0
        var calls = 0
        val engine = MockEngine {
            calls++
            val status = when (mode) { 1 -> HttpStatusCode.NotFound; 2 -> HttpStatusCode.ServiceUnavailable; else -> HttpStatusCode.BadRequest }
            val body = when (mode) {
                1 -> "{\"reason\":\"missing\"}"
                2 -> "null"
                3 -> "{"
                4 -> "{}"
                5 -> "{\"message\":\"\"}"
                else -> "{\"message\":\"invalid\",\"state\":\"future\",\"nested\":{\"code\":7}}"
            }
            respond(body, status, headersOf("Content-Type", "application/json"))
        }
        val client = HttpClient(engine) { expectSuccess = true; install(cancelPlugin); install(ContentNegotiation) { json(codec) } }
        try {
            val api = DefaultApi(client, "https://example.test/")
            suspend fun error(): ResponseException {
                try { api.alpha(); error("Missing caller-owned error validation") }
                catch (failure: ResponseException) { return failure }
            }
            val original = error()
            val typed = original.errorBody<AlphaError400Dto>()!!
            check(typed.message == "invalid" && typed.state!!.serialName == "future" && typed.nested!!.code == 7)
            check(original.errorBody<JsonObject>() != null)
            check(original.errorBody<AlphaErrorDefaultDto>()?.message == "invalid")
            check(runCatching { original.errorBody<InlineErrorCancelDecode>() }.exceptionOrNull() === cancellation)
            mode = 1; check(error().errorBody<AlphaError4XXDto>()?.reason == "missing")
            mode = 2; check(error().errorBody<AlphaError5XXDto?>() == null)
            mode = 3; check(error().errorBody<AlphaError400Dto>() == null)
            mode = 4; check(error().errorBody<AlphaError400Dto>() == null)
            mode = 5
            val bounded = error().errorBody<AlphaError400Dto>()
            check((bounded == null) == checked)
            mode = 0
            try { api.borrowed(); error("Missing borrowed error response") }
            catch (failure: ResponseException) { check(failure.errorBody<BorrowedError400Dto>()?.message == "invalid") }
            val lenient = client.config { expectSuccess = false }
            try { DefaultApi(lenient, "https://example.test/").alpha() }
            finally { lenient.close() }
            check(calls == 8)
            val decoded: AlphaError400Dto = codec.decodeFromString("{\"message\":\"given\"}")
            check(decoded.state === AlphaError400Dto.State.READY)
        } finally { client.close(); engine.close() }
        println("InlineErrors Json defaults=$defaults nulls=$nulls typed-status-borrowed-required-nullable-default-cancellation-caller-validation passed")
    }
}
