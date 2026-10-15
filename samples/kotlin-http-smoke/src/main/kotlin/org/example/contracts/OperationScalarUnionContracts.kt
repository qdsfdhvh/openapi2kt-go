package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.*
import org.example.scalarunions.ApiClient
import org.example.scalarunions.models.StateDto

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun operationScalarUnionContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            val text = (request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString()
            when (calls) {
                1 -> {
                    check(request.url.encodedPath == "/open" && text == "future-value")
                    respond("\"future-reply\"", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                2 -> {
                    check(request.url.encodedPath == "/maybe" && text == "null")
                    respond("", HttpStatusCode.NoContent)
                }
                3 -> {
                    check(request.url.encodedPath == "/maybe" && text == "outside-enum")
                    respond("", HttpStatusCode.NoContent)
                }
                4 -> {
                    check(request.url.encodedPath == "/items")
                    check(codec.parseToJsonElement(text) == JsonArray(listOf(JsonPrimitive("known"), JsonPrimitive("outside"), JsonNull)))
                    respond("[\"reply\",\"outside\",null]", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                5 -> {
                    check(request.url.encodedPath == "/maps")
                    val values = codec.parseToJsonElement(text).jsonArray.single().jsonObject
                    check(values["x"] == JsonPrimitive("outside") && values["n"] == JsonNull)
                    respond("[[{\"x\":\"future\",\"n\":null}]]", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                6 -> {
                    check(request.url.encodedPath == "/identity")
                    check(codec.parseToJsonElement(text) == JsonArray(listOf(JsonPrimitive("new-shared"), JsonNull)))
                    respond("[\"reply-shared\",null]", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                else -> error("Unexpected operation scalar union request")
            }
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http).default
            // The root String reader/writer is Ktor raw text, unlike list/map JSON strings.
            check(api.openValue("future-value") == "\"future-reply\"")
            api.maybeValue()
            api.maybeValue("outside-enum")
            check(api.itemValues(listOf("known", "outside", null)) == listOf("reply", "outside", null))
            check(api.mapValues(listOf(mapOf("x" to "outside", "n" to null))) == listOf(listOf(mapOf("x" to "future", "n" to null))))
            val result = api.identities(listOf(StateDto.Serializer.fromSerialName("new-shared"), null))
            check(result[0]?.serialName == "reply-shared" && result[1] == null)
            check(calls == 6)
            println("Operation scalar union contracts passed: defaults=$defaults/nulls=$nulls; open/nullable enum scalar folds, list/map nulls, raw root String, named component identity; $calls requests")
        } finally {
            http.close()
            engine.close()
        }
    }
}
