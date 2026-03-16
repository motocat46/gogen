# pkg/writer — 测试文档

## 测试文件

| 文件 | 覆盖范围 |
|------|---------|
| `writer_test.go` | 文件写入、增量跳过、安全保护、Clean 孤儿清理、Check 验证模式 |

## 测试命令

### 快速验证
```bash
go test ./pkg/writer/... -count=1 -v
```

### 含竞态检测
```bash
go test ./pkg/writer/... -count=1 -race
```

## 覆盖的测试场景

| 场景 | 说明 |
|------|------|
| 首次写入 | 文件不存在时，格式化并创建，返回 `written=true` |
| 增量跳过 | 生成内容与磁盘完全一致时，返回 `written=false` |
| 内容变更后写入 | 生成内容与磁盘不一致时，覆盖写入，返回 `written=true` |
| 安全保护 | 目标文件存在但不含 gogen 标记时，返回错误，不覆盖 |
| DryRun 模式 | 只打印路径，不实际写入，返回 `written=false` |
| 输出目录自动创建 | OutputDir 不存在时自动 mkdir -p |
| Clean 删除孤儿文件 | 对应生成文件存在时删除；不存在时无操作 |
| Check 一致性验证 | 文件最新时返回 `upToDate=true`；需更新返回 `false` |
| Check code=nil | 文件不存在返回最新；文件存在且含 gogen 标记返回非最新 |
| OutputFilename | 结构体名转小写 + 后缀拼接正确 |
| IsGogenGenerated | 含标记返回 true；不含返回 false；前 1KB 内检测 |

## 注意事项

- writer 测试创建临时目录（`t.TempDir()`），测试结束后自动清理
- 格式化行为依赖 `golang.org/x/tools/imports`，需要网络可达或 module cache 已有相关依赖
