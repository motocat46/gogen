package contradictions

// Contradictions 包含互斥的 tag 组合。
type Contradictions struct {
	A string `gogen:"readonly,writeonly"` // 互斥 → Error
	B int    `gogen:"-,plain"`            // - 不能组合 → Error
	C string `gogen:"readonly,dirty"`     // readonly + dirty 无效 → Warning
}
