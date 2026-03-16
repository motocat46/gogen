# pkg/model — 测试文档

## 测试文件

| 文件 | 覆盖范围 |
|------|---------|
| `model_test.go` | TypeKind.String()、TypeInfo 构造、FieldDef 方法、StructDef 方法冲突检查、ParseFieldConfig |

## 测试命令

### 快速验证
```bash
go test ./pkg/model/... -count=1 -v
```

### 含竞态检测
```bash
go test ./pkg/model/... -count=1 -race
```

## 覆盖的测试场景

| 场景 | 说明 |
|------|------|
| TypeKind.String() | 所有 Kind 的字符串表示正确 |
| ParseFieldConfig | `"-"` / `"readonly"` / `"writeonly"` / `"plain"` / `"override"` 正确解析 |
| ParseFieldConfig 组合 | `"readonly,plain"` 等逗号分隔的组合 |
| ParseFieldConfig 无 tag | 返回零值 `FieldConfig{}` |
| IsReadable / IsWritable | Skip/Readonly/WriteOnly 各种组合的正确结果 |
| CanGenerateMethod | 三层检查：字段名冲突、手写方法冲突、提升方法冲突 |
| CanGenerateMethodOverride | 两层检查：字段名冲突、手写方法冲突（跳过提升方法） |
| ReceiverType | 非泛型结构体返回 `"Name"`；泛型返回 `"Name[K, V]"` |
| ActiveFields | 过滤 Skip=true 的字段 |
