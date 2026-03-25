# pkg — 子包索引

gogen 各功能模块一览，按数据流顺序排列：

| 子包 | 职责 | 文档 |
|------|------|------|
| [loader](loader/) | 使用 `go/packages` 加载包，内置两阶段错误恢复 | [README](loader/README.md) · [DESIGN](loader/DESIGN.md) · [TEST](loader/TEST.md) |
| [analyzer](analyzer/) | `go/types` 语义分析，输出 `model.StructDef` | [README](analyzer/README.md) · [DESIGN](analyzer/DESIGN.md) · [TEST](analyzer/TEST.md) |
| [model](model/) | 领域模型（`TypeInfo` / `FieldDef` / `StructDef`），分析层与生成层的契约 | [README](model/README.md) · [DESIGN](model/DESIGN.md) · [TEST](model/TEST.md) |
| [generator](generator/) | Registry 模式，按 TypeKind 分发生成器，输出 Go 源码 | [README](generator/README.md) · [DESIGN](generator/DESIGN.md) · [TEST](generator/TEST.md) |
| [writer](writer/) | `goimports` 格式化 + 增量写入 + 孤儿文件清理 | [README](writer/README.md) · [DESIGN](writer/DESIGN.md) · [TEST](writer/TEST.md) |
| [config](config/) | 加载 `.gogen.yaml` 配置文件 | [README](config/README.md) · [DESIGN](config/DESIGN.md) · [TEST](config/TEST.md) |
| [annotations](annotations/) | 统一解析结构体文档注释中的 gogen 注解（plain / dirty / nodirty / modify），供 analyzer 和 linter 共用 | [README](annotations/README.md) |
| [linter](linter/) | 静态检查 struct tag 和注解有效性（拼写、矛盾组合、dirty 方法引用、modify 无效配置） | [README](linter/README.md) · [TEST](linter/TEST.md) |

数据流：`loader` → `analyzer` → `model` → `generator` → `writer`；`config` 在 CLI 层注入各阶段。`annotations` 被 `analyzer` 和 `linter` 共用；`linter` 独立于生成流程运行。
