# pkg/annotations — 测试文档

## 测试文件

| 文件 | 覆盖范围 |
|------|---------|
| `annotations_test.go` | `ParseStructAnnotations` 全场景（13 case）+ `MethodSetContains` 完整测试（6 case） |
| `testmain_test.go` | goroutine leak 检测（goleak） |

## 测试命令

### 快速验证
```bash
go test ./pkg/annotations/... -count=1 -v
```

### 含竞态检测
```bash
go test ./pkg/annotations/... -count=1 -race
```

## 覆盖的测试场景

### ParseStructAnnotations

| 场景 | 预期行为 |
|------|---------|
| 空文档 | 返回零值 `StructAnnotations{}` |
| `gogen:plain` | `Plain = true` |
| `gogen:nodirty` | `NoDirty = true` |
| `gogen:dirty`（无值） | `DirtyMethod = "MakeDirty"` |
| `gogen:dirty=CustomDirty` | `DirtyMethod = "CustomDirty"` |
| `gogen:dirty=`（空值） | 不生效，`DirtyMethod = ""` |
| `gogen:modify=Apply` | `ModifyMethod = "Apply"` |
| `gogen:modify=`（空值） | 不生效，`ModifyMethod = ""` |
| 多注解组合 | 各字段独立设置，互不干扰 |
| 忽略无关行 | 非 gogen: 前缀的行被跳过 |
| 行首尾空格裁剪 | `TrimSpace` 后正常解析 |
| `gogen:dirty` 后置 `gogen:dirty=XXX` | 后者覆盖前者 |
| `nodirty` 与 `dirty` 共存 | 解析层两者都记录，优先级由调用方处理 |

### MethodSetContains

| 场景 | 预期行为 |
|------|---------|
| 包含匹配的零参无返回值方法 | 返回 `true` |
| 指定方法名不存在 | 返回 `false` |
| 无任何方法的类型 | 返回 `false` |
| 方法存在但有参数 | 返回 `false`（签名不匹配） |
| 方法存在但有返回值 | 返回 `false`（签名不匹配） |
| 自定义方法名 | 按名精确匹配，其他名不匹配 |
