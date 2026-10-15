package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.example.bodies.BodyApiClient
import org.example.bodies.models.WidgetDto

@OptIn(ExperimentalSerializationApi::class)
suspend fun bodyContracts() {
    for (defaults in listOf(false, true)) {
        for (nulls in listOf(false, true)) {
            bodyContracts(Json { encodeDefaults = defaults; explicitNulls = nulls })
        }
    }
}

@OptIn(ExperimentalSerializationApi::class)
private suspend fun bodyContracts(codec: Json) {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        val text = when (val body = request.body) {
            is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
            is OutgoingContent.NoContent -> ""
            else -> error("Unexpected outgoing content: ${body::class}")
        }
        check(request.body.contentType?.match(ContentType.Application.Json) == true) { "content type: ${request.body.contentType}" }
        when (calls) {
            1 -> {
                check(request.method == HttpMethod.Post)
                check(request.url.parameters["input"] == "query-input")
                val obj = Json.parseToJsonElement(text).jsonObject
                check(obj["id"]?.jsonPrimitive?.content == "one")
                // Shared components retain ordinary defaults, controlled by the
                // caller's Json configuration, not a request-only Patch policy.
                check(("label" in obj) == (codec.configuration.encodeDefaults && codec.configuration.explicitNulls)) { obj.toString() }
            }
            2 -> {
                check(request.method == HttpMethod.Put)
                check(text == "null") { "optional body: $text" }
            }
            3 -> check(Json.parseToJsonElement(text).jsonObject["id"]?.jsonPrimitive?.content == "two")
            4 -> check(text == "7")
            // Ktor's default ContentNegotiation ignores String: the generated
            // setBody(String) call retains raw string wire, not JSON quoting.
            5 -> check(text == "raw string")
            6 -> check(text == "replacement")
            else -> error("Unexpected body request")
        }
        respond("", HttpStatusCode.NoContent)
    }
    val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
    try {
        val api = BodyApiClient(http).default
        api.createCreate("query-input", WidgetDto("one", null))
        api.optional()
        api.optional(WidgetDto("two"))
        api.number(7)
        api.text("raw string")
        api.createCreate(input = WidgetDto("unused")) { setBody("replacement") }
        check(calls == 6)
        println("HTTP body contracts passed: component/scalar, required/optional null, reserved input, raw String and builder override")
    } finally {
        http.close()
        engine.close()
    }
}
