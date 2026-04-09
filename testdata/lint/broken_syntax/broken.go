package broken_syntax

// 故意引入类型错误，用于覆盖 Lint 加载错误路径。
var _ = undefinedSymbol // undefined: undefinedSymbol
