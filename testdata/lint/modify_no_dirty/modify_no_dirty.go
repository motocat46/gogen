package modify_no_dirty

// NoEffect：指定了 gogen:modify=Apply 但没有启用 dirty tracking，Modify 不会生成。
//
// gogen:modify=Apply
type NoEffect struct {
	Score int64
}

// NoDirtyWithModify：nodirty 显式禁用，modify= 同样无效。
//
// gogen:nodirty
// gogen:modify=Update
type NoDirtyWithModify struct {
	Score int64
}
