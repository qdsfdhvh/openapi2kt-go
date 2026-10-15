# 直接嵌套 String enum 集合字段默认值

[直接嵌套 native enum 默认值](nested-native-enum-collection-defaults.md)另有普通 direct inline outer-own 角色，terminal 为既有 Int/Long/Float/Double/Boolean enum，保持 native wire、原 exact default membership 与既有 checked membership/bounds；不使用本页 lexical String enum 或 Unrecognized API，也不扩张 String bounds/uniqueness。

普通 DTO 字段可以在原已接受的 finite direct inline List / typed `Map<String, T>` 树的外层声明自己的 `default`，terminal 叶为 plain、无 format 的显式 String 或 typeless String enum。List/List、List/Map、Map/List、Map/Map 和更长有限混合路径使用同一规则。default 不会授权原本不支持的 schema。

```json
{
  "type": "array",
  "items": {
    "additionalProperties": {
      "type": ["string", "null"],
      "enum": [null, "ready", "waiting", "null"]
    }
  },
  "default": [{"first": "ready", "nil": null}, {}]
}
```

该 schema 作为可选普通 `labels` 字段时，保留真实词法 enum `LabelsInnerValue`，字段类型为 `List<Map<String, LabelsInnerValue?>>?`。叶仍拥有原 entries、`serialName`、`fromSerialName`、serializer 和 `Unrecognized` API；不会投影成 `List<Map<String, String>>`。缺失 JSON 使用 typed initializer，explicit null 保持 null，显式集合整体替换默认集合。非 null initializer 每次求值通过绑定最终实际 enum TypeRef 的 typed JSON helper 解码独立集合树；空集合保留完整泛型，null 保留 nullable 值。List/Map 的公开接口不承诺深度不可变。

原六种 builtin 叶和 date/date-time 叶分别遵循[嵌套 builtin](nested-builtin-collection-defaults.md)与[嵌套 temporal](nested-temporal-collection-defaults.md)合同，它们的 literal、codec 与检查保持原义。[单层 String enum 集合默认值](string-enum-collection-defaults.md)及其[叶 bounds](string-enum-collection-bounds.md)是独立范围。

## 原词表、nullability 与字段 presence

每个非 null default 叶必须是属于原始 enum 词表的 JSON String，两种 validation 设置均如此。未知 default 在原 member Pointer 拒绝；运行时仍可通过原 `Unrecognized` 或 `fromSerialName` 接收并回写未知 String。`--validation` 不会关闭 String enum 词表，也不会新增 native enum membership 检查。

每层 List、Map 和 enum 叶的显式 nullable intent 独立。default 中的 null 需要对应 schema 节点显式 nullable；普通字段 optional 所产生的 Kotlin nullable view 本身不授权 null 注解。叶沿用[显式 nullable String null-member 语法](string-enum-null-members.md)：至多一个 null 且至少一个 String，保留原成员索引，null 计入 240 个 raw members 上限。JSON null 与词表中的 String `"null"` 保持区别。容器 nullable 不传给内层叶，内层 nullable 不创建 Patch。

required 字段，包括 required nullable 字段，完整验证注解后仍保留 required constructor 参数，没有 initializer。`explicitNulls=false` 保留 serializer 原有的 missing-required-null 推断。ordinary inline request 中 optional explicitly nullable 字段完整验证后仍为 `Patch.Absent`；缺失键不会变成默认更新。Patch.Null 和 Patch.Value 保留 bare JSON wire。见[请求 presence](request-presence.md)。

普通 shared/nested DTO、ordinary inline request/response/SSE payload DTO 可以消费这个字段 profile。原已支持的 ordinary object allOf 保留 later-field override，并在覆盖或 initializer suppression 前验证每个原 discarded 注解；非法词表、default、约束或预算不能随覆盖而消失。不会增加跨贡献 collection/enum checks 聚合。union/hierarchy 字段保留原排除。

## 完整 schema 验证与 source fidelity

只授予普通 direct inline 字段自己的 outer default 角色。empty/null 注解同样先验证整个 schema 树、所有容器和完整原词表。内层容器/叶的 default、root/named/ref/wrapper/target/chain defaults 和 default inheritance 均不属于此 profile。引用一个自身有合法 direct 字段的 DTO 不会授予引用叶或其 collection root 默认值。

树中只使用原已接受的 inline List 和 typed Map grammar；typed dictionary 可以省略 `type`，或声明 object/nullable-object，且不能引入非空 fixed properties/required。open/Boolean dictionary、untyped JSON、prefixItems、composition wrapper、recursive tree 和 general intersection 保持排除。新增 terminal 仅 plain unformatted String enum：numeric/Boolean enum、native/formatted/provider/const/value-class/DTO/union/hierarchy 叶及引用/named 叶默认值保持原拒绝边界。

原 JSON kind、词表成员和 schema 节点是验证依据。错误位于原始 `/default/<index>/<escaped-key>/...` Pointer；每层 Map 按原 source order 检查，成功后分别排序该层 keys。`$ref`、`type`、`/`、`~` 等 dictionary keys 始终是 opaque data，不当作 schema 或引用解释。数组顺序和 enum 原 String token 保留。

## Validation 与资源

`--validation` 下，每层 List 的 min/maxItems、typed Map 的 min/maxProperties 同时约束原 default 与 supplied typed 值。empty 集合仍须满足最小数量；null 只跳过自身 non-null 检查。constructor、Data copy、automatic serde、Patch.Value、typed HTTP pre/postflight 与 SSE payload 使用既有 recursive count checks。裸集合赋值、独立 enum codec、后续 mutation 和 requestBuilder 替换不是 schema validation 屏障。

**嵌套 String enum 叶的 minLength/maxLength/pattern 和 numeric bounds 仍拒绝**，包括 empty/null default。[单层 String enum bounds](string-enum-collection-bounds.md)不授权嵌套角色。enum 列表和容器列表的 `uniqueItems:true` 仍拒绝；`uniqueItems:false` 仅沿用 validation 下 otherwise-supported arrays 的 effect-free no-op grammar。既有 builtin6/temporal literal、checks 和 uniqueness 边界保持不变。没有 validation 的 constrained schema 保持拒绝；所有 NoDrop 检查仍执行。

资源上限保持不变：输入 8 MiB、source depth 48、checked collection edges 16、analysis work 16,384、raw enum members 240、serialized constructor slots 240。每个原 default 在 canonical allocation 前独立 preflight 整棵树的 16,384 JSON values（含容器/null）和 32 KiB compact JSON；canonical JSON 另有独立 32 KiB 上限。keys 与 scalar 内容都计 bytes，default 值遍历也受 depth 48 限制。required/Patch/discarded 注解同样计预算。超限在原 Pointer 报错；生成失败不返回部分 artifacts/name-lock/warnings，native 输出仍按既有事务保留旧 owned 与 handwritten 文件。单项宽度/深度/bytes 边界不构成最大宽度×最大深度、compiler heap 或平台保证。

## 实际类型、优化与兼容

Initializer helper 绑定优化/重定位后的真实 lexical enum owner 和最终 TypeRef。原叶声明仍是独立 references 的来源；不会创建伪 enum owner 或借字符串替换生成 classifier。dedup、merge、hoist 和[name-lock](name-lock.md)使用实际 type/nullability/check tree、原词表及有效 canonical default identity。每层 Map key 重排保持输出与锁身份；改变有效 default、nullable 层、保留 counts 或实际 enum String vocabulary/order 不能复用不同的冻结 shape。required/Patch 注解未产生 initializer 时不引入 runtime default identity。

这是对原先返回 `E_SCHEMA_UNSUPPORTED` 的 direct nested enum 字段自身默认值的增量准入。本合同只覆盖 direct inline 字段；同一 nested enum tree 形状经 ordinary direct `$ref` 字段自身 default 的准入由 [referenced String enum collection field defaults](referenced-string-enum-collection-defaults.md) 另行授予，本页 scope 排除不因此失效。既有 default-free nested enum、single-layer defaults、builtin6/temporal controls 的 Kotlin API、wire、输出和锁格式保持不变。新输入的消费者需要重新生成并编译。范围关联 M01/M03/M04/M07/M08、已有 M12/V05 和 typed A03/A05/A08 消费者；general D03、其他 default/constraint 角色、平台与发布验收仍开放。

## 验证入口与证据状态

Feature 名为 `nested-string-enum-collection-defaults`。[合成 fixture](../tests/fixtures/http/nested-string-enum-collection-defaults.json)与[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedStringEnumCollectionDefaultsContracts.kt)定义消费输入和断言。Core 入口是[typed trees](../core/nested_string_enum_collection_defaults_test.go)、[grammar/guards/resources](../core/nested_string_enum_collection_defaults_guards_test.go)与[实际 owner/optimizer/identity](../core/nested_string_enum_collection_defaults_identity_test.go)；[native process tests](../internal/cli/nested_string_enum_collection_defaults_process_test.go)检查 core/CLI parity、只读与失败保旧。验证矩阵包含四种树、Data/Plain × validation off/on、原成员 Pointers、required/Patch/discarded 验证、空/null/nullable grammar、opaque keys、每层 canonicalization、资源与 frozen-lock identity。最小 empty nested String enum 输入保留为原样正例，其他 builtin/temporal 拒绝边界保持。

专用 route 为 `bash scripts/test.sh feature nested-string-enum-collection-defaults`，其[feature script](../scripts/kotlin/features/nested-string-enum-collection-defaults.sh)生成新鲜 Data/Plain × validation 项目，编译并运行实际 enum classifier/descriptor、四种 Json 设置的 missing/null/default/unknown、fresh tree、constructor/copy/serde counts、优化/冻结锁、typed HTTP 和 fragmented live SSE 生命周期检查。该合同也接入 `models` 和完整 `http` 门禁。

source/default 深度预算约束生成器的处理工作量；深层 lexical enum binary names 仍可能超过主机目录的单文件名上限。feature 的 source48 探针编译到 jar 并让原 JavaExec 加载同一 jar，保留完整 classifier 与原树深度。该 route 的产物配置不表示其他目录输出配置、Android/KMP、最低依赖或所有资源组合都已验收。

`OPENAPI_KT_PREPARE_ONLY=1` 仅执行 native generation/check 与消费项目准备，不能记为 Kotlin compilation/runtime PASS。一次有界 feature 通过也不关闭完整 D03 或新发布候选门槛。
