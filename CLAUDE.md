# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gogen` is a Go code generator tool that automatically generates accessor methods (getters/setters) for struct fields. It uses `go/types` for semantic analysis and `text/template` for code generation.

### Core Architecture

```
CLI (main.go + cobra)
  → pkg/loader/    go/packages 加载包，含两阶段错误恢复
  → pkg/analyzer/  go/types 语义分析 → model.StructDef
  → pkg/model/     领域模型（TypeInfo / FieldDef / StructDef）
  → pkg/generator/ Registry 模式，per-kind 生成器
  → pkg/writer/    文件写入 + golang.org/x/tools/imports 格式化
  → pkg/config/    .gogen.yaml 配置文件加载
```

### Key Design Decisions

- `go/types` 替代手工 AST 类型解析：支持泛型、类型别名、跨文件引用、interface/func 类型
- Generator Registry：新增类型支持只需实现 `MethodGenerator` 接口并注册
- 嵌入提升方法检测（`collectPromotedMethods`）：不覆盖通过嵌入字段提升的方法，保护接口实现
- 增量生成：格式化后与磁盘内容对比，内容相同则跳过写入
- 孤儿文件清理：结构体删除后自动清理对应 `*_access.go`
- 并行化：Generate+Write 阶段用 `errgroup`，并发数限制为 `runtime.NumCPU()`

### Supported Field Types

| Kind | 生成方法 |
|---|---|
| basic（含 interface/func） | Get/Set |
| pointer | Get/Set |
| struct | Get/Set |
| generic instance | Get/Set |
| slice | Elem/Len/Cap/Range/Add/Del |
| array | Elem/Len/Range/SetElem |
| map | Val/Range/SetKV/DelKV |
| chan | 跳过（封装弊大于利） |

## Development Commands

```bash
go build ./...                                     # 构建
go test ./... -count=1 -race                       # 全量测试（含竞态检测）
go run . ./...                                     # 对当前项目运行生成
go run . --no-default-excludes ./testdata/examples # 更新黄金文件
go run . check ./...                               # 验证生成文件是否最新（CI 用）
go run . --dry-run ./...                           # 预览模式
```

### struct tag 控制

```
gogen:"-"         跳过此字段
gogen:"readonly"  只生成 getter
gogen:"writeonly" 只生成 setter
```

## Code Style Notes

- 中文注释，Apache 2.0 license header
- 生成文件含 "// Code generated ... DO NOT EDIT." 标记
- module: `github.com/motocat46/gogen`，Go 1.24+

## 与 Claude 协作偏好

**工作方式:**
- 修改代码前必须先用 Read 工具理解现有实现
- 修改后运行 `go build ./...`、`go test ./... -count=1` 验证
- 中文回答技术问题

**代码规范:**
- 中文注释，严格遵循 Go 最佳实践
- 完整错误处理，不暴露 slice/map 的整体 getter（只提供细粒度操作）
- 不生成时间戳（确保增量对比可靠）
