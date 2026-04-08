package valid

// DirtyBase 提供 MakeDirty 方法。
type DirtyBase struct{}

func (d *DirtyBase) MakeDirty() {}

// Valid 合法的 gogen 注解，lint 应无任何问题。
// MakeDirty 通过嵌入 DirtyBase 提升，types.NewMethodSet(*Valid) 能正确查到。
//
// gogen:dirty
type Valid struct {
	DirtyBase         // MakeDirty() 通过此嵌入提升到 *Valid
	ID    int64       `gogen:"-"`
	Name  string      `gogen:"readonly"`
	Score int64       `gogen:"writeonly"`
	Tags  []string    `gogen:"plain,override"`
}

// ValidModify：gogen:modify= 与 dirty tracking 同时生效，合法。
//
// gogen:dirty=MarkChanged
// gogen:modify=Apply
type ValidModify struct {
	DirtyBase
	Name string
}

func (v *ValidModify) MarkChanged() {}

// 以下非 struct 类型声明用于覆盖 lintPackage 中跳过非 struct 的分支。

// Status 是枚举类型，不是 struct，lintPackage 应跳过。
type Status int

const (
	StatusActive   Status = 1
	StatusInactive Status = 2
)

// Tags 是切片类型别名，lintPackage 应跳过。
type Tags []string
