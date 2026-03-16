# pkg/loader — 测试文档

## 测试文件

| 文件 | 覆盖范围 |
|------|---------|
| `loader_test.go` | 包加载、patterns 规范化、排除路径过滤、两阶段错误恢复 |

## 测试命令

### 快速验证
```bash
go test ./pkg/loader/... -count=1 -v
```

### 含竞态检测
```bash
go test ./pkg/loader/... -count=1 -race
```

## 覆盖的测试场景

| 场景 | 说明 |
|------|------|
| 正常包加载 | `./...` 模式成功加载多个包 |
| 单文件模式 | `./foo.go` 自动转为 `file=` 格式，加载整个包 |
| ExcludePaths 纯目录名 | `"mock"` 匹配路径中任意层级的 mock 目录 |
| ExcludePaths 绝对路径 | 前缀匹配，精确排除指定目录 |
| 两阶段恢复：有问题的生成文件 | 旧 `*_access.go` 有编译错误时，overlay 后重新加载成功 |
| 两阶段恢复：用户代码错误 | 用户代码有真实编译错误时，正确返回错误而非静默忽略 |
| ExtractFileFilter | 从 patterns 正确提取文件绝对路径 |
| no Go files 静默跳过 | 不含 .go 文件的目录不触发错误 |

## 注意事项

loader 测试需要真实的 Go 模块环境，测试数据使用项目自身的 `testdata/examples` 包。
