package dirty_missing

// DirtyMissing 引用了不存在的 dirty 方法。
//
// gogen:dirty=NonExistentMethod
type DirtyMissing struct {
	Score int64
}

// FieldDirty 字段级引用不存在的 dirty 方法。
type FieldDirty struct {
	Score int64 `gogen:"dirty=AlsoMissing"`
}
