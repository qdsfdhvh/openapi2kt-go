# 直接嵌套 native enum 集合字段默认值

普通 DTO 字段可以在原已接受的 finite direct inline List / typed `Map<String, T>` 树外层声明自己的 `default`，terminal 叶仅为既有 integer/number/Boolean enum，映射为 Int、Long、Float、Double 或 Boolean。List/List、List/Map、Map/List、Map/Map 和更长有限混合路径使用同一规则。default 不授权原本不支持的 schema，也不创建 enum wrapper 或新 codec。

```json
{
  "type": "array",
  "items": {
    "additionalProperties": {
      "type": ["integer", "null"],
      "enum": [1, 2]
    }
  },
  "default": [{"first": 1, "nil": null}, {}]
}
```

该 schema 作为普通可选 `codes` 字段时，类型为 `List<Map<String, Int?>>?`，有 typed initializer。缺失 JSON 使用默认值，explicit null 保持 null，显式集合整体替换默认树。每次 initializer 求值通过绑定最终实际 TypeRef 的 typed JSON helper 解码独立树；空集合保留完整泛型，null 保持 nullable 值。公开 List/Map 接口不承诺深度不可变。numeric/Boolean JSON wire 保持 native scalar，不使用 String enum 的 `serialName`、entries 或 `Unrecognized` API。

[单层 native enum 集合默认值](native-enum-collection-defaults.md)、[嵌套 builtin](nested-builtin-collection-defaults.md)、[嵌套 temporal](nested-temporal-collection-defaults.md)和[嵌套 String enum](nested-string-enum-collection-defaults.md)分别保留其原合同。

## 原始词表与 nullable/presence

所有非 null default 叶都必须有正确原 JSON kind、可表示为映射的 Kotlin scalar，并严格属于**原始 enum 词表**，validation off/on 均如此。numeric membership 在 Float/Double rounding 之前使用有界 exact source 数值；enum `[1]` 不接纳 Float default `1.000000001`，即使映射结果相同。整数原 token grammar、Int/Long 范围、native enum 重复规则、数值精度/exponent 边界保持不变。

每层 List、Map 和 terminal 的显式 nullable intent 独立。注解 null 需要对应 schema 节点显式 nullable；optional 字段造成的 Kotlin nullable view 不授权注解 null。容器 nullable 不传给叶，叶 nullable 不创建 Patch。nullable 叶不必在 enum 中列出 null；若原词表含一个 null，则仍仅由既有 [validation-only null-member 合同](native-enum-null-members.md)接纳，至少一个非 null scalar、最多一个 null，保原成员索引，null 计入 raw enum 240 上限。null 不拯救空的非 null numeric-bound 交集。

required 字段（含 required nullable）完整验证注解后仍保留 required constructor 参数，没有 initializer；`explicitNulls=false` 的 missing-required-null 推断不构成严格 required-key validation。ordinary inline request 的 optional explicitly nullable 字段完整验证后仍为 `Patch.Absent`，缺失键不变成默认更新；Patch.Null/Value 保留 bare wire。见[请求 presence](request-presence.md)。

普通 shared/nested DTO、ordinary inline request/response/SSE payload DTO 可消费这个字段 profile；List/Map 内的普通 DTO 仍保持既有 nominal validation barrier。ordinary object allOf 保留 later-field override：在覆盖、required/Patch suppression 或任何优化前，验证每个原 discarded 注解、完整词表、约束与资源预算。不能用覆盖删除 enforcing checks；既有 checked NoDrop 可先在原 enforcing enum Pointer 拒绝，再进入 discarded default 遍历，而不是改写其错误优先级。不扩张跨贡献 collection/enum checks 聚合或 union/hierarchy 字段角色。

## Validation、原始 source 与 NoDrop

validation off 时 supplied 值保留 native scalar 近似与原叶 `W_ENUM_NATIVE_APPROXIMATION`，**严格 annotation membership 不等于关闭运行时词表**。`--validation` 时既有 mapped membership/bounds metadata 沿 recursive typed consumer tree 检查非 null supplied 值，checked source 不再发该近似 warning。constructor、Data copy、automatic serde、Patch.Value、typed HTTP pre/postflight 与 SSE payload 使用原检查边界。裸集合赋值、独立 alias/scalar codec、后续 mutation 或 requestBuilder 替换不成为 schema validation 屏障；原 HTTP/SSE wire 与生命周期合同不变。

`--validation` 下每层 List min/maxItems、typed Map min/maxProperties，以及既有 numeric enum inclusive/exclusive bounds 和 Int/Long factors 同时约束 default 与 supplied 值。default 即使属于原词表也须满足 retained bounds。Float/Double supplied membership 和 bounds 保持 mapped finite-value 语义，不新增 epsilon 或 exact runtime decimal 比较。floating enum `multipleOf`、Boolean numeric/String bounds、enum 或容器 `uniqueItems:true` 保持拒绝；`uniqueItems:false` 仅沿用 validation 下 otherwise-supported array 的 effect-free no-op grammar。**不扩张 String bounds/pattern、enum uniqueness 或一般约束聚合。** 无 validation 的 constrained schema 仍拒绝。ignored extras、raw/stringification projection、discarded checks 的 NoDrop guards 保留。

empty/null default 也先验证整个 schema 树及完整原词表，再遍历值。错误保留原 `/default/<index>/<escaped-key>/...` Pointer、原 enum 成员索引或约束 Pointer；每层 Map 按 source order 检查，成功后才分别 canonicalize keys。`$ref`、`type`、`/`、`~` 等 default keys 始终是 opaque data，不当作 schema/ref 解读。数组顺序保留。

## 范围与资源

只增加普通 direct inline 字段自己的 outer default。内层容器/叶、root/named/ref/wrapper/target/chain default、default inheritance 仍不属于本 profile。允许的 typed Map 可省略 type，或声明 object/nullable-object；非空 fixed properties/required、Boolean/open dictionary、untyped JSON、prefixItems、composition wrapper、recursive tree 保持原边界。不能由相同 native classifier 推导 caller-native/provider、const、value-class、DTO、union、hierarchy 或 named/borrowed terminal 的新 default 准入。

所有 caps 不变：输入 **8 MiB**；source depth **48**；checked collection edges **16**；analysis work **16,384**；serialized constructor slots 和 raw enum members 各 **240**。每个原 default 在 canonical allocation 前独立 preflight **16,384 JSON values**（含容器/null）和 **32 KiB compact JSON**；成功 canonical JSON 另有独立 **32 KiB** 上限，keys/scalar 内容都计 bytes，值遍历也受 depth48 限制。required/Patch/discarded 注解同样计预算。超限在原 Pointer 拒绝，不返回部分 artifacts/name-lock/warnings；native CLI 按既有事务保留旧 owned 与 handwritten 文件。单项 source48/checked16/slots240/bytes32KiB 探针不证明最大宽度×最大深度、compiler heap 或平台保证。

## 实际类型、优化与迁移

Helper 绑定 dedup/merge/hoist 后最终实际 scalar/container TypeRef，不伪造 owner，也不替换 Kotlin 文本拼出 classifier。实际 alias/type、独立 nullability、retained check tree 和 effective canonical default 共同参与优化及 [name-lock](name-lock.md)身份。每层 Map key 重排保持输出和锁身份；改变有效 default、nullable 层或 retained checks 不能复用不同的 frozen shape。native enum 的 runtime mapped-check 身份与 default 的 strict source membership 是两个判断域；不把所有 raw 数字拼写当作新的运行时身份。required/Patch 注解没有 initializer 时不新增 runtime default identity。原 default-free、single-layer 和其他叶合同的输出、occurrences、opaque lock format、ownership/CAS 均不改变。

迁移影响：原先在原叶 enum 返回 `E_SCHEMA_UNSUPPORTED` 的此类 direct nested own-default 输入现在可生成，需要重新生成并编译消费者。已有 accepted 输入的 native Kotlin call sites、numeric/Boolean wire 与 validation-off 近似保持；validation 仍检查既有 mapped membership/bounds，不新增一般运行时 validator。范围关联 M01/M03/M04/M07/M08 与既有 M12/V05/A03/A05/A08，**不是完整 general D03/D04/D05、统一 candidate、平台或发布验收**。

## 验证入口

[native process/source tests](../internal/cli/nested_native_enum_collection_defaults_process_test.go)、[bounds/null/资源](../internal/cli/nested_native_enum_collection_defaults_limits_test.go)及 [Unix FIFO/CAS](../internal/cli/nested_native_enum_collection_defaults_unix_test.go)定义 Core/actual CLI parity、Data/Plain × validation、原 Pointer/kind/exact membership、required/Patch/discarded 预验证、逐层 canonicalization、只读/no-op/check/validate bytes/modes/mtimes 与失败保旧。可用 `go test ./internal/cli -run '^TestNestedNativeEnumCollectionDefaults'` 执行；这是 Go/native/source 证据，不是 Kotlin compilation/runtime。

[Core typed trees](../core/nested_native_enum_collection_defaults_test.go)、[原 grammar/guards](../core/nested_native_enum_collection_defaults_guards_test.go)、[独立资源边界](../core/nested_native_enum_collection_defaults_limits_test.go)、[实际优化/锁身份](../core/nested_native_enum_collection_defaults_identity_test.go)和[合成 fixture](../tests/fixtures/http/nested-native-enum-collection-defaults.json)保留原接受/拒绝边界；[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedNativeEnumCollectionDefaultsContracts.kt)提供消费断言。feature 入口为 `bash scripts/test.sh feature nested-native-enum-collection-defaults`，提供 Data/Plain × validation 四 profile。`bash scripts/test.sh models` 经 [exporter](../core/smoke_test.go)仅导出 fixture 的普通模型，将 `NestedNativeEnumOwnDefaultsModelContracts.kt` 断言接入既有 `EnumContextContracts.main`（`runEnumContextContracts`），不新增模型 runtime suite；`bash scripts/test.sh http` 生成/check 完整 fixture，由既有 `runHttpContracts` 对应的 `HttpContracts` suspend registry 调用完整 HTTP 消费者 body。消费检查需分别验证四种 Json 设置的 missing/null/default/replacement、fresh tree、required/Patch、constructor/copy/serde checks、actual optimizer aliases/descriptors、typed HTTP 与 fragmented live SSE failure closure/reuse。`OPENAPI_KT_PREPARE_ONLY=1` 仅控制 native generation/check 和项目准备；不能将准备或其他 feature 的既有 JVM 结果记为本 feature Kotlin compilation/runtime PASS。一般平台、最低依赖组合、统一 candidate 与 release gates 继续独立验收。
