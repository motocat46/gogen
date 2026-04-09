package multi_file_errors

// StructB1 至 StructB7：与 a.go 合计 14 个 issue，超过插入排序阈值（12），
// 触发 pdqsort 递归路径，确保 return -1 和 return 1 两个分支都被覆盖。

type StructB1 struct{ F1 int `gogen:"unknownoption"` }
type StructB2 struct{ F2 int `gogen:"unknownoption"` }
type StructB3 struct{ F3 int `gogen:"unknownoption"` }
type StructB4 struct{ F4 int `gogen:"unknownoption"` }
type StructB5 struct{ F5 int `gogen:"unknownoption"` }
type StructB6 struct{ F6 int `gogen:"unknownoption"` }
type StructB7 struct{ F7 int `gogen:"unknownoption"` }
