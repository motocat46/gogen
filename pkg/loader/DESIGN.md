# pkg/loader — 设计文档

> go/types 加载策略的决策背景见 [DECISIONS.md D-001](../../DECISIONS.md#d-001-类型分析gotypes-替代手工-ast-解析)（包括加载变慢的代价权衡）。本文档记录两阶段加载的实现细节。

## 关键设计决策

### 1. 以"包"为分析单元，而非单个文件

与 `stringer`、`mockgen` 等官方工具保持一致：分析粒度是整个包，而非单个 `.go` 文件。

**原因**：
- 结构体方法可能分散在包内多个文件中，必须整包加载才能正确收集手写方法
- 嵌入的外部类型（`time.Time`、自定义结构体）需要完整的导入解析
- go/types 的类型解析本身就是以包为单位的

**单文件模式的处理**：用户传入 `./foo.go` 时，内部转为 `file=<abs>` 格式，go/packages 会加载整个包，但 analyzer 层通过 `FileFilter` 只处理 `foo.go` 中的结构体。

### 2. 两阶段加载：解决旧生成文件导致的死锁

**问题**：gogen 生成的 `*_access.go` 如果有编译错误（如修改了结构体字段类型），会导致整个包无法编译，进而 gogen 无法重新加载包来修复错误——死锁。

**两阶段方案**：

```
阶段1：正常加载，收集所有包错误
      ↓
      分类错误：
      ├── 错误直接指向 *_{suffix}.go 文件
      │     → 构建 overlay（替换为空包声明）
      └── 其他错误
            → 如无 overlay，直接报错（用户代码有问题）

阶段2（仅在有 overlay 时执行）：
      用 overlay 替换有问题的 *_{suffix}.go，重新加载
      → 跳过 overlay 目录的残余错误（重新生成后会消失）
      → 其他目录的错误仍然报告
```

**overlay 的安全性**：只替换**确认为 gogen 生成**的文件（通过 `readGogenFilePkg` 检查 `Code generated` 标记），不会影响用户手写的同名文件。

### 两阶段加载：otherErrs 与 overlay 共存时的处理

**情况描述：** 阶段1同时发现了 `*_access.go` 引起的错误（构建了 `overlay`）和其他错误（`otherErrs`）。

**当前行为：** `otherErrs` 被丢弃，直接进入阶段2重新加载。

**理由：** 阶段1的 `otherErrs` 通常是 cascade 错误——`*_access.go` 损坏后导致的连锁编译失败，而不是用户代码的真实错误。阶段2用空文件替换损坏的 `*_access.go` 后，这些 cascade 错误会自然消失。若阶段2结束后仍有错误，`remainErrs` 收集逻辑会捕获并上报。

**已知风险：** 极少数情况下，用户代码可能同时存在真实编译错误和 `*_access.go` 损坏，此时阶段1的真实错误会被丢弃，阶段2的 `remainErrs` 过滤可能不完整地报告错误（仅报告非 overlay 包中的错误）。这属于"在复杂错误场景下优先完成生成，异常报告可能不完整"的设计取舍。

### 3. normalizePatterns：单文件模式的 file= 转换

`go/packages` 对 `./foo.go` 路径的处理：作为合成包 `command-line-arguments`，只包含该单一文件，丢失完整包上下文，导致：
- 同包其他文件中的类型无法解析
- 跨文件引用的结构体字段类型报错

通过转换为 `file=<绝对路径>` 格式，go/packages 以正确的完整包为单位加载。

### 4. filterExcludedPackages：只过滤顶层包

排除路径只应用于用户显式指定的包（顶层），不影响这些包的依赖。否则：
- 如果用户指定 `./...`，排除 `vendor` 是合理的
- 但依赖关系解析仍然需要访问 vendor 中的包

`packages.Visit` 遍历的是全图（含传递依赖），`filterExcludedPackages` 只从顶层切片中移除，不影响内部依赖解析。

---

## 已知限制

- 两阶段加载只能处理 gogen 自身生成文件（`Code generated` 标记）导致的错误，对用户代码中的编译错误会正常报告退出
- `readGogenFilePkg` 读取文件头判断是否为 gogen 生成，若文件头被意外损坏（如被其他工具覆盖部分内容），可能无法正确识别
