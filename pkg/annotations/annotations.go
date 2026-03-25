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
// 创建日期: 2026/3/25

// Package annotations 提供统一的 gogen 结构体注解解析和方法集查询能力。
//
// 替代了原先散落在 pkg/analyzer 和 pkg/linter 两处的重复实现，
// 并修复了 linter 缺少 gogen:modify= 支持的 bug。
package annotations

import (
	"go/types"
	"strings"
)

// StructAnnotations 保存从结构体文档注释解析出的所有 gogen 注解。
type StructAnnotations struct {
	Plain        bool
	DirtyMethod  string // "" 表示不注入；"MakeDirty" 为默认；自定义名为指定值
	NoDirty      bool   // gogen:nodirty 显式禁用
	ModifyMethod string // Modify 方法名，默认由调用方决定（通常 "Modify"），gogen:modify=Xxx 可覆盖
}

// ParseStructAnnotations 统一解析结构体文档注释中的 gogen 注解，
// 支持 plain / dirty / nodirty / modify 四类注解。
//
// doc 已由 ast.CommentGroup.Text() 剥离 "//" 前缀，每行独立匹配，避免前缀误判。
// 解析规则：
//   - gogen:plain   → Plain = true
//   - gogen:nodirty → NoDirty = true
//   - gogen:dirty   → DirtyMethod = "MakeDirty"（若尚未被 gogen:dirty=XXX 覆盖）
//   - gogen:dirty=X → DirtyMethod = X（X 非空时生效，覆盖前面的 gogen:dirty）
//   - gogen:modify=X → ModifyMethod = X（X 非空时生效）
func ParseStructAnnotations(doc string) StructAnnotations {
	var ann StructAnnotations
	for line := range strings.SplitSeq(doc, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "gogen:plain":
			ann.Plain = true
		case line == "gogen:nodirty":
			ann.NoDirty = true
		case line == "gogen:dirty":
			// 仅当尚未被带值的 dirty= 覆盖时设置默认值
			if ann.DirtyMethod == "" {
				ann.DirtyMethod = "MakeDirty"
			}
		case strings.HasPrefix(line, "gogen:dirty="):
			if name, _ := strings.CutPrefix(line, "gogen:dirty="); name != "" {
				ann.DirtyMethod = name
			}
		case strings.HasPrefix(line, "gogen:modify="):
			if name, _ := strings.CutPrefix(line, "gogen:modify="); name != "" {
				ann.ModifyMethod = name
			}
		}
	}
	return ann
}

// MethodSetContains 检查 *named 类型的方法集是否包含名为 methodName 的零参无返回值方法。
//
// 用于 dirty 自动检测：方法集含 MakeDirty() 时自动注入；
// 也用于 linter 验证 gogen:dirty=XXX 指定的方法是否实际存在。
func MethodSetContains(named *types.Named, methodName string) bool {
	mset := types.NewMethodSet(types.NewPointer(named))
	sel := mset.Lookup(nil, methodName)
	if sel == nil {
		return false
	}
	sig, ok := sel.Type().(*types.Signature)
	if !ok {
		return false
	}
	return sig.Params().Len() == 0 && sig.Results().Len() == 0
}
