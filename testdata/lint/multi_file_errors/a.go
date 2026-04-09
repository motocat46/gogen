package multi_file_errors

// StructA1 至 StructA7：每个 struct 贡献一个 Error，
// 与 b.go 的 7 个 issue 合计 14 个，超过插入排序阈值（12），
// 触发 pdqsort 递归路径，覆盖 return -1 和 return 1 两个分支。

type StructA1 struct{ F1 int `gogen:"raedonly"` }
type StructA2 struct{ F2 int `gogen:"raedonly"` }
type StructA3 struct{ F3 int `gogen:"raedonly"` }
type StructA4 struct{ F4 int `gogen:"raedonly"` }
type StructA5 struct{ F5 int `gogen:"raedonly"` }
type StructA6 struct{ F6 int `gogen:"raedonly"` }
type StructA7 struct{ F7 int `gogen:"raedonly"` }
