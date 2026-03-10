// 版权所有(Copyright)[yangyuan]
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// 作者:  yangyuan
// 创建日期: 2025/7/31

package model

// TypeKind 表示字段的类型种类，用于生成层选择对应的生成策略
type TypeKind int

const (
	KindBasic       TypeKind = iota // int, string, bool, float64 等基础类型
	KindPointer                     // *T 指针类型
	KindSlice                       // []T 切片类型
	KindArray                       // [N]T 数组类型
	KindMap                         // map[K]V 映射类型
	KindStruct                      // 具名结构体类型
	KindGeneric                     // List[T]、Result[K,V] 等泛型实例化类型
	KindUnsupported                 // interface/func/chan 等，跳过生成
)

// String 返回 TypeKind 的可读名称，方便调试
func (k TypeKind) String() string {
	switch k {
	case KindBasic:
		return "basic"
	case KindPointer:
		return "pointer"
	case KindSlice:
		return "slice"
	case KindArray:
		return "array"
	case KindMap:
		return "map"
	case KindStruct:
		return "struct"
	case KindGeneric:
		return "generic"
	case KindUnsupported:
		return "unsupported"
	default:
		return "unknown"
	}
}

// TypeInfo 描述一个字段的完整类型信息，与 go/types 解耦
//
// 设计说明：
//   - TypeStr 由 go/types.TypeString() 渲染，对所有合法 Go 类型都正确
//   - Elem/Key/Value 保留了类型的层次结构，供生成层递归使用
//   - IsAlias 区分 type X = T（别名）和 type X T（新类型），
//     别名可以展开底层类型，新类型必须保留原名以通过编译
type TypeInfo struct {
	Kind     TypeKind
	TypeStr  string      // 完整类型字符串，如 "[]string"、"map[string]int32"
	Elem     *TypeInfo   // slice/array/pointer 的元素类型
	Key      *TypeInfo   // map 的 key 类型
	Value    *TypeInfo   // map 的 value 类型
	ArrayLen string      // array 的长度，如 "8"
	TypeArgs []*TypeInfo // 泛型类型参数，如 List[int] 中的 int
	IsAlias  bool        // 是否为类型别名（type X = T）
}
