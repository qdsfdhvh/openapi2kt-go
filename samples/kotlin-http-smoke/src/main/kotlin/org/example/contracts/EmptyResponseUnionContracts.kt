package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.*
import org.example.emptyresponses.ApiClient

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun emptyResponseUnionContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            val (path, body) = when (calls) {
                1 -> "/unit" to "not-json"
                2 -> "/object" to "null"
                3 -> "/object" to "{\"id\":\"one\",\"memo\":null}"
                4 -> "/ref" to "null"
                5 -> "/ref" to "{\"id\":\"two\"}"
                6 -> "/scalar" to "\"outside\""
                7 -> "/many" to "true"
                8 -> "/typed-empty" to "[1,2]"
                9 -> "/single-empty" to "not-json"
                10 -> "/object" to "{\"future\":true}"
                else -> error("Unexpected empty response union request")
            }
            check(request.url.encodedPath == path)
            respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http).default
            // JSON-bearing 200 remains preferred over typed201 even when projected Unit.
            check(api.emptyResult() == Unit)
            check(api.oneObject() == null)
            val local = api.oneObject()
            check(local?.id == "one" && local.memo == null)
            check(api.oneRef() == null)
            check(api.oneRef()?.id == "two")
            check(api.scalarRaw() == JsonPrimitive("outside"))
            check(api.manyRaw() == JsonPrimitive(true))
            // Explicit type:object is not an empty/void branch; fallback remains raw JSON.
            check(api.typedEmpty() == JsonArray(listOf(JsonPrimitive(1), JsonPrimitive(2))))
            check(api.singleEmpty() == Unit)
            // Narrowed nullable DTO projection is not union/schema validation:
            // an unknown object can still fail its required-field decoder.
            check(runCatching { api.oneObject() }.isFailure)
            check(calls == 10)
            println("Empty response union contracts passed: defaults=$defaults/nulls=$nulls; Unit/media preference, nullable borrowed object/ref, raw scalar/multiple kinds, typed-empty distinction and missing-field failure; $calls requests")
        } finally {
            http.close()
            engine.close()
        }
    }
}
