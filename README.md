# gogen — Go 结构体访问器代码生成器

为 Go 结构体字段自动生成 getter/setter 等访问器方法的 CLI 工具。

## 特性

- 基于 `go/types` 语义分析，支持泛型、类型别名、跨文件类型引用
- 自动跳过已有手写实现的方法（无冲突生成）
- 嵌入提升方法检测：不覆盖通过嵌入字段提升的方法，保护接口实现
- 增量生成：文件内容未变时跳过写入
- 孤儿文件清理：结构体删除后自动清理对应的生成文件
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
| 基础类型 `int` / `string` / 自定义类型 | `GetField()`, `SetField()` |
| 指针 `*T` | `GetField()`, `SetField()` |
| 结构体 `T` | `GetField()`, `SetField()` |
| 泛型实例 `List[T]` | `GetField()`, `SetField()` |
| 切片 `[]T` | `GetFieldElem()`, `GetFieldLen()`, `GetFieldCap()`, `RangeField()`, `SetFieldElem()`, `AddFieldElem()`, `DelFieldElem()` |
| 数组 `[N]T` | `GetFieldElem()`, `GetFieldLen()`, `RangeField()`, `SetFieldElem()` |
| Map `map[K]V` | `GetFieldVal()`, `RangeField()`, `SetFieldKV()`, `DelFieldKV()` |
| `interface{}` / `func` / `chan` | 跳过，不生成 |

设计原则：切片和 map 不提供整体 getter（`GetEmails() []string`），只提供细粒度操作，强制封装。

## struct tag 控制

在字段上添加 `gogen` tag 控制生成行为：

```go
type User struct {
    ID       int64  `gogen:"-"`        // 跳过，不生成任何方法
    Name     string `gogen:"readonly"` // 只生成 getter
    password string `gogen:"writeonly"`// 只生成 setter（小写字段默认跳过）
    Age      int                       // 默认：生成 getter + setter
}
```

| Tag | 效果 |
|---|---|
| `gogen:"-"` | 跳过此字段 |
| `gogen:"readonly"` | 只生成 getter（Get/Range/Elem/Len/Cap/Val） |
| `gogen:"writeonly"` | 只生成 setter（Set/Add/Del/SetKV/DelKV） |

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

或在项目根目录的 Makefile 中：

```makefile
generate:
	go generate ./...
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
func (this *Container[T]) GetTagsElem(index int) string { ... }
// ...
```

## 项目结构

```
gogen/
├── main.go                  # CLI 入口（cobra）
├── pkg/
│   ├── loader/              # go/packages 包加载 + 两阶段恢复
│   ├── analyzer/            # go/types 语义分析 → model.StructDef
│   ├── model/               # 领域模型（TypeInfo / FieldDef / StructDef）
│   ├── generator/           # Registry 模式，按类型分发生成器
│   ├── writer/              # 文件写入 + goimports 格式化
│   └── config/              # .gogen.yaml 配置文件加载
└── testdata/examples/       # 测试用例 + 黄金文件
```

## 系统要求

- Go 1.24+

## 许可证

Apache License 2.0
