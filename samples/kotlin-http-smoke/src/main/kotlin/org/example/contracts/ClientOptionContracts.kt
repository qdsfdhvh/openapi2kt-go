package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.http.HttpStatusCode
import org.example.options.ChosenClient

suspend fun clientOptionContracts() {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        check(request.url.toString() == "https://api.example.test/v1/health")
        respond("", HttpStatusCode.NoContent)
    }
    val http = HttpClient(engine)
    try {
        val client = ChosenClient(http)
        check(client.userAdmin === client.userAdmin)
        client.default.getHealth()
        check(calls == 1)
        println("HTTP client option contracts passed: explicit client name, excluded API, remaining groups and base path")
    } finally {
        http.close()
        engine.close()
    }
}
