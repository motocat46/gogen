package multi_file

// MakeDirty 标记结构体脏状态（定义在另一个文件中，验证跨文件 dirty 方法检测）。
func (m *MultiFileStruct) MakeDirty() {}
