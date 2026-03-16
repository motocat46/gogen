# pkg/analyzer — 设计文档

> 本文档记录实现细节和陷阱。工具链选型（go/types vs 手工 AST）的决策背景见 [DECISIONS.md D-001](../../DECISIONS.md#d-001-类型分析gotypes-替代手工-ast-解析)；提升方法检测的决策背景见 [DECISIONS.md D-002](../../DECISIONS.md#d-002-嵌入提升方法检测为什么不能查外层类型的方法集)。

## 设计演进

### v1：纯 AST 手工解析
最初仅依赖 `go/ast` 解析类型字符串，手工拼接 `*T`、`[]T`、`map[K]V` 等。问题：
- 泛型类型（`List[int]`）解析复杂且容易出错
- 类型别名（`type X = T`）无法区分别名和新类型
- 跨文件、跨包的具名类型（`time.Time`）需要手动解析导入路径

### v2：go/types 语义分析（当前）
改用 `go/types` 进行类型解析，AST 只用于提取注释和 struct tag。详见 DECISIONS.md D-001。

---

## 核心分析策略

### 双层信息合并

```
AST（go/ast）          ←→     go/types（类型系统）
  ↓                              ↓
结构体注释、字段注释           字段的精确类型（TypeInfo）
struct tag                    手写方法集（ManualMethods）
字段排列顺序                   提升方法集（PromotedMethods）
```

两层信息在 `analyzeFields` 中通过字段名合并：AST 提供名字和 tag，go/types 提供类型语义。

### buildTypeInfo：类型识别状态机

`buildTypeInfo` 递归将 `types.Type` 转换为 `model.TypeInfo`：

```
types.Type
├── *types.Basic         → KindBool / KindNumeric / KindBasic
├── *types.Pointer       → KindPointer，递归解析 Elem
├── *types.Slice         → KindSlice，递归解析 Elem
├── *types.Array         → KindArray，递归解析 Elem，记录 ArrayLen
├── *types.Map           → KindMap，递归解析 Key 和 Value
├── *types.Alias         → 透传到 Rhs()，但保留别名 TypeStr（IsAlias=true）
├── *types.Named
│   ├── 有 TypeArgs      → KindGeneric（泛型实例化，如 List[int]）
│   ├── 底层为 Struct    → KindStruct
│   └── 其他            → 递归解析底层类型，保留具名 TypeStr
│                          （type Status string → KindBasic，TypeStr="Status"）
├── *types.Interface     → KindInterface
├── *types.Signature     → KindFunc
├── *types.TypeParam     → KindBasic（泛型类型参数 T、K、V）
└── *types.Chan          → KindUnsupported
```

### qualifierFor：跨包类型的包名前缀

`types.TypeString` 需要一个 `Qualifier` 函数决定是否添加包名前缀：
- 与当前包相同的类型：省略前缀（`User` 而非 `mypackage.User`）
- 跨包类型：保留包名（`time.Time`、`sync.Mutex`）

生成的代码直接可用于目标包，无需额外处理导入路径（由 `goimports` 在 writer 层自动补全）。

---

## 提升方法检测的设计（核心陷阱）

详细背景见 DECISIONS.md D-002，这里补充技术细节。

### 为什么不查外层类型的方法集

```go
// 反直觉的陷阱
mset := types.NewMethodSet(types.NewPointer(named)) // ← 错误！
```

如果外层类型已有同名直接方法（包括上次 gogen 生成的方法），外层方法集中该方法的 `sel.Index()` 长度为 1（直接方法），不是通过嵌入提升的。这样查询无法发现"提升方法被直接方法遮蔽"的情况，导致无法自愈（删除直接方法后提升语义才恢复）。

### 正确做法：直接遍历嵌入字段

```go
func collectPromotedMethods(named *types.Named) map[string]bool {
    underlying := named.Underlying().(*types.Struct)
    for field := range underlying.Fields() {
        if !field.Anonymous() { continue }
        ft := field.Type()
        if ptr, ok := ft.(*types.Pointer); ok { ft = ptr.Elem() }
        // 用 *ft 的方法集：包含 ft 自身及所有深层嵌入的方法（递归覆盖）
        mset := types.NewMethodSet(types.NewPointer(ft))
        for sel := range mset.Methods() {
            promoted[sel.Obj().Name()] = true
        }
    }
}
```

直接查嵌入字段类型的方法集，不受外层直接方法的影响，即使外层已有同名方法也能正确检测到提升。

### 为什么用 *ft 而非 ft 的方法集

`types.NewMethodSet(ft)` 只包含值接收者方法；`types.NewMethodSet(types.NewPointer(ft))` 包含值接收者和指针接收者的全集。用 `*ft` 确保不漏报指针接收者定义的方法。

---

## isExcluded：排除规则的两种匹配模式

排除路径规则的两种匹配逻辑（含路径分隔符 → 前缀匹配；纯目录名 → 路径分段精确匹配）与 loader 层一致，规范定义见 [`pkg/loader/DESIGN.md`](../loader/DESIGN.md#4-filterexcludedpackages只过滤顶层包)。

---

## 已知限制

1. **只处理导出字段**：非导出字段（小写开头）不生成访问器。如有需求需要修改 `analyzeFields` 中的 `nameIdent.IsExported()` 判断。

2. **不处理匿名嵌入字段本身**：嵌入字段（`Anonymous()=true`）不生成访问器，只参与提升方法检测。

3. **chan 类型**：`buildTypeInfo` 返回 `KindUnsupported`，生成层跳过。若需支持，在 generator 层注册 `KindUnsupported` 的处理器（会影响所有 chan 字段）。
