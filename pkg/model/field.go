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

import (
	"reflect"
	"strings"
)

// FieldConfig 保存从 struct tag `gogen:"..."` 解析出的生成控制配置
//
// 支持的 tag 格式（逗号分隔，可组合）：
//
//	`gogen:"-"`         跳过此字段，不生成任何方法
//	`gogen:"readonly"`  只生成 getter，不生成 setter 及写操作方法
//	`gogen:"writeonly"` 只生成 setter，不生成 getter
//	`gogen:"plain"`     简单模式：只生成核心 Get/Set，跳过 Add/Sub/Toggle/Has 等扩展方法
//	`gogen:"override"` 覆盖模式：忽略嵌入提升方法检查，强制生成该字段的访问器
type FieldConfig struct {
	Skip      bool `gogen:"plain"` // 跳过此字段
	Readonly  bool `gogen:"plain"` // 只读：只生成 getter
	WriteOnly bool `gogen:"plain"` // 只写：只生成 setter
	Plain     bool `gogen:"plain"` // 简单模式：跳过扩展方法，只保留核心访问器
	Override  bool `gogen:"plain"` // 覆盖模式：忽略嵌入提升方法检查，强制生成（仍受字段名/手写方法约束）
}

// ParseFieldConfig 从原始 struct tag 字符串解析 FieldConfig。
// rawTag 为完整的 tag 字符串，如 `json:"name" gogen:"readonly"`
func ParseFieldConfig(rawTag string) FieldConfig {
	val := reflect.StructTag(rawTag).Get("gogen")
	if val == "" {
		return FieldConfig{}
	}

	cfg := FieldConfig{}
	for p := range strings.SplitSeq(val, ",") {
		p = strings.TrimSpace(p)
		switch p {
		case "-":
			cfg.Skip = true
		case "readonly":
			cfg.Readonly = true
		case "writeonly":
			cfg.WriteOnly = true
		case "plain":
			cfg.Plain = true
		case "override":
			cfg.Override = true
		}
	}
	return cfg
}

// FieldDef 描述结构体中一个字段的完整信息
type FieldDef struct {
	Name    string      // 字段名，如 "Name"、"UserID"
	Type    *TypeInfo   // 字段类型的完整描述
	Config  FieldConfig // 从 struct tag 解析的生成配置
	Doc     string      // 字段上方的文档注释（field.Doc）
	Comment string      // 字段行尾注释（field.Comment）
}

// IsReadable 判断是否应生成读操作方法（getter/range 等）
func (f *FieldDef) IsReadable() bool {
	return !f.Config.Skip && !f.Config.WriteOnly
}

// IsWritable 判断是否应生成写操作方法（setter/add/del 等）
func (f *FieldDef) IsWritable() bool {
	return !f.Config.Skip && !f.Config.Readonly
}
