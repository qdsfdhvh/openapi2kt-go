package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.*
import org.example.requestfields.ApiClient
import org.example.requestfields.api.models.ValuesInputDto
import org.example.requestfields.models.Patch
import org.example.requestfields.models.SharedDto

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun requestFieldUnionContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var calls = 0
        val future = buildJsonObject { put("kind", "future"); put("extra", 9) }
        val array = JsonArray(listOf(JsonPrimitive("outside-kind")))
        val engine = MockEngine { request ->
            calls++
            check(request.url.encodedPath == "/values")
            val row = codec.parseToJsonElement((request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString()).jsonObject
            val decoded = codec.decodeFromJsonElement(ValuesInputDto.serializer(), row)
            when (calls) {
                1 -> {
                    check(row["choice"] == JsonPrimitive("raw"))
                    check(("nullableChoice" in row) == nulls)
                    check(("optionalPlain" in row) == (defaults && nulls))
                    check("optionalPatch" !in row)
                    check(decoded.nullableChoice == null && decoded.optionalPatch == Patch.Absent)
                }
                2 -> {
                    check(row["optionalPatch"] == JsonNull && decoded.optionalPatch == Patch.Null)
                    val child = row.getValue("child").jsonObject
                    check(child["choice"] == JsonPrimitive(false) && child["patch"] == JsonNull)
                    val item = row.getValue("rows").jsonArray.single().jsonObject
                    check(item["choice"] == array && item["patch"] == JsonPrimitive(7))
                    check(decoded.rows?.single()?.patch == Patch.Value(JsonPrimitive(7)))
                    val value = row.getValue("map").jsonObject.getValue("key").jsonObject
                    check(value.getValue("choice").jsonObject["future"] == JsonPrimitive(true))
                    check("patch" !in value)
                    val variant = row.getValue("variant").jsonObject
                    check(variant["kind"] == JsonPrimitive("a"))
                    check(("nullablePayload" in variant) == nulls)
                    check(("payload" in variant) == (defaults && nulls))
                    check(variant.getValue("detail").jsonObject["patch"] == array)
                    // Mixed fields are opaque JSON, not newly generated branch DTOs.
                    check(row.getValue("mixed").jsonObject["unknown"] == JsonPrimitive(true))
                    check(row["arrays"] == array)
                    check(row.getValue("kinds").jsonArray == JsonArray(listOf(JsonPrimitive(true), JsonNull, future)))
                    check(decoded.kinds?.get(1) == JsonNull)
                    val values = row.getValue("values").jsonObject
                    check(values["clear"] == JsonNull && values["bool"] == JsonPrimitive(true) && values["future"] == future)
                    check(decoded.values?.containsKey("clear") == true && decoded.values?.get("clear") == JsonNull)
                    check(row.getValue("shared").jsonObject["choice"] == JsonPrimitive("component"))
                }
                3 -> {
                    check(row["choice"] == JsonPrimitive(1))
                    check(row["optionalPatch"] == JsonPrimitive(Long.MAX_VALUE))
                    check(decoded.optionalPatch == Patch.Value(JsonPrimitive(Long.MAX_VALUE)))
                    check(row["optionalPlain"] == JsonPrimitive(false))
                    check(row["mixed"] == JsonPrimitive("future"))
                    check(row["arrays"] == JsonPrimitive(false))
                    check(row["variant"] == future)
                    check(decoded.variant is ValuesInputDto.UnknownVariant)
                    check(row.getValue("shared").jsonObject["choice"] == JsonPrimitive(true))
                }
                4 -> {
                    // JsonElement can hold a JSON null even when the schema union
                    // has no null branch. This projection does not validate membership.
                    check(row["choice"] == JsonNull && decoded.choice == JsonNull)
                    check(row["nullableChoice"] == JsonPrimitive(true))
                    // Value(JsonNull) and Patch.Null have the same wire representation.
                    check(row["optionalPatch"] == JsonNull && decoded.optionalPatch == Patch.Null)
                }
                else -> error("Unexpected request field union call")
            }
            respond("", HttpStatusCode.NoContent)
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http).default
            api.values(ValuesInputDto(choice = JsonPrimitive("raw"), nullableChoice = null))
            api.values(ValuesInputDto(
                choice = JsonPrimitive(false), nullableChoice = null,
                optionalPatch = Patch.Null,
                child = ValuesInputDto.Child(JsonPrimitive(false), Patch.Null),
                rows = listOf(ValuesInputDto.RowsInner(array, Patch.Value(JsonPrimitive(7)))),
                map = mapOf("key" to ValuesInputDto.MapValue(buildJsonObject { put("future", true) })),
                mixed = buildJsonObject { put("unknown", true) }, arrays = array,
                kinds = listOf(JsonPrimitive(true), JsonNull, future),
                values = buildJsonObject { put("clear", JsonNull); put("bool", true); put("future", future) },
                variant = ValuesInputDto.Variant.A(nullablePayload = null,
                    detail = ValuesInputDto.Variant.A.Detail(JsonPrimitive(1.5), Patch.Value(array))),
                shared = SharedDto(SharedDto.StringChoice("component")),
            ))
            api.values(ValuesInputDto(
                choice = JsonPrimitive(1), nullableChoice = null,
                optionalPatch = Patch.Value(JsonPrimitive(Long.MAX_VALUE)), optionalPlain = JsonPrimitive(false),
                mixed = JsonPrimitive("future"), arrays = JsonPrimitive(false),
                variant = ValuesInputDto.UnknownVariant(future),
                shared = SharedDto(SharedDto.BooleanChoice(true)),
            ))
            api.values(ValuesInputDto(choice = JsonNull, nullableChoice = JsonPrimitive(true), optionalPatch = Patch.Value(JsonNull)))
            check(calls == 4)
            println("Request-only field union contracts passed: defaults=$defaults/nulls=$nulls; raw kinds/unknowns, required nullable, Patch/variant/child/list/map, component wrappers and JsonNull carrier boundary; $calls requests")
        } finally {
            http.close()
            engine.close()
        }
    }
}
