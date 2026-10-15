@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import org.example.dataarrays.models.ArraysDto as DataArrays
import org.example.plainarrays.models.ArraysDto as PlainArrays
import org.example.dataarrays.models.CountDto as DataCount
import org.example.plainarrays.models.CountDto as PlainCount
import org.example.plainarrays.models.OnlyDto
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement

private fun plain(a: DataArrays) = PlainArrays(a.bytes,a.shorts,a.ints,a.longs,a.singles,a.doubles,a.chars,a.flags,a.items,a.opaque,a.optionalItems)
private fun reject(block: () -> Unit) { try { block(); error("Invalid count/plain input accepted") } catch (e: Throwable) { check(generateSequence(e){it.cause}.any{it is IllegalArgumentException || it is SerializationException}) } }
fun plainStyleArrayContracts() {
 val bytes=byteArrayOf(-128,0,127)
 val a=DataArrays(bytes,shortArrayOf(Short.MIN_VALUE,Short.MAX_VALUE),intArrayOf(Int.MIN_VALUE,Int.MAX_VALUE),longArrayOf(Long.MIN_VALUE,Long.MAX_VALUE),floatArrayOf(Float.NaN,0f,-0f),doubleArrayOf(Double.NaN,0.0,-0.0),charArrayOf('a','中'),booleanArrayOf(true,false),arrayOf("sample",null),bytes,null)
 val variants=listOf(a,a.copy(bytes=bytes.copyOf()),a.copy(shorts=a.shorts.copyOf()),a.copy(ints=a.ints.copyOf()),a.copy(longs=a.longs.copyOf()),a.copy(singles=a.singles.copyOf()),a.copy(doubles=a.doubles.copyOf()),a.copy(chars=a.chars.copyOf()),a.copy(flags=a.flags.copyOf()),a.copy(items=a.items.copyOf()),a.copy(opaque=bytes.copyOf()),a.copy(optionalItems=a.items),a.copy(optionalItems=arrayOf("sample",null)))
 for(x in variants){check(x.hashCode()==plain(x).hashCode());for(y in variants)check((x==y)==(plain(x)==plain(y)))}
 for(defaults in listOf(false,true))for(nulls in listOf(false,true)){
 val json=Json{encodeDefaults=defaults;explicitNulls=nulls;allowSpecialFloatingPointValues=true;ignoreUnknownKeys=true}
 check(json.encodeToJsonElement(a)==json.encodeToJsonElement(plain(a)))
 val decoded=json.decodeFromJsonElement(PlainArrays.serializer(),json.encodeToJsonElement(plain(a)))
 check(decoded.bytes.contentEquals(a.bytes)&&decoded.ints.contentEquals(a.ints)&&decoded.items.contentEquals(a.items))
 check(json.encodeToJsonElement(decoded)==json.encodeToJsonElement(plain(a)))
 val count=PlainCount("ok","reserved",null)
 check(count==PlainCount("ok","reserved",null)&&count.hashCode()==DataCount("ok","reserved",null).hashCode());check(count.toString()=="CountDto");check(count._openapiKtPlainHash=="reserved")
 check(json.encodeToJsonElement(count)==json.encodeToJsonElement(DataCount("ok","reserved",null)))
 check(json.decodeFromJsonElement(PlainCount.serializer(),json.encodeToJsonElement(count))==count)
 reject{PlainCount("x","reserved",null)};reject{json.decodeFromString(PlainCount.serializer(),"{\"id\":\"ok\"}")};reject{json.decodeFromString(PlainCount.serializer(),"{\"id\":\"ok\",\"_openapiKtPlainHash\":\"x\",\"kind\":\"count\",\"other\":null,\"extra\":1}")}
 val only=OnlyDto();check(only!=OnlyDto());check(json.encodeToJsonElement(only)==json.parseToJsonElement("{\"kind\":\"only\"}"))
 println("PlainArrays Json defaults=$defaults nulls=$nulls all-static-provider-arrays-versus-Any-nullable-referenceEquals-hashCount-methodCollision-constIdentity checks passed")
 }
}
