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

// StructDef 描述一个结构体的完整信息，是分析层输出、生成层输入的核心数据结构
//
// 设计说明：
//   - 与 AST 和 go/types 完全解耦，生成层只依赖此结构
//   - Dir 和 PackageName 用于决定生成文件的写入位置和 package 声明
type StructDef struct {
	Name            string          // 结构体名，如 "Cache"
	TypeParams      string          // 泛型类型参数（仅含参数名），如 "[K, V]"；非泛型时为空字符串
	PackageName     string          // 所在包名，如 "model"
	PackagePath     string          // 包的导入路径，如 "github.com/foo/bar/model"
	Dir             string          // 源文件所在目录的绝对路径，用于输出文件
	Fields          []*FieldDef     // 字段列表（已过滤掉 Skip=true 的字段由生成层处理）
	Doc             string          // 结构体文档注释
	ManualMethods   map[string]bool // 已在手写（非生成）文件中定义的方法名集合
	FieldNames      map[string]bool // 结构体所有字段名集合（含不导出字段）
	PromotedMethods map[string]bool // 通过嵌入字段可访问的方法名集合（提升方法）
	DirtyMethod     string          // 结构体级 dirty 方法名；"" 表示无标注或未检测到
	NoDirty         bool            `gogen:"plain"` // true = 显式禁用（gogen:nodirty），优先级最高，压过字段级 dirty tag
}

// ReceiverType 返回方法接收者中使用的类型名称：
//   - 非泛型结构体返回 "Cache"
//   - 泛型结构体返回 "Cache[K, V]"（仅类型参数名，不含约束）
func (s *StructDef) ReceiverType() string {
	if s.TypeParams == "" {
		return s.Name
	}
	return s.Name + s.TypeParams
}

// CanGenerateMethod 判断指定方法名是否可以安全生成（三层检查）：
//  1. 与字段名相同（Go 编译器禁止方法名与字段名同名）
//  2. 手写文件已有同名方法（避免重复声明编译错误）
//  3. 嵌入提升方法同名（生成会覆盖提升语义，可能破坏接口实现）
func (s *StructDef) CanGenerateMethod(name string) bool {
	return !s.FieldNames[name] && !s.ManualMethods[name] && !s.PromotedMethods[name]
}

// CanGenerateMethodOverride 判断指定方法名是否可以安全生成（override 模式，两层检查）：
//  1. 与字段名相同（Go 编译器禁止方法名与字段名同名）
//  2. 手写文件已有同名方法（避免重复声明编译错误）
//
// 与 CanGenerateMethod 的区别：跳过第三层「嵌入提升方法」检查，
// 允许用户通过 gogen:"override" 显式覆盖嵌入提升的方法。
func (s *StructDef) CanGenerateMethodOverride(name string) bool {
	return !s.FieldNames[name] && !s.ManualMethods[name]
}

// ActiveFields 返回未被跳过（Skip=false）的字段列表
func (s *StructDef) ActiveFields() []*FieldDef {
	var result []*FieldDef
	for _, f := range s.Fields {
		if !f.Config.Skip {
			result = append(result, f)
		}
	}
	return result
}
