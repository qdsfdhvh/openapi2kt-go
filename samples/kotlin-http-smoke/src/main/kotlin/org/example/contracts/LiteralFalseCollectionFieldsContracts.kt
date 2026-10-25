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
import org.example.literalcollections.LiteralCollectionsClient
import org.example.literalcollections.models.Holder as HolderRoot
import org.example.literalcollections.models.OnlyContainers as OnlyRoot
import org.example.literalcollections.models.RootNullable as NullableOwnerRoot

private fun literalCollectionBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> literalCollectionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic collection body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun literalFalseCollectionFieldsContracts() {
    val value=HolderRoot(rows=emptyList(),nullable=null,optional=emptyList(),mapping=mapOf("empty" to emptyList(),"null" to null),nested=listOf(mapOf("empty" to emptyList())),kotlinx=emptyList(),org=emptyList())
    check(value.blocked == null && value.rows.isEmpty())
    val descriptor=HolderRoot.serializer().descriptor
    check(!descriptor.isElementOptional(descriptor.getElementIndex("rows")))
    check(descriptor.isElementOptional(descriptor.getElementIndex("nullable")))
    val rows=OnlyRoot.serializer().descriptor
    check(rows.isElementOptional(0) && rows.getElementDescriptor(0).isNullable)
    check(!rows.getElementDescriptor(0).getElementDescriptor(0).isNullable)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
        check(codec.encodeToString(OnlyRoot()) == if(defaults && nulls) "{\"rows\":null}" else "{}")
        check(codec.encodeToString(OnlyRoot(rows=emptyList())) == "{\"rows\":[]}")
        check(codec.decodeFromString<OnlyRoot>("{\"rows\":null}").rows == null)
        check(codec.decodeFromString<NullableOwnerRoot?>("null") == null)
        val body=codec.encodeToString(value)
        val fields=codec.parseToJsonElement(body).jsonObject
        check("blocked" !in fields)
        val roundTrip=codec.decodeFromString<HolderRoot>(body)
        check(roundTrip.rows.isEmpty() && roundTrip.optional!!.isEmpty())
        check(roundTrip.mapping!!.containsKey("null") && roundTrip.mapping!!["null"] == null)
        for(bad in listOf("{\"rows\":[null]}","{\"rows\":[1]}","{\"rows\":[{}]}")) {
            check(runCatching {codec.decodeFromString<OnlyRoot>(bad)}.exceptionOrNull() is SerializationException)
        }
        check(runCatching {codec.decodeFromString<HolderRoot>("{\"rows\":[],\"nested\":[{\"key\":[null]}]}")}.isFailure)
        var badResponse=false
        val engine=MockEngine {request ->
            check(request.url.encodedPath == "/collections")
            check(literalCollectionBody(request.body) == body)
            respond(if(badResponse) "{\"rows\":[null]}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) {install(ContentNegotiation) {json(codec)}}
        try {
            val api=LiteralCollectionsClient(http,"http://example.test/").objects
            check(api.send(value).rows.isEmpty())
            badResponse=true
            check(runCatching {api.send(value)}.isFailure)
        } finally {http.close()}
        println("LiteralFalseCollections defaults=$defaults nulls=$nulls empty-list/map/nullable-containers/strict-leaf/ordinary-presence/http passed")
    }
}
