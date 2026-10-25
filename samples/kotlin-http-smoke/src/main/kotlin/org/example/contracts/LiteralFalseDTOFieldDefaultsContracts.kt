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
import org.example.literalfielddefaults.LiteralFieldDefaultsClient
import org.example.literalfielddefaults.models.Optional as OptionalRoot
import org.example.literalfielddefaults.models.Required as RequiredRoot
import org.example.literalfielddefaults.models.Target as TargetRoot
import org.example.literalfielddefaults.models.RootNullable as NullableOwnerRoot

private fun literalFieldDefaultBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> literalFieldDefaultBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic field default body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun literalFalseDTOFieldDefaultsContracts() {
    val first=OptionalRoot()
    val second=OptionalRoot()
    check(first.value !== second.value && first.value?.labels !== second.value?.labels)
    check(first.value?.labels == listOf("field") && second.value?.labels == listOf("field"))
    check(first.without !== second.without && first.without?.blocked == null)
    check(first.plain == null && first.nullable == null)
    check(TargetRoot.default().labels == listOf("target"))
    check(NullableOwnerRoot.default() == null)
    val required=RequiredRoot(value=checkNotNull(first.value),nullable=null)
    check(required.value.labels == listOf("field") && required.nullable == null)
    val descriptor=RequiredRoot.serializer().descriptor
    check(!descriptor.isElementOptional(descriptor.getElementIndex("value")))
    check(!descriptor.isElementOptional(descriptor.getElementIndex("nullable")))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val body=codec.encodeToString(first)
        val roundTrip=codec.decodeFromString<OptionalRoot>(body)
        check(roundTrip.value?.labels == listOf("field") && roundTrip.without?.blocked == null)
        val fields=codec.parseToJsonElement(codec.encodeToString(required)).jsonObject
        check(fields.getValue("value").jsonObject.keys == setOf("labels"))
        for(name in listOf("blocked","reference")) {
            check(runCatching { codec.decodeFromString<OptionalRoot>("{\"value\":{\"$name\":null}}") }.exceptionOrNull() is SerializationException)
        }
        var badMember: String?=null
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/consumer")
            check(literalFieldDefaultBody(request.body) == body)
            val response=badMember?.let { "{\"value\":{\"$it\":null}}" } ?: body
            respond(response,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=LiteralFieldDefaultsClient(http,"http://example.test/").objects
            check(api.send(first).value?.labels == listOf("field"))
            for(name in listOf("blocked","reference")) {
                badMember=name
                check(runCatching { api.send(first) }.isFailure)
            }
            badMember=null
            check(api.send(first).value?.labels == listOf("field"))
        } finally { http.close() }
        println("LiteralFalseDTOFieldDefaults defaults=$defaults nulls=$nulls independent-field/root/required/mixed-strict-codecs/http passed")
    }
}
