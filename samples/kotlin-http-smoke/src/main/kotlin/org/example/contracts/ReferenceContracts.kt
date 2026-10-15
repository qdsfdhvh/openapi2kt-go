package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.example.references.ReferenceApiClient
import org.example.references.models.WidgetDto

suspend fun referenceContracts() {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        check(request.method == HttpMethod.Post)
        check(request.url.encodedPath == "/widgets/a%2Fb")
        check(request.url.parameters["limit"] == Long.MAX_VALUE.toString())
        check(request.headers["X-Trace"] == "from-array")
        val body = (request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString()
        check(Json.parseToJsonElement(body).jsonObject["id"]?.jsonPrimitive?.content == "input")
        respond("""{"id":"output"}""", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
    }
    val http = HttpClient(engine) { install(ContentNegotiation) { json() } }
    try {
        val response = ReferenceApiClient(http).default.send(
            id = "a/b", limit = Long.MAX_VALUE, xTrace = "from-array", input = WidgetDto("input"),
        )
        check(response.id == "output" && calls == 1)
        println("HTTP reference contracts passed: inherited/overridden parameters, chains, escaped/array pointers, typed request/response and opaque examples")
    } finally {
        http.close()
        engine.close()
    }
}
