package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import org.example.literalcollectionrootdefaults.LiteralCollectionRootDefaultsClient
import org.example.literalcollectionrootdefaults.models.Holder as HolderRoot
import org.example.literalcollectionrootdefaults.models.Only as OnlyRoot

private fun literalCollectionDefaultBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> literalCollectionDefaultBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic default body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun literalFalseCollectionRootDefaultsContracts() {
    val first=HolderRoot.default()
    val second=HolderRoot.default()
    check(first !== second && first.falseValue == "seed")
    check(first.mapping!!["empty"] == null)
    check(OnlyRoot.default().rows!!.isEmpty())
    check(org.example.literalcollectionrootdefaults.models.RootNullable.default() == null)
    check(org.example.literalcollectionrootdefaults.models.NullableField.default().rows == null)
    val absent=org.example.literalcollectionrootdefaults.models.NoFieldDefault()
    val supplied=org.example.literalcollectionrootdefaults.models.NoFieldDefault.default()
    check(absent.rows == null && supplied.rows!!.isEmpty())
    check(first.optional!!.isEmpty() && first.optional !== second.optional)
    check(first.mapping !== second.mapping && first.mapping!!["description"] !== second.mapping!!["description"])
    check(first.mapping!!["description"]!!.containsKey("description") && first.mapping!!["description"]!!["description"] == null)
    check(first.nested == listOf(emptyList<Nothing>()) && first.nested !== second.nested)
    check(first.nullable == null && first.blocked == null)
    val descriptor=HolderRoot.serializer().descriptor
    check(!descriptor.isElementOptional(descriptor.getElementIndex("required")))
    check(!descriptor.isElementOptional(descriptor.getElementIndex("requiredNull")))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
        check(codec.encodeToString(OnlyRoot()) == if(defaults) "{\"rows\":[]}" else "{}")
        check(codec.encodeToString(OnlyRoot(rows=null)) == if(nulls) "{\"rows\":null}" else "{}")
        check(codec.decodeFromString<OnlyRoot>("{}").rows!!.isEmpty())
        check(runCatching {codec.decodeFromString<OnlyRoot>("{\"rows\":[null]}")}.isFailure)
        val body=codec.encodeToString(first)
        check("blocked" !in codec.parseToJsonElement(body).jsonObject)
        val decoded=codec.decodeFromString<HolderRoot>(body)
        check(decoded.required.isEmpty() && decoded.optional!!.isEmpty() && decoded.mapping!!["description"]!!.containsKey("description"))
        var bad=false
        val engine=MockEngine {request ->
            check(request.url.encodedPath == "/defaults")
            check(literalCollectionDefaultBody(request.body) == body)
            respond(if(bad) "{\"required\":[null],\"requiredNull\":null}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) {install(ContentNegotiation) {json(codec)}}
        try {
            val api=LiteralCollectionRootDefaultsClient(http,"http://example.test/").objects
            check(api.send(first).required.isEmpty())
            bad=true
            check(runCatching {api.send(first)}.isFailure)
        } finally {http.close()}
        println("LiteralFalseCollectionRootDefaults defaults=$defaults nulls=$nulls typed-fresh-default/required-suppression/nullable/metadata-data/http passed")
    }
}
