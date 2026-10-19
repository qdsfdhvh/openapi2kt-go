# JSON 首版交付验收清单

## 目标与状态

交付一个离线、确定、失败保旧输出的 **OpenAPI JSON 3.0/3.1 → kotlinx.serialization 模型 + 有界 Ktor HTTP/SSE 客户端**生成器。规格由外部工具提供本地文件或 stdin；消费工程显式调用本地 CLI，不需要生成器 Gradle 插件。

**这是收敛后的候选发布清单，不是“完整首版已完成”的报告。** `0.2.0-alpha.1` 有界候选已完成 R02–R05 的原生、完整 JVM profile、安全与固定工具链验收；R06–R08 发布门槛仍未全部关闭。既有 alpha 资产不包含近期全部能力。整理此表不批准版本、平台支持、发布操作，也不自动把广义规划项标为完成。

当前候选固定为现有合同的**有界 JSON profile**，按下节边界统一验收。原 [完整首版 DoD](development-plan.md#84-首个完整版本-definition-of-done)继续作为后续路线图；本候选不以补齐全部引用图为退出条件，也不替代 `1.0.0` 的 [版本策略](versioning.md)要求。

本清单维护候选验收与发布门槛；完整路线图的功能缺口、已有实现和范围外工作统一见[实现状态与剩余工作](remaining-work.md)。

### 冻结的候选边界

| 维度 | 本候选约定 |
|---|---|
| 成熟度 | **有界 `0.2.0-alpha.1` 已发布 prerelease**（tag `v0.2.0-alpha.1`，源自 `e898e46`）。冻结验收范围不自动晋级 beta、rc 或完整首版。发布资产为六个 CLI 归档与 `SHA256SUMS`；历史 `0.1.0-alpha.1` 包不变。后续合并到 `main` 的改动不在该 tag 内 |
| 输入与功能 | 本地 JSON OpenAPI 3.0/3.1 的全部已有受支持 profile，含 file/stdin、本地多 spec、现有模型/引用/nullable/组合/校验、HTTP/SSE、输出事务、命名锁、warning policy 与 metrics；以 J01–J09 链接的合同和现有正例/拒绝测试为准，不撤销已有支持 |
| 默认与选项 | 保留 `data_class` 默认与 validation/优化默认关闭。执行 models、HTTP 与当前测试入口全部 feature routes 中现有的 data/plain、validation off/on、单项及已声明组合；锁与共享 runtime 的 core/native/Kotlin 合同保留。完整选项笛卡尔组合不在承诺内 |
| CLI 宿主 | **macOS ARM64** 原生编译并运行。Linux、Windows、macOS x86_64 及其他架构留在后续平台验收；旧 alpha CI 与交叉构建不作为本候选证据 |
| Kotlin 目标与引擎 | **Kotlin/JVM，JVM 17 字节码，JDK 21 运行**；serialization 运行断言、Ktor MockEngine wire 与真实 ByteChannel/SSE parser 合同。Android、KMP metadata/JS/Native 和其他实际网络引擎留在后续验收 |
| 固定工具链组合 | Go **1.27.2**、Gradle **9.3.1**、Kotlin/serialization compiler plugin **2.4.10**、kotlinx.serialization **1.11.0**、Ktor **3.5.2**、kotlinx.datetime **0.7.1**、已安装的 JDK **21**。这是本候选验收组合，不是已经证明的所有最低依赖版本 |
| 资源与可靠性 | 保留现有输入、schema/source-work、checked-edge、constructor/method 等限制及编译资源设置；独立宽度/深度探针不扩成宽×深最大组合保证。不得提高 heap、删负例或放宽检查取得通过 |
| 路线图差异 | 一般引用/nullable graph、通用组合与 discriminator、其余模型/约束/HTTP 投影、泛化优化与锁迁移、跨平台消费及 YAML 继续按第 2–3 节和原路线图处理。本轮只关闭此边界内的缺陷与验收证据 |

完整本地产品门禁为 `bash scripts/test.sh all`，固定执行 unit → models → HTTP/SSE → [测试入口](../scripts/test.sh)中 `features` 数组列出的全部 route。使用 `bash scripts/test.sh --list` 获取当前完整名称；新增 feature 应进入该入口，不在此重复维护 route 数量或一份冻结的旧列表。

覆盖组包括 Boolean/tuple/递归 JSON、DTO/enum/union/hierarchy/scalar 引用与借用集合、组合/nullable、默认值/const/enum/校验，以及结构优化/排除规则。models/HTTP 内既有 dedup、hoist、qualification、plain、Patch、shared runtime 与 name-lock profile 同时保留。具体功能与入口对应关系见[已有实现表](remaining-work.md#1-已有实现不再作为整项未开发功能)。

Q01–Q05 采用对应现有合同的上下文、Unknown、命名/锁、wire 与近似诊断策略；Q06 在本候选限于上表真实执行组合。统一验收必须绑定同一精确源码、输入、消费者、CLI 和工具链；具体 hashes、日志、JDK patch、资源参数与通过状态记录在本地验收材料。执行状态以第 4 节关闭表为准，未勾选的发布门槛仍需独立证据。

### 状态如何判定

- **已有实现/测试入口**：源码和契约可定位；不等于指定发布候选已经跑过。
- **候选通过**：同一冻结源码、输入、CLI、消费者、工具链和选项有成功运行证据；缓存、失败尝试及未执行项须明确。
- **待关闭**：缺实现、证据或范围决定。必须修复，或经明确批准后写入该候选的限制；不能删除负例、放宽检查或提高资源上限来“完成”。

下方 R01–R05 已关闭；结果限于本候选和明确执行的 profile。完整本地 `all` 门禁已通过，覆盖 unit、fresh models、HTTP/SSE 和当前全部 83 个 feature routes。使用既有 Go 缓存和已准备的离线依赖，Kotlin 消费者从 fresh 输出编译运行；相关 source/input/consumer 内容校验不变，失败尝试与详细绑定记录保留在本地验收材料。[PR #6 CI](https://github.com/qdsfdhvh/openapi2kt-go/pull/6/checks)另验证三宿主原生 CLI 和包安装。发布门槛仍需单独复核。功能合同定义支持边界，本清单不扩张它们。

## 1. 产品门槛与证据入口

| ID / 关联矩阵 | 首版候选必须守住的契约 | 已有入口；候选关闭证据 |
|---|---|---|
| J01 / F01、F03、T03 | 本地 JSON 与 direct `--input -`；strict JSON、8 MiB、EOF、IO 诊断；旧下载命令/配置拒绝；core 不做 IO | [file/stdin](local-input.md)、[core](../core/generate.go)、[原生输入测试](../internal/cli/local_input_process_test.go)。file/stdin 产物与报告一致，help/错误不消费流，失败不改变旧输出 |
| J02 / F04、F06、F07、F10 | 仅合同接受的 named/internal refs、DTO/alias/递归与组合；保 canonical constructor/serializer、nullable 优先级、每 hop/cache height；不支持者保原 Pointer 报错 | [内部 DTO 合同](internal-dto-references.md)、[root bridge 守卫](../core/root_dto_bridge_test.go)、[模型语义](model-semantics.md)。正例、透明环/排除形状/unused source、48/49 与 240/241 边界；不将有界 profile 标为整项完成 |
| J03 / M01–M13、V00–V06、Q01–Q05 | 支持的模型、enum/union/字典/特殊标量、请求 Patch、校验与已记录近似映射；构造/API 与 wire 都兼容 | [P1](p1-contract.md)、[请求 presence](request-presence.md)、[模型消费者](../samples/kotlin-smoke/README.md)。fresh 生成并编译/运行；相关 `encodeDefaults × explicitNulls` 四组合、Absent/Null/Value、known/unknown 与 nullable narrowing |
| J04 / A01–A07 | 已支持 path/query/header、JSON body、响应优先级、调用方 client/callback 与错误体；有损 projection 必须保持 warning/context，不宣称通用递归编码 | [HTTP 合同](http-client.md)、[HTTP fail-closed](../core/http_contracts_test.go)、[真实引用测试](../core/http_references_test.go)。fresh call-site 与受控引擎 URL/header/body/response/error/cancellation 断言 |
| J05 / A08 | 有界 SSE typed cold Flow、共用 payload resolver、cursor/header、POST、分块数据、关闭/取消/复用；不生成自动重连策略 | [SSE 合同](sse.md)、[SSE 消费者](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/SSEContracts.kt)。真实 Ktor parser 下分块 UTF-8/CRLF、多行、未知/坏 known data、未闭合 EOF、仍开连接的取消及后续复用 |
| J06 / O01–O05 | default-off 去重/hoist/merge、qualification、plain class 只承诺各自 profile；明确 alias nested-classifier、class/descriptor/package 与 plain no-copy 的迁移影响 | [dedup](deduplicate-models.md)、[hoist](hoist-models.md)、[merge](merge-models.md)、[plain](model-style.md)。默认、单项、已声明交互与全开 profile；保 source/noDrop/opaque owner/命名冲突，不能以文本相似证明 wire/API 相同 |
| J07 / F02、O03、T01–T02 | 本地多 spec、defaults/覆盖、partial retention/reconcile、共享 runtime、manifest/name-lock 版本与 CAS、正常失败回滚；只删 owned 文件 | [配置](configuration.md)、[命名锁](name-lock.md)、[ownership](output-ownership.md)、[batch](output-batches.md)。[native parity](../internal/cli/config_parity_process_test.go)、[multi-root preflight/check](../internal/cli/config_multi_root_guards_process_test.go)、[故障回滚](../internal/output/batch_test.go)；check/no-op/mtime/锁忙/手写文件/旧输出保护 |
| J08 / T05–T06 | 稳定诊断、可定位 warning、deny/allow policy、opt-in metrics；同输入/选项/状态确定性输出，默认报告兼容 | [warning policy](warning-policy.md)、[metrics](cli-metrics.md)、[原生 CLI 测试](../internal/cli/process_test.go)。text/JSON/exit code、原 Pointer、失败不回显输入或凭据，metrics 不进入产物 hash |
| J09 / T04、Q06 | native CLI 独立运行；生成 Kotlin 在**声明过并实际执行**的宿主/target/工具链上可用 | [消费工程](../samples/kotlin-http-smoke/README.md)、[CI/包边界](releases.md)。host CLI 与 Kotlin target 分别验收；交叉编译、生成/check、JVM 编译都不能替代另一个层次的运行证据 |

J02/J03/J06 的所有现有受支持 profile 均须保留；不能为了缩表悄悄撤销现有正例。各上下文里的 JsonElement/Unit/参数文本投影、nullable carrier collapse、raw String/ByteArray 与 bigint wire 差异以对应合同为准。

## 2. 先关闭的缺口：修复还是明确后置

按顺序处理，先冻结边界再选开发切片；此列表本身不批准后置。

| 优先级 | 当前缺口/不完整项 | 收敛办法与关闭条件 |
|---|---|---|
| P0 范围 | 广义 F/M/A/O/T 矩阵尚未整项完成 | 当前候选采用上节冻结的有界 JSON alpha、macOS ARM64 与 JVM；广义完整 DoD 保留为后续路线图。任何新增 profile、宿主、target 或成熟度晋级都需重新定义边界并补证据 |
| P0 统一验收 | 各切片成功不等于同一源码的完整 profile 集成功 | 对冻结候选跑默认 models/HTTP、所有受支持 feature routes 与声明的选项交互；核实际输入/消费者/binary/hash，不拼接不同源码的通过记录 |
| P1 模型边界 | 一般引用/nullable 图、超出对象 product 子集的组合/层级、一般 default/const/enum/约束聚合仍未完整；详见 [D01–D06](remaining-work.md#2-尚未完成的功能覆盖) | 对象 union-in-allOf、自/互递归完整 JSON covers、primitive List/Map 默认值、nullable structured const、显式 nullable null enum 成员及 referenced/nested allOf bounds/counts 已有实现，不能再列为整项未做。只因真实消费需求扩展最小缺口，其余保留当前拒绝/近似合同 |
| P1 HTTP/SSE 边界 | 部分结构参数/请求响应仍有有损投影；更多 media/style/组合未接受；SSE 无自动 reconnect/backoff/resume 保证 | 固定现有合同与 warning，并用 wire/streaming 证据守住。若候选必须支持超出边界的用例，再独立实现/测试；调用方 Ktor 策略不冒充生成器保证 |
| P1 优化/锁生命周期 | 泛化 union/hierarchy/递归优化、partial/disjoint root move、一般partial/multiowner锁迁移仍未完整；详见 [D08–D09](remaining-work.md#2-尚未完成的功能覆盖) | exact/portable regex/literal suffix exclusions、CLI/config 锁持久化、显式 disable/reset 与 [named-owner rename](output-owner-rename.md) 及[完整 root move](output-root-move.md)与[完整同root v3→v4 adoption](ownership-migration.md#locked-owner-adoption) 已有合同，须纳入现有门禁。其他生命周期需独立迁移合同；不得删除或修改 manifest/锁绕过错误 |
| P1 资源 | 独立宽度/深度探针不证明宽×深×选项×平台笛卡尔上界；未证明编译资源或性能收益 | 公布已执行的资源 profile 与硬限制；保留失败和 warnings，固定 compiler/heap/caps。不以分析通过承诺所有极端组合可编译；性能声明另按 [性能契约](validation-and-performance.md)测量 |
| P1 平台 | 完整 feature 证据以 JVM 为主；[catalog 平台探针](../samples/kotlin-platform-smoke/README.md)已有 data/plain 的 JVM/JS/macOS Native 运行和 metadata/Android AAR 编译结果，但 Android 真机、跨平台完整 feature、最低依赖与其他 CLI 宿主未全部执行 | 先决定实际发布 targets；对要宣称的逐一跑。未执行者明确未验证，不借旧 alpha 的 CI 或六目标 cross-build 证明当前候选 |
| P1 发布 | 源码已标识 `0.2.0-alpha.1`，原生包安装和候选模式扫描已有证据；最终发布审查、tag/包绑定和授权仍待关闭，旧 alpha 不自动更新 | 完成下节 R01–R08。内置下载移除是 CLI/config breaking change，需测试、migration 和正确版本分类，不覆盖旧资产 |

## 3. 明确外置或不承诺的能力

- **规格获取外置**：HTTP 下载、鉴权、代理、redirect/retry/cache、输入文件原子替换由 curl/CI 等工具负责；不添加内置 downloader、URL input 或子进程 wrapper。producer 完整 JSON 后失败不能回滚生成，调用方应观察 pipeline 状态；需要更强保证时先成功获取文件。
- **首期 JSON**：YAML 是后续 Loader 阶段；external file/HTTP `$ref`、完整 OpenAPI/JSON Schema 认证、所有 3.1 关键字、Swagger 2.0 不在现有承诺内。
- **消费工程接入**：使用本地 CLI 和源集配置；正式 Gradle 插件、自动安装 CLI/Ktor 插件、应用鉴权流程、GUI/其他生成语言不交付。
- **没有新增容错承诺**：输出保证是合同定义的正常失败回滚，不是跨 root/文件系统 crash atomicity 或自动恢复。配置不提供 stdin replay；SSE 默认无生成器自动重连。不得把这些限制隐藏在“支持 stream/事务/SSE”字样里。
- **其他未接入 HTTP 功能**：multipart、cookie、callbacks、webhooks、links 等继续按现有合同/诊断处理；不把“HTTP 客户端子集”宣传成完整客户端。

Android/KMP 或更广引用图是否后置属于候选范围决定，不在此处自动认定它们永久不需要。

## 4. 发布候选关闭表

每项必须绑定同一候选；下面的空框表示**尚未在此清单下整体验收**，不是否定已有实现或历史切片证据。

- [x] **R01 本地源码候选范围冻结**：采用上节有界 JSON `0.x alpha`、macOS ARM64 CLI 与 Kotlin/JVM、固定工具链/默认值/现有选项交互和 Q01–Q06 策略；原完整 DoD 的差异明确留在路线图。实际发布版本身份须另按 R07 绑定，本项不批准晋级或发布。
- [x] **R02 源码与原生门禁**：冻结精确 source/input/consumer/binary 绑定；格式、vet、完整 Go/unit/process/race、Python tooling、相关 bounded fuzz 通过；失败证据保留，注明 Go cache。当前候选要求 macOS ARM64 原生执行；仅在以后扩张宿主范围时增加对应 Linux/Windows 等实际运行门槛。
- [x] **R03 Kotlin 完整 profile**：fresh 默认 models、HTTP/SSE、全部受支持 feature suites 通过；四 Json/Call-site/wire/stream closure/canonical identity 断言仍在；声明的单项/pairwise/全开交互矩阵有可重复证据。脚本 `all` 不是所有选项笛卡尔组合的自动证明。
- [x] **R04 安全与确定性**：旧产物/手写文件/mtime、partial/reconcile、共享文件、check 无写入、busy locks、正常回滚及遗留事务拒绝守卫通过；同输入/选项/状态的文件集/bytes 不漂移。有意 API/wire 差异逐条测试与迁移说明。
- [x] **R05 平台与工具链组合**：当前候选的 Kotlin/JVM、受控 Ktor 引擎与上表固定依赖组合真实编译并运行相应断言；额外 catalog 平台探针的 data/plain JVM/JS/macOS Native 运行及 metadata/Android AAR 编译结果见[探针范围](../samples/kotlin-platform-smoke/README.md)，不扩大为跨平台完整 profile 或 Android 真机支持。声明最低版本时再补最低组合验证，不将固定组合外推为版本区间。host CLI、交叉构建和 Kotlin runtime 三类证据分开。
- [x] **R06 精确发布候选审查**：[candidate scan](publication.md)在要发布的候选上无未解决 findings；跟踪删除/候选集合差异要解释，不能将工作区 live-file 检查冒充 exact tree/history/package 检查。保 license/第三方 attribution、依赖安全审查与只含公开产品内容；扫描不等于完整安全审计。 **已执行**（tag `v0.2.0-alpha.1`，`e898e46`）：candidate scan 0 findings；`go mod verify` 通过；`go mod vendor` 零差异复现 `vendor/`；`govulncheck` 无已知漏洞；随包许可与第三方归属与源文件逐字节一致。
- [x] **R07 版本与包安装**：[版本分类](versioning.md)和 [next-alpha migration](releases/next-alpha.md) 审查完成；最终 CLI/tag/README/RELEASE.json/archives 一致；准确包成员/checksum、支持宿主上的从包安装→help/file/stdin/validate/generate/check 验证齐全。对包中没有的 fixture 不依赖外部 checkout；历史 tag/assets 不覆盖。旧版本标识的本地源码预检包只能证明打包/安装流程，不能充作新版本发布资产。 **已执行**：版本面一致（CLI 常量、`--version`、`RELEASE.json`、归档名前缀、tag）；六个归档各 10 成员。执行证据：CI 在每个提交上为各自的 native target 构建并做隔离安装检查（`darwin-arm64`/`linux-amd64`/`windows-amd64`）；本候选另对已发布字节执行了 `darwin-arm64`（原生）、`linux-arm64`（原生 arm64 容器）、`linux-amd64`（x86-64 容器与 ARM64 模拟）。`darwin-amd64` 与 `windows-arm64` **未执行**，仅成员/元数据/可执行性验证。release 流水线现增加 `verify-native` job，在各自宿主上验证**已发布归档**；本 tag 早于该 job。详见[安装说明](installation.md)。
- [x] **R08 发布授权**：操作者明确授权对应 Git 写入、tag/push/upload/发布/可见性操作；CI、draft assets、notes 与 unsigned-package 限制审查完成。测试全绿不等于已获这些权限。 **已授权并执行**：操作者在会话中明确要求发版；tag、push、归档上传与 prerelease 发布均已完成，资产为六个归档与 `SHA256SUMS`。

### 本地门禁入口

已有匹配 `go.mod` 的 Go、缓存好的 Gradle/JDK/Kotlin 依赖时，在仓库中运行：

```bash
# 可用 GO/GOFMT/GRADLE 指定已有工具；不自动获取工具或依赖。
bash scripts/test.sh --list
bash scripts/test.sh all
```

[脚本入口](../scripts/test.sh)的 `all` 实际为 unit（format/vet/Go/race/Python）→ fresh models → fresh HTTP/SSE → 所有列出的 feature routes。生成/Go 使用离线开关，Kotlin 使用缓存依赖与 `--offline`；缺工具或失败不能绕过，保持既有 heap/caps。它不执行跨宿主验证、Android/KMP、打包安装、history 审查或发布授权。

[CI 工作流](../.github/workflows/ci.yml)目前提供三宿主 Go、源码安装/生成/check、原生包打包与隔离安装/file/stdin 检查，以及 tooling/candidate 检查；**不编译或运行 Kotlin**。工作流存在不是已经执行成功的证据。发布门禁另见 [release contract](releases.md)。

## 5. 停止条件

候选发布前，R01–R08 均需关闭。经明确批准的范围调整可以收窄要验收的 profile/host/target，并同步 notes/文档/对应测试；不能豁免纯生成、旧输出保护、真实证据、许可/安全边界或发布授权。阻塞时保留最后一个成功来源、失败日志、未验证项与所需工具/范围决定；不得静默改产品保证、删失败输入、伪造 green 或自动开始不相关开发。

本清单的维护只更新产品边界与可验证入口；具体运行结果、机器路径和任务日志留在本地验收记录，不进入公共文档。
