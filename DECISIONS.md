# gogen 设计决策记录

记录项目开发过程中的关键决策、方案对比与取舍依据，避免维护时重复思考已经解决过的问题。

---

## D-001 类型分析：go/types 替代手工 AST 解析

**背景**
早期版本手工遍历 `ast.TypeSpec` 来解析字段类型，对泛型、类型别名、跨文件类型引用处理脆弱。

**方案对比**
- 手工 AST：代码简单，但 `ast.Ident` 只有名字字符串，无法区分 `int` 和自定义 `type MyInt int`，泛型参数更无法处理
- `go/types`：语义层分析，类型完全解析，`types.Named`/`types.TypeParam`/`types.Alias` 一网打尽

**选择**
使用 `go/types`（通过 `go/packages` 加载）。

**代价**
加载速度变慢（需要完整类型检查），对含编译错误的包需要两阶段恢复策略。

---

## D-002 嵌入提升方法检测：为什么不能查外层类型的方法集

**背景**
需要阻止 gogen 为与嵌入提升方法同名的字段生成方法，否则会破坏接口实现。

典型场景：
```
LostMine { Speed float32; *Unit }
Unit → Entity { Speed float32 + 手写 GetSpeed()/SetSpeed(speed, accel float32) }
IEntity { GetSpeed() float32; SetSpeed(speed, accel float32) }
```
gogen 若生成单参数 `SetSpeed`，`LostMine` 就不再满足 `IEntity`，编译报错。

**错误方案：查外层类型的方法集**
```go
// 看起来合理，实则有 bootstrap 陷阱
mset := types.NewMethodSet(types.NewPointer(named))
for i := range mset.Len() {
    sel := mset.At(i)
    if len(sel.Index()) > 1 { // Index 长度 > 1 表示提升方法
        promoted[sel.Obj().Name()] = true
    }
}
```
**陷阱**：若旧的 `*_access.go` 文件已存在并生成了有冲突的 `GetSpeed()`，加载时外层类型的方法集会把它识别为**直接方法**（`Index` 长度为 1），提升来源被掩盖，gogen 无法自愈，每次运行都会重新生成错误代码。

**正确方案：直接遍历嵌入字段的方法集**
```go
func collectPromotedMethods(named *types.Named) map[string]bool {
    underlying := named.Underlying().(*types.Struct)
    promoted := make(map[string]bool)
    for i := range underlying.NumFields() {
        field := underlying.Field(i)
        if !field.Anonymous() {
            continue
        }
        ft := field.Type()
        if ptr, ok := ft.(*types.Pointer); ok {
            ft = ptr.Elem()
        }
        // 查嵌入字段本身的方法集，完全不受外层直接方法干扰
        mset := types.NewMethodSet(types.NewPointer(ft))
        for j := range mset.Len() {
            promoted[mset.At(j).Obj().Name()] = true
        }
    }
    return promoted
}
```

**为什么用 `*T` 的方法集而非 `T`**
`*T` 的方法集 = T 的值接收者方法 + 指针接收者方法，是完整超集。若 `Entity.SetSpeed` 是指针接收者，用 `T` 的方法集会漏掉它。

**深层嵌入的处理**
`*Unit` 的方法集会自动包含从 `Entity` 提升上来的方法，无需递归手动处理，`go/types` 帮你展开。

---

## D-003 slice/map 不暴露整体 getter

**背景**
是否为切片字段生成 `GetEmails() []string`？

**决策**
不生成。只提供细粒度操作（`GetEmailsAt`、`AppendEmails` 等）。

**理由**
整体 getter 返回内部切片的引用，调用方可以直接 `s.GetEmails()[0] = "hack"` 修改内部状态，封装形同虚设。`GetEmailsCopy()` 提供安全的浅拷贝版本（用 `slices.Clone`）供确实需要完整数据的场景使用。

**代价**
用起来比直接 `s.Emails` 稍麻烦，但这正是强制封装的目的。如果不需要封装，不用 gogen 即可。

---

## D-004 Delete 而非 Remove：切片删除方法的命名

**背景**
切片删除方法叫 `RemoveFieldAt` 还是 `DeleteFieldAt`？

**争论过程**
- `remove` 语义上是"取出元素，元素还在"（如 `list.remove(item)` 返回被移除的元素）
- `delete` 语义上是"销毁，元素消失"
- `slices.Delete` 的注释里用的是 "removes" 来描述行为，但函数名却是 `Delete`

**关键发现**
`slices.Delete` 的实现会对释放的尾部槽位调用 `clear()`，将引用置零，防止 GC 泄漏（对 `[]*T` 或 `[]interface{}` 字段尤其重要）。这个"清零"步骤是破坏性的，与 remove 的"取出"语义不符。

**选择**
用 `Delete`，并且直接调用 `slices.Delete()` 而非手写 `append` 实现，保证清零语义。命名与底层调用保持一致，减少心智负担。

---

## D-005 At 后缀：index 操作方法的命名约定

**背景**
切片删除方法最初叫 `DeleteField(index int)`，后改为 `DeleteFieldAt(index int)`。

**问题**
`DeleteField(index)` 直觉上容易误读为"删除名为 Field 的切片字段本身"，而非"删除 Field 切片中 index 位置的元素"。

**约定**
所有 index 参数的操作统一加 `At` 后缀：`GetFieldAt` / `SetFieldAt` / `DeleteFieldAt`，与 Go 标准库中 `bytes.Index` 等的 `At` 语义保持一致。

---

## D-006 plain 模式：为什么需要，哪些方法被裁剪

**背景**
某些字段语义上不适合暴露所有操作：
- 唯一 ID（`int64`）不应该有 `Add`/`Sub`
- 状态枚举（`bool`）不应该有 `Toggle`
- 只做遍历的 map 字段不需要 `GetLen`/`GetKeys`/`GetCopy`

**机制**
字段级 tag `gogen:"plain"` 或结构体文档注释中加 `// gogen:plain` 批量启用。

**裁剪规则**

| 类型 | 被裁剪的方法 |
|---|---|
| bool | Toggle |
| 数值 | Add / Sub |
| 指针/接口/func | Has |
| 切片 | GetLen / Has / GetCopy |
| 数组 | GetLen |
| map | GetValOrDefault / Has / HasKey / GetLen / GetKeys / GetCopy |

**Ensure 不裁剪**
map 的 `EnsureField` 在 plain 模式下保留，因为惰性初始化是基础操作（常用于 ORM AfterFind 钩子），不属于"扩展查询能力"。

---

## D-007 containsAnnotation 用逐行精确匹配

**背景**
结构体文档注释中检测 `gogen:plain` 标注，最初用 `strings.Contains(doc, "gogen:plain")`。

**问题**
`strings.Contains` 会把 `gogen:plaintext`、`gogen:plainMode` 等误判为 plain 标注。

**修复**
逐行 `TrimSpace` 后做精确相等比较：
```go
func containsAnnotation(doc, annotation string) bool {
    for line := range strings.SplitSeq(doc, "\n") {
        if strings.TrimSpace(line) == annotation {
            return true
        }
    }
    return false
}
```

---

## D-008 chan 类型跳过不生成

**决策**
`chan` 类型字段不生成任何方法。

**理由**
Channel 的封装收益极低：`GetChan()` 返回 channel 引用后调用方可以直接收发，和直接访问字段没有区别；而 `SendToField(v T)` / `RecvFromField() T` 会封装掉 `select` 能力，反而限制用法。Channel 的正确使用方式通常需要配合 `context`、`select` 等，难以通过简单 getter/setter 封装。

---

## D-009 override tag：强制覆盖提升方法检查

**背景**
D-002 的提升方法保护在极少数场景下过于严格——用户明确知道自己要覆盖嵌入提升的方法。

**机制**
字段标注 `gogen:"override"` 后，`CanGenerateMethod` 退化为 `CanGenerateMethodOverride`，跳过提升方法检查，但仍保留字段名冲突和手写方法冲突两层检查。

**注意**
override 只是跳过"提升方法同名"这一层保护，如果用 override 生成的方法与嵌入类型实现的接口方法签名不同，调用方通过接口访问时拿到的将是 override 生成的方法，接口实现由嵌入类型提供这一假设被打破，需要使用者自己负责语义正确性。

---

## D-010 增量生成：内容对比跳过写入

**背景**
每次运行 gogen 是否都重写所有 `*_access.go` 文件？

**决策**
格式化后与磁盘内容对比，内容相同则跳过写入，不更新文件的 mtime。

**理由**
- 避免触发不必要的 `go build` 重编译
- `go:generate` 场景下不污染 git diff
- 幂等性：多次运行结果完全一致

**前提**
生成代码不含时间戳，模板输出确定性强。
