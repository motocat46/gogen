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
	Gold  int64       `gogen:"dirty=MakeDirty"` // 字段级覆盖，方法存在
}
