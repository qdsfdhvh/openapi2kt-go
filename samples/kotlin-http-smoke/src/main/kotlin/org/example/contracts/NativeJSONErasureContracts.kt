package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.nativejsonerasure.NativeJSONErasureProbeApiClient

private fun nativeJSONBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nativeJSONBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected JSON classifier body ${body::class}")
}
suspend fun nativeJSONErasureContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var expected: JsonElement = JsonNull
        val routes = mutableListOf<String>()
        val engine = MockEngine { request ->
            routes += request.url.encodedPath
            check(codec.parseToJsonElement(nativeJSONBody(request.body)) == expected)
            respond("", HttpStatusCode.NoContent)
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = NativeJSONErasureProbeApiClient(http, "http://example.test/").default
            for (value in listOf(JsonNull, JsonPrimitive("future"), JsonPrimitive(42), JsonPrimitive(true), JsonArray(listOf(JsonNull, JsonPrimitive(1))), JsonObject(mapOf("unknown" to JsonPrimitive("kept"))))) {
                expected = value
                api.send(value); api.send2(value); api.send3(value)
                check(routes.takeLast(3) == listOf("/a-element", "/b-raw", "/c-element-alias"))
            }
            for (value in listOf(JsonObject(emptyMap()), JsonObject(mapOf("unknown" to JsonArray(listOf(JsonNull, JsonPrimitive(true))))))) {
                expected = value
                api.putPut(value); api.putPut2(value); api.putPut3(value)
                check(routes.takeLast(3) == listOf("/d-object", "/e-bag", "/f-object-alias"))
            }
            check(routes.size == 24)
        } finally { http.close() }
        println("NativeJSONErasure Json defaults=$defaults nulls=$nulls canonicalclassifier-suffix-native-raw-object-alias JSONwire checks passed")
    }
}
