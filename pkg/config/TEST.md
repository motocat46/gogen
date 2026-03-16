# pkg/config — 测试文档

## 测试文件

| 文件 | 覆盖范围 |
|------|---------|
| `config_test.go` | YAML 解析、文件不存在、格式错误、各字段正确读取 |

## 测试命令

### 快速验证
```bash
go test ./pkg/config/... -count=1 -v
```

### 含竞态检测
```bash
go test ./pkg/config/... -count=1 -race
```

## 覆盖的测试场景

| 场景 | 预期行为 |
|------|---------|
| 文件不存在 | 返回 `File{}, nil`（不报错） |
| 空文件 | 返回 `File{}, nil` |
| 完整配置 | 所有字段正确解析 |
| 仅部分字段 | 未设置字段为零值 |
| YAML 格式错误 | 返回 error |
| suffix 字段 | 正确读取字符串 |
| output 字段 | 正确读取字符串 |
| excludes 字段 | 正确读取字符串切片 |
| no-default-excludes | 正确读取 bool |
