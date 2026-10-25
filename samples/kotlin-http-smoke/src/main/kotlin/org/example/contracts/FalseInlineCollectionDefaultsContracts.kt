package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import org.example.falseinlinedefaults.FalseInlineDefaultsClient
import org.example.falseinlinedefaults.models.Holder as HolderRoot

private fun falseInlineDefaultBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> falseInlineDefaultBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline collection body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseInlineCollectionDefaultsContracts() {
    val first=HolderRoot(rows=emptyList(),nullableRows=listOf(null))
    check(first.mapping?.isEmpty() == true && first.empty?.isEmpty() == true && first.rootNull == null)
    check(first.nested?.first()?.containsKey("description") == true)
    val second=HolderRoot(rows=emptyList(),nullableRows=null)
    check(first.nested !== second.nested && first.nested?.first() !== second.nested?.first())
    check(first.rows.isEmpty() && first.nullableRows == listOf(null))
    check(HolderRoot(rows=emptyList(),nullableRows=null).mapping?.isEmpty() == true)
    val descriptor=kotlinx.serialization.serializer<HolderRoot>().descriptor
    check(!descriptor.isElementOptional(descriptor.getElementIndex("rows")))
    check(!descriptor.isElementOptional(descriptor.getElementIndex("nullableRows")))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val body=codec.encodeToString(first)
        val decoded=codec.decodeFromString<HolderRoot>(body)
        check(decoded.rows.isEmpty() && decoded.nullableRows == listOf(null))
        check(decoded.nested?.first()?.get("description") == null)
        check(codec.decodeFromString<HolderRoot>("{\"rows\":[],\"nullableRows\":null,\"mapping\":null}").mapping == null)
        for(value in listOf("null","true","0","{}","[]","\"x\"")) {
            check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":[$value],\"nullableRows\":null}") }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":[],\"nullableRows\":null,\"mapping\":{\"k\":$value}}") }.exceptionOrNull() is SerializationException)
            if(value != "null") check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":[],\"nullableRows\":[$value]}") }.exceptionOrNull() is SerializationException)
        }
        check(runCatching { codec.decodeFromString<HolderRoot>("{}") }.isFailure)
        val missing=runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":[]}") }
        if(nulls) check(missing.isFailure) else check(missing.getOrThrow().nullableRows == null)
        val defaultsValue=codec.decodeFromString<HolderRoot>("{\"rows\":[],\"nullableRows\":null}")
        check(defaultsValue.nested?.first()?.containsKey("description") == true)
        check(defaultsValue.nested !== first.nested)
        check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":null}") }.isFailure)
        var bad=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/collections")
            check(falseInlineDefaultBody(request.body) == body)
            respond(if(bad) "{\"rows\":[null],\"nullableRows\":null}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=FalseInlineDefaultsClient(http,"http://example.test/").collections
            check(api.send(first).rows.isEmpty())
            bad=true
            check(runCatching { api.send(first) }.isFailure)
        } finally {http.close()}
        println("FalseInlineCollectionDefaults defaults=$defaults nulls=$nulls ownliteral/fresh/nullable/strict-leaf/required/http passed")
    }
}
