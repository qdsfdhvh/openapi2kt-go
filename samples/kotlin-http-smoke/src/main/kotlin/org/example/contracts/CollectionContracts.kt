package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.http.HttpStatusCode
import io.ktor.http.decodeURLPart
import org.example.collections.CollectionApiClient
import org.example.collections.models.State

suspend fun collectionContracts() {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        when (calls) {
            1 -> {
                val pathValue = request.url.encodedPath.removePrefix("/collections/")
                check(pathValue.decodeURLPart() == "a/b,c d") { request.url.toString() }
                check(!pathValue.contains('/') && !pathValue.contains(' '))
                val q = request.url.parameters
                check(q.getAll("repeat") == listOf("ready", "future value"))
                check(q["csv"] == "1,2")
                check(q["spaces"] == "red blue green")
                check(q["pipes"] == "x|y")
                check(q["indexed[0]"] == "true" && q["indexed[1]"] == "false")
                check(q.getAll("expanded") == listOf("one", "two"))
                check(q["region"] == "west" && q["key with space"] == "a&b")
                check("attributes" !in q.names())
                check(request.headers["X-States"] == "on hold,ready")
            }
            2 -> {
                check(request.url.parameters.isEmpty())
                check(request.headers["X-States"] == null)
            }
            3 -> {
                check(request.url.parameters["csv"] == "")
                check("repeat" !in request.url.parameters.names())
                check(request.headers["X-States"] == "")
            }
            4 -> check(request.url.parameters.getAll("values") == listOf("text"))
            5 -> check(request.url.parameters.getAll("values") == listOf("7"))
            6 -> check(request.url.parameters.isEmpty())
            7 -> check(request.url.parameters.getAll("values") == listOf("required"))
            else -> error("Unexpected request")
        }
        respond("", HttpStatusCode.NoContent)
    }
    val http = HttpClient(engine)
    try {
        val api = CollectionApiClient(http).default
        val state = State.Serializer::fromSerialName
        api.search(
            ids = listOf("a/b", "c d"),
            repeat = listOf(state("ready"), state("future value")),
            csv = listOf(1, 2),
            spaces = listOf("red blue", "green"),
            pipes = listOf("x", "y"),
            indexed = listOf(true, false),
            expanded = listOf("one", "two"),
            attributes = linkedMapOf("region" to "west", "key with space" to "a&b"),
            xStates = listOf(state("on hold"), state("ready")),
        )
        api.search(listOf("single"))
        api.search(listOf("single"), repeat = emptyList(), csv = emptyList(), xStates = emptyList())
        api.same(listOf("text"))
        api.same2(listOf(7))
        api.nullable(null)
        api.nullable(listOf("required"))
        check(calls == 7)
        println("HTTP collection contracts passed: repeat/CSV/space/pipe/indexed/map, enum/path/header, null/empty and generic erasure")
    } finally {
        http.close()
        engine.close()
    }
}
