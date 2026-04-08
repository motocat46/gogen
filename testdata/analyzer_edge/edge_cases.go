// Package analyzer_edge 提供 analyzer 边界场景测试用例。
// 独立于 testdata/examples，避免影响 golden file 测试。
package analyzer_edge

// WithUnexported 混合了导出和非导出字段。
// analyzer 应只收集导出字段（Name、Score），跳过非导出字段（id、secret）。
type WithUnexported struct {
	id     int64  // 非导出，应被跳过
	Name   string // 导出，应被收集
	secret string // 非导出，应被跳过
	Score  int    // 导出，应被收集
}

// UnknownTagOption 字段带有未知 gogen tag 选项。
// analyzer 应发出警告（写到 stderr），但仍正常收集该字段。
type UnknownTagOption struct {
	Name  string `gogen:"typo_option"` // 未知选项，触发 warning 路径
	Value int    `gogen:"dirty="`      // 空值 dirty=，触发另一条 warning 路径
}
