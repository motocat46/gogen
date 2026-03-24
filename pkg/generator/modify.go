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
	"fmt"
	"os"
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
// {{ .ModifyMethod }} 在 fn 中修改结构体内容，fn 执行完毕后自动调用 {{ .DirtyMethod }}()，若 fn 发生 panic 则不调用。
// 适用于所有类型的字段变更，包括嵌入的自定义结构体和第三方类型。
func (this *{{ .ReceiverType }}) {{ .ModifyMethod }}(fn func(*{{ .ReceiverType }})) {
	fn(this)
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
func (g *ModifyGenerator) Generate(s *model.StructDef) ([]byte, error) {
	if s.ModifyMethod == "" {
		return nil, nil
	}
	// 使用 CanGenerateMethodOverride：允许覆盖从嵌入类型提升的同名方法。
	// 典型场景：DirtyBase.Modify(fn func(*DirtyBase)) 被 Child.Modify(fn func(*Child)) 覆盖，
	// 覆盖是正确行为（Child 的 Modify 签名更具体，更有用）。
	// 仍然检查手写方法冲突（ManualMethods），保护用户自定义的 Modify 实现。
	//
	// 当提升方法来自非 dirty-base 的第三方嵌入类型时，覆盖可能出乎意料，因此打印 Warning。
	if s.PromotedMethods[s.ModifyMethod] {
		fmt.Fprintf(os.Stderr, "%s.%s: [Warning] 生成的 %s() 覆盖了嵌入类型提升的同名方法；"+
			"若需保留原提升方法，请手写 %s() 或通过 gogen:modify=Xxx 改用其他名称\n",
			s.PackagePath, s.Name, s.ModifyMethod, s.ModifyMethod)
	}
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
