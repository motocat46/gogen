# pkg/analyzer — 测试文档

## 测试文件

| 文件 | 覆盖范围 |
|------|---------|
| `analyzer_test.go` | 类型提取正确性、字段解析、提升方法检测、排除路径过滤、生成文件自动跳过 |

## 测试命令

### 快速验证
```bash
go test ./pkg/analyzer/... -count=1 -v
```

### 含竞态检测
```bash
go test ./pkg/analyzer/... -count=1 -race
```

## 覆盖的测试场景

| 场景 | 说明 |
|------|------|
| 基础类型提取 | bool/int/float/string 等基础字段的 TypeKind 和 TypeStr 正确识别 |
| 指针/slice/array/map | 复合类型的递归解析，Elem/Key/Value 指针正确填充 |
| 泛型结构体 | `type Pair[K, V any]` 的 TypeParams 提取，TypeParam 字段的 KindBasic 分类 |
| 类型别名 | `type MyTime = time.Time`，TypeStr 保留别名名称，IsAlias=true |
| 跨包类型 | `time.Time`、`sync.Mutex`：TypeStr 含包名前缀 |
| 具名类型底层解析 | `type Status string` → KindBasic；`type Tags []string` → KindSlice |
| 嵌入提升方法检测 | 值嵌入、指针嵌入、多层深度嵌入的提升方法集正确收集 |
| 手写方法识别 | 手写文件中的方法进入 ManualMethods，生成文件中的方法不进入 |
| 生成文件自动跳过 | `*_access.go`（含 Code generated 标记）不被分析 |
| 字段 tag 解析 | `gogen:"-"` / `"readonly"` / `"writeonly"` / `"plain"` / `"override"` 正确解析 |
| 结构体级 plain | `// gogen:plain` 注释传播到所有字段 |
| ExcludePaths 过滤 | 纯目录名（任意层级匹配）和绝对路径前缀匹配 |
| FileFilter 过滤 | 只处理指定文件中的结构体 |
| 非导出字段跳过 | 小写开头字段不出现在 Fields 中 |

## 注意事项

analyzer 测试依赖 `loader.Load` 加载真实包，需要在有效的 Go 模块根目录下运行（项目本身满足此条件）。

测试数据来自 `testdata/examples/types.go`，修改该文件后需同步更新测试预期。
