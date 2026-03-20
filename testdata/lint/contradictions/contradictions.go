package contradictions

// Contradictions 包含互斥的 tag 组合。
// MakeDirty 存在，确保字段 C 的 checkDirtyRef 通过，只暴露矛盾组合 Warning。
type Contradictions struct {
	A string `gogen:"readonly,writeonly"` // 互斥 → Error
	B int    `gogen:"-,plain"`            // - 不能组合 → Error
	C string `gogen:"readonly,dirty"`     // readonly + dirty 无效 → Warning（仅此一条）
}

func (c *Contradictions) MakeDirty() {}
