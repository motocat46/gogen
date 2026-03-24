package bad_tags

// BadTags 包含拼写错误和无效的 gogen tag。
type BadTags struct {
	ID    int64  `gogen:"raedonly"`      // 拼写错误 → Error (suggest "readonly")
	Name  string `gogen:"unknownoption"` // 完全未知选项 → Error
	Score int    `gogen:"dirty="`        // 未知选项（字段级 dirty 已废弃）→ Error
}
