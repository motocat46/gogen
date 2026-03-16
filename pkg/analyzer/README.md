# pkg/analyzer — 使用文档

## 功能简介

`analyzer` 包负责将 `go/packages` 加载的包信息语义分析，提取结构体定义，转换为生成层所需的 `model.StructDef` 领域模型。

## 快速上手

```go
import (
    "github.com/motocat46/gogen/pkg/analyzer"
    "github.com/motocat46/gogen/pkg/loader"
)

// 1. 先用 loader 加载包（必须包含完整类型信息）
pkgs, err := loader.Load(dir, loader.Config{}, "./...")
if err != nil {
    return err
}

// 2. 分析所有包，提取结构体定义
structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
if err != nil {
    return err
}

// 3. 遍历结构体，交给 generator 生成代码
for _, s := range structs {
    fmt.Printf("结构体: %s，字段数: %d\n", s.Name, len(s.Fields))
}
```

## 排除特定路径

```go
structs, err := analyzer.Analyze(pkgs, analyzer.Config{
    // 排除目录名（匹配路径中任意一段，支持嵌套任意层级）
    ExcludePaths: []string{"mock", "mocks", "proto"},

    // 排除绝对路径前缀（精确匹配目录或文件）
    // ExcludePaths: []string{"/project/internal/gen"},
})
```

## 只处理指定文件

```go
// 当用户指定 ./foo.go 时，loader 会加载整个包（确保类型解析正确），
// 但 analyzer 只处理 foo.go 中定义的结构体
structs, err := analyzer.Analyze(pkgs, analyzer.Config{
    FileFilter: []string{"/absolute/path/to/foo.go"},
})
```

通常通过 `loader.ExtractFileFilter` 自动提取，无需手动构造。

## API 说明

### `Analyze(pkgs []*packages.Package, cfg Config) ([]*model.StructDef, error)`

分析一组已加载的包，返回所有结构体的领域模型列表。

**自动跳过规则（无需配置）：**
- 含 `// Code generated ... DO NOT EDIT.` 标记的文件（mockgen/protobuf/gogen 等工具生成的文件）
- 非导出字段（小写开头）
- 匿名嵌入字段（用于嵌入提升，不产生访问器）
- 非结构体类型（interface、type alias 等）

**Config 字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `FileFilter` | `[]string` | 文件绝对路径列表；非空时只分析这些文件中的结构体 |
| `ExcludePaths` | `[]string` | 排除的路径前缀列表；纯目录名匹配路径中任意一段 |

### 返回值 `model.StructDef` 的关键字段

| 字段 | 含义 |
|------|------|
| `Name` | 结构体名，如 `"User"` |
| `TypeParams` | 泛型参数（含括号），如 `"[K, V]"`；非泛型为空 |
| `PackageName` | 包名，如 `"model"` |
| `Dir` | 源文件所在目录绝对路径（输出文件写入此处） |
| `Fields` | 字段列表（含 tag 解析结果、类型信息） |
| `ManualMethods` | 手写方法名集合（生成时跳过） |
| `PromotedMethods` | 嵌入提升方法名集合（生成时跳过，除非 override） |

## 注意事项

- `Analyze` 依赖 `loader.LoadMode` 中指定的加载模式（包含 NeedSyntax + NeedTypes + NeedTypesInfo），如果使用自定义加载方式必须包含这三项
- 同一结构体的字段注释和 struct tag 来自 AST，类型信息来自 `go/types`，两者在 `analyzeFields` 中合并
- 结构体文档注释的 `gogen:plain` 注解由 analyzer 检测，并向下传播到所有字段的 `Config.Plain`
