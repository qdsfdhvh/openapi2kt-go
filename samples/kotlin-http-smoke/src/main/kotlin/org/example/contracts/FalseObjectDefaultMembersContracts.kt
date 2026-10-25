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
import org.example.falseobjectdefaults.FalseObjectDefaultsClient
import org.example.falseobjectdefaults.models.OptionalOnly as OptionalRoot
import org.example.falseobjectdefaults.models.Mixed as MixedRoot
import org.example.falseobjectdefaults.models.Nullable as NullableRoot
import org.example.falseobjectdefaults.models.Required as RequiredRoot
import org.example.falseobjectdefaults.models.RootNullable as NullableOwnerRoot
import org.example.falseobjectdefaults.models.Consumer as ConsumerRoot

private fun falseObjectDefaultBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> falseObjectDefaultBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic object factory body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseObjectDefaultMembersContracts() {
    val optional=OptionalRoot.default()
    check(optional.blocked == null)
    val mixed=MixedRoot.default()
    check(mixed.maybe == null && mixed.blocked == null && mixed.label == "synthetic")
    val nullable=NullableRoot.default()
    check(nullable.description == null)
    val required=RequiredRoot.default()
    check(required.value == null)
    check(NullableOwnerRoot.default() == null)
    check(ConsumerRoot().value == null)
    check(OptionalRoot.default() !== optional)
    check(MixedRoot.default() !== mixed)
    check(RequiredRoot.default() !== required)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.encodeToString(optional) == "{}")
        check(codec.decodeFromString<OptionalRoot>("{}").blocked == null)
        check(runCatching { codec.decodeFromString<OptionalRoot>("{\"blocked\":null}") }.exceptionOrNull() is SerializationException)
        val fields=codec.parseToJsonElement(codec.encodeToString(mixed)).jsonObject
        check(fields.size == if(defaults && nulls) 2 else 1)
        check("blocked" !in fields)
        check(codec.decodeFromString<MixedRoot>(codec.encodeToString(mixed)).label == "synthetic")
        check(codec.decodeFromString<NullableRoot>(codec.encodeToString(nullable)).description == null)
        val body=codec.encodeToString(required)
        check(body == if(nulls) "{\"value\":null}" else "{}")
        var badResponse=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/required")
            check(falseObjectDefaultBody(request.body) == body)
            respond(if(badResponse) "{\"value\":true}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=FalseObjectDefaultsClient(http,"http://example.test/").objects
            check(api.send(required).value == null)
            badResponse=true
            check(runCatching { api.send(required) }.isFailure)
        } finally { http.close() }
        println("FalseObjectDefaultMembers defaults=$defaults nulls=$nulls own-fresh-objectfactory/nullable-member/strictabsence/required/sourcecounts/http passed")
    }
}
