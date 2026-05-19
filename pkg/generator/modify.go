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
// 创建日期: 2026/3/23

package generator

import (
	"bytes"
	"text/template"

	"github.com/motocat46/gogen/pkg/model"
)

// modifyTmplStr 生成 Modify() 方法。
//
// 设计说明：
//   - Modify() 是 dirty tracking 的唯一入口，适用于所有字段类型（含自定义结构体、第三方类型）
//   - 调用方在 fn 中自由修改结构体的任何字段或嵌入对象，Modify 在 fn 返回后统一调用 dirty 方法
//   - 方法名默认 "Modify"，可通过结构体文档注释 gogen:modify=Xxx 自定义
const modifyTmplStr = `
// {{ .ModifyMethod }} 在 fn 中修改结构体内容，fn 执行完毕后自动调用 {{ .DirtyMethod }}()。
// 适用于所有类型的字段变更，包括嵌入的自定义结构体和第三方类型。
//
// 示例（obj 为该结构体的变量）：
//
//	obj.{{ .ModifyMethod }}(func() {
//	    obj.SetXxx(newValue)
//	})
//
// 注意：
//   - fn 是无参闭包，编译器不验证闭包内操作的目标；作用域内有多个对象时，
//     确保闭包引用的是调用 {{ .ModifyMethod }} 的那个变量，而非其他同类对象。
//   - fn 发生 panic 时不调用 {{ .DirtyMethod }}()。
//   - 嵌套调用 {{ .ModifyMethod }} 会触发多次 {{ .DirtyMethod }}()，仅在后者幂等时无害。
func (this *{{ .ReceiverType }}) {{ .ModifyMethod }}(fn func()) {
	fn()
	this.{{ .DirtyMethod }}()
}
`

var modifyTmpl = template.Must(template.New("modify").Parse(modifyTmplStr))

// ModifyGenerator 为有 dirty 配置的结构体生成 Modify() 方法。
// 实现 StructGenerator 接口，在字段级方法之后追加至同一 *_access.go 文件。
type ModifyGenerator struct{}

func (g *ModifyGenerator) Name() string { return "modify" }

// Generate 生成 Modify() 方法代码。
// 条件：s.ModifyMethod 非空（由 analyzer 在分析阶段填充，dirty 未启用时为空）。
// 若方法名已存在于手写文件或与字段名冲突，静默跳过。
// 若覆盖了嵌入类型提升的同名方法，打印 Warning 提示用户确认行为是否符合预期。
func (g *ModifyGenerator) Generate(s *model.StructDef, log func(string)) ([]byte, error) {
	if s.ModifyMethod == "" {
		return nil, nil
	}
	// 使用 CanGenerateMethodOverride：允许覆盖从嵌入类型提升的同名方法。
	// 典型场景：DirtyBase.Modify(fn func()) 被 Child.Modify(fn func()) 覆盖，
	// 覆盖是正确行为（Child 的 Modify 绑定了正确的 dirty 方法，语义更具体）。
	// 仍然检查手写方法冲突（ManualMethods），保护用户自定义的 Modify 实现。
	// 注：不打印 Warning——Modify() 语义固定（fn + dirty），覆盖提升方法与 Reset() 一样是必然正确的行为。
	if !s.CanGenerateMethodOverride(s.ModifyMethod) {
		return nil, nil
	}
	var buf bytes.Buffer
	err := modifyTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"ModifyMethod": s.ModifyMethod,
		"DirtyMethod":  s.DirtyMethod,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
