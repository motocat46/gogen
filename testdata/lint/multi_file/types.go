package multi_file

// MultiFileStruct 的结构体定义和 dirty 方法分散在同一包的不同文件。
//
// gogen:dirty=MakeDirty
type MultiFileStruct struct {
	Score int64
	Name  string
}
