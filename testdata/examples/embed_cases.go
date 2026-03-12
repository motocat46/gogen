// 嵌入提升方法检测的测试用例。
// 覆盖三类场景：非冲突字段正常生成、冲突字段跳过、接口实现保护。
package examples

// ── 共用基础类型 ──────────────────────────────────────────────

// BaseWithMethods 带有手写方法的基础结构体，用于测试方法提升场景。
type BaseWithMethods struct{ Count int }

func (b *BaseWithMethods) GetCount() int  { return b.Count }
func (b *BaseWithMethods) SetCount(v int) { b.Count = v }

// ── 第一类：非冲突字段正常生成 ──────────────────────────────

// EmbedByValue 值嵌入（单层）。
// GetCount/SetCount 来自 BaseWithMethods 的提升，应被阻止生成。
// GetName/SetName 无冲突，应正常生成。
type EmbedByValue struct {
	Name string
	BaseWithMethods
}

// EmbedByPointer 指针嵌入（单层）。
// GetCount/SetCount 来自 *BaseWithMethods 的提升，应被阻止生成。
// GetName/SetName 无冲突，应正常生成。
type EmbedByPointer struct {
	Name string
	*BaseWithMethods
}

// MidLevel 中间层，嵌入 *BaseWithMethods，用于测试深层嵌入。
type MidLevel struct{ *BaseWithMethods }

// EmbedDeep 深层嵌入（多层：EmbedDeep → MidLevel → *BaseWithMethods）。
// GetCount/SetCount 应被阻止生成（来自深层提升）。
// GetID/SetID 无冲突，应正常生成。
type EmbedDeep struct {
	ID int
	MidLevel
}

// SecondBase 第二个嵌入源，带手写 GetVal 方法。
type SecondBase struct{ Val string }

func (s *SecondBase) GetVal() string { return s.Val }

// MultipleEmbeds 包含两个嵌入源。
// GetCount/SetCount 来自 *BaseWithMethods，GetVal 来自 *SecondBase，均应被阻止生成。
type MultipleEmbeds struct {
	*BaseWithMethods
	SecondBase
}

// ── 第二类：字段名与提升方法名冲突 ──────────────────────────

// FieldSameAsPromoted 字段名 Count 与提升方法 GetCount/SetCount 同名。
// GetCount/SetCount 应被阻止生成（已由提升方法提供）。
// GetOtherField/SetOtherField 无冲突，应正常生成。
type FieldSameAsPromoted struct {
	Count      int    // 想生成 GetCount/SetCount，但已被提升 → 跳过
	OtherField string // 无冲突，应正常生成 GetOtherField/SetOtherField
	*BaseWithMethods
}

// ── 第三类：接口实现保护（最严格场景）────────────────────────

// ISpeedProvider 接口，SetSpeed 为双参数签名，与 gogen 生成的单参数版本不同。
type ISpeedProvider interface {
	GetSpeed() float32
	SetSpeed(speed, acceleration float32) // 双参数，gogen 生成的是单参数，两者签名不同
}

// SpeedEntity 带有手写多参数 SetSpeed 的结构体。
type SpeedEntity struct{ Speed float32 }

func (e *SpeedEntity) GetSpeed() float32                    { return e.Speed }
func (e *SpeedEntity) SetSpeed(speed, acceleration float32) { e.Speed = speed }

// EmbedWithInterface 通过嵌入 *SpeedEntity 满足 ISpeedProvider 接口。
// Speed 字段与提升方法 GetSpeed/SetSpeed 同名 → 均应跳过生成。
// 修复前：gogen 生成单参数 SetSpeed，覆盖提升，破坏 ISpeedProvider 实现（编译失败）。
// 修复后：gogen 跳过 GetSpeed/SetSpeed，接口由提升方法满足（编译通过）。
type EmbedWithInterface struct {
	Speed float32  // 与提升方法同名：GetSpeed/SetSpeed 均应跳过
	*SpeedEntity   // 提升 GetSpeed()/SetSpeed(float32, float32)
}

// 编译时断言：EmbedWithInterface 满足 ISpeedProvider 接口。
// 若 gogen 错误生成了单参数 SetSpeed，此断言会在编译时报错。
var _ ISpeedProvider = (*EmbedWithInterface)(nil)

// ── 第四类：override 强制覆盖提升方法 ────────────────────────────

// OverrideEmbed 通过 gogen:"override" 强制覆盖提升方法。
// Count 字段的 GetCount/SetCount 本会因提升而跳过，
// 但 override tag 允许显式生成（覆盖提升语义）。
type OverrideEmbed struct {
	Count int `gogen:"override"`
	Name  string
	*BaseWithMethods
}
