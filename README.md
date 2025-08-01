# GoGen - Go 结构体访问器代码生成器

一个强大的 Go 代码生成工具，自动为结构体字段生成访问器方法（getter/setter）。

## 功能特性

- 🚀 **智能解析**：使用 Go AST 解析结构体定义
- 🎯 **多类型支持**：支持基础类型、切片、数组、映射和嵌套结构体
- 📦 **批量处理**：可同时处理多个 Go 文件
- ⚡ **自动格式化**：生成的代码自动使用 `goimports` 格式化
- 🛠️ **CLI 工具**：提供完整的命令行界面

## 支持的访问器类型

| 字段类型 | 生成方法 |
|---------|---------|
| **基础类型** | `Get<Field>()`, `Set<Field>()` |
| **切片** | 元素访问、长度查询、添加/删除操作 |
| **数组** | 元素访问、长度查询、元素设置 |
| **映射** | 键值访问、遍历、设置/删除操作 |
| **结构体** | Get/Set 方法，支持指针和值类型 |

## 快速开始

### 安装

```bash
git clone <repository>
cd gogen
go build
```

### 基本用法

```bash
# 为单个文件生成代码
./gogen user.go

# 为多个文件生成代码
./gogen *.go

# 指定输出目录
./gogen --output ./generated *.go

# 预览模式（不实际生成文件）
./gogen --dry-run user.go

# 详细输出
./gogen --verbose user.go
```

## 命令选项

| 选项 | 简写 | 描述 |
|-----|------|------|
| `--output` | `-o` | 指定输出目录 |
| `--package` | `-p` | 指定生成代码的包名 |
| `--verbose` | `-v` | 显示详细输出 |
| `--dry-run` |  | 预览模式，不实际生成文件 |

## 示例

假设有以下结构体：

```go
type User struct {
    Name    string
    Age     int
    Emails  []string
    Profile map[string]interface{}
}
```

生成的访问器方法包括：

```go
// 基础类型访问器
func (u *User) GetName() string { return u.Name }
func (u *User) SetName(name string) { u.Name = name }

// 切片操作方法
func (u *User) GetEmails() []string { return u.Emails }
func (u *User) AddEmail(email string) { u.Emails = append(u.Emails, email) }

// 映射操作方法
func (u *User) GetProfile() map[string]interface{} { return u.Profile }
func (u *User) SetProfileValue(key string, value interface{}) { u.Profile[key] = value }
```

## 项目结构

```
gogen/
├── main.go                                # CLI 入口点
├── pkg/gen/
│   ├── generate_struct_access.go         # 核心生成逻辑
│   └── generate_struct_access_backup.go  # 备份文件
├── go.mod                                # 模块定义
└── CLAUDE.md                            # 项目说明
```

## 技术实现

- **AST 解析**：使用 `go/parser` 和 `go/ast` 包解析 Go 源码
- **模板生成**：使用 `text/template` 生成代码
- **代码格式化**：集成 `goimports` 自动格式化
- **CLI 框架**：基于 `github.com/spf13/cobra` 构建

## 系统要求

- Go 1.24+
- `goimports` 工具（用于代码格式化）

## 许可证

Apache License 2.0

## TODO
并行化

---

**作者**: yangyuan  
**创建时间**: 2025年