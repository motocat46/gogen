# pkg/model — 设计文档

## 设计原则

model 包是整个系统的稳定核心——分析层（analyzer）写入，生成层（generator）读取，两层均依赖 model 但互不依赖。

### 与 go/types 解耦

model 不包含任何 `go/ast` 或 `go/types` 的类型，生成层可以在不依赖 Go 工具链的环境中运行（如未来的 IDE 插件、LSP 集成场景）。

### TypeInfo 的递归结构

TypeInfo 以树状结构表示复合类型：

```
map[string][]int32
→ TypeInfo{
    Kind: KindMap,
    TypeStr: "map[string][]int32",
    Key:   &TypeInfo{KindBasic, "string"},
    Value: &TypeInfo{
        KindSlice, "[]int32",
        Elem: &TypeInfo{KindNumeric, "int32"}
    }
  }
```

生成层通过递归访问 Elem/Key/Value 获取内层类型字符串，用于生成方法参数和返回值。

### StructDef 不做业务决策

`StructDef` 只存储分析结果，不判断"应该生成什么"——这是生成层的职责。`CanGenerateMethod` / `CanGenerateMethodOverride` 是纯查询方法（只读），不修改 StructDef 状态。

### FieldDef.Fields 包含 Skip 字段

`StructDef.Fields` 保留所有字段（含 `Config.Skip=true`），由生成层通过 `ActiveFields()` 过滤。这个设计是为了让调用方在需要时能访问完整字段列表（如诊断、调试工具），而不是让 model 做隐式过滤。
