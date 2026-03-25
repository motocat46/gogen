# gogen — Go 结构体访问器代码生成器

为 Go 结构体字段自动生成 getter/setter 等访问器方法的 CLI 工具。

## 特性

- 基于 `go/types` 语义分析，支持泛型、类型别名、跨文件类型引用
- 自动跳过已有手写实现的方法（无冲突生成）
- 嵌入提升方法检测：不覆盖通过嵌入字段提升的方法，保护接口实现
- 增量生成：文件内容未变时跳过写入
- 孤儿文件清理：结构体删除后自动清理对应的生成文件
- `Reset()` 方法生成：将所有字段重置为零值，slice/map 重置为 nil，释放底层内存（语义与 `proto.Reset()` 一致）
- Dirty tracking（opt-in）：生成 `Modify(fn func(*T))` 作为唯一变更入口，fn 完成后自动调用业务层 dirty 方法；支持自动检测 `MakeDirty()`、结构体注解两种触发方式
- 支持 `.gogen.yaml` 配置文件
- 与 `//go:generate` 无缝集成

## 安装

```bash
go install github.com/motocat46/gogen@latest
```

或从源码构建：

```bash
git clone https://github.com/motocat46/gogen
cd gogen
go build -o gogen .
```

## 快速开始

```bash
# 处理当前目录
gogen .

# 处理当前目录及所有子包
gogen ./...

# 预览模式（不写入文件）
gogen --dry-run ./...

# 详细输出
gogen --verbose ./...
```

## 生成方法一览

| 字段类型 | 生成的方法 |
|---|---|
| `bool` / 底层为 bool 的类型 | `GetField()`, `SetField()`, `ToggleField()` |
| `int`/`float`/`uint` 等数值类型 | `GetField()`, `SetField()`, `AddField(delta)`, `SubField(delta)` |
| `string` / 自定义字符串类型 | `GetField()`, `SetField()` |
| 指针 `*T` | `GetField()`, `SetField()`, `HasField()` |
| `interface{}` / `any` / 具名接口 | `GetField()`, `SetField()`, `HasField()` |
| `func` 类型 | `GetField()`, `SetField()`, `HasField()` |
| 结构体 `T` | `GetField()`, `SetField()` |
| 泛型实例 `List[T]` | `GetField()`, `SetField()` |
| 切片 `[]T` | `GetFieldAt()`, `GetFieldLen()`, `RangeField()`, `HasField()`, `GetFieldCopy()`, `SetFieldAt()`, `AppendField()`, `DeleteFieldAt()` |
| 数组 `[N]T` | `GetField()`, `GetFieldAt()`, `GetFieldLen()`, `RangeField()`, `SetFieldAt()` |
| Map `map[K]V` | `GetFieldVal()`, `GetFieldValOrDefault()`, `RangeField()`, `HasField()`, `HasFieldKey()`, `GetFieldLen()`, `GetFieldKeys()`, `GetFieldCopy()`, `EnsureField()`, `SetFieldVal()`, `DeleteFieldKey()` |
| `chan` | 跳过，不生成 |

设计原则：切片和 map 不提供整体 getter（`GetEmails() []string`），只提供细粒度操作，强制封装。`GetFieldCopy()` 使用 `slices.Clone`/`maps.Clone` 返回浅拷贝，保护内部状态。`EnsureField()` 对 map 字段做懒初始化，适合在 ORM `AfterFind` 等钩子中调用。

**结构体级方法（所有结构体自动生成）：**

| 方法 | 签名 | 说明 |
|------|------|------|
| `Reset()` | `Reset()` | 将所有字段重置为零值；若启用了 dirty tracking，末尾追加 dirty 调用。已有手写或提升的 `Reset()` 时静默跳过 |
| `Modify()` | `Modify(fn func(*T))` | dirty tracking 的唯一变更入口；fn 执行完毕后自动调用 dirty 方法，若 fn panic 则不调用。仅在启用 dirty tracking 时生成 |

## struct tag 控制

在字段上添加 `gogen` tag 控制生成行为：

```go
type User struct {
    ID       int64  `gogen:"-"`         // 跳过，不生成任何方法
    Name     string `gogen:"readonly"`  // 只生成读方法（Get/Range/GetAt 等）
    Age      int    `gogen:"writeonly"` // 只生成写方法（Set/Append 等）
    Score    int    `gogen:"plain"`     // 简单模式：只生成 Get/Set，跳过 Add/Sub
    Tags     []string                   // 默认：生成全套方法
}
```

| Tag | 效果 |
|---|---|
| `gogen:"-"` | 跳过此字段，不生成任何方法 |
| `gogen:"readonly"` | 只生成读方法（Get/GetAt/Range/GetLen/GetVal 等） |
| `gogen:"writeonly"` | 只生成写方法（Set/SetAt/Append/SetVal 等） |
| `gogen:"plain"` | 简单模式：只保留核心访问器，跳过扩展方法（见下表） |
| `gogen:"override"` | 覆盖模式：忽略嵌入提升方法检查，强制生成该字段的访问器 |

**plain 模式各类型对比：**

| 字段类型 | 默认方法 | plain 模式 |
|---|---|---|
| `bool` | Get / Set / **Toggle** | Get / Set |
| 数值类型 | Get / Set / **Add / Sub** | Get / Set |
| 指针 / 接口 / func | Get / Set / **Has** | Get / Set |
| 切片 `[]T` | GetAt / **GetLen** / Range / **Has / GetCopy** / SetAt / Append / DeleteAt | GetAt / Range / SetAt / Append / DeleteAt |
| 数组 `[N]T` | Get / GetAt / **GetLen** / Range / SetAt | Get / GetAt / Range / SetAt |
| map `map[K]V` | GetVal / **GetValOrDefault** / Range / **Has / HasKey / GetLen / GetKeys / GetCopy** / Ensure / SetVal / DeleteKey | GetVal / Range / Ensure / SetVal / DeleteKey |

> `plain` 适合语义上不应暴露扩展操作的字段，如唯一 ID（不应 Add/Sub）、状态枚举（不应 Toggle）等。

**结构体级 plain（批量应用）：**

当一个结构体有多个字段都需要 plain 模式时，在结构体文档注释中加 `gogen:plain`，无需逐字段打 tag：

```go
// PlayerStats 玩家统计数据。
//
// gogen:plain
type PlayerStats struct {
    PlayerID int64   // 自动 plain → 只生成 Get/Set，不会有 Add/Sub
    RoomID   int64
    Score    float64 `gogen:"readonly"` // 字段级 tag 仍然有效
}
```

**override — 强制覆盖提升方法：**

默认情况下，gogen 不会为与嵌入提升方法同名的字段生成方法（保护接口实现）。若确实需要覆盖提升方法，使用 `gogen:"override"`：

```go
type Vehicle struct {
    Speed float32 `gogen:"override"` // 强制生成 GetSpeed/SetSpeed，覆盖嵌入提升
    *SpeedEntity
}
```

> 注意：`override` 仅跳过提升方法检查，仍遵守字段名冲突和手写方法冲突规则。

## 命令行选项

```
gogen [patterns...] [flags]

Flags:
  -o, --output string         输出目录（默认：与源文件同目录）
      --suffix string         生成文件名后缀（默认：access → user_access.go）
  -v, --verbose               显示详细输出
      --dry-run               预览模式，不实际写入文件
      --exclude stringArray   额外排除路径（支持目录名或路径前缀，可多次指定）
      --no-default-excludes   禁用内置默认排除（vendor、testdata、mock、mocks）

子命令:
  gogen version               打印版本号
  gogen init                  在当前目录生成 .gogen.yaml 配置文件模板
  gogen check [patterns...]   验证生成文件是否最新，不写入（CI 适用，过期则非零退出）
  gogen lint [patterns...]    检查 gogen tag 和注解有效性（不生成代码，有 Error 级诊断时非零退出）
```

patterns 格式与 `go/packages` 一致：`.`、`./...`、`pkg/model`，也支持直接传入 `.go` 文件路径。

## 配置文件

在项目根目录运行 `gogen init` 生成配置文件模板：

```bash
gogen init
```

生成的 `.gogen.yaml`：

```yaml
# suffix: access        # 文件名后缀，默认 access
# output: ""            # 输出目录，默认与源文件同目录
# excludes:             # 额外排除路径
#   - proto
#   - internal/generated
# no-default-excludes: false
```

CLI 参数优先级高于配置文件。`excludes` 是追加关系（配置文件 + CLI 合并）。

## go:generate 集成

在需要生成代码的文件中添加注释：

```go
//go:generate gogen .
```

**推荐做法**：将 `//go:generate` 放在包的入口文件（如 `doc.go` 或 `model.go`），而非散落在每个文件：

```go
// doc.go
//go:generate gogen .

package model
```

在项目根目录的 Makefile 中统一触发：

```makefile
.PHONY: generate
generate:
	go generate ./...
```

**CI 集成**：在 CI 流水线中使用 `gogen check` 验证生成文件是否最新，避免手动忘记重新生成：

```yaml
# GitHub Actions 示例
- name: Check generated files
  run: go run github.com/motocat46/gogen check ./...
```

```bash
# 或直接使用已安装的 gogen
gogen check ./...   # 生成文件过期时以非零状态退出
```

**Lint 集成**：使用 `gogen lint` 静态检查 struct tag 和注解，可在 CI 中接入：

```yaml
# GitHub Actions 示例
- name: Lint gogen tags
  run: go run github.com/motocat46/gogen lint ./...
```

```bash
# 或直接使用已安装的 gogen
gogen lint ./...    # tag/注解有 Error 级别问题时以非零状态退出
```

## Dirty Tracking

生成 `Modify(fn func(*T))` 作为唯一的变更入口。调用方在 fn 内修改结构体任意字段（包括嵌入的自定义结构体和第三方类型），fn 执行完毕后 gogen 统一调用 dirty 方法。**默认不生成（opt-in）**。

### 触发方式

| 方式 | 写法 | 说明 |
|------|------|------|
| 自动检测 | 结构体方法集包含零参 `MakeDirty()` | 含通过嵌入提升的 |
| 显式注解 | `// gogen:dirty` | 使用默认方法名 `MakeDirty()` |
| 自定义方法名 | `// gogen:dirty=MarkChanged` | 指定业务层方法名 |
| 禁用 | `// gogen:nodirty` | 最高优先级，覆盖自动检测 |
| 自定义 Modify 名 | `// gogen:modify=Apply` | 默认生成 `Modify()`，可改为任意名称 |

### 使用方式

```go
// 场景 1：自动检测（嵌入含 MakeDirty() 的类型）
type DirtyBase struct{}
func (d *DirtyBase) MakeDirty() {}

type Player struct {
    DirtyBase   // gogen 自动检测到 MakeDirty()
    Gold  int64
    Queue *ItemQueue  // 第三方/自定义类型，同样支持
}

// 生成：
func (p *Player) Modify(fn func(*Player)) {
    fn(p)
    p.MakeDirty()
}

// 调用：
player.Modify(func(p *Player) {
    p.Gold += 100          // 基础类型
    p.Queue.Enqueue(item)  // 自定义类型，无需额外处理
})

// 场景 2：自定义 dirty 方法名
// gogen:dirty=MarkChanged
type Entity struct {
    Name string
}
func (e *Entity) MarkChanged() {}
// 生成：func (e *Entity) Modify(fn func(*Entity)) { fn(e); e.MarkChanged() }

// 场景 3：禁用
// gogen:nodirty
type ReadOnlyView struct {
    DirtyBase
    Score float64
}
// 不生成 Modify()

// 场景 4：自定义 Modify 方法名
// gogen:dirty
// gogen:modify=Apply
type Command struct {
    Target string
}
func (c *Command) MakeDirty() {}
// 生成：func (c *Command) Apply(fn func(*Command)) { fn(c); c.MakeDirty() }
```

### 设计说明

setter（`SetGold`、`AppendTags` 等）作为字段操作工具独立存在，不触发 dirty。
所有需要标记 dirty 的修改统一通过 `Modify()` 完成：

```go
// ✓ 需要 dirty 时，走 Modify
player.Modify(func(p *Player) {
    p.SetGold(100)   // 或直接 p.Gold = 100，效果相同
})

// ✓ 只读或内部逻辑，直接用 setter/getter
value := player.GetGold()
```

## 嵌入提升方法保护

gogen 自动检测通过嵌入字段提升的方法，不生成同名方法，避免破坏接口实现：

```go
type SpeedEntity struct{ Speed float32 }
func (e *SpeedEntity) GetSpeed() float32                    { return e.Speed }
func (e *SpeedEntity) SetSpeed(speed, accel float32)        { e.Speed = speed }

type Vehicle struct {
    Speed float32   // gogen 想生成 GetSpeed/SetSpeed
    *SpeedEntity    // 但这两个方法已由 SpeedEntity 提升 → 自动跳过
}

// 接口实现由提升方法满足，不被 gogen 破坏
type IVehicle interface {
    GetSpeed() float32
    SetSpeed(speed, accel float32)
}
var _ IVehicle = (*Vehicle)(nil) // 编译通过
```

## 泛型支持

```go
type Container[T any] struct {
    Item T
    Tags []string
}

// 生成：
func (this *Container[T]) GetItem() T           { return this.Item }
func (this *Container[T]) SetItem(Item T)       { this.Item = Item }
func (this *Container[T]) GetTagsAt(index int) string { ... }
// ...
```

## 项目结构

```
gogen/
├── main.go                  # CLI 入口（cobra）
├── DECISIONS.md             # 设计决策记录（关键 trade-off 与方案选择依据）
├── pkg/                     # 各功能子包（见 pkg/README.md）
│   ├── loader/              # go/packages 包加载 + 两阶段恢复
│   ├── analyzer/            # go/types 语义分析 → model.StructDef
│   ├── model/               # 领域模型（TypeInfo / FieldDef / StructDef）
│   ├── generator/           # Registry 模式，按类型分发生成器
│   ├── writer/              # 文件写入 + goimports 格式化
│   ├── config/              # .gogen.yaml 配置文件加载
│   ├── annotations/         # gogen 注解统一解析（analyzer + linter 共用）
│   └── linter/              # struct tag 和注解静态检查
└── testdata/examples/       # 测试用例 + 黄金文件
```

## 开发者文档

| 文档 | 内容 |
|------|------|
| [DECISIONS.md](DECISIONS.md) | 关键设计决策记录（D-001 ~ D-022），记录 trade-off 与方案选择依据 |
| [pkg/README.md](pkg/README.md) | 各子包一览（loader / analyzer / model / generator / writer / config）|
| [pkg/loader/DESIGN.md](pkg/loader/DESIGN.md) | 两阶段加载机制、以包为分析单元的原因 |
| [pkg/analyzer/DESIGN.md](pkg/analyzer/DESIGN.md) | go/types 类型识别状态机、提升方法检测陷阱 |
| [pkg/generator/DESIGN.md](pkg/generator/DESIGN.md) | Registry 模式、plain 过滤实现、扩展新类型生成器 |
| [pkg/writer/DESIGN.md](pkg/writer/DESIGN.md) | 增量跳过算法、安全保护、goimports 集成 |
| [pkg/generator/README.md](pkg/generator/README.md) | 各生成器方法完整列表（含 plain/权限控制列） |
| [pkg/generator/TEST.md](pkg/generator/TEST.md) | 黄金文件清单及如何添加新测试场景 |

## 系统要求

- Go 1.24+

## 许可证

Apache License 2.0
