# pkg/annotations — gogen 注解解析

## 功能简介

`annotations` 包提供统一的 gogen 结构体注解解析能力，供 `pkg/analyzer` 和 `pkg/linter` 共用。

在此包提取之前，两个包各自维护了一份几乎相同的实现，linter 还因此漏掉了 `gogen:modify=` 的解析支持。

## API 说明

### `ParseStructAnnotations(doc string) StructAnnotations`

解析结构体文档注释中的 gogen 注解。`doc` 是 `ast.CommentGroup.Text()` 的输出（已去除 `//` 前缀），逐行匹配，不会误判行内文本。

```go
ann := annotations.ParseStructAnnotations(doc)
```

### `StructAnnotations`

| 字段 | 触发注解 | 含义 |
|------|---------|------|
| `Plain bool` | `gogen:plain` | 该结构体所有字段使用 plain 模式 |
| `DirtyMethod string` | `gogen:dirty` / `gogen:dirty=XXX` | dirty 方法名；`""` 表示未指定 |
| `NoDirty bool` | `gogen:nodirty` | 显式禁用 dirty tracking（最高优先级） |
| `ModifyMethod string` | `gogen:modify=XXX` | 自定义 Modify 方法名；`""` 表示使用默认值 `"Modify"` |

**注解优先级（由调用方决定）**：`nodirty` > 显式 `dirty=XXX` > `dirty`（默认 `MakeDirty`）> 自动检测。

### `MethodSetContains(named *types.Named, methodName string) bool`

检查 `*named` 的方法集中是否包含名为 `methodName` 的零参无返回值方法（含嵌入提升方法）。

```go
// 检查是否自动检测到 MakeDirty()
if annotations.MethodSetContains(typesNamed, "MakeDirty") {
    dirtyMethod = "MakeDirty"
}
```

## 使用示例

```go
import "github.com/motocat46/gogen/pkg/annotations"

doc := "gogen:dirty=MarkChanged\ngogen:modify=Apply"
ann := annotations.ParseStructAnnotations(doc)

// ann.DirtyMethod  == "MarkChanged"
// ann.ModifyMethod == "Apply"
// ann.NoDirty      == false
// ann.Plain        == false
```

## 注意事项

- `gogen:dirty` 与 `gogen:dirty=XXX` 同时出现时，**后者覆盖前者**（按文档顺序，最后一个 `dirty=XXX` 生效）
- `gogen:modify=XXX` 仅在 dirty tracking 生效时才有意义；`gogen lint` 会对无效配置发出 Warning
- `MethodSetContains` 使用 `types.NewPointer(named)` 检查指针接收者的方法集，确保嵌入提升方法可被识别
