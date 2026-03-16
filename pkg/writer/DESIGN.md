# pkg/writer — 设计文档

> 增量生成的决策背景（为什么不写时间戳、为什么做内容对比）见 [DECISIONS.md D-010](../../DECISIONS.md#d-010-增量生成内容对比跳过写入)。本文档记录实现细节。

## 关键设计决策

### 1. 增量跳过：相同内容不写入

写入前将格式化结果与磁盘文件逐字节对比，内容相同则跳过（返回 `written=false`）：

```
生成代码 → goimports 格式化 → bytes.Equal(formatted, existing) ? 跳过 : 写入
```

**为什么可靠**：
- 生成代码不含时间戳（详见 [DECISIONS.md D-010](../../DECISIONS.md#d-010-增量生成内容对比跳过写入)），相同输入产生相同字节
- goimports 格式化是确定性的：相同输入 → 相同输出
- 因此"相同输入 → 相同字节"的链条完整，增量对比完全可信

**收益**：
- git diff 干净：未修改的结构体不产生无意义的文件变更
- CI 快速：`gogen check` 在无变更时立即通过
- 文件系统友好：不触发不必要的 mtime 更新

### 2. 安全保护：不覆盖手写文件

若目标路径的文件存在但不含 `Code generated ... DO NOT EDIT.` 标记，判定为手写文件，拒绝覆盖并返回描述性错误：

```
文件存在 && !IsGogenGenerated(existing)
  → 返回 error（含修复建议）
```

防止用户误将 gogen 输出目录（`--output`）与手写代码目录重叠，导致业务逻辑丢失。

### 3. 进程内格式化：不依赖外部命令

```go
formatted, err := imports.Process(outputPath, code, nil)
```

`golang.org/x/tools/imports.Process` 等价于 `goimports`，在进程内执行。

**优势**：
- 不依赖 `goimports` 是否安装，任何 `go build` 环境均可运行
- 格式化结果确定，不受不同版本 goimports 的行为差异影响
- 自动推断和整理 import 块，无需生成器手动管理 import

### 4. Clean：孤儿文件清理

当结构体所有方法均已有手写实现时，generator 返回 `nil`（无需生成）。但上次运行可能已经生成过 `*_access.go`，这个旧文件与手写方法会产生重复声明编译错误。

`Clean` 负责删除这个孤儿文件，保持包的编译正确性：

```go
// main.go 中的调用模式
if code == nil {
    writer.Clean(structDef, writerCfg)
} else {
    writer.Write(structDef, code, writerCfg)
}
```

### 5. Check 模式：CI 验证

`Check` 与 `Write` 逻辑相同，但只做比对，不写入：

```
gogen check ./...
  → 对每个结构体调用 Check
  → 任何 upToDate=false → 以非零状态码退出
```

用于 CI pipeline 验证"已提交的生成文件是否与当前代码一致"。

---

## IsGogenGenerated 的实现细节

只检查文件前 1 KB：

```go
const headerBytes = 1024
header := content
if len(header) > headerBytes {
    header = header[:headerBytes]
}
return bytes.Contains(header, []byte("Code generated")) &&
    bytes.Contains(header, []byte("DO NOT EDIT"))
```

**为什么只检查 1 KB**：Go 约定生成文件的标记必须在文件最前面（通常是第一行）。检查 1 KB 足够覆盖任何合法的文件头，同时避免对大文件进行全文搜索。

---

## 已知限制

- `goimports` 需要能访问模块 cache（`GOPATH/pkg/mod`）来推断跨包 import 路径。在离线环境中如果 cache 缺失，格式化可能失败。
- `Clean` 不检查文件是否含 gogen 标记就删除——假设调用方（main.go）已通过 loader 的两阶段逻辑确认该文件是 gogen 生成的。如果手动调用 `Clean` 需要先用 `IsGogenGenerated` 确认。
