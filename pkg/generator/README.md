# pkg/generator — 使用文档

## 功能简介

`generator` 包是代码生成的核心层，使用 Registry 模式管理各类型字段的生成策略，将 `model.StructDef` 转换为 Go 源码字符串。

## 快速上手

```go
import (
    "github.com/motocat46/gogen/pkg/generator"
    "github.com/motocat46/gogen/pkg/model"
)

// 1. 创建包含所有内置生成器的注册表
reg := generator.NewRegistry()

// 2. 为一个结构体生成完整的访问器文件内容
code, err := reg.GenerateStruct(structDef)
if err != nil {
    // 处理错误
}
if code == nil {
    // 所有字段均已有手写实现，无需生成文件
}

// 3. 注册自定义生成器（覆盖或扩展）
reg.Register(model.KindBasic, &MyCustomGenerator{})
```

## 各生成器生成的方法

### BasicGenerator（KindBasic / KindStruct / KindGeneric）

适用类型：`string`、自定义类型（底层为 basic）、具名结构体、泛型实例、TypeParam（泛型类型参数）

| 方法 | 签名 | tag 控制 |
|------|------|---------|
| `GetField()` | `GetXxx() T` | readonly / plain 均生成 |
| `SetField()` | `SetXxx(v T)` | readonly 时跳过 |

### BoolGenerator（KindBool）

适用类型：`bool`

| 方法 | 签名 | plain 模式 |
|------|------|-----------|
| `GetField()` | `GetXxx() bool` | ✓ 生成 |
| `SetField()` | `SetXxx(v bool)` | ✓ 生成 |
| `ToggleField()` | `ToggleXxx()` | ✗ 跳过 |

### NumericGenerator（KindNumeric）

适用类型：`int`、`int8`~`int64`、`uint`~`uint64`、`float32`、`float64`、`complex64`、`complex128`

| 方法 | 签名 | plain 模式 |
|------|------|-----------|
| `GetField()` | `GetXxx() T` | ✓ 生成 |
| `SetField()` | `SetXxx(v T)` | ✓ 生成 |
| `AddField()` | `AddXxx(delta T)` | ✗ 跳过 |
| `SubField()` | `SubXxx(delta T)` | ✗ 跳过 |

### NilableGenerator（KindPointer / KindInterface / KindFunc）

适用类型：`*T`、`interface{}`、`any`、具名接口、`func(...)`

| 方法 | 签名 | plain 模式 |
|------|------|-----------|
| `GetField()` | `GetXxx() T` | ✓ 生成 |
| `SetField()` | `SetXxx(v T)` | ✓ 生成 |
| `HasField()` | `HasXxx() bool` | ✗ 跳过 |

### SliceGenerator（KindSlice）

适用类型：`[]T`

| 方法 | 签名 | plain 模式 | 需要读权限 | 需要写权限 |
|------|------|-----------|-----------|-----------|
| `GetFieldAt(index int)` | `GetXxxAt(int) T` | ✓ 生成 | ✓ | - |
| `GetFieldLen()` | `GetXxxLen() int` | ✗ 跳过 | ✓ | - |
| `RangeField(fn)` | `RangeXxx(func(int, T) bool)` | ✓ 生成 | ✓ | - |
| `HasField()` | `HasXxx() bool` | ✗ 跳过 | ✓ | - |
| `GetFieldCopy()` | `GetXxxCopy() []T` | ✗ 跳过 | ✓ | - |
| `SetFieldAt(index, elem)` | `SetXxxAt(int, T)` | ✓ 生成 | - | ✓ |
| `AppendField(elems...)` | `AppendXxx(...T)` | ✓ 生成 | - | ✓ |
| `DeleteFieldAt(index)` | `DeleteXxxAt(int)` | ✓ 生成 | - | ✓ |

### ArrayGenerator（KindArray）

适用类型：`[N]T`（长度固定，不支持 Append/Delete）

| 方法 | 签名 | plain 模式 | 需要读权限 | 需要写权限 |
|------|------|-----------|-----------|-----------|
| `GetField()` | `GetXxx() [N]T` | ✓ 生成 | ✓ | - |
| `GetFieldAt(index int)` | `GetXxxAt(int) T` | ✓ 生成 | ✓ | - |
| `GetFieldLen()` | `GetXxxLen() int` | ✗ 跳过 | ✓ | - |
| `RangeField(fn)` | `RangeXxx(func(int, T) bool)` | ✓ 生成 | ✓ | - |
| `SetFieldAt(index, elem)` | `SetXxxAt(int, T)` | ✓ 生成 | - | ✓ |

### MapGenerator（KindMap）

适用类型：`map[K]V`

| 方法 | 签名 | plain 模式 | 需要读权限 | 需要写权限 |
|------|------|-----------|-----------|-----------|
| `GetFieldVal(key)` | `GetXxxVal(K) (V, bool)` | ✓ 生成 | ✓ | - |
| `GetFieldValOrDefault(key, def)` | `GetXxxValOrDefault(K, V) V` | ✗ 跳过 | ✓ | - |
| `RangeField(fn)` | `RangeXxx(func(K, V) bool)` | ✓ 生成 | ✓ | - |
| `HasField()` | `HasXxx() bool` | ✗ 跳过 | ✓ | - |
| `HasFieldKey(key)` | `HasXxxKey(K) bool` | ✗ 跳过 | ✓ | - |
| `GetFieldLen()` | `GetXxxLen() int` | ✗ 跳过 | ✓ | - |
| `GetFieldKeys()` | `GetXxxKeys() []K` | ✗ 跳过 | ✓ | - |
| `GetFieldCopy()` | `GetXxxCopy() map[K]V` | ✗ 跳过 | ✓ | - |
| `EnsureField()` | `EnsureXxx() map[K]V` | ✓ 生成 | - | ✓ |
| `SetFieldVal(key, value)` | `SetXxxVal(K, V)` | ✓ 生成 | - | ✓ |
| `DeleteFieldKey(key)` | `DeleteXxxKey(K)` | ✓ 生成 | - | ✓ |

## API 说明

### `NewRegistry() *Registry`

创建并返回已注册所有内置生成器的 Registry 实例。

### `(*Registry).Register(kind model.TypeKind, g MethodGenerator)`

注册自定义生成器，若该 Kind 已有注册则覆盖。用于扩展新类型或替换内置策略。

### `(*Registry).GenerateStruct(s *model.StructDef) ([]byte, error)`

为一个结构体生成完整的访问器文件内容（含文件头和所有字段的方法）。

- 返回 `nil, nil`：所有字段均被跳过或已有手写实现，调用方应跳过写文件
- 返回 `[]byte, nil`：未格式化的原始 Go 源码（需通过 writer 层格式化后写入）
- 返回 `nil, error`：生成过程中出现错误

### `MethodGenerator` 接口

```go
type MethodGenerator interface {
    Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error)
}
```

实现此接口并调用 `Register` 即可支持新的 TypeKind。

## 注意事项

- 生成的代码**不含** `package` 声明和 `import`，由 `GenerateStruct` 统一添加文件头，由 `writer` 层通过 `goimports` 自动推断 import
- 生成文件不含时间戳，确保相同输入在任何环境产生相同字节（幂等性）
- 方法名冲突检查（字段名、手写方法、提升方法）由 `model.StructDef.CanGenerateMethod` 完成，生成器通过 `resolveCanGen` 调用
- `KindUnsupported`（chan）不注册任何生成器，会被静默跳过
