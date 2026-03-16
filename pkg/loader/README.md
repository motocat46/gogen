# pkg/loader — 使用文档

## 功能简介

`loader` 包负责使用 `go/packages` 加载 Go 包的完整类型信息（AST + 类型系统），是整个分析流程的第一步。内置两阶段加载，解决旧生成文件编译错误导致的死锁问题。

## 快速上手

```go
import "github.com/motocat46/gogen/pkg/loader"

// 加载当前目录及所有子包
pkgs, err := loader.Load(workDir, loader.Config{}, "./...")
if err != nil {
    return err
}

// 加载单个文件（自动扩展到整个包）
pkgs, err := loader.Load(workDir, loader.Config{}, "./model/user.go")
```

## patterns 格式

| 格式 | 含义 |
|------|------|
| `"."` | 当前目录的包 |
| `"./..."` | 当前目录及所有子包（递归） |
| `"pkg/model"` | 指定包路径 |
| `"./foo.go"` | 单个 `.go` 文件（内部自动转为 `file=` 格式，加载整个包） |
| `"file=/abs/foo.go"` | 显式 `file=` 格式 |

## 排除特定目录

```go
pkgs, err := loader.Load(workDir, loader.Config{
    ExcludePaths: []string{
        "mock",    // 纯目录名：匹配路径中任意一段的 mock 目录
        "mocks",
        "vendor",
        "testdata",
        "/abs/path/to/proto", // 绝对路径前缀
    },
}, "./...")
```

## 自定义生成文件后缀

```go
pkgs, err := loader.Load(workDir, loader.Config{
    GeneratedSuffix: "gen", // 匹配 *_gen.go（默认为 "access"）
}, "./...")
```

## 提取文件过滤器

当用户指定具体文件时，需要告知 analyzer 只处理那些文件中的结构体：

```go
// 从 patterns 中提取显式指定的 .go 文件绝对路径
fileFilter := loader.ExtractFileFilter(workDir, patterns)

// 传给 analyzer
structs, err := analyzer.Analyze(pkgs, analyzer.Config{
    FileFilter: fileFilter,
})
```

## API 说明

### `Load(dir string, cfg Config, patterns ...string) ([]*packages.Package, error)`

加载指定 patterns 的包。

- `dir`：工作目录，相对路径以此为基准解析
- `cfg`：加载配置（排除路径、生成文件后缀）
- `patterns`：包模式，支持上述所有格式

内部采用**两阶段加载**：
1. 无 overlay 正常加载，收集包含错误的包
2. 若发现某些 `*_{suffix}.go` 文件有编译错误，仅对这些文件使用 overlay（替换为空包声明）重新加载

这解决了"旧生成文件有错误 → 包无法编译 → gogen 无法重新生成"的死锁。

### `ExtractFileFilter(dir string, patterns []string) []string`

从 patterns 中提取显式指定的 `.go` 文件绝对路径列表。
返回空列表表示未指定具体文件（处理所有加载到的结构体）。

### `LoadMode` 常量

```go
const LoadMode = packages.NeedName |
    packages.NeedFiles |
    packages.NeedSyntax |    // AST，用于读取注释和 struct tag
    packages.NeedTypes |     // 类型系统
    packages.NeedTypesInfo | // 完整类型映射（TypesInfo.Defs）
    packages.NeedImports     // 导入信息，用于跨包类型解析
```

## 注意事项

- loader 本层只负责加载，不做任何业务判断（不过滤生成文件、不分析结构体）
- `GeneratedSuffix` 空时默认为 `"access"`，与 writer 层的 `DefaultSuffix` 保持一致
- 排除路径的匹配规则：含路径分隔符 → 前缀匹配；纯目录名 → 路径分段精确匹配
- 单文件模式（`./foo.go`）内部转为 `file=<abs>` 格式，确保加载整个包以获得完整上下文
