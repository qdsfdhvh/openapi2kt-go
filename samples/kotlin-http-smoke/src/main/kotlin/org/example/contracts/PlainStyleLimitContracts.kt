package org.example.contracts

import org.example.plainlimits.models.Wide240Dto
import org.example.plainlimits.models.Wide120Dto
import org.example.plainlimits.models.A47Dto
import org.example.plainlimits.models.DeepDto
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement

fun plainStyleLimitContracts(){
 val json=Json{encodeDefaults=true;explicitNulls=true}
 val big=JsonObject((0 until 240).associate{String.format("f%03d",it) to JsonPrimitive(1L)})
 val wide=json.decodeFromJsonElement(Wide240Dto.serializer(),big)
 val second=json.decodeFromJsonElement(Wide240Dto.serializer(),big)
 check(wide==second&&wide.hashCode()==second.hashCode());check(wide.toString()=="Wide240Dto");check(json.encodeToJsonElement(wide)==big);check(Wide240Dto.serializer().descriptor.elementsCount==240)
 val narrow=JsonObject((0 until 120).associate{String.format("f%03d",it) to JsonPrimitive(2L)})
 val longs=json.decodeFromJsonElement(Wide120Dto.serializer(),narrow);check(longs==json.decodeFromJsonElement(Wide120Dto.serializer(),narrow));check(longs.toString()=="Wide120Dto");check(json.encodeToJsonElement(longs)==narrow);check(Wide120Dto.serializer().descriptor.elementsCount==120)
 for(clazz in listOf(Wide240Dto::class.java,Wide120Dto::class.java)){check(clazz.methods.none{it.name=="copy"||it.name.startsWith("component")})}
 val root=A47Dto("root");check(root==A47Dto("root")&&root.toString()=="ZRecordDto");check(json.encodeToJsonElement(root)==json.parseToJsonElement("{\"id\":\"root\"}"))
 fun deep(value:String):String = "{\"values\":"+"[".repeat(16)+"\"$value\""+"]".repeat(16)+"}"
 val tree=json.parseToJsonElement(deep("safe"));val leaf=json.decodeFromJsonElement(DeepDto.serializer(),tree);check(leaf==json.decodeFromJsonElement(DeepDto.serializer(),tree));check(json.encodeToJsonElement(leaf)==tree)
 try{json.decodeFromString(DeepDto.serializer(),deep(""));error("invalid 16-edge plain leaf accepted")}catch(cause:Throwable){check(generateSequence(cause){it.cause}.any{it is IllegalArgumentException||it is SerializationException})}
 println("PLAIN STYLE INDEPENDENT 240 BOXED/120 LONG/48 ALIAS/16 CHECKED COMPILE-RUNTIME PASS")
}
