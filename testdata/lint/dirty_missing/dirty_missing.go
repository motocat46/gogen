package dirty_missing

// DirtyMissing 引用了不存在的 dirty 方法。
//
// gogen:dirty=NonExistentMethod
type DirtyMissing struct {
	Score int64
}

