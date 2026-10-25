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
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import org.example.literalrootdefaults.LiteralRootDefaultsClient
import org.example.literalrootdefaults.models.OptionalOnly as OptionalRoot
import org.example.literalrootdefaults.models.Mixed as MixedRoot
import org.example.literalrootdefaults.models.RootNullable as NullableOwnerRoot
import org.example.literalrootdefaults.models.Consumer as ConsumerRoot

private fun literalRootDefaultBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> literalRootDefaultBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic factory body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun literalFalseRootDefaultsContracts() {
    val optional=OptionalRoot.default()
    check(optional.blocked == null)
    val first=MixedRoot.default()
    val second=MixedRoot.default()
    check(first !== second && first.labels !== second.labels)
    check(first.labels == listOf("seed") && second.labels == listOf("seed"))
    check(first.blocked == null && first.kotlin == null && first.kotlinx == null && first.org == null)
    check(NullableOwnerRoot.default() == null)
    check(ConsumerRoot().value == null)
    check(OptionalRoot.default() !== optional)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.encodeToString(optional) == "{}")
        check(codec.encodeToString<NullableOwnerRoot?>(NullableOwnerRoot.default()) == "null")
        check(codec.decodeFromString<NullableOwnerRoot?>("null") == null)
        check(runCatching { codec.decodeFromString<NullableOwnerRoot?>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<OptionalRoot>("{\"blocked\":null}") }.exceptionOrNull() is SerializationException)
        check(runCatching { codec.decodeFromString<MixedRoot>("{\"kotlinx\":null}") }.exceptionOrNull() is SerializationException)
        val fields=codec.parseToJsonElement(codec.encodeToString(first)).jsonObject
        check(fields.keys == setOf("labels"))
        check(codec.decodeFromString<MixedRoot>(codec.encodeToString(first)).labels == listOf("seed"))
        val body=codec.encodeToString(first)
        var badResponse=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/mixed")
            check(literalRootDefaultBody(request.body) == body)
            respond(if(badResponse) "{\"blocked\":null}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=LiteralRootDefaultsClient(http,"http://example.test/").objects
            check(api.send(first).labels == listOf("seed"))
            badResponse=true
            check(runCatching { api.send(first) }.isFailure)
        } finally { http.close() }
        println("LiteralFalseRootDefaults defaults=$defaults nulls=$nulls fresh-factory/strictabsence/nullable-owner/http passed")
    }
}
