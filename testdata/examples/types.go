// 综合测试文件：覆盖所有支持的字段类型和边界场景
package examples

import (
	"time"
)

// --- 类型定义（用于测试具名类型/别名）---

// UserID 具名类型（新类型，不等价于 int64）
type UserID int64

// Status 具名字符串类型
type Status string

// Tags 具名切片类型
type Tags []string

// Metadata 具名 map 类型
type Metadata map[string]string

// MyTime 类型别名（完全等价于 time.Time）
type MyTime = time.Time

// --- 跨包嵌入的结构体（模拟被嵌入的基础结构）---

// BaseInfo 基础信息结构体，用于测试嵌入
type BaseInfo struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// --- 主测试结构体 ---

// AllTypes 覆盖所有支持的字段类型
type AllTypes struct {
	// --- 基础类型 ---
	FieldBool       bool
	FieldInt        int
	FieldInt8       int8
	FieldInt16      int16
	FieldInt32      int32
	FieldInt64      int64
	FieldUint       uint
	FieldUint8      uint8
	FieldUint16     uint16
	FieldUint32     uint32
	FieldUint64     uint64
	FieldFloat32    float32
	FieldFloat64    float64
	FieldComplex64  complex64
	FieldComplex128 complex128
	FieldString     string
	FieldByte       byte // byte = uint8
	FieldRune       rune // rune = int32

	// --- 指针类型 ---
	FieldPtrInt    *int
	FieldPtrString *string
	FieldPtrStruct *BaseInfo

	// --- 具名类型（新类型） ---
	FieldUserID UserID
	FieldStatus Status

	// --- 类型别名 ---
	FieldMyTime MyTime

	// --- 跨包具体类型 ---
	FieldTime     time.Time
	FieldDuration time.Duration

	// --- 切片类型 ---
	FieldSliceInt    []int
	FieldSliceString []string
	FieldSliceStruct []*BaseInfo
	FieldTags        Tags // 具名切片类型

	// --- 数组类型（固定长度）---
	FieldArray8   [8]int
	FieldArrayStr [4]string

	// --- Map 类型 ---
	FieldMapStrInt    map[string]int
	FieldMapStrStruct map[string]*BaseInfo
	FieldMetadata     Metadata // 具名 map 类型

	// --- 嵌套复合类型 ---
	FieldMapSlice map[string][]string // map value 是 slice
	FieldSliceMap []map[string]int    // slice elem 是 map

	// --- struct tag 控制 ---
	FieldSkip      string `gogen:"-"`                    // 应跳过，不生成任何方法
	FieldReadonly  string `gogen:"readonly"`             // 只生成 getter
	FieldWriteOnly string `gogen:"writeonly"`            // 只生成 setter
	FieldWithJSON  string `json:"name" gogen:"readonly"` // 多个 tag 共存

	// --- interface / func 类型：生成 Get/Set ---
	FieldInterface interface{}      // 生成 GetFieldInterface/SetFieldInterface
	FieldFunc      func(int) string // 生成 GetFieldFunc/SetFieldFunc
	FieldAny       any              // 生成 GetFieldAny/SetFieldAny（any = interface{}）

	// --- 不支持的类型（应跳过，不能生成错误代码）---
	FieldChan chan int // 跳过：chan 封装弊大于利，Send/Recv 有阻塞/方向性语义
}

// SliceOnly 只有切片字段，测试切片生成器的完整性
type SliceOnly struct {
	Names  []string
	Scores []float64
	Items  []*BaseInfo
}

// MapOnly 只有 map 字段，测试 map 生成器的完整性
type MapOnly struct {
	Index  map[int]string
	Config map[string]interface{} // value 是 interface，生成 key/value 类型要正确
	Nested map[string][]int       // value 是 slice
}

// ArrayOnly 测试各种数组长度写法（go/types 会将长度解析为整数）
type ArrayOnly struct {
	Fixed8   [8]byte
	Fixed16  [16]byte
	Fixed256 [256]int32
}

// TagControl 专门测试 struct tag 控制逻辑
type TagControl struct {
	ReadWrite int // 无 tag：同时生成 getter + setter
	ReadOnly  int `gogen:"readonly"`
	WriteOnly int `gogen:"writeonly"`
	Skip      int `gogen:"-"`
	// plain 模式测试：只生成核心 Get/Set，跳过扩展方法
	PlainBool  bool           `gogen:"plain"` // Get/Set，无 Toggle
	PlainInt   int            `gogen:"plain"` // Get/Set，无 Add/Sub
	PlainPtr   *BaseInfo      `gogen:"plain"` // Get/Set，无 Has
	PlainSlice []string       `gogen:"plain"` // At/Range/SetAt/Append/Delete，无 Len/Has/GetCopy
	PlainMap   map[string]int `gogen:"plain"` // Val/Range/SetVal/DeleteKey，无 Has 系列
}

// EmbedOther 测试嵌入其他结构体的字段（命名嵌入，非匿名）
type EmbedOther struct {
	Base    BaseInfo  // 命名字段，应生成 GetBase/SetBase
	BasePtr *BaseInfo // 指针，应生成 GetBasePtr/SetBasePtr
}

// PlainStruct 测试结构体级 plain 注释：所有字段应以 plain 模式生成。
//
// gogen:plain
type PlainStruct struct {
	ID    int64
	Score float64
	Name  string
	Flag  bool
}
