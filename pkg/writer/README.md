# pkg/writer — 使用文档

## 功能简介

`writer` 包负责将生成的原始 Go 代码格式化并写入文件，支持增量跳过（内容相同不写入）、安全保护（不覆盖手写文件）、dry-run 预览，以及孤儿文件清理。

## 快速上手

```go
import "github.com/motocat46/gogen/pkg/writer"

cfg := writer.Config{
    Suffix:  "access", // 生成文件后缀，空时默认 "access"
    Verbose: true,     // 输出写入日志
}

// 写入生成代码
written, err := writer.Write(structDef, code, cfg)
if err != nil {
    return err
}
if written {
    fmt.Println("文件已更新")
} else {
    fmt.Println("内容无变化，跳过写入")
}

// 清理不再需要的旧文件（当结构体所有方法均已手写时）
if code == nil {
    err = writer.Clean(structDef, cfg)
}
```

## Config 字段说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `OutputDir` | string | 空（与源文件同目录） | 统一输出目录 |
| `Suffix` | string | `"access"` | 文件名后缀（不含下划线和 .go） |
| `DryRun` | bool | false | 只打印路径，不写入 |
| `Verbose` | bool | false | 输出写入/删除日志 |

## 生成文件命名规则

`{小写结构体名}_{suffix}.go`

```go
cfg.OutputFilename("UserInfo")  // → "userinfo_access.go"
cfg.OutputFilename("Cache")     // → "cache_access.go"
```

## API 说明

### `Write(s *model.StructDef, code []byte, cfg Config) (written bool, err error)`

格式化并写入生成代码。

- 格式化：使用 `golang.org/x/tools/imports`（等价于 `goimports`），进程内完成，不依赖外部命令
- 安全检查：目标文件存在但不含 `Code generated` 标记时，拒绝覆盖并返回错误
- 增量跳过：格式化后与磁盘内容逐字节对比，相同则返回 `false, nil`

### `Check(s *model.StructDef, code []byte, cfg Config) (upToDate bool, err error)`

检查是否需要写入，不实际写文件。用于 `gogen check` 命令（CI 验证）。

- `code == nil`：检查对应文件是否应该被删除（文件存在且含 gogen 标记 → 不是最新）
- `code != nil`：检查格式化结果是否与磁盘内容一致

### `Clean(s *model.StructDef, cfg Config) error`

删除结构体对应的生成文件（若存在）。当结构体所有方法均已有手写实现时调用，防止旧的生成文件与手写方法产生重复声明编译错误。

### `IsGogenGenerated(content []byte) bool`

检查文件内容是否含有 `Code generated ... DO NOT EDIT.` 标记。只检查前 1 KB，避免读取大文件。

## 注意事项

- `OutputDir` 为空时，文件写入到 `model.StructDef.Dir`（结构体源文件所在目录）
- goimports 格式化是幂等的：相同输入始终产生相同字节，使增量对比完全可靠
- DryRun 模式下 `Clean` 也只打印路径，不执行实际删除
