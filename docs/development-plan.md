# openapi2kt-go 开发计划

本计划定义完整路线图；当前已有可用的 JSON 模型生成、限定 HTTP/SSE、本地配置和安全输出，但尚未完成全部阶段及完整版本验收。

先按目的阅读：

- **查看还有什么没做**：[实现状态与剩余工作](remaining-work.md)，区分功能缺口、统一验收与后续／范围外工作。
- **查看阶段进度**：[第 7 节](#7-开发顺序与阶段验收)，按 P0–P8 汇总已有实现和剩余退出条件。
- **查看精确输入/API/wire 边界**：[兼容矩阵](compatibility-matrix.md)及其中链接的功能合同；有界 profile 不代表整个矩阵 ID 完成。
- **交付当前 alpha**：[JSON 发布候选清单](json-release-checklist.md)，范围固定为现有 JSON profile、macOS ARM64 CLI 和 Kotlin/JVM。R01 范围冻结不等于 R02–R08 运行和发布门槛通过。

下面架构和完整 DoD 保留为长期设计；具体 CLI/config 示例遵循现有接口。新增功能合同应同步更新剩余工作，不再在本页堆叠重复的增量状态段落。

## 1. 目标与发布边界

用 Go 构建独立运行的 OpenAPI → Kotlin 客户端生成器。首期 JSON 及新增的[有界 YAML Adapter](yaml-input.md)均接入同一语义流水线，支持本文约定的 OpenAPI 3.0/3.1 子集。

产品质量分三层验收：

1. **wire 行为**：请求参数编码、JSON 序列化/反序列化、Patch 三态、枚举和 union fallback、SSE 事件与取消行为。
2. **Kotlin 使用面**：包布局、类型与方法命名、可空性、默认值、构造方式、辅助 runtime、配置开关。
3. **工具行为**：多规格、file/stdin 输入边界、共享命名锁、增量输出、过期文件清理、失败回滚、Gradle 源集与任务接入。

以类型、调用方式和 wire 行为作为契约，不将某种源码排版视为正确性的依据。发布后的行为变更必须有契约测试及升级说明。

首个完整版本包含 SSE、体积优化、命名锁、file/stdin 输入和本地 CLI 的 Kotlin 消费接入；规格下载不属于产品范围；正式 Gradle 插件不在交付范围。可提前发布能力受限的 alpha，但需明确其支持范围。YAML 不属于首期范围。

### 不在首期承诺内

- 完整 OpenAPI/JSON Schema validator、Swagger 2.0、所有 3.1 关键字。
- 任意外部文件/HTTP `$ref`、远端引用自动递归下载。
- 服务端代码、其他客户端语言、GUI、动态模板插件系统。
- OAuth 流程实现、应用业务鉴权、自动安装应用的 Ktor 插件。
- multipart、cookie、callbacks、webhooks、links 等暂未纳入契约的能力。
- 正式 Gradle 生成插件、自动下载或安装 CLI 的插件机制；构建示例只消费本地 CLI 输出。

遇到不支持的影响 wire 的构造要给定位诊断；不能生成一个缺参数或返回 `Unit` 的“成功结果”。已确认的近似映射见[兼容矩阵](compatibility-matrix.md)。

## 2. 设计原则与验证状态

本文主要定义完整版本的设计，不是全功能通过报告。P1 已有原生 Go core/CLI、单 owner 安全写出、Go API/进程/事务测试和完整已接受 serializer 的 Kotlin 编译/运行 smoke；所有更大范围的支持声明仍需要对应证据。

- 开放枚举、union、字典、value class、Patch、SSE 和模型压缩均属于产品范围。
- 命名与 shape 复用需要全局规划；请求/响应上下文会影响类结构，不能只对原始 schema 做哈希去重。
- 生成只接收本地文件/显式stdin；下载、鉴权、重试和缓存由外部工具负责，不内置网络适配或curlwrapper，编译不触发隐式联网。
- 输出清单和异常回滚是工具的核心可靠性契约。
- inline 对象请求及其递归 inline 子树的 `required + nullable` 使用 `T?`，不设默认值；`explicitNulls=true` 时，即使 `encodeDefaults=false` 也发送 null 键。共享组件、响应与根 union 请求仍保留普通 `T? = null`；`explicitNulls=false` 不保证 null 键，必须分别测试这些上下文及 JSON 配置。
- 日期时间已映射到 `kotlin.time.Instant` 和 `kotlinx.datetime.LocalDate`；见[特殊标量合同](special-scalars.md)。最低依赖版本仍须由实际编译矩阵确定。

测试数据和变更验证要求见[贡献指南](../CONTRIBUTING.md)。

## 3. 总体架构

```text
本地 CLI 适配层
          │ 配置合并、bounded 本地file/stdin读取（规格获取是外部工具）
          ▼
Document Loader：JSON / 有界 YAML Adapter
          ▼
Raw Document + Source Map
          ▼
OpenAPI 解码 → 本地引用图 → 规范化 → 用途分析
          ▼
类型/operation 计划 → shape 收集 → 稳定命名/命名锁
          ▼
Kotlin IR → 可选去重/合并 → 模型风格 → Kotlin emitter
          ▼
Artifact Set + 新命名锁 + Diagnostics + Metrics
          ▼
Output Transaction：校验、暂存、提交、清理、失败恢复
```

### 3.1 Go module 与包边界

公开小型 core API，复杂决策留在内部包，CLI/文件事务单独适配：

```text
go.mod                    # 当前仅依赖标准库
core/                     # 公开 request/result/diagnostic API 与合同测试
cmd/openapi-kt/            # native CLI 入口
internal/
  document/               # ordered JSON、精确数值与输入限额
  naming/                 # Kotlin 名称、literal、Pointer
  schema/                 # catalog、source views、规范化与有界图
  kotlin/                 # typed declarations 的稳定输出
  diagnostic/             # 安全结构化诊断
  cli/                    # input/flags/report/exit
  output/                 # manifest、锁、同盘暂存、回滚
  testutil/               # 测试专用原 fixture 清单
tests/fixtures/           # 独立合成 JSON 与完整 Kotlin 快照
samples/                  # Kotlin serializer consumer smoke
scripts/                  # 离线验证与发布检查
```

生成内核不依赖 Gradle/JVM，不联网、读环境或写文件；Go tests 无需 Rust/Cargo。Kotlin 编译/runtime 使用显式 smoke harness；Kotlin 工程通过显式调用本地 CLI 和配置生成源目录接入；不新增正式 Gradle 生成插件。

### 3.2 小 Interface，复杂 Implementation 留在 Module 内

核心对外保留一个生成入口，概念契约如下（非可直接编译的 API）：

```text
generate(GenerationRequest {
    document_bytes, input_format, source_label,
    options, previous_name_lock
}) -> Result<GenerationArtifacts, Diagnostics>
```

结果包含相对文件路径与内容、新命名锁、生成摘要、诊断；不直接暴露几十个 builder。初期中间 IR 是内部实现，不承诺稳定公共 API。`source_label` 默认使用相对路径；报告不含原文或绝对路径。

CLI 的职责是加载文件/配置、调用内核、提交结果。核心用同一入口完成内存快照测试，文件事务另做故障注入测试。不要为尚不存在的十种后端设计插件框架。

### 3.3 JSON / YAML 的真正扩展点

- `InputFormat = Auto | Json | Yaml` 已实现 JSON 和[有界 YAML profile](yaml-input.md)，通过 vendored `go.yaml.in/yaml/v3` v3.0.4 解析节点，不按行拼凑。
- Loader 只负责语法 → ordered JSON 兼容树及安全诊断。两个格式进入同一个引用解析、规范化和生成入口。
- `auto` 按 `.json`/`.yaml`/`.yml` 后缀选择 parser；无后缀（含 stdin）是 JSON，未知后缀要求 `--input-format`。禁止 JSON 失败后降级为另一种解释。
- 语义定位保留 JSON Pointer；YAML node 错误提供可用行列，原 parser 错误替换为安全消息，不伪造 general source span。
- YAML 的重复 key/非 String key、anchor rebinding/cycle、alias node/byte/depth budgets、merge/tag/隐式日期/多文档/UTF8 策略以合同为准；完整 YAML 仍不承诺。
- emitter 无 YAML 分支。同义接受输入的 ordered tree、文件/warnings/诊断 parity 与 fresh Kotlin serde/Patch/HTTP 消费通过独立 fixture 检查。

### 3.4 schema 与引用图

当前实现独立的 [terminal builtin scalar schema Pointer 子集](model-semantics.md#terminal-scalar-schema-pointers)，及 [finite List/typed Map targets](model-semantics.md#finite-collection-schema-pointers)：沿真实 component schema edges 借用原节点/source；集合支持直接有限嵌套和 terminal internal scalar-ref 叶，保留 typed checks/root override/inner nullability；[internal collection reference chains](model-semantics.md#internal-collection-reference-chains)另扩张有限 collection-ref hops/嵌套引用，跨 ref/container 原节点 active path 有界拒绝环，不引入 nominal 身份、root component alias 或 HTTP schema anchors。另有 [internal scalar reference chains](model-semantics.md#internal-scalar-reference-chains) 的有限透明 hop 解析：原 terminal/source、最近 nullable override、累计 consumer 高度与请求 Patch；循环仍保持历史排除诊断。既有 named/HTTP references 不变；下述 SchemaId/SCC 是更广图设计，不是该有界实现的能力承诺。完整 F04/V05/V06 与非 YAML 后续计划仍开放。独立 240 浅集合字段和单字段 16 层编译验证不保证广深组合：深泛型宽模型仍可能超出 Kotlin/JVM compiler/class-file 字符串资源，这一既有 direct schema 亦可触发的资源问题须继续处理，不通过新增 cap 或提高 heap 规避。

[Direct internal ordinary DTO references](internal-dto-references.md) 另有 typed-use 有界增量：direct component 的直接 properties 对象树中，内部 `$ref` 绑定既有 nested DTO canonical FQName，保留 nominal 递归屏障与 nullable/Patch。有限内部 ref chain→这些 DTO 已有独立 nearestnullable/fullhop/cacheheight 增量；direct List/Map-owned DTO targets另复用nested Inner/Value与root top-level peer，保持container/leaf nullable独立。独立[component-root aliases→internal DTO](internal-dto-references.md#component-root-aliases-to-internal-dtos)保exported typealias与canonical物理owner/ctor/serializer、最近nullable及完整named/internal hop/cacheheight；未使用的alias也需final constructor-bearing确认，不复制catalog。独立[internal chains→named ordinary DTO](internal-dto-references.md#finite-internal-chains-to-named-ordinary-dtos)支持有限内部hop到direct named构造DTO、真实catalog/canonical顶层owner、nearestnullable/最终namededge与alias缓存预算；另有[internal chains→named DTO aliases](internal-dto-references.md#finite-internal-chains-through-named-dto-aliases)保existing directnamed/internal canonicalowner、nearestaliasnullable/累计全named-internalhop；另有[root aliases→internal chains→named DTO bridges](internal-dto-references.md#component-root-aliases-through-internal-chains-to-named-dtos)共用catalogDFS、canonicalphysicalowner、own/internal/namedintent与全hop/cacheheight；nonordinaryalias/shape expansion不扩张。Hierarchy 与 composition-owned targets、借用集合中的 DTO/enum/union/hierarchy/scalar leaves 已有独立合同，见[兼容矩阵](compatibility-matrix.md)。Ambiguous owners、fixed extras scratch 与更广 shape expansion 仍未覆盖；不等于完整 F04。

IR 必须区分：字段是否 required、值是否 nullable、是否有 default、是否 const，以及 schema 是任意值、不可满足、对象还是字典；不能把这些挤进一个 `Option<T>`。

- 引用使用 `SchemaId`/节点索引，保留来源 Pointer；用访问状态/SCC 控制递归，禁止无限展开。
- 只在 schema/parameter/response 等合法位置解释 `$ref`，不改写 examples/default 中同名的用户数据。
- 本地 Pointer 解码 `~0`/`~1`，引用缺失明确报错；外部引用明确不支持。
- 区分 `UsageContext = RequestBody | ResponseBody | Parameter | SharedComponent | SsePayload`。
- 共享 component 的 canonical 构造器、nullable 默认值与 serializer 保持；普通 HTTP DTO 的 readOnly/writeOnly 字段另按[定向模型合同](directional-models.md)生成 operation-owned Request/PublicResponse 视图，包含受影响的引用、集合和名义递归。无定向需求的引用仍共享模型。一般 Patch 上下文仍用于 inline 对象请求及其递归 inline 子树；定向 Request clone 使用相同 presence 表，根 union 请求、response 和 deepObject 不开启 Patch。
- 规范化保留有损变换诊断。生成器支持某种“近似表示”不等于验证了 JSON Schema 全部约束。

### 3.5 Kotlin IR 与 emitter

Go emitter 使用**小型、强类型声明模型**，统一生成策略，而不是到处重复字符串拼接，也不必做完整 Kotlin AST。

至少覆盖 `TypeRef`、declaration、property、constructor、annotation、函数、常量、类型参数、nullable、import、KDoc、受控代码片段。统一负责：

- Kotlin 标识符、关键字、大小写不敏感文件名碰撞。
- 字符串中的引号、反斜杠、`$`、换行；数值后缀和负数边界。
- 正则字面量与 KDoc 中的 `*/`，防止规格内容破坏源码结构。
- import 歧义、嵌套类型和 typealias 引用。
- 无构造参数的 const-only 模型，不生成非法 `data class`。

[模型风格](model-style.md)已接通 `data_class`/`plain_class` 的 typed owned IR 样式与显式 value methods；结构合并已有下方初始有界 profile，非完整 O04。[Initial resolved ordinary model merging](merge-models.md) 已在 typed IR 上接通 default-off merge/exact FQ exclusions、component-first canonical owners/acyclic resolved references 与 source/noDrop/alias migration；general graphs仍待；[portable regex/literal suffix exclusions](merge-exclusions.md)已实现full-FQ-match与short-generated-name suffix，exact旧语义不变。模型风格和结构合并在 IR 上进行，不对生成后的 Kotlin 文本做正则替换。合并先于包含类名字符串的 plain-class `toString` 输出。

### 3.6 确定性和命名锁

1. 收集所有 components、operation 输入/输出、参数、错误体、SSE payload 的 shape。
2. shape fingerprint 包含类型、wire 名、required/default/nullable、序列化器、校验、请求上下文；通常排除纯说明文字。
3. 集中分配符号，考虑组件名、runtime 保留名、别名、大小写折叠冲突。
4. 读取有版本的命名锁，优先匹配结构，再按已定义 occurrence 规则识别变化；不依赖随机哈希顺序。
5. 输出重命名报告与新锁，在文件提交成功时一起更新。

命名锁格式及 fingerprint 规则要版本化。遇到不支持的格式版本必须报错；后续升级提供转换或显式重建流程，并输出类型名差异清单，不能静默当空锁。

保持已约定的属性声明顺序和 union 分支优先级；只对无序集合确定性排序。“乱序测试”不得打乱具有语义的列表。

## 4. 当前配置与 CLI 契约

以下命令和字段已实现；完整配置/lifecycle的剩余边界见[本地配置](configuration.md)和[剩余工作](remaining-work.md)。

### 4.1 命令

```bash
openapi-kt generate --config openapi-kt.toml
openapi-kt generate --config openapi-kt.toml --spec catalog
openapi-kt generate --config openapi-kt.toml --check
openapi-kt validate --input tests/fixtures/http/basic.json --input-format json
# 外部工具可供流；stdin 不用于多spec配置。
curl -fsSL https://api.example.test/openapi.json | openapi-kt validate --input -
```

- direct `generate`/`validate`/`--check` 读本地file或显式 `--input -` JSON/YAML（auto stdin 为 JSON，YAML stream 须显式format）；配置仅localfile、拒绝stdin marker。移除旧download/URL/header配置；外部producer与退出状态由调用方负责。
- `--check` 计算 Kotlin 文件、manifest、命名锁差异，**不写回**。
- `validate` 是“本工具支持子集及可生成性检查”，不是通用规范认证。
- [同 root named-owner rename](output-owner-rename.md)已支持一次性 repeatable `--rename-owner OLD:NEW`，仅 generate/validate --config；full 映射每个当前 spec，partial 唯一对应 NEW，identity 验证后 no-op。保持现有 v2/v4、exact Kotlin 与锁语义；只更新 metadata、不 reconcile，保留 config 外 peers。
- 退出码：0 成功、1 输入/配置/生成失败、2 CLI 用法错误、3 check 有差异、4 输入IO或文件提交/存储/rename安全拒绝。
- `--diagnostic-format text|json`；机器报告写 stdout，进度日志写 stderr。诊断有稳定 code、severity、阶段、相对来源和 Pointer。
- [CLI warning policy](warning-policy.md) 已实现 `generate`/`validate`/`generate --check` 的 `--deny-warnings` 与 repeatable exact `--allow-warning CODE`；成功生成后在 output IO 前以原 Pointer 拒绝未放行 warnings，保留全部 warnings。默认行为不变；[opt-in CLI metrics](cli-metrics.md) 已实现实际 read/generate/warning_policy/check-or-commit 的 coarse timing 和 planned byte/file counts，纯 core/输出 hash 不变；[配置聚合](cli-metrics.md#configuration-aggregation)已接通 config/read/generate/policy/batch 累计和同root-path helper去重计数，core实际options/parse/analyze/transform/emit子阶段与Darwin/Linux process-lifetime peak RSS已接通；resolve/name交织在analyze内，独立分摊/phase memory仍不提供，不能全局吞异常。

### 4.2 中性配置示例

```toml
config_version = 1

[defaults]
output_dir = "build/generated/openapi"
model_style = "data_class"
model_name_suffix = "Dto"
runtime_package = "org.example.generated.runtime"
deduplicate_models = false # initial operation ordinary DTO/enum subset; see deduplicate-models.md
hoist_shared_shapes = true
qualify_hoisted_names = false
lock_hoisted_names = true
merge_identical_models = false

[[specs]]
name = "catalog"
input = "catalog.json"
input_format = "json"
package_name = "org.example.catalog"
api_client_name = "CatalogApiClient"
exclude_apis = []
```

`catalog.json` 由调用方提供，路径相对配置文件。命名锁由输出 manifest 管理并保存在输出根，不接受自定义 `hoisted_names_lock_file` 路径；具体 version3/4 文件名与生命周期见[命名锁合同](name-lock.md)。

规格获取由外部工具负责，本地配置没有download字段/表。上方外部producer域名仅作示例，不用于CI；无fetch回归只用loopback。

优先级：内置默认 < `[defaults]` < 单 spec < 明确 CLI 覆盖。路径相对配置文件目录解析，而不是调用时 cwd。CLI 生成选项覆盖要求只选择一个 spec（或配置只有一个 spec）；不能在多 spec 下含糊地覆盖 `--package-name`。

[去重](deduplicate-models.md)、[hoist](hoist-models.md)、[merge](merge-models.md)和[命名锁](name-lock.md)均已有 core/CLI/config 有界实现，ordinary inline error DTO 也参与这些操作。当前支持 runtime package、API 排除、client name、hoist qualification/lock，以及 exact FQ、portable regex 和 literal suffix merge exclusions。广义结构优化、partial/disjoint inventory root move和一般多owner迁移仍待；[完整同root single-owner v3→v4 adoption](ownership-migration.md#locked-owner-adoption)与[完整现有 root 显式迁移](output-root-move.md)已提供同版本 v1–v4 原始 bytes/属性/完整 owner/shared/lock 状态保留与两侧 CAS/rollback；显式 disable/reset 与旧/新命名及产物差异报告、[同 root v2/v4 named-owner rename](output-owner-rename.md)已提供，后者要求 Kotlin membership/bytes 和锁 mode/profile/allocations/occurrences 不变，见 D08–D10。配置未知字段报错；不提供插件 DSL 或 `autoGenerateOnCompile` 开关，消费工程自行显式调用本地 CLI。

配置排除正则已采用[可移植 ASCII 子集](merge-exclusions.md)，对生成的 FQ root name 做完整匹配；lookaround/backreference 等明确拒绝。literal suffix 对生成的 short classifier 做大小写敏感匹配。schema `pattern` 则沿用独立的 Kotlin 校验合同，不能与名称排除规则混为一谈。

## 5. 输出、输入和 Gradle

### 5.1 输出事务

[Named-owner adapter](output-ownership.md) 已有 version2 membership/shared-byte/partial-retention/reconcile/lock-time-replan/rollback；Direct --input 保持未启用锁时的 v1。[disjoint-root batch](output-batches.md)、[exact-match v1 ownership adoption](ownership-migration.md)、[共享 runtime](runtime-package.md)和[version3/4 命名锁事务](name-lock.md)均已接通。[遗留事务检测并拒绝](recovery-rejection.md)已实现；[named-owner rename](output-owner-rename.md)已提供同 root v2/v4 metadata-only 事务，精确 manifest+全部 owned lock bytes 在 preflight 和完整 sorted lock set 内作 CAS，保留所有 peers/shared memberships、不 reconcile、不写 Kotlin。[显式完整 root move](output-root-move.md)已提供具体 source/target 的同版本 v1–v4 bytes/属性/归属保留与完整排序两侧锁/CAS/正常失败 rollback；partial/disjoint inventory move、一般partial/multiowner锁迁移和自动恢复仍未提供；[完整同root v3→v4 adoption](ownership-migration.md#locked-owner-adoption)已有独立metadata-only绑定/失败保旧合同，不承诺 crash atomicity。

- 先完成生成及路径冲突检查，再碰目标文件；严禁先删 output 再解析。
- manifest 记录 owner/spec、相对文件路径、内容 hash、共享 runtime owner 集合及格式版本。
- 只删除 manifest 管理的旧文件；保护手写文件，拒绝覆写不归本工具所有的同名文件，提供明确迁移流程。
- 共享 runtime 同包必须同输出根，内容一致；全部 owner 删除后才清理。
- `--spec` 单独生成不能把没选中的其他 spec 当作已删除。普通完整配置用于 owner 判定；清空规格也要有显式 reconcile 路径。显式 owner rename 的 full/partial 均保留当前 config 不存在的 peer owners。
- output 迁移后的旧根不能自动发现。[显式完整 root move](output-root-move.md)只处理选定的源／目标与全 manifest-owned 范围；保留各侧 unowned 文件、原 root/目录与协调 sidecar，不递归清空目录。partial/disjoint inventory 或历史 root 发现仍不支持。
- 对输出根加跨进程锁，跨多个根按稳定顺序加锁；共享根的 spec 串行提交。
- 路径禁止绝对路径、`..`、越界符号链接；包名/规格名不可变成任意磁盘路径。
- staging 在目标同一文件系统；受控提交失败恢复旧代码、manifest 和输出根内由 manifest 管理的命名锁。
- 不承诺跨文件/文件系统 crash-atomic 更新。检测遗留事务后拒绝继续，故障后保留证据，不盲删备份；没有自动 recover 命令。
- 相同内容不改写，保持 mtime；成功后清理暂存文件。

### 5.2 本地 file/stdin 与外部获取

[输入契约](local-input.md)固定direct `--input -` 单流、EOF/8MiB+1 detection/strictJSON/readerror与同core输出；本地多spec配置保留，不消费/replay stdin。旧download命令与下载元数据明确拒绝，这是CLI/config破坏性迁移，不改变generated Kotlin/wire或manifest/config版本。

下载URL/鉴权/代理/redirect/retry/cache/输入原子替换属于curl/CI等外部工具，不增加内置fetch或子进程wrapper，也不拉取external `$ref`。调用方须观察上游退出状态；有效JSON后producer失败不能回滚已完成生成，需要时先完成外部文件获取再生成。

### 5.3 本地 CLI 消费接入

- 不开发或发布正式 Gradle 生成插件；生成器只提供本地 native CLI 和 Go core。
- 使用文档给出显式 CLI 生成与 JVM `main`、KMP `commonMain`、Android 生成源目录配置；可提供手写 `Exec` 示例，但不能借此宣称平台已验收。
- 规格获取完全外置；编译不触发隐式联网，也不自动获取 latest CLI。
- 消费工程自行控制是否在编译前调用 CLI，固定本地 executable 版本、OS/arch 和 checksum；已有生成源目录无需自动任务也可使用。
- 多 spec、共享输出和命名锁的并发/事务规则由 CLI 测试证明，不引入插件 DSL、TestKit 或插件缓存承诺。
- 平台声明以真实 freshly-generated 编译和运行结果为准；Gradle/JDK 只是消费工程验证工具，不是生成器运行依赖。

## 6. 技术选型与兼容基线

当前技术选型与后续扩展边界：

| 位置 | 建议 | 约束 |
|---|---|---|
| JSON / 数据结构 | `encoding/json` Token decoder + ordered internal AST | 保留有意义的声明顺序、数值精度，schema slot 类型不丢失 |
| CLI / 配置 | 已实现封闭 flags parser、固定版本 TOML 1.1 库读取现有封闭 schema/local配置+disjoint-root batch；固定 byte/depth/work 预算 | 配置 schema 版本化、未知字段失败 |
| 错误 | `core.Diagnostics` implements `error` | 对外稳定诊断 code，不依赖错误字符串做协议 |
| 输入适配 | 标准库 `io`/`os` boundedfile/stdin | 获取交外部工具，不用HTTP/子进程下载适配；core保持纯byte输入 |
| 测试 | Go `testing`、native fuzz、真实进程/临时 FS | 最小 fixture、可审查快照，生成结果必须编译和运行 |
| fingerprint | 稳定 hash（如 SHA-256） | hash 前明确 canonicalization 与版本，不用默认 HashMap hasher |
| YAML | vendored `go.yaml.in/yaml/v3` v3.0.4 | YAML 官方组织维护、纯 Go；保留 MIT/Apache/NOTICE，产品另管 JSON-compatible tree 与 alias 预算 |

P0 固定 Go toolchain、Kotlin、serialization、Ktor、datetime、Gradle、JDK、AGP 的**真实可编译组合**，提交版本清单和 lock。日期类型至少验证 `kotlin.time.Instant` 与 `kotlinx.datetime.LocalDate` 的 serialization 支持；不能只写“支持 Kotlin 1.9+”。

默认 runtime 由生成 Kotlin 源码提供。SSE 首期采用 `client.sse` 后端，调用方配置 Ktor SSE/ContentNegotiation；更换流式后端时需重新验证事件、取消与重连契约。

## 7. 开发顺序与阶段验收

| 阶段 | 当前实现状态 | 尚未关闭的工作与退出条件 |
|---|---|---|
| P0 契约与工具链 | 已有矩阵、合成 fixture、命名/nullable/SSE/引用等合同及候选工具链组合 | 完整矩阵仍有开放边界；最低依赖、其他平台和公开 Go module 坐标待确认。当前候选范围已冻结，不能外推成完整 P0 验收 |
| P1 最小纵切 | 已实现 Go core/CLI、JSON Loader、基础模型、Kotlin emitter、单 owner 安全写出；已有 Go/进程/事务和 Kotlin 测试入口 | 基础纵切见 [P1 合同](p1-contract.md)。不把完整配置或一般模型语义计入该纵切；当前有界候选 R02–R05 已通过，新候选仍需重新绑定验收证据 |
| P2 模型语义 | 已有 enum/nullable/特殊标量/value class/Patch、named/internal refs、借用集合、union/allOf/hierarchy、默认值、const 与 opt-in 校验的多项有界实现 | [D01–D06](remaining-work.md#2-尚未完成的功能覆盖)：一般引用/nullable 图、组合/所有权、一般 default/const/enum 与约束聚合、Boolean/tuple 边界。退出需对应 M01–M13/API/wire 合同与新候选运行证据 |
| P3 HTTP 与 SSE | 已有 tag/client/method/参数、JSON body/response、typed inline error DTO、SSE Flow 与 wire/分块/取消测试入口 | [D07](remaining-work.md#2-尚未完成的功能覆盖)：更广参数/style/media/组合和 error roots；当前近似映射保持明确。A01–A08 的完整阶段验收未关闭 |
| P4 规模与稳定性 | 已有 dedup/hoist/merge、exact/regex/suffix exclusions、plain class、共享 runtime、core 与 CLI/config 命名锁、显式 disable/reset 及同 root owner rename | [D08–D09](remaining-work.md#2-尚未完成的功能覆盖)：一般结构优化与锁生命周期；选项交互、资源和性能证据分别验收，不由单个 profile 推导 |
| P5 工具可靠性 | 已有多 spec/config、manifest/reconcile、metadata-only owner rename、正常失败回滚、并发锁、file/stdin、check、warning policy、metrics、遗留事务检测并拒绝 | [D09–D10](remaining-work.md#2-尚未完成的功能覆盖)：迁移生命周期与超出现有 envelope 的配置需求（封闭 schema 的 TOML 语法已完成）；统一可靠性/确定性证据见 R02/R04。当前合同不承诺自动 recover 或 crash atomicity |
| P6 本地消费生态 | 已有本地 CLI 的 JVM 消费工程及历史宿主 CI/打包入口 | Android、KMP metadata/JS/Native、其他宿主/引擎与最低依赖组合仍需实际验收；不交付正式 Gradle 插件。新候选的 host/runtime 证据分别关闭 R05 |
| P7 发布 | 已有文档、MIT/attribution、打包/安装检查工具及历史 alpha 资产 | 当前有界候选 R02–R05 已通过；R06–R08 的最终发布候选审查、版本/tag/包绑定与发布授权仍待关闭。历史资产不自动包含近期实现 |
| P8 YAML | 已实现有界 Loader Adapter、core/native/config 与同义 JSON/YAML fixture | 完整 YAML/merge/非 JSON tags/keys/general source spans、各宿主新候选证据仍开放；不改模型/API generator，不重写历史 JSON 候选范围 |

依赖链是 P0 → P1 → P2 → P3 → P4 → P5 → P6 → P7；输入校验与安全写出从 P0/P1 开始，不等到最后补。按“一个 fixture → 一条语义 → 编译/运行证明”迭代，不先堆完整 parser 再集中测试。

不先承诺速度倍数或工期。P1/P2 完成后按生成器耗时、Kotlin 编译耗时和复杂规则返工量估算剩余工作。

## 8. 验证策略与完成定义

### 8.1 四层验证

1. Go 单元/性质测试：规范化、引用、命名、literal escaping、冲突、递归/深度限额；fuzz 不崩溃、不挂死。
2. 生成快照：独立编写的最小 JSON，断言文件集合与关键声明；更新快照必须说明语义变化，不能整批 accept 代替审查。
3. Kotlin 编译/运行：serialization round-trip、未知 enum/union、Patch 三态、required nullable、plain equality、请求 wire、SSE。
4. 工具集成：临时文件系统故障、stdin pipe/EOF/read-limit/no-fetch、本地 CLI 多 spec、命名锁与 check 无写入、消费工程源集接入。

P1 普通 DTO 以 Go 单测和生成快照为主，模板首次落地或实质变更时做最小 Kotlin 编译 smoke；Patch、union、自定义 serializer 等复杂语义进入后续运行测试。完整阶段默认 `data_class` 全量跑；优化选项做 pairwise + 全开组合，尤其 hoist×merge、hoist×Patch、plain×enum/union、runtime×多 spec。`encodeDefaults` / `explicitNulls` 组合单独成表；不把默认 Json 的正确性外推到所有用户自定义配置。

### 8.2 CI 分层

- PR：gofmt、go vet、Go unit/process/race tests、fixture 快照、JVM 编译/运行、公开材料扫描。当前工作流已配置 Go 后置的独立 Kotlin/JVM models、HTTP/SSE 与选定 profile job，显式准备依赖后运行离线门禁；具体提交的 CI 通过状态仍需对应 run 证明。
- 主分支：本地 CLI 消费集成、Android 编译、KMP metadata + JVM + JS、macOS 上 Native smoke；以真正通过的 target 声明支持。
- 发布：Linux/macOS/Windows CLI smoke，归档 checksum，依赖许可/漏洞检查，包内容与安装验证。

Go module 与 `scripts/kotlin/models.sh` 已可执行；当前源码最低要求为 Go 1.27.2，CI 和打包通过 `go.mod` 选择工具链。既有 Kotlin 编译/运行证据的组合为 Go 1.26.5、Kotlin 2.4.10、serialization 1.11.0、Gradle 9.3.1、JDK 21，宿主 macOS ARM64；提高 Go 版本不自动构成新的 Kotlin 编译/运行证据。Go 三宿主 CI 与首个 Draft Alpha 的六目标打包已有实际执行；Android/KMP 和完整运行语义矩阵仍是后续建设目标。

### 8.3 性能

用原创合成规格生成小/中/大规模样本，维度包括 operation 数、重复 shape、递归和 union 深度。分别记录 parse/resolve/analyze/name/emit/commit 耗时、峰值 RSS、文件/声明/行数、Kotlin 编译耗时和产物大小。只比较同机同输入同选项，不能拿 Go 生成速度代替 Kotlin 编译收益。

### 8.4 首个完整版本 Definition of Done

下列为广义完整版本标准；[JSON 候选清单](json-release-checklist.md)的有界交付不能自动替代它。若调整范围，需列明差异并明确批准，按 [版本策略](versioning.md)选择成熟度；文档收敛不等于 `1.0.0` 或发布授权。

- 兼容矩阵全部首期项有自动测试，所有行为差异都有明确记录和批准。
- 默认与优化模式输出在支持工具链上编译；运行测试覆盖 wire，而不只有字符串匹配。
- JSON file/stdin 生成离线可重复；规格获取外置、旧download/config明确拒绝；失败不破坏旧产物；不会误删手写文件。
- Kotlin 消费工程可通过本地 CLI 显式生成并配置源集完成编译；不要求或交付正式 Gradle 生成插件。
- 样本、快照、依赖及发布包通过贡献与发布检查。
- YAML 增量有明确接受/拒绝及预算合同，JSON/YAML 同义树共享分析与生成；完整 YAML 不属于首期 DoD。

## 9. 待确定决策

默认路线：**行为契约优先、JSON 首期、Go 本地 CLI、合成测试样例**。

1. 项目许可证已确定为 [MIT](../LICENSE)。产品版本与预览阶段按[版本规则](versioning.md)管理；公开 Go module 发布坐标仍待确定，未改变当前 module 路径。
2. 当前方法命名、配置排除正则子集、命名锁格式已分别有合同；显式 disable/reset 生命周期已有迁移报告与事务，同 root version2/4 named-owner rename 已有[独立合同](output-owner-rename.md)。[显式完整 root move](output-root-move.md)也已有同版本 v1–v4 完整原始状态迁移合同；partial/disjoint inventory move 与一般partial/multiowner升级流程仍待设计和实现；[完整同root v3→v4 single-owner adoption](ownership-migration.md#locked-owner-adoption)已有独立exact-preservation合同，D09 不整体关闭，见 [D09](remaining-work.md#2-尚未完成的功能覆盖)。
3. 正式 Gradle 插件不在交付范围；本地 CLI 与显式消费工程源集配置足够，平台覆盖单独按执行证据验收。
4. 首批 release binary 与 Kotlin/KMP targets，以真正通过的构建矩阵声明支持。

以上决策及验证进展应随实现更新，不将设计目标表述为已实现能力。
