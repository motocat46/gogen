# gogen 设计决策记录

记录项目开发过程中的关键决策、方案对比与取舍依据，避免维护时重复思考已经解决过的问题。

**与各模块文档的分工**：本文件聚焦"为什么做这个决策"；实现细节（代码、算法、陷阱）在各模块的 `DESIGN.md` 中：[analyzer](pkg/analyzer/DESIGN.md) · [generator](pkg/generator/DESIGN.md) · [loader](pkg/loader/DESIGN.md) · [writer](pkg/writer/DESIGN.md)。

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
直接查每个嵌入字段（`Anonymous()`=true）的 `*T` 方法集，完全不受外层直接方法干扰。用 `*T` 而非 `T` 确保包含指针接收者方法（完整超集）；`go/types` 自动展开深层嵌入，无需递归。

实现细节见 [`pkg/analyzer/DESIGN.md`](pkg/analyzer/DESIGN.md#提升方法检测的设计核心陷阱)。

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
各类型在 plain 模式下保留的方法见 [README.md §plain 模式各类型对比](README.md#struct-tag-控制)。

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

## D-011 Reset 方法语义

**决策**
为每个结构体生成 `Reset()` 方法，语义与 `proto.Reset()` 一致：`*this = T{}`，slice/map 字段重置为 nil（释放底层内存）。

**实现细节**
- 若结构体或其嵌入字段已有手写/提升的 `Reset()` 方法，gogen 静默跳过，不覆盖
- 嵌入类型会提升 `Reset()`：若 `DirtyBase` 有 `Reset()`，嵌入它的结构体不会再生成一个，需用手写或不嵌入来避免冲突
- 若结构体启用了 dirty 注入（`DirtyMethod != ""`），生成的 `Reset()` 末尾追加一次 dirty 调用，语义：重置也是一次写入，通知状态已变

**取舍**
`Reset()` 有时被 DirtyBase 的提升方法遮蔽，导致嵌入型 dirty 结构体无法得到 Reset。测试场景中，若需要 Reset + dirty 同时生效，应让结构体自行实现 `MakeDirty()`（不依赖嵌入提升），或为 DirtyBase 提供手写 Reset 阻止 gogen 生成。

---

## D-012 Dirty 注入：opt-out 默认不注入，三层优先级

**背景**
为写方法（Set/Add/Sub/Toggle/SetAt/Append/Delete/SetVal/DeleteKey）末尾注入业务层脏标记调用，减少手写样板。

**决策：opt-out（默认不注入）**
- 未显式配置且方法集中无 `MakeDirty()` 时，不注入，维持现有行为
- 仅在以下情况下注入：
  1. 结构体方法集包含 `MakeDirty()`（自动检测）
  2. 文档注释含 `// gogen:dirty` 或 `// gogen:dirty=XXX`（显式配置）

**三层优先级（高→低）**
1. `gogen:nodirty`（结构体注解）：禁用所有 dirty 注入，字段级 tag 也失效
2. `gogen:"dirty=XXX"`（字段 tag）：覆盖结构体级 dirty 方法名
3. 结构体级 dirty 方法（自动检测 or `gogen:dirty=XXX`）：所有字段共享

**Ensure 不注入**
`EnsureField`（map 惰性初始化）只在字段为 nil 时初始化，自身具有幂等性，不需要 dirty。

**幂等检查**
早期版本在 Set 类方法中生成幂等检查（见 D-013，已移除）。

---

## D-013 移除 Set 方法的幂等检查

**背景**
早期版本在 Set/SetAt 方法中，若字段类型可比较（`go/types.Comparable()`），且启用了 dirty 注入，则生成 `if current == new { return }` 前置检查，意图减少无意义的 dirty 通知。

**问题**
- `go/types.Comparable()` 对指针和 interface 也返回 true，但 `==` 比较的是地址，而非内容，语义与"值未变化"不符；interface 的动态类型若不可比较（如含 slice），`==` 还会 runtime panic
- 修复方案（拆出 `IsValueComparable` 只对基本类型生效）导致基本类型有幂等检查、其他类型没有，行为不一致，认知负担更重
- 幂等检查假设"调用者可能用相同值调用 setter"，但在设计良好的业务代码中这极少发生；调用者有责任在修改前自行判断

**决策：移除所有幂等检查**
生成的 setter 只做一件事：赋值（+ dirty 通知）。简单、一致、无类型特例。若调用者需要幂等语义，在调用层自行检查。

**同步移除** `TypeInfo.IsComparable` 字段及 analyzer 中 `types.Comparable()` 调用，因为该字段的唯一用途就是幂等检查。

---

## D-014 不实现 validate 注入

**背景**
讨论过为 setter 注入验证钩子（类似 dirty 注入），在赋值前调用用户提供的 `ValidateXxx(newValue)` 方法。

**决策：不实现**
- dirty 是跨字段的结构性横切关注点，所有写方法共享同一模式，gogen 注入是合理抽象
- validate 是字段级的领域逻辑，每个字段规则不同，不具备结构性均匀性
- 调用层自行验证更清晰：验证逻辑就在调用处，不需要跳转到 `ValidateXxx` 实现才能理解 setter 行为
- gogen 的边界：生成结构性访问样板，不生成领域逻辑的注入点

---

## D-015 Diff-aware Dirty：当前设计已足够，不扩展带参注入

**背景**
讨论是否扩展 dirty 注入为带参形式（如 `p.MarkDirty(GoldBit)`），以支持使用者实现字段级变更追踪（Diff-aware dirty）。

**决策：维持零参设计，不扩展**
零参 + 字段级方法名覆盖（`gogen:"dirty=MarkGoldDirty"`）已能完整支持 diff-aware dirty：
- 每个字段可以路由到不同方法，各自实现任意追踪逻辑
- 带参注入会强制所有字段共享同一个方法签名，per-field 的差异化行为只能在方法内部 switch，表达能力反而退化

**使用者实现 diff-aware dirty 的方案**
字段级 tag 覆盖，每个字段指向各自的方法：
```go
Gold  int64 `gogen:"dirty=MarkGoldDirty"`
Level int32 `gogen:"dirty=MarkLevelDirty"`
```
或在结构体级 `MakeDirty()` 内部维护 dirty bit，由统一入口处理——gogen 不越界。

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

---

## D-016 不生成 String() 方法

**背景**
讨论过为结构体生成 `String() string`（实现 `fmt.Stringer`），改善调试和日志可读性。

**已识别的实现陷阱**
- 在 String() 内部把 `this` 传给 fmt 函数会触发无限递归
- 指针字段若展开内容，嵌套结构体互相引用时会循环调用
- nil pointer 必须加守卫，否则 `fmt.Println(nilPtr)` 会 panic
- 敏感字段（密码、token）需要新的 tag 语义，增加设计复杂度
- `[]byte` 字段输出格式无通用答案

**决策：暂不实现**
当前项目的实际使用场景极少，`%+v` 已能满足基本调试需求，引入 String() 生成的设计复杂度与收益不成比例。待有明确高频需求时再重新评估。

---

## D-017 不生成并发安全访问器

**背景**
讨论过为结构体生成带 `sync.RWMutex` 的读写访问器（`gogen:"concurrent"`），解决多 goroutine 访问同一对象的数据竞争问题。

**决策：不实现**
- Go 惯用的同步手段是 channel 和更高维度的并发原语，直接使用读写锁的场景很少
- 生成单字段级别的锁无法保护复合操作，用户可能误以为加了标注就线程安全，实际仍有竞争窗口
- 收益与复杂度不成比例

---

## D-018 不生成观察者/事件模式

**背景**
讨论过生成 `OnXxxChanged(fn func(old, new T))` 订阅方法，在字段变化时触发注册的回调。

**决策：不实现**
dirty 注入已解决"有变化时通知"的核心需求。观察者模式的额外能力（old/new 值、运行时注册/取消）属于业务层逻辑，由业务层在 `MakeDirty()` 实现中自行处理更合适。gogen 不应承担回调生命周期管理（注册、取消、调用顺序）的设计复杂度。

---

## D-019 不生成跨语言绑定（TypeScript 等）

**背景**
讨论过从 Go 结构体生成 TypeScript 接口定义，解决前后端数据契约同步问题。

**决策：不实现**
gogen 的定位是 Go 访问器代码生成器，输入和输出均为 Go 源码。跨语言生成需要维护类型映射表、处理目标语言的模块系统，且存在 int64 精度等语言边界问题。该需求已有成熟的独立工具（tygo、go-typescript、protobuf+grpc-gateway 等），不在 gogen 的边界内。

---

## D-020 不生成 Clone() / DeepCopy() 方法

**背景**
讨论过为含 slice/map/pointer 字段的结构体生成深拷贝方法，解决手写深拷贝繁琐的问题。

**决策：不实现**
Go 生态中已有专为此设计的成熟工具 [k8s.io/code-generator/cmd/deepcopy-gen](https://github.com/kubernetes/code-generator)，广泛用于 k8s 生态及各类 Go 项目。该工具支持递归深拷贝、`+k8s:deepcopy-gen` 注解控制、接口适配等完整能力，gogen 重新实现收益极低。有深拷贝需求的项目应直接引入 deepcopy-gen。

---

## D-021 不生成 Equal() 方法

**背景**
讨论过生成 `Equal(other *T) bool` 逐字段比较方法，用于测试断言和变更检测。

**决策：暂不实现**
- 实际需要结构体级 Equal() 的场景较少；测试中更常用 `reflect.DeepEqual` 或 `cmp.Equal`（google/go-cmp）
- 浅比较与深比较语义取舍复杂（slice/map 元素是否递归比较、pointer 是地址比较还是值比较）
- `func` / `chan` 字段无法比较，需要跳过规则，引入额外设计复杂度
- 待有明确高频需求时再重新评估。
