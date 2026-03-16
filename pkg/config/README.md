# pkg/config — 使用文档

## 功能简介

`config` 包负责从项目目录加载 `.gogen.yaml` 配置文件，提供对 CLI 参数的持久化配置支持。

**优先级规则：CLI 参数 > 配置文件 > 内置默认值**

## 配置文件示例

在项目根目录（与 `go.mod` 同级）创建 `.gogen.yaml`：

```yaml
# 生成文件的后缀名（不含下划线和 .go），默认 access
suffix: access

# 统一输出目录；为空时与源文件同目录（默认行为）
output: ""

# 额外排除的路径列表
# 纯目录名：匹配路径中任意一段（如 "mock" 匹配所有层级的 mock 目录）
# 路径前缀：含路径分隔符时使用前缀匹配
excludes:
  - mock
  - mocks
  - proto
  - vendor

# 禁用内置默认排除列表（vendor、testdata、pb 等），默认 false
no-default-excludes: false
```

## 快速上手

```go
import "github.com/motocat46/gogen/pkg/config"

// 从工作目录加载 .gogen.yaml
cfg, err := config.Load(workDir)
if err != nil {
    return err // 文件格式错误
}
// 文件不存在时 cfg 为零值，err 为 nil

// 合并到 CLI 参数（CLI 优先）
suffix := cliSuffix
if suffix == "" && cfg.Suffix != "" {
    suffix = cfg.Suffix
}
excludes := append(cliExcludes, cfg.Excludes...)
```

## API 说明

### `Load(dir string) (File, error)`

从指定目录加载 `.gogen.yaml`。

- 文件不存在：返回 `File{}, nil`（正常情况，不报错）
- 文件存在但 YAML 格式错误：返回 error
- 成功：返回解析后的 `File`

### `File` 结构体

| 字段 | YAML key | 类型 | 对应 CLI 参数 | 说明 |
|------|----------|------|------------|------|
| `Suffix` | `suffix` | string | `--suffix` | 生成文件名后缀 |
| `Output` | `output` | string | `--output` | 统一输出目录 |
| `Excludes` | `excludes` | []string | `--exclude`（可多次） | 额外排除路径 |
| `NoDefaultExcludes` | `no-default-excludes` | bool | `--no-default-excludes` | 禁用内置排除 |

### `FileName` 常量

```go
const FileName = ".gogen.yaml"
```

## 注意事项

- 配置文件只影响本次运行，不是持久状态；每次运行都重新加载
- `Excludes` 与 CLI `--exclude` 是**追加**关系（`append(cliExcludes, cfg.Excludes...)`），不是覆盖
- 配置文件不支持通配符（`*`）；使用纯目录名实现"任意层级"的模糊匹配
