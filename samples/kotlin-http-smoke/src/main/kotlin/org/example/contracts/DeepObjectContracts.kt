package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.request.parameter
import io.ktor.http.HttpStatusCode
import kotlinx.serialization.json.Json
import org.example.deepquery.DeepQueryApiClient
import org.example.deepquery.api.models.RequiredFilterDto
import org.example.deepquery.api.models.SearchRecordsFilterDto
import org.example.deepquery.models.StateDto
import org.example.deepbindings.DeepBindingApiClient
import org.example.deepbindings.api.models.*

private suspend fun deepBindingContracts() {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        val query = request.url.parameters
        when (calls) {
            1 -> check(request.url.encodedPath == "/scan" && query["id"] == "query" && query["filter[memo]"] == "first")
            2 -> check(request.url.encodedPath == "/scan/path" && "id" !in query && query["filter[memo]"] == "second")
            3 -> check(request.method.value == "GET" && query["f[memo]"] == "get")
            4 -> check(request.method.value == "POST" && query["f[memo]"] == "post")
            else -> error("Unexpected binding request")
        }
        respond("", HttpStatusCode.NoContent)
    }
    val http = HttpClient(engine)
    try {
        val api = DeepBindingApiClient(http).default
        api.getScan("query", GetScanFilterDto(memo = "first"))
        api.getScan2("path", GetScan2FilterDto(memo = "second"))
        api.getInherited(GetInheritedFilterDto(memo = "get"))
        api.postInherited(PostInheritedFilterDto(memo = "post"))
        check(calls == 4)
    } finally {
        http.close()
        engine.close()
    }
}

suspend fun deepObjectContracts() {
    deepBindingContracts()
    val decoded = Json.decodeFromString(SearchRecordsFilterDto.serializer(), """{"term":"x"}""")
    check(decoded.token == null && decoded.state == null && decoded.memo == null)
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        val query = request.url.parameters
        when (calls) {
            1 -> {
                check(request.url.encodedPath == "/records/a%2Fb")
                check(query["limit"] == "3" && request.headers["X-Trace"] == "trace")
                check(query["filter[term]"] == "snow 雪")
                check(query["filter[state]"] == "on hold")
                check(query["filter[mode]"] == "quick")
                check(query["filter[snake_case]"] == "a&b")
                check(query["filter[when]"] == "false")
                check(query["filter[kind]"] == "fixed")
                check("filter[token]" !in query && "filter[memo]" !in query)
                check(query.names().size == 7)
            }
            2 -> {
                check(query.names() == setOf("limit") && query["limit"] == "4")
                check(request.headers["X-Trace"] == "without-filter")
            }
            3 -> {
                check(query.names() == setOf("filter[term]", "filter[kind]"))
                check(query["filter[term]"] == "x")
            }
            4 -> check(request.url.encodedPath == "/required" && query.isEmpty())
            5 -> {
                check(query["criteria[token]"] == "null?")
                check(query["criteria[state]"] == "future value")
                check(query["criteria[memo]"] == "")
                check(query["manual"] == "last")
            }
            else -> error("Unexpected deepObject request")
        }
        respond("", HttpStatusCode.NoContent)
    }
    val http = HttpClient(engine)
    try {
        val api = DeepQueryApiClient(http).default
        val filter = SearchRecordsFilterDto(
            term = "snow 雪", state = StateDto.Serializer.fromSerialName("on hold"),
            mode = SearchRecordsFilterDto.Mode.Serializer.fromSerialName("quick"),
            snake_case = "a&b", `when` = false,
        )
        api.records("a/b", 3, filter, "trace")
        api.records("plain", 4, null, "without-filter")
        api.records("plain", filter = decoded)
        api.required(RequiredFilterDto())
        api.required(RequiredFilterDto(token = "null?", state = StateDto.Serializer.fromSerialName("future value"), memo = "")) {
            parameter("manual", "last")
        }
        check(calls == 5)
        println("DeepObject contracts passed: nullable filters and members, enum wire values, constants, property spelling, positional calls and final callback; $calls requests")
    } finally {
        http.close()
        engine.close()
    }
}
