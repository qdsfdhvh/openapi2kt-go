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
import org.example.falseinlinecollections.FalseInlineCollectionsClient
import org.example.falseinlinecollections.models.Holder as HolderRoot

private fun falseInlineCollectionBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> falseInlineCollectionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline collection body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseInlineCollectionFieldsContracts() {
    val first=HolderRoot(rows=emptyList(),nullableRows=listOf(null),mapping=emptyMap(),nested=listOf(mapOf("null" to null)))
    check(first.rows.isEmpty() && first.nullableRows == listOf(null))
    check(HolderRoot(rows=emptyList(),nullableRows=null).mapping == null)
    val descriptor=kotlinx.serialization.serializer<HolderRoot>().descriptor
    check(!descriptor.isElementOptional(descriptor.getElementIndex("rows")))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val body=codec.encodeToString(first)
        val decoded=codec.decodeFromString<HolderRoot>(body)
        check(decoded.rows.isEmpty() && decoded.nullableRows == listOf(null))
        check(decoded.nested?.single()?.get("null") == null)
        check(codec.decodeFromString<HolderRoot>("{\"rows\":[],\"nullableRows\":null,\"mapping\":null}").mapping == null)
        for(value in listOf("null","true","0","{}","[]","\"x\"")) {
            check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":[$value]}") }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":[],\"mapping\":{\"k\":$value}}") }.exceptionOrNull() is SerializationException)
            if(value != "null") check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":[],\"nullableRows\":[$value]}") }.exceptionOrNull() is SerializationException)
        }
        check(runCatching { codec.decodeFromString<HolderRoot>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderRoot>("{\"rows\":null}") }.isFailure)
        var bad=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/collections")
            check(falseInlineCollectionBody(request.body) == body)
            respond(if(bad) "{\"rows\":[null]}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=FalseInlineCollectionsClient(http,"http://example.test/").collections
            check(api.send(first).rows.isEmpty())
            bad=true
            check(runCatching { api.send(first) }.isFailure)
        } finally {http.close()}
        println("FalseInlineCollectionFields defaults=$defaults nulls=$nulls empty/nullable/strict-leaf/required/http passed")
    }
}
