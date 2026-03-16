# pkg/config — 设计文档

## 设计说明

### 优先级模型

```
CLI 参数（最高）
    ↓ 空值时从配置文件读取
配置文件（.gogen.yaml）
    ↓ 配置文件无此项时
内置默认值（最低）
```

合并逻辑在 `main.go` 中完成，config 包只负责解析，不做合并。

**例外**：`Excludes` 是追加而非覆盖——CLI 的 `--exclude` 和配置文件的 `excludes` 同时生效，最终合并到一个列表。

### 为什么文件不存在不报错

配置文件是可选的，对于简单项目（不需要任何自定义配置）完全不需要创建 `.gogen.yaml`。如果文件不存在就报错，会给不需要配置文件的用户带来不必要的摩擦。

```go
if errors.Is(err, os.ErrNotExist) {
    return File{}, nil // 正常情况，不报错
}
```

### 配置文件放置位置

约定放在 gogen 运行目录（通常是 `go.mod` 所在的项目根目录）。CLI 的 `--config` 参数未实现，当前固定为 `dir + "/.gogen.yaml"`，与主流 Go 工具（`.golangci.yml`、`.goreleaser.yaml` 等）的约定一致。
