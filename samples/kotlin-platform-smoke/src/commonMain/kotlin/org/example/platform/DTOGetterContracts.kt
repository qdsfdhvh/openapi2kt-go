package org.example.platform

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.platformgetters.models.DefaultHolder
import org.example.platformgetters.models.Holder
import org.example.platformgetters.models.InlineHolder
import org.example.platformgetters.models.Item
import org.example.platformgetters.models.NullableHolder

internal fun dtoGetterContracts(json: Json) {
    val first=Item(id="a")
    val second=Item(id="b")
    check(first.state?.serialName=="ready" && first.version==0 && first.flag==false)
    check(first.kind=="fixed" && !first.pinnedFlag && first.pinnedNumber==0.0)
    val valid=Holder(values=listOf(first,second))
    check(json.decodeFromString<Holder>(json.encodeToString(valid))==valid)
    check(DefaultHolder().values?.single()?.state?.serialName=="ready")
    check(json.decodeFromString<DefaultHolder>("{}").values?.single()?.kind=="fixed")
    check(runCatching{Holder(values=listOf(first,Item(id="a",version=0)))}.isFailure)
    check(runCatching{json.decodeFromString<Holder>("""{"values":[{"id":"a"},{"id":"a","state":"ready","kind":"fixed"}]}""")}.isFailure)
    check(runCatching{Holder(values=listOf(Item(id="e",state=Item.State.READY),Item(id="e",state=Item.State.Unrecognized("ready"))))}.isFailure)
    check(Holder(values=listOf(Item(id="e",state=Item.State.READY),Item(id="e",state=Item.State.Unrecognized("future")))).values?.size==2)
    check(runCatching{NullableHolder(values=listOf(null,null))}.isFailure)
    check(NullableHolder(values=listOf(null,first)).values?.size==2)
    check(runCatching{Holder(values=listOf(Item(id="n",state=null),Item(id="n",state=null)))}.isFailure)
    check(Holder(values=listOf(Item(id="n",state=null),Item(id="n",state=Item.State.READY))).values?.size==2)
    check(runCatching{Item(id="bad",version=2)}.isFailure)
    check(runCatching{Item(id="bad",weight=2f)}.isFailure)
    check(runCatching{Holder(values=listOf(Item(id="z",weight=0f),Item(id="z",weight=-0f)))}.isFailure)
    check(runCatching{Holder(values=listOf(Item(id="z",amount=Double.NaN)))}.isFailure)
    val changed=json.decodeFromString<Item>("""{"id":"a","kind":"changed","pinnedFlag":true,"pinnedNumber":1.0}""")
    check(changed.kind=="changed" && changed.pinnedFlag && changed.pinnedNumber==1.0)
    check(json.encodeToString(changed).contains("changed"))
    check(Holder(values=listOf(first,changed)).values?.size==2)
    val same=json.decodeFromString<Item>("""{"id":"a","kind":"fixed","pinnedFlag":false,"pinnedNumber":-0.0}""")
    check(runCatching{Holder(values=listOf(first,same))}.isFailure)
    val left=kotlin.uuid.Uuid.parse("550E8400-E29B-41D4-A716-446655440000")
    val right=kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
    check(runCatching{Holder(values=listOf(Item(id="u",uuid=left),Item(id="u",uuid=right)))}.isFailure)
    val earlier=kotlin.time.Instant.parse("2024-01-01T00:00:00Z")
    val offset=kotlin.time.Instant.parse("2024-01-01T01:00:00+01:00")
    check(runCatching{Holder(values=listOf(Item(id="i",instant=earlier),Item(id="i",instant=offset)))}.isFailure)
    check(json.decodeFromString<InlineHolder>("""{"values":[{"id":"a"},{"id":"b"}]}""").values?.size==2)
    check(runCatching{json.decodeFromString<InlineHolder>("""{"values":[{"id":"a"},{"id":"a"}]}""")}.isFailure)
}
