# Changelog

所有版本的变更记录。格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

---

## [Unreleased]

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
