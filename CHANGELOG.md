# Changelog

所有版本的变更记录。格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

---

## [Unreleased]

### 测试

- **E2E 测试框架**：新增 `e2e/` 包，`TestMain` 编译一次 gogen 二进制，14 个测试覆盖所有 CLI 子命令：`generate`（含幂等性、`--dry-run`、`--suffix`）、`check`（最新/过期两种状态）、`lint`（Error/Warning/clean 三种退出码）、`init`（创建/已存在）、`version`。
- **单元测试补全**：为各包补充 17 个测试文件，覆盖率从 50.5% 提升至 54.4%；涵盖 `MethodSetContains`、`CanGenerateMethodOverride`、`ActiveFields`、`ModifyGenerator.Generate`、`ResetGenerator.Generate`、`formatDoc`、`isExcluded`/`isExcludedPath`、`Severity.String`、`Issue.String`、`extractDocText`、`packageDir` 等此前零覆盖函数。
- **analyzer 集成测试**：新增 `ExcludePaths` 和 `FileFilter` 配置的集成测试；新增 `testdata/analyzer_edge/` 包，覆盖非导出字段跳过和未知 tag 选项警告路径（`analyzeFields` 覆盖率 73.5% → 97.1%）。
- **覆盖率系统性提升（pkg/ 整体 94.8%）**：
  - `linter`（92.5% → 97.5%）：提取 `compareIssues` 为包私有函数，内部测试覆盖排序三分支（`-1`/`1`/column）；新增 `empty_tag_value`/`broken_syntax`/`multi_file_errors` testdata，覆盖空 tag 值跳过、LoadError、跨文件排序路径。
  - `loader`（91.3% → 96.5%）：新增 `TestLoad_NoGoFilesSkipped`，覆盖 phase 1/2 的 `isNoGoFilesError` 静默跳过分支。
  - `writer`（88.9% → 95.2%）：新增格式化错误测试（`Write`/`Check` 传入无效 Go 代码），覆盖 `imports.Process` 错误路径。
  - `analyzer`（93.1% → 95.0%）：新增 `Holder{Items Container[int]}` 测试结构体，覆盖 `buildTypeInfo` 中 `*types.Named{TypeArgs>0}` → `KindGeneric` 路径。

---

## [v0.4.1] — 2026-03-25

### 修复

- **linter 缺少 `gogen:modify=` 解析**：`docAnnotations` 结构体未包含 `ModifyMethod` 字段，导致 linter 无法识别 `gogen:modify=XXX` 注解，现已通过统一注解解析修复。

### 新特性

- **linter：`gogen:modify=` 无效配置 Warning**：`gogen:modify=XXX` 在 dirty tracking 未启用时（无 `gogen:dirty`、无 `MakeDirty()`、或设置了 `gogen:nodirty`）不会生成任何方法，linter 现在对此发出 Warning。

### 内部变更

- 新建 `pkg/annotations/` 统一注解解析包，消除 `pkg/analyzer` 与 `pkg/linter` 约 100 行重复实现。
- 新增 `pkg/annotations/README.md`、`pkg/linter/README.md`、`pkg/linter/TEST.md`。

---

## [v0.4.0] — 2026-03-24

### 破坏性变更

- **移除 Set 方法的幂等检查**：所有 `Set*` / `SetAt*` / `SetVal*` 方法不再生成 `if this.Field == value { return }` 检查。原因：指针/接口类型的 `==` 比较地址而非值，接口比较可能 runtime panic；幂等检查是调用方责任，生成层不应隐含此语义。详见 [D-013](DECISIONS.md)。
- **Dirty tracking 改用统一 `Modify()` 入口**：不再为每个写方法末尾注入 dirty 调用。升级后需将散落的 setter 调用改为 `Modify()` 包裹：
  ```go
  // 旧用法（不再支持）
  player.SetName("x")   // setter 内部自动调用 MakeDirty()
  // 新用法
  player.Modify(func(p *Player) { p.SetName("x") })
  ```
- **字段级 `gogen:"dirty=XXX"` tag 废除**：现为 lint Error，请改用结构体注解 `// gogen:dirty` 或 `// gogen:dirty=MethodName`。

### 新特性

- **`Modify()` 方法生成**：为启用 dirty tracking 的结构体生成 `Modify(fn func(*T))` 方法，fn 执行后统一调用 dirty 方法；fn panic 时不调用。方法名可通过 `// gogen:modify=Apply` 自定义。
- **`Reset()` 生成**：为所有结构体自动生成 `Reset()` 方法（`*this = T{}`），将所有字段重置为零值（slice/map 重置为 nil 释放内存）。已有手写 `Reset()` 时跳过生成并输出 `[Info]` 说明原因。
- **`gogen lint` 子命令**：静态检查 struct tag 和注解；捕获拼写错误（附近似建议）、矛盾组合（`readonly+writeonly`）、dirty 方法引用错误；Error 级别问题时以非零退出码退出，可接入 CI。

### 修复

- **嵌入场景 `Reset()` 生成正确**：改用 `CanGenerateMethodOverride`，外层结构体嵌入了已生成 `Reset()` 的内层结构体时，不再被提升方法阻挡，始终生成正确的 `*this = T{}` 形式。
- **并发输出顺序确定**：`[Info]` 等诊断消息由 log 回调传递，在 mutex 保护下与 `✅` 一起刷出，不再可能出现在汇总行之后。
- 移除 `Modify()` 覆盖提升方法时的 Warning 噪音。

### 内部变更

- `structAnnotations` 改为包私有类型（原 `StructAnnotations`），不对外暴露
- 新增设计决策 D-011 ~ D-019
- 新增并发正确性命题测试、边界测试和性能基准（生成全部结构体 ~522 µs/op，Apple M4）

---

## [v0.2.0] — 2026-03-12

### 新特性

- **结构体级 `plain` 注解**：在结构体文档注释中加 `// gogen:plain`，批量为所有字段应用 plain 模式，无需逐字段打 tag
- **`gogen:"override"` tag**：强制为字段生成访问器，忽略嵌入提升方法检查（默认行为是保护提升方法，不覆盖）

### 变更

- 重命名：`DeleteField` → `DeleteFieldAt`（slice），统一 At 后缀命名约定
- 重命名：`DelFieldKey` → `DeleteFieldKey`（map），与 slice 命名风格保持一致
- slice 底层改用 `slices.Delete()`（含 clear 语义，释放尾部元素引用，避免内存泄漏）
- 修复 README 泛型示例中的方法名错误，补充 override 运行时测试

---

## [v0.1.5] — 2026-03-12

### 新特性

- **`gogen:"plain"` tag**：简单模式，为字段只生成核心访问器，跳过扩展方法（bool 不生成 Toggle、数值不生成 Add/Sub、slice/map 不生成 Has/GetLen/GetCopy 等）
- **`EnsureXxx()`**（map）：对 map 字段做懒初始化，返回已初始化的 map，适合在 ORM `AfterFind` 等钩子中调用

---

## [v0.1.0] — 2026-03-11

### 初始发布

核心代码生成功能：

- 基于 `go/types` 语义分析，支持泛型、类型别名、跨文件类型引用
- 自动跳过已有手写实现的方法（无冲突生成）
- 嵌入提升方法检测：不覆盖通过嵌入字段提升的方法，保护接口实现
- 增量生成：文件内容未变时跳过写入
- 孤儿文件清理：结构体删除后自动清理对应的生成文件
- struct tag 控制：`gogen:"-"` / `gogen:"readonly"` / `gogen:"writeonly"`
- 支持 `.gogen.yaml` 配置文件
- `gogen check` 子命令（CI 验证生成文件是否最新）
- 支持字段类型：bool、数值、string、指针、接口、func、结构体、泛型实例、slice、数组、map（chan 跳过）
