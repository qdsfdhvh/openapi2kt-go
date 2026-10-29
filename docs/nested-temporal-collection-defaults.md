# 直接嵌套 temporal 集合字段默认值

Component root annotations use the separate [finite nested component factory contract](nested-component-defaults.md); these factories do not become field initializers.

[直接嵌套 native enum 字段默认值](nested-native-enum-collection-defaults.md)另有 ordinary outer-own 角色，terminal 为既有 Int/Long/Float/Double/Boolean enum，保 native scalar wire、原词表与 checked membership/bounds；不改变本页 temporal grammar、lexeme 身份或 bounds/uniqueness 排除。

[Referenced temporal collection 字段默认值](referenced-temporal-collection-field-defaults.md)另有独立 ordinary direct `$ref` 字段自身注解角色，经原已接受的 named/internal finite List/typed Map 路径绑定 actual LocalDate/Instant；本页 direct-inline 语法与叶引用排除不变。

普通 DTO 字段可以在 finite direct inline List / typed `Map<String, T>` 树的外层声明自己的 `default`，直接 String `format: date` / `date-time` 叶分别使用 `kotlinx.datetime.LocalDate` / `kotlin.time.Instant`。原有六种 builtin 叶继续遵循[嵌套 builtin 默认值合同](nested-builtin-collection-defaults.md)；plain unformatted String enum 叶的外层字段默认值另遵循[嵌套 String enum 合同](nested-string-enum-collection-defaults.md)，保真实词法 enum 与运行时开放词表，不授予嵌套 enum String bounds/pattern。List/List、List/Map、Map/List、Map/Map 和更长有限混合路径遵循相同规则。

```json
{
  "type": "array",
  "items": {"additionalProperties": {"type": "string", "format": "date"}},
  "default": [{"a": "2024-02-29"}]
}
```

可选普通字段生成 `List<Map<String, LocalDate>>?` 与 typed initializer。缺失 JSON 使用默认值，explicit null 保持 null，显式提供的集合整体替换默认集合。每次 constructor/default 求值通过既有 typed JSON helper 与标准 temporal serializers 解码一棵独立集合树；空集合保留完整泛型类型。公开 List/Map 接口不承诺深度不可变。

每一层容器与叶的原 nullable 独立。注解中的 null 必须有对应 schema 的显式 nullable；字段 optional 本身不授权注解为 null。required 字段仍没有 initializer，但完整注解须先验证。`explicitNulls=false` 保留既有 missing-required-null 推断。inline request 的可选显式 nullable 字段仍为 `Patch.Absent`，不会把 default 变成缺失键更新；Null/Value 的 bare wire 保留，内层 nullable 不创建 Patch。见[请求 presence](request-presence.md)。

## 原始注解、ISO 与身份

所有原 schema 节点都要验证，即使 default 为 empty/null。本合同只授予普通 DTO 字段自己的 outer default 角色。普通 named DTO 直接 List/Map 字段上的 date/date-time 子节点自身注解现按独立的[子节点合同](inline-builtin-collection-child-defaults.md)校验；更深层容器/叶、target/chain/root default、named/ref 集合默认值、`$ref` temporal 叶、native-date/provider/value-class、formatted enum/const/union/hierarchy 叶及组合包装保持各自的准入限制（inline DTO object 叶改用独立的[object collection 合同](nested-object-collection-defaults.md)）；plain unformatted String enum 叶仅由上方独立合同授予 outer field default，不改变 temporal grammar/checks。本合同不扩大 general schema roles、intersection 或 default inheritance。

LocalDate 使用既有合法四位年份 ISO date 规则，包括真实闰日检查。Instant 除既有 RFC3339 date 检查外，共用[单层集合限制](primitive-array-defaults.md#date-and-timestamp-elements)：四位年份、uppercase T/Z、两位时间字段、点后最多九位小数，Z 或范围在 ±18:00 的合法 offset。逗号小数、单数字小时、过量精度、坏闰日和范围外 offset 在 generation 时拒绝，防止 initializer 首次执行时才失败。不会新增 timezone 等价 default identity 或日期排序规则。

原 default 成员必须有正确 JSON kind。错误位于原始 `/default/<index>/<escaped-key>/...` Pointer；每层 Map 按原 source order 验证，成功后才分别 canonicalize key 顺序。`$ref`、`type`、`/`、`~` 等 key 始终是 opaque data。Array 顺序与 temporal 原字符串参与 default identity。Instant runtime wire 沿用 ISO UTC 规范化，源字符串不保证原样回写；同一 instant 的不同源 offset/小数拼写不视为同一 default 身份。既有 exact integer 与 mapped Float/Double 身份保持原义。

Helper 使用合法 JVM 方法名中的 LocalDate/Instant key，serializer 泛型绑定最终实际 TypeRef。dedup/merge/hoist/name-lock 使用既有 typed tree/check/default 域；canonical keys 重排保持输出与锁，改变有效 default、nullable 层或保留检查会改变 shape。required/Patch 没有 initializer 的注解不产生 runtime default identity。锁格式、occurrence 与 canonical replay 规则不变。

普通 shared/nested DTO、ordinary inline request/response/SSE payload DTO 可消费此 profile。既有 ordinary object allOf 保留 later-field override；原 discarded 注解仍须验证，不能通过覆盖或输出抑制删除非法注解/检查。union/hierarchy 字段和跨贡献 collection checks 聚合保持拒绝。

## Validation 与资源

`--validation` 下，每层 List min/maxItems、typed Map min/maxProperties 检查同时作用于原 default 与 supplied typed 值。null 只跳过自己的 non-null checks，empty 集合仍须符合最小数量并验证整个叶 grammar。constructor、Data copy、automatic serde、Patch.Value、typed HTTP 与 SSE 使用既有 recursive checks。

Temporal `uniqueItems:true` 沿用[映射值唯一性合同](temporal-unique.md)，包含 default 中等价 Instant 的 offset/fraction 判重；Temporal String/numeric bounds 仍拒绝，包括 empty/null default；源类型为 String 不引入 String length/pattern。`uniqueItems:false` 沿用既有 validation metadata/no-op 合同。原 builtin 叶检查与所有 NoDrop guards 保留。裸集合赋值、独立 builtin codec、调用方后续 mutation 和 requestBuilder 替换不形成 schema validation 屏障。

原预算完全不变：输入 8 MiB、source depth48、checked collection depth16、serialized slots240、analysis work16384，以及现有 schema/alias/name-lock caps。每个原 default 整树在 canonical allocation 前独立 preflight 16,384 values（含 containers/null）与 32 KiB compact JSON；canonical JSON 再独立限制 32 KiB。key 与标量内容均计 bytes，required/Patch/discarded 注解同样计预算。失败不返回部分文件/锁/warnings，native generation 不改旧 owned 输出或 handwritten 文件。未提高 compiler heap，不承诺最大宽度×最大深度或 Android/KMP。

## 兼容与验证入口

这些直接嵌套 temporal 默认值过去返回 `E_SCHEMA_UNSUPPORTED`；既有单层 temporal/default-free 集合的 API、wire 与锁身份保留。该增量只关闭 D03 的 direct nested temporal 子项，扩张 M02/M03/M04/M07，沿用 M12/V05 和 A03/A05/A08 的 typed consumer 合同。一般 default refs、nominals、roots 等仍 open。

[Core 合同](../core/nested_temporal_collection_defaults_test.go)保留原最小 List/Map/date 输入与两个旧 temporal 负例的 verbatim graduation，验证四种树、更长路径、原 Pointer/kind 与资源。[guards/identity](../core/nested_temporal_collection_defaults_guards_test.go)验证 excluded roles、validate-before-suppression、deep counts、实际 optimizer classifier 和冻结锁。[native tests](../internal/cli/nested_temporal_collection_defaults_process_test.go)检查实际 CLI/core 源与锁 parity、occurrences、只读/no-op bytes/modes/mtimes、失败与 8 MiB 边界保旧。

`bash scripts/test.sh feature nested-temporal-collection-defaults` freshly generates Data/Plain × validation 四个项目，运行[消费者](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedTemporalCollectionDefaultsContracts.kt)的四 Json 配置、constructor/copy/serde/default/presence、actual aliases/classifiers/descriptors、optimizer/lock、typed HTTP wire/非法响应和 fragmented live SSE failure closure/client reuse。独立 source48/checked16/slots240/default32KiB consumers 使用原资源上限。完整 `http` 入口接入 unconstrained [fixture](../tests/fixtures/http/nested-temporal-collection-defaults.json)。`OPENAPI_KT_PREPARE_ONLY=1` 只进行 native generation/check 与项目准备，不是 Kotlin compilation/runtime 证据。其他平台、最低依赖与完整 D03 矩阵仍需独立验收。
