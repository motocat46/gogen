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
	this.{{ .DirtyMethod }}() // 需业务层实现此方法
{{- end }}
}
`

var resetTmpl = template.Must(template.New("reset").Parse(resetTmplStr))

// ResetGenerator 为结构体生成 Reset() 方法。
// 实现 StructGenerator 接口，在字段级方法之后追加至同一 *_access.go 文件。
type ResetGenerator struct{}

func (g *ResetGenerator) Name() string { return "reset" }

// Generate 生成 Reset() 方法代码。
// 若 s.CanGenerateMethod("Reset") 返回 false（手写文件已有 Reset，或提升方法冲突），
// 静默跳过，与现有字段生成器行为一致。
func (g *ResetGenerator) Generate(s *model.StructDef) ([]byte, error) {
	if !s.CanGenerateMethod("Reset") {
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
