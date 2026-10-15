package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.*
import org.example.nullableunions.NullableUnionApiClient
import org.example.nullableunions.api.models.RequiredChoiceInputDto
import org.example.nullableunions.api.models.UnknownRequiredChoiceInputDto

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun nullableRequestUnionContracts() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            val body = codec.parseToJsonElement((request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString())
            when(calls) {
                1 -> {
                    check(request.url.encodedPath == "/required")
                    val row = body.jsonObject
                    check(row["kind"] == JsonPrimitive("left"))
                    check(("memo" in row) == (defaults && nulls))
                }
                2 -> {
                    check(request.url.encodedPath == "/required")
                    check(body == JsonNull)
                }
                3 -> {
                    check(request.url.encodedPath == "/optional")
                    check(body == JsonNull)
                }
                else -> error("Unexpected nullable union request")
            }
            respond("",HttpStatusCode.NoContent)
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = NullableUnionApiClient(http).default
            api.requiredChoice(RequiredChoiceInputDto.Left())
            // Content dispatch remains object-only; nullable schema does not
            // silently change the existing decoder's non-object behavior.
            check(runCatching { codec.decodeFromString<RequiredChoiceInputDto>("null") }.exceptionOrNull() is IllegalArgumentException)
            val rawNull = UnknownRequiredChoiceInputDto(JsonNull)
            check(rawNull.rawElement == JsonNull)
            api.requiredChoice(rawNull)
            api.optionalChoice()
            check(calls == 3)
            println("Nullable request union contracts passed: defaults=$defaults/nulls=$nulls, nominal required input, ordinary root defaults, raw null Unknown and optional null; $calls requests")
        } finally {
            http.close()
            engine.close()
        }
    }
}
