package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.*
import org.example.responsekinds.ResponseKindApiClient

suspend fun responseKindContracts() {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        val (path, body) = when (calls) {
            1 -> "/choice" to """{"kind":"left","id":"one"}"""
            2 -> "/choice" to "42"
            3 -> "/kind" to "\"text\""
            4 -> "/kind" to "false"
            5 -> "/nullable-kind" to "null"
            6 -> "/open" to "\"future\""
            7 -> "/void" to "null"
            else -> error("Unexpected response-kind request")
        }
        check(request.url.encodedPath == path)
        respond(body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
    }
    val http = HttpClient(engine) { install(ContentNegotiation) { json() } }
    try {
        val api = ResponseKindApiClient(http).default
        check(api.choice().jsonObject["id"]?.jsonPrimitive?.content == "one")
        // A raw JSON projection intentionally does not validate union membership.
        check(api.choice() == JsonPrimitive(42))
        check(api.kind() == JsonPrimitive("text"))
        check(api.kind() == JsonPrimitive(false))
        val nullable: JsonElement = api.nullableKind()
        check(nullable === JsonNull)
        check(api.openValue() == "\"future\"")
        check(api.empty() == Unit)
        check(calls == 7)
        println("Response kind contracts passed: raw object/primitive unions, JsonNull, scalar normalization and preferred void response; $calls requests")
    } finally {
        http.close()
        engine.close()
    }
}
