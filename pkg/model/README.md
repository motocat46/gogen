# pkg/model — 使用文档

## 功能简介

`model` 包是 gogen 的领域模型层，定义了分析层输出、生成层输入的核心数据结构。与 `go/ast` 和 `go/types` 完全解耦，确保生成层只依赖稳定的领域模型。

## 核心类型

### TypeKind — 字段类型分类

```go
const (
    KindBasic       // string、TypeParam（泛型参数）等基础类型 → Get/Set
    KindBool        // bool → Get/Set/Toggle
    KindNumeric     // int/float/uint/complex → Get/Set/Add/Sub
    KindPointer     // *T → Get/Set/Has
    KindSlice       // []T → GetAt/GetLen/Range/Has/GetCopy/SetAt/Append/DeleteAt
    KindArray       // [N]T → Get/GetAt/GetLen/Range/SetAt
    KindMap         // map[K]V → GetVal/Range/Has/GetLen/SetVal/DeleteKey 等
    KindStruct      // 具名结构体（如 time.Time） → Get/Set
    KindGeneric     // 泛型实例（如 List[int]） → Get/Set
    KindInterface   // interface{}/any/具名接口 → Get/Set/Has
    KindFunc        // func 类型字段 → Get/Set/Has
    KindUnsupported // chan 等，跳过生成
)
```

### TypeInfo — 字段类型的完整描述

```go
type TypeInfo struct {
    Kind     TypeKind   // 类型分类，决定选用哪个生成器
    TypeStr  string     // 完整类型字符串，如 "[]string"、"map[string]int32"
    Elem     *TypeInfo  // slice/array/pointer 的元素类型
    Key      *TypeInfo  // map 的 key 类型
    Value    *TypeInfo  // map 的 value 类型
    ArrayLen string     // [N]T 中的 N，如 "8"
    TypeArgs []*TypeInfo // 泛型类型实参，如 List[int] 中的 int
    IsAlias  bool       // 是否为类型别名（type X = T）
}
```

**典型示例：**

```go
// []string
TypeInfo{Kind: KindSlice, TypeStr: "[]string", Elem: &TypeInfo{KindBasic, "string", ...}}

// map[string]int32
TypeInfo{Kind: KindMap, TypeStr: "map[string]int32",
    Key: &TypeInfo{KindBasic, "string"},
    Value: &TypeInfo{KindNumeric, "int32"}}

// *time.Time
TypeInfo{Kind: KindPointer, TypeStr: "*time.Time",
    Elem: &TypeInfo{KindStruct, "time.Time"}}
```

### FieldDef — 字段描述

```go
type FieldDef struct {
    Name    string      // 字段名，如 "Tags"
    Type    *TypeInfo   // 字段类型的完整描述
    Config  FieldConfig // 从 struct tag 解析的生成控制
    Doc     string      // 字段文档注释（已去除 // 前缀）
    Comment string      // 字段行尾注释
}
```

**FieldConfig — struct tag 控制：**

```go
type FieldConfig struct {
    Skip      bool // gogen:"-"         跳过此字段
    Readonly  bool // gogen:"readonly"  只生成 getter
    WriteOnly bool // gogen:"writeonly" 只生成 setter
    Plain     bool // gogen:"plain"     只生成核心 Get/Set，跳过扩展方法
    Override  bool // gogen:"override"  忽略嵌入提升方法检查，强制生成
}
```

**常用方法：**

```go
f.IsReadable()  // !Skip && !WriteOnly
f.IsWritable()  // !Skip && !Readonly
```

### StructDef — 结构体描述

```go
type StructDef struct {
    Name            string
    TypeParams      string          // 泛型参数，如 "[K, V]"；非泛型为空
    PackageName     string          // 包名，如 "model"
    PackagePath     string          // 导入路径，如 "github.com/foo/bar/model"
    Dir             string          // 源文件目录（输出文件写入此处）
    Fields          []*FieldDef     // 所有字段（含 Skip=true 的字段）
    Doc             string          // 结构体文档注释
    ManualMethods   map[string]bool // 手写方法名
    FieldNames      map[string]bool // 所有字段名（含不导出）
    PromotedMethods map[string]bool // 嵌入提升方法名
}
```

**常用方法：**

```go
// 接收者类型字符串（泛型/非泛型）
s.ReceiverType()  // "Cache" 或 "Cache[K, V]"

// 方法名冲突检查（三层：字段名 + 手写方法 + 提升方法）
s.CanGenerateMethod("GetName")

// override 模式（两层：字段名 + 手写方法，跳过提升方法检查）
s.CanGenerateMethodOverride("GetEmbeddedField")

// 获取未被跳过的字段列表（Skip=false）
s.ActiveFields()
```

## ParseFieldConfig — 解析 struct tag

```go
tag := `json:"name" gogen:"readonly,plain"`
cfg := model.ParseFieldConfig(tag)
// cfg.Readonly == true, cfg.Plain == true
```

## 注意事项

- `StructDef.Fields` 包含 `Skip=true` 的字段，生成层通过 `ActiveFields()` 过滤；若需遍历全部字段，直接用 `Fields`
- `TypeInfo.TypeStr` 由 `types.TypeString` 生成，对于跨包类型含包名前缀（如 `"time.Time"`），在目标包中直接可用
- `IsAlias` 仅标记 `type X = T` 形式的别名；`type X T`（新类型）的 `IsAlias=false`
