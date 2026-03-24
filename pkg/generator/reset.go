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
// 创建日期: 2026/3/18

package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/motocat46/gogen/pkg/model"
)

// resetTmplStr 生成 Reset() 方法。
// 使用 ReceiverType（含泛型参数，如 Container[T]{}）作为零值字面量，
// 与 proto.Reset() 语义一致：slice/map 重置为 nil，释放底层内存。
const resetTmplStr = `
// Reset 将所有字段重置为零值。
// slice 和 map 字段重置为 nil，释放底层内存。
func (this *{{ .ReceiverType }}) Reset() {
	*this = {{ .ReceiverType }}{}
{{- if .DirtyMethod }}
	this.{{ .DirtyMethod }}()
{{- end }}
}
`

var resetTmpl = template.Must(template.New("reset").Parse(resetTmplStr))

// ResetGenerator 为结构体生成 Reset() 方法。
// 实现 StructGenerator 接口，在字段级方法之后追加至同一 *_access.go 文件。
type ResetGenerator struct{}

func (g *ResetGenerator) Name() string { return "reset" }

// Generate 生成 Reset() 方法代码。
// 使用 CanGenerateMethodOverride 而非 CanGenerateMethod：允许覆盖嵌入类型提升的 Reset()。
// 理由：提升的 Reset() 只清零嵌入部分的字段，不会清零外层结构体的其他字段，几乎必然是错的；
// 而生成的 *this = T{} 语义唯一且正确，无需 Warning——与 ModifyGenerator 不同，Reset 无歧义。
// 手写的 Reset() 受保护：检测到时打印 [Info] 提示原因，不静默跳过，让用户明确知晓未生成的原因。
func (g *ResetGenerator) Generate(s *model.StructDef, log func(string)) ([]byte, error) {
	if s.ManualMethods["Reset"] {
		log(fmt.Sprintf("%s.%s: [Info] 检测到手写 Reset()，跳过生成；手写实现可能包含自定义逻辑（非全字段清零），gogen 不覆盖",
			s.PackagePath, s.Name))
		return nil, nil
	}
	if !s.CanGenerateMethodOverride("Reset") {
		// 唯一剩余情况：字段名与 Reset 冲突（极罕见）
		return nil, nil
	}
	var buf bytes.Buffer
	dirtyMethod := s.DirtyMethod
	if s.NoDirty {
		dirtyMethod = ""
	}
	err := resetTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"DirtyMethod":  dirtyMethod, // 由 analyzer 在分析阶段填充；NoDirty=true 时强制为空
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
