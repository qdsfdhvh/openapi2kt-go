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
import org.example.structural.StructuralValueApiClient
import org.example.structural.models.RecordDto

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun structuralValueContracts() {
    val codec = Json { explicitNulls = false; encodeDefaults = false }
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        if (calls == 2) {
            check(request.url.encodedPath == "/nested")
            respond("""[[{"id":"known","extra":9,"memo":null},null]]""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        } else {
            val body = codec.parseToJsonElement((request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString())
            when (calls) {
                1 -> {
                    check(request.url.encodedPath == "/rows")
                    val row = body.jsonArray[0].jsonObject
                    check(row["memo"] == JsonNull && row["extra"] == JsonPrimitive(5))
                }
                3 -> {
                    check(request.url.encodedPath == "/refs")
                    check(body.jsonArray[0].jsonArray[0].jsonObject["id"] == JsonPrimitive("wire"))
                }
                4 -> {
                    check(request.url.encodedPath == "/maps")
                    check(body.jsonArray[0].jsonObject["clear"] == JsonNull)
                    check(body.jsonArray[0].jsonObject["value"] == JsonPrimitive("future"))
                }
                else -> error("Unexpected structural request")
            }
            respond("",HttpStatusCode.NoContent)
        }
    }
    val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
    try {
        val api = StructuralValueApiClient(http).default
        api.sendRows(listOf(mapOf("memo" to null, "extra" to JsonPrimitive(5))))
        val nested = api.nested()
        check(nested[0][1] == null)
        val first = checkNotNull(nested[0][0])
        check(first["id"] == JsonPrimitive("known") && first["extra"] == JsonPrimitive(9))
        check("memo" in first && first["memo"] == null)
        api.sendRefs(listOf(listOf(RecordDto("wire"))))
        api.sendMaps(listOf(mapOf("clear" to null, "value" to "future")))
        check(calls == 4)
        println("Structural value contracts passed: raw map fields/extras/nulls, nested nullable response values, component refs and scalar enum maps; $calls requests")
    } finally {
        http.close()
        engine.close()
    }
}
