// 泛型结构体的测试用例。
// 覆盖：单类型参数、多类型参数、混合普通字段的泛型结构体。
package examples

// ── 单类型参数 ─────────────────────────────────────────────────

// Container 单类型参数泛型容器
type Container[T any] struct {
	// Item 存储的元素
	Item T
	// Size 当前大小
	Size int
}

// ── 多类型参数 ─────────────────────────────────────────────────

// Pair 两个不同类型值的泛型对
type Pair[K comparable, V any] struct {
	// Key 键
	Key K
	// Value 值
	Value V
}

// ── 泛型实例作为字段（覆盖 buildTypeInfo KindGeneric 分支）─────

// Holder 持有一个已实例化的泛型容器，用于覆盖 buildTypeInfo 中的
// *types.Named{TypeArgs().Len() > 0} → KindGeneric 路径。
type Holder struct {
	// Items 存储整数的泛型容器
	Items Container[int]
}
