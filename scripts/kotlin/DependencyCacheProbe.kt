package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.request.get
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.coroutines.runBlocking
import kotlinx.datetime.LocalDate
import kotlinx.serialization.Serializable

@Serializable
private data class DependencyCacheProbe(val value: String)

// Exercise the declared compiler/serialization/runtime classpaths with an
// in-memory transport. This is dependency preparation, not generator acceptance.
fun main() = runBlocking {
    val engine = MockEngine {
        respond("""{"value":"ready"}""", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
    }
    val client = HttpClient(engine) {
        install(ContentNegotiation) { json() }
        install(SSE)
    }
    try {
        check(client.get("https://example.test/dependency-probe").body<DependencyCacheProbe>().value == "ready")
        check(LocalDate.parse("2026-01-01").year == 2026)
    } finally { client.close(); engine.close() }
    println("Kotlin compiler, serialization, datetime, coroutines and Ktor dependencies ready")
}
