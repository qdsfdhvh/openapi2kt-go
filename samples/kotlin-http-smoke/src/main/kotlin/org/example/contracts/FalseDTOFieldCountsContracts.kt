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
import org.example.falsedtocounts.FalseDTOFieldCountsClient
import org.example.falsedtocounts.models.Consumer as ConsumerRoot
import org.example.falsedtocounts.models.RequiredConsumer as RequiredRoot
import org.example.falsedtocounts.models.Target as TargetRoot

private fun falseDTOCountBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> falseDTOCountBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic DTO field default body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseDTOFieldCountsContracts() {
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
        val target=first.alias!!
        if(!defaults || !nulls) {
            check(runCatching { codec.encodeToString(target) }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.encodeToString(required) }.exceptionOrNull() is SerializationException)
        } else {
            check(codec.parseToJsonElement(codec.encodeToString(target)).jsonObject.size >= 1)
            check(codec.decodeFromString<TargetRoot>(codec.encodeToString(target)).value == null)
        }
        check(runCatching { codec.decodeFromString<TargetRoot>("{}") }.exceptionOrNull() is SerializationException)
        for(value in listOf("true","1","{}","[]","\"x\"")) {
            check(runCatching { codec.decodeFromString<ConsumerRoot>("{\"target\":{\"value\":$value}}") }.exceptionOrNull() is SerializationException)
        }
        check(runCatching { codec.decodeFromString<ConsumerRoot>("{\"target\":{\"value\":null,\"blocked\":null}}") }.exceptionOrNull() is SerializationException)
        check(runCatching { codec.decodeFromString<RequiredRoot>("{}") }.isFailure)
        val wire=ConsumerRoot(alias=null)
        val body=codec.encodeToString(wire)
        var bad=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/defaults")
            check(falseDTOCountBody(request.body) == body)
            respond(if(bad) "{\"target\":{\"value\":true}}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=FalseDTOFieldCountsClient(http,"http://example.test/").defaults
            check(api.send(wire).target?.label == "synthetic")
            bad=true
            check(runCatching { api.send(wire) }.isFailure)
        } finally {http.close()}
        println("FalseDTOFieldCounts defaults=$defaults nulls=$nulls fresh/required/absence/alias/http passed")
    }
}
