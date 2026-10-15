package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import org.example.operationexcluded.NoOperationsClient
import org.example.operationcollisions.CollisionTypesApiClient
import org.example.operationcollisions.api.models.GetRowsModeDto
import org.example.operationcollisions.api.models.GetRows2ModeDto
import org.example.operationtypes.OperationTypesApiClient
import org.example.operationtypes.api.models.FetchInputParamDto
import org.example.operationtypes.api.models.FetchOrderDto
import org.example.operationtypes.api.models.FetchResponseDto
import org.example.operationtypes.api.models.ListItemDto
import org.example.operationtypes.api.models.LookupResponseDto
import org.example.operationtypes.models.StateDto

suspend fun operationModelContracts() {
    var calls = 0
    var nullableCalls = 0
    val engine = MockEngine { request ->
        calls++
        val body = when (request.url.encodedPath) {
            "/record" -> {
                check(request.url.parameters.getAll("order") == listOf("low", "future order"))
                check(request.headers["input"] == "full")
                """{"id":"one","state":"future","detail":{"note":"nested"}}"""
            }
            "/records" -> """[{"record":{"id":"two"}}]"""
            "/lookup" -> """{"first":[{"id":"three"}]}"""
            "/rows", "/rows/path" -> {
                check(request.url.parameters.getAll("mode") == listOf("b"))
                if (request.url.encodedPath == "/rows") check(request.url.parameters["x"] == "query")
                """{"id":"renamed"}"""
            }
            "/numbers" -> "[1,2]"
            "/peers" -> """{"any":{"label":"peer"}}"""
            "/maybe-rows" -> if (++nullableCalls == 1) """[null,{"id":"nested"}]""" else "null"
            "/nullable-map" -> "null"
            else -> error("Unexpected URL")
        }
        respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
    }
    val http = HttpClient(engine) { install(ContentNegotiation) { json() } }
    try {
        NoOperationsClient(http) // Exclusion leaves a valid client with no local model dependencies.
        val api = OperationTypesApiClient(http).default
        val record: FetchResponseDto = api.fetch(
            order = listOf(FetchOrderDto.Serializer.fromSerialName("low"), FetchOrderDto.Serializer.fromSerialName("future order")),
            input = FetchInputParamDto.Serializer.fromSerialName("full"),
        )
        val state: StateDto = record.state
        check(record.id == "one" && state.serialName == "future" && record.detail?.note == "nested")
        val rows: List<ListItemDto> = api.listList()
        check(rows.single().record.id == "two")
        val dictionary: LookupResponseDto = api.lookup()
        check(dictionary["first"]?.single()?.id == "three")
        check(api.numbers() == listOf(1, 2))
        check(api.peers().getValue("any").label == "peer")
        val maybeRows = api.maybeRows()
        check(maybeRows?.size == 2 && maybeRows[0] == null && maybeRows[1]?.id == "nested")
        check(api.maybeRows() == null && api.nullableMap() == null && calls == 8)
        val collision = CollisionTypesApiClient(http).default
        check(collision.getRows("query", listOf(GetRowsModeDto.Serializer.fromSerialName("b"))).id == "renamed")
        check(collision.getRows2("path", listOf(GetRows2ModeDto.Serializer.fromSerialName("b"))).id == "renamed")
        check(calls == 10)
        println("HTTP operation model contracts passed: inline enum/list wire, reserved model suffix, response nesting/component identity, array item and dictionary aliases")
    } finally {
        http.close()
        engine.close()
    }
}
