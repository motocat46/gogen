# pkg/analyzer — 测试文档

## 测试文件

| 文件 | 覆盖范围 |
|------|---------|
| `analyzer_test.go` | 类型提取正确性、字段解析、提升方法检测、排除路径过滤、生成文件自动跳过 |
| `analyzer_internal_test.go` | 私有函数 `isExcluded` 的全边界场景（8 case）|
| `typematrix_test.go` | 类型矩阵单元测试：直接对各 TypeKind 做字段级断言，独立于黄金文件 |

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

### typematrix_test.go（类型矩阵直接断言）

| 测试函数 | 验证内容 |
|---------|---------|
| `TestTypeMatrix_Basic` | string → KindBasic，无 Elem |
| `TestTypeMatrix_Bool` | bool → KindBool |
| `TestTypeMatrix_Numeric` | int/int64/float64/uint32 → KindNumeric，TypeStr 精确 |
| `TestTypeMatrix_Pointer` | *int 和 *struct → KindPointer，Elem 正确递归 |
| `TestTypeMatrix_Slice` | []int 和 []*struct → KindSlice，Elem 种类正确 |
| `TestTypeMatrix_Array` | [8]int → KindArray，ArrayLen="8"，Elem 正确 |
| `TestTypeMatrix_Map` | map[string]int → KindMap，Key/Value 正确 |
| `TestTypeMatrix_Struct` | time.Time → KindStruct，TypeStr 含包名前缀 |
| `TestTypeMatrix_Interface` | interface{} 和 any → KindInterface |
| `TestTypeMatrix_Func` | func(int) string → KindFunc |
| `TestTypeMatrix_Chan` | chan int → KindUnsupported |
| `TestTypeMatrix_NamedTypeUnderlying` | 具名类型底层解析：TypeStr 保留具名名称，Kind 由底层决定 |
| `TestTypeMatrix_TypeAlias` | type MyTime = time.Time → IsAlias=true，Kind=KindStruct |
| `TestTypeMatrix_GenericTypeParam` | T/K/V → KindBasic，TypeStr 为参数名 |
| `TestTypeMatrix_NestedComposite` | map[string][]string 和 []map[string]int 的递归解析 |
| `TestTypeMatrix_GenericInstance` | Container[int] → KindGeneric，TypeArgs 递归解析（int → KindNumeric） |

## 注意事项

analyzer 测试依赖 `loader.Load` 加载真实包，需要在有效的 Go 模块根目录下运行（项目本身满足此条件）。

测试数据来自 `testdata/examples/types.go`，修改该文件后需同步更新测试预期。
