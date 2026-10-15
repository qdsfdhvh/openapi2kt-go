package org.example.contracts
import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import org.example.nullableint.ApiClient

suspend fun nullableIntResponseContracts() {
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        var response="null"
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        val engine=MockEngine { respond(response,HttpStatusCode.OK,headersOf("Content-Type","application/json")) }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=ApiClient(http,"http://example.test/").default
            check(api.number()==null)
            for(value in listOf(Int.MIN_VALUE,Int.MAX_VALUE)) { response=value.toString();check(api.number()==value && api.plain()==value) }
            response="null";check(api.text()=="null")
            for(invalid in listOf("2147483648","{}","{")) {
                response=invalid
                val result=runCatching { api.number() }
                check(result.isFailure)
            }
            response="2";check(api.number()==2)
            println("NullableIntNoSSE Json defaults=$defaults nulls=$nulls twoargctor-null-range-rejection-reuse-rawString checks passed")
        } finally { http.close() }
    }
}
