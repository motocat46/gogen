# pkg/linter — gogen 静态检查

## 功能简介

`linter` 包对 Go 结构体的 gogen struct tag 和文档注释注解做静态检查，在不生成任何代码的情况下提前发现配置错误。

**检查项一览：**

| 检查 | 级别 | 示例 |
|------|------|------|
| 未知 tag 选项（含拼写建议） | Error | `gogen:"raedonly"` → 是否指的是 `"readonly"`？ |
| 矛盾 tag 组合 | Error | `gogen:"readonly,writeonly"`、`gogen:"-,plain"` |
| 字段级 dirty tag（已废弃） | Error | `gogen:"dirty=MakeDirty"` → 改用结构体注解 |
| `gogen:dirty=XXX` 方法不存在 | Error | 指定方法不在 `*T` 的方法集中，生成代码将无法编译 |
| `gogen:modify=XXX` 无 dirty tracking | Warning | modify= 指定了但 dirty tracking 未启用，该注解不会生效 |

## 快速上手

```go
import "github.com/motocat46/gogen/pkg/linter"

issues, err := linter.Lint(workDir, linter.Config{
    ExcludePaths: []string{"vendor", "testdata"},
}, "./...")
if err != nil {
    return err
}
for _, iss := range issues {
    fmt.Println(iss) // file:line:col: [error/warning] message
}
```

## API 说明

### `Lint(dir string, cfg Config, patterns ...string) ([]Issue, error)`

加载 `patterns` 指定的包并对所有 struct 执行全部检查，返回按文件位置排序的问题列表。

- `dir`：工作目录，用于解析相对 pattern 和配置文件路径
- `patterns`：与 `go/packages` 格式一致（`.`、`./...`、`pkg/model` 等）
- 返回的 `error` 仅表示加载/类型检查失败，问题本身通过 `[]Issue` 返回

### `Issue`

```go
type Issue struct {
    Pos      token.Position // 文件:行:列
    Severity Severity       // Error 或 Warning
    Message  string
}
```

`Issue.String()` 输出 go vet 风格：`path/to/file.go:12:3: [error] 字段 Name：...`

### `Severity`

| 常量 | 含义 |
|------|------|
| `Error` | 会导致编译错误或语义错误，`gogen lint` 以非零状态退出 |
| `Warning` | 无害但可能不符合预期，不影响退出码 |

### `Config`

```go
type Config struct {
    ExcludePaths []string // 排除路径（纯目录名或绝对路径前缀）
}
```

调用方负责在传入前合并默认排除与用户排除（参考 `main.go` 的 `buildExcludePaths`）。

## CLI 用法

```bash
# 检查当前项目，发现 Error 时非零退出
gogen lint ./...

# 接入 CI（GitHub Actions）
- run: gogen lint ./...
```

## 注意事项

- linter 只做静态检查，**不生成、不修改任何文件**
- 仅检查包级 struct 声明（与代码生成范围一致，函数体内的局部类型不在范围内）
- 注解解析依赖 `pkg/annotations` 包，与 `pkg/analyzer` 的解析行为完全一致
