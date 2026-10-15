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
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.example.errors.ErrorContractApiClient
import org.example.errors.api.errorBody
import org.example.errors.models.ProblemDto

private class CancelDecode

suspend fun errorContracts() {
    val cancellation = CancellationException("synthetic decode cancellation")
    val cancelPlugin = createClientPlugin("SyntheticCancelDecode") {
        transformResponseBody { _, _, type ->
            if (type.type == CancelDecode::class) throw cancellation
            null
        }
    }
    var calls = 0
    val engine = MockEngine {
        calls++
        val status = when (calls) {
            1 -> HttpStatusCode.OK
            3 -> HttpStatusCode.ServiceUnavailable
            6 -> HttpStatusCode.Conflict
            else -> HttpStatusCode.BadRequest
        }
        val body = when (calls) {
            1 -> "\"ok\""
            2, 8 -> """{"message":"invalid"}"""
            3 -> "<html>unavailable</html>"
            4 -> "{"
            6 -> """{"reason":"conflict"}"""
            else -> "{}"
        }
        respond(body,status,headersOf("Content-Type",if (calls == 3) "text/html" else "application/json"))
    }
    val http = HttpClient(engine) {
        expectSuccess = true
        install(cancelPlugin)
        install(ContentNegotiation) { json() }
    }
    try {
        val api = ErrorContractApiClient(http).default
        // Ktor's String body reader returns JSON string text including quotes.
        check(api.fetch() == "\"ok\"")
        for (index in 2..7) {
            val error = runCatching { api.fetch() }.exceptionOrNull()
            check(error is ResponseException)
            when (index) {
                2 -> {
                    check(error.errorBody<ProblemDto>()?.message == "invalid")
                    check(error.errorBody<JsonObject>()?.get("message")?.jsonPrimitive?.content == "invalid")
                }
                3, 4, 7 -> check(error.errorBody<ProblemDto>() == null)
                5 -> check(runCatching { error.errorBody<CancelDecode>() }.exceptionOrNull() === cancellation)
                6 -> check(error.errorBody<JsonObject>()?.get("reason")?.jsonPrimitive?.content == "conflict")
            }
        }
        val lenient = http.config { expectSuccess = false }
        try {
            check(ErrorContractApiClient(lenient).default.fetch() == """{"message":"invalid"}""")
        } finally {
            lenient.close()
        }
        check(calls == 8)
        println("Error contracts passed: typed/ref and inline body, saved reread, malformed/HTML/missing fields, cancellation identity and caller-owned validation; $calls requests")
    } finally {
        http.close()
        engine.close()
    }
}
