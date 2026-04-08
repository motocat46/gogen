# pkg/linter — 测试文档

## 测试文件列表

| 文件 | 覆盖范围 |
|------|---------|
| `linter_test.go` | 端到端测试：加载 testdata/lint/ 下各场景目录，验证 Error/Warning 数量 |

## 测试数据目录（testdata/lint/）

每个子目录是一个独立的 Go 包，包含触发特定检查项的结构体定义：

| 子目录 | 场景 | 预期结果 |
|--------|------|---------|
| `bad_tags/` | 未知 tag 选项（含拼写建议）、字段级 dirty tag（已废弃） | 3 Error |
| `contradictions/` | `readonly+writeonly`、`-+plain` 矛盾组合 | 2 Error |
| `dirty_missing/` | `gogen:dirty=NonExistentMethod` 指定的方法不存在 | 1 Error |
| `modify_no_dirty/` | `gogen:modify=XXX` 但未启用 dirty tracking | 2 Warning |
| `valid/` | 合法的 gogen 注解（含 `gogen:dirty`、`gogen:modify=`） | 0 Error，0 Warning |
| `multi_file/` | dirty 方法定义在同包不同文件，验证跨文件类型检查解析 | 0 Error，0 Warning |

## 可执行测试命令

### 全量测试
```bash
go test ./pkg/linter/... -count=1 -v
```

### 含竞态检测
```bash
go test ./pkg/linter/... -count=1 -race
```

### 仅跑指定场景
```bash
go test ./pkg/linter/... -count=1 -run TestLint/modify= -v
```

## 如何添加新检查场景

1. 在 `testdata/lint/` 下创建新子目录（即新 Go 包），添加触发场景的结构体定义
2. 在 `linter_test.go` 的 `cases` 中新增一条，填入预期的 Error/Warning 数量
3. 先运行测试确认**失败**（TDD 红阶段）
4. 在 `checks.go` 中实现对应检查，从 `checkStruct` 调用
5. 再次运行测试确认**通过**（TDD 绿阶段）
