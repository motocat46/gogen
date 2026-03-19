# pkg/generator — 测试文档

## 测试文件列表

| 文件 | 覆盖范围 |
|------|---------|
| `golden_test.go` | 全链路黄金文件对比测试：loader→analyzer→generator，验证所有结构体的生成输出与提交的黄金文件逐字节一致 |
| `plain_test.go` | plain 模式专项测试：验证 `gogen:"plain"` 时各类型只生成核心方法（Get/Set），扩展方法（Toggle/Add/Sub/Has/Copy 等）被正确跳过 |

## 黄金文件清单

黄金文件位于 `testdata/examples/`，每个文件对应一个或多个结构体的生成输出：

| 黄金文件 | 覆盖的结构体 | 重点测试场景 |
|---------|-----------|------------|
| `alltypes_access.go` | `AllTypes` | 所有 TypeKind 全覆盖（basic/bool/numeric/pointer/slice/array/map/struct/generic/interface/func） |
| `baseinfo_access.go` | `BaseInfo` | 基础字段类型（string/int/bool） |
| `sliceonly_access.go` | `SliceOnly` | 切片字段：GetAt/GetLen/Range/Has/GetCopy/SetAt/Append/DeleteAt |
| `maponly_access.go` | `MapOnly` | map 字段：GetVal/GetValOrDefault/Range/Has/HasKey/GetLen/GetKeys/GetCopy/Ensure/SetVal/DeleteKey |
| `arrayonly_access.go` | `ArrayOnly` | 数组字段：Get/GetAt/GetLen/Range/SetAt |
| `plainstruct_access.go` | `PlainStruct` | gogen:plain 注解，只生成核心 Get/Set |
| `tagcontrol_access.go` | `TagControl` | gogen:"readonly" / "writeonly" / "-" |
| `embedbyvalue_access.go` | `EmbedByValue` | 值嵌入：提升方法保护，不覆盖嵌入字段提升的方法 |
| `embedbypointer_access.go` | `EmbedByPointer` | 指针嵌入：提升方法保护 |
| `embeddeep_access.go` | `EmbedDeep` | 深层嵌入（多层传递）的提升方法保护 |
| `embedother_access.go` | `EmbedOther` | 嵌入外部包类型 |
| `basewithmethods_access.go` | `BaseWithMethods` | 手写方法存在时，生成器跳过同名方法 |
| `secondbase_access.go` | `SecondBase` | 继承链中第二个嵌入 |
| `fieldsameaspromoted_access.go` | `FieldSameAsPromoted` | 字段名与提升方法同名的冲突处理 |
| `overrideembed_access.go` | `OverrideEmbed` | gogen:"override" 强制覆盖嵌入提升方法 |
| `pair_access.go` | `Pair` | 泛型结构体 `Pair[K, V]` |
| `container_access.go` | `Container` | 泛型结构体（含 slice/map 字段） |
| `speedentity_access.go` | `SpeedEntity` | 数值类型的 Add/Sub 方法 |
| `autodirtyplayer_access.go` | `AutoDirtyPlayer` | 自动检测 dirty（嵌入含 MakeDirty），数值/slice/map 写方法注入；数值 Set 幂等检查，slice SetAt 幂等检查（string 可比较），Append/Delete/SetVal/DeleteKey 无幂等检查 |
| `customdirtyentity_access.go` | `CustomDirtyEntity` | 自定义 dirty 方法名 `gogen:dirty=MarkChanged` |
| `nodirtyplayer_access.go` | `NoDirtyPlayer` | `gogen:nodirty` 最高优先级：禁用所有注入，字段级 tag 失效 |
| `fieldoverrideentity_access.go` | `FieldOverrideEntity` | 字段级覆盖：`ModuleScore` 使用 `MarkModule()`，`Gold` 使用结构体级 `MakeDirty()` |
| `autodirtycollections_access.go` | `AutoDirtyCollections` | 集合类型 dirty 注入：slice/map 写方法直接注入；数组 SetAt 有幂等检查（int32 可比较）|
| `resetwithdirtyplayer_access.go` | `ResetWithDirtyPlayer` | Reset + dirty 交互：`Reset()` 末尾注入 dirty 调用 |

## 可执行测试命令

### 快速验证（推荐日常使用）
```bash
go test ./pkg/generator/... -count=1 -v
```

### 全量测试（含竞态检测）
```bash
go test ./... -count=1 -race
```

### 仅运行黄金文件测试
```bash
go test ./pkg/generator/... -count=1 -run TestGoldenFiles -v
```

### 仅运行 plain 模式测试
```bash
go test ./pkg/generator/... -count=1 -run TestPlain -v
```

### 更新黄金文件（修改了生成逻辑后执行）
```bash
go run . --no-default-excludes ./testdata/examples
```

更新后需重新运行测试验证，然后提交新的黄金文件。

## 如何添加新的测试场景

1. 在 `testdata/examples/types.go`（通用类型）或对应主题文件（如 `dirty_cases.go`）中添加测试结构体
2. 运行 `go run . --no-default-excludes ./testdata/examples` 生成对应的 `*_access.go`
3. 检查生成内容是否符合预期
4. 运行 `go test ./pkg/generator/... -count=1 -run TestGoldenFiles` 确认通过
5. 提交源文件和新生成的 `*_access.go`

> **dirty 场景注意事项**：含 `MakeDirty()` 的嵌入类型（如 `DirtyBase`）若被 gogen 处理，会为其生成 `Reset()` 方法，进而通过提升机制阻止嵌入结构体生成自己的 `Reset()`。解决方案：在源文件中为该基础类型手写一个空的 `Reset()` 方法，阻止 gogen 生成。

## 黄金文件比对规则

`normalizeForCompare` 在比对前规范化以下差异（不影响代码语义）：
- 去掉 import 语句（`imports.Process` 自动推断，内存生成不产生）
- 折叠连续空行（`gofmt` 在声明之间加空行，模板不加）

其余内容（方法签名、方法体、注释）必须逐字节相同。
