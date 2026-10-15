package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import org.example.projections.ProjectionApiClient

suspend fun projectionContracts() {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        val body = when (calls) {
            1 -> { check(request.url.encodedPath == "/enum"); "\"future\"" }
            2 -> { check(request.url.encodedPath == "/enum-list"); "[\"a\",\"future\"]" }
            3 -> { check(request.url.encodedPath == "/nullable-object"); "null" }
            4 -> { check(request.url.encodedPath == "/nullable-object"); """{"id":"local"}""" }
            5 -> { check(request.url.encodedPath == "/nullable-ref"); "null" }
            6 -> { check(request.url.encodedPath == "/nullable-ref"); """{"id":"shared"}""" }
            7 -> {
                check(request.url.encodedPath == "/send")
                check((request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString() == "future")
                ""
            }
            else -> error("Unexpected projection request")
        }
        respond(body,if (calls == 7) HttpStatusCode.NoContent else HttpStatusCode.OK,headersOf("Content-Type","application/json"))
    }
    val http = HttpClient(engine) { install(ContentNegotiation) { json() } }
    try {
        val api = ProjectionApiClient(http).default
        // Root String is deliberately Ktor's raw text, not an enum serializer.
        check(api.enumValue() == "\"future\"")
        check(api.enumList() == listOf("a", "future"))
        check(api.nullableObject() == null)
        check(api.nullableObject()?.id == "local")
        check(api.nullableRef() == null)
        check(api.nullableRef()?.id == "shared")
        api.sendEnum("future")
        check(calls == 7)
        println("Operation projection contracts passed: scalar/array enums, raw String body, nullable local/ref responses; $calls requests")
    } finally {
        http.close()
        engine.close()
    }
}
