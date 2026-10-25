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
import kotlinx.serialization.json.jsonObject
import org.example.falsedtodefaults.FalseDTOFieldDefaultsClient
import org.example.falsedtodefaults.models.Consumer as ConsumerRoot
import org.example.falsedtodefaults.models.RequiredConsumer as RequiredRoot
import org.example.falsedtodefaults.models.Target as TargetRoot

private fun falseDTOFieldBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> falseDTOFieldBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic DTO field default body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseDTOFieldDefaultsContracts() {
    val first=ConsumerRoot()
    check(first.target?.value == null && first.target?.label == "synthetic")
    check(first.alias != null && first.alias?.value == null)
    check(first.absent == null && first.nullable == null)
    check(ConsumerRoot().target !== first.target && ConsumerRoot().alias !== first.alias)
    val required=RequiredRoot(TargetRoot(value=null))
    check(required.target.value == null)
    check(!kotlinx.serialization.serializer<RequiredRoot>().descriptor.isElementOptional(0))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val decoded=codec.decodeFromString<ConsumerRoot>("{}")
        check(decoded.target?.label == "synthetic" && decoded.alias != null)
        check(decoded.target !== first.target)
        val explicit=codec.decodeFromString<ConsumerRoot>("{\"target\":null,\"alias\":null}")
        check(explicit.target == null && explicit.alias == null)
        val body=codec.encodeToString(first)
        val fields=codec.parseToJsonElement(body).jsonObject
        check(("target" in fields) == (defaults || first.target != ConsumerRoot().target))
        check(("alias" in fields) == (defaults || first.alias != ConsumerRoot().alias))
        check(("absent" in fields) == (defaults && nulls))
        for(value in listOf("true","1","{}","[]","\"x\"")) {
            check(runCatching { codec.decodeFromString<ConsumerRoot>("{\"target\":{\"value\":$value}}") }.exceptionOrNull() is SerializationException)
        }
        check(runCatching { codec.decodeFromString<ConsumerRoot>("{\"target\":{\"value\":null,\"blocked\":null}}") }.exceptionOrNull() is SerializationException)
        check(runCatching { codec.decodeFromString<RequiredRoot>("{}") }.isFailure)
        var bad=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/defaults")
            check(falseDTOFieldBody(request.body) == body)
            respond(if(bad) "{\"target\":{\"value\":true}}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=FalseDTOFieldDefaultsClient(http,"http://example.test/").defaults
            check(api.send(first).target?.label == "synthetic")
            bad=true
            check(runCatching { api.send(first) }.isFailure)
        } finally {http.close()}
        println("FalseDTOFieldDefaults defaults=$defaults nulls=$nulls fresh/required/absence/alias/http passed")
    }
}
