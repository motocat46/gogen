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

package generator

import (
	"bytes"
	"text/template"

	"github.com/motocat46/gogen/pkg/model"
)

// boolTmplStr 适用于 bool 类型（含以 bool 为底层类型的具名类型）：
// 生成 Get/Set/Toggle 三个方法。
const boolTmplStr = `
{{ if and .Any .Doc }}{{ .Doc }}
{{ end -}}
{{ if .GetField -}}
// Get{{ .FieldName }} 获取 {{ .FieldName }}
func (this *{{ .ReceiverType }}) Get{{ .FieldName }}() {{ .TypeStr }} {
	return this.{{ .FieldName }}
}
{{ end -}}
{{ if .SetField -}}
// Set{{ .FieldName }} 设置 {{ .FieldName }}
func (this *{{ .ReceiverType }}) Set{{ .FieldName }}({{ .FieldName }} {{ .TypeStr }}) {
{{- if .SetIdempotent }}
	if this.{{ .FieldName }} == {{ .FieldName }} {
		return
	}
{{- end }}
	this.{{ .FieldName }} = {{ .FieldName }}
{{- if .SetDirtyMethod }}
	this.{{ .SetDirtyMethod }}() // 需业务层实现此方法
{{- end }}
}
{{ end -}}
{{ if .Toggle -}}
// Toggle{{ .FieldName }} 翻转 {{ .FieldName }} 的布尔值
func (this *{{ .ReceiverType }}) Toggle{{ .FieldName }}() {
	this.{{ .FieldName }} = !this.{{ .FieldName }}
{{- if .ToggleDirtyMethod }}
	this.{{ .ToggleDirtyMethod }}() // 需业务层实现此方法
{{- end }}
}
{{ end }}`

var boolTmpl = template.Must(template.New("bool").Parse(boolTmplStr))

// BoolGenerator 为 bool 类型字段生成 Get/Set/Toggle 方法。
type BoolGenerator struct{}

func (g *BoolGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	canGen := resolveCanGen(s, f)
	fn := f.Name
	r, w := f.IsReadable(), f.IsWritable()
	getField := r && canGen("Get"+fn)
	setField := w && canGen("Set"+fn)
	toggle := !f.Config.Plain && w && canGen("Toggle"+fn)

	// dirty 注入：计算每个写方法的有效 dirty 方法名
	effectiveDM := model.EffectiveDirtyMethod(f, s)
	setDirtyMethod := ""
	setIdempotent := false
	if setField {
		setDirtyMethod = effectiveDM
		setIdempotent = setDirtyMethod != "" && f.Type.IsComparable
	}
	toggleDirtyMethod := ""
	if toggle {
		toggleDirtyMethod = effectiveDM
	}

	var buf bytes.Buffer
	err := boolTmpl.Execute(&buf, map[string]any{
		"ReceiverType":      s.ReceiverType(),
		"FieldName":         fn,
		"TypeStr":           f.Type.TypeStr,
		"Doc":               formatDoc(f.Doc),
		"GetField":          getField,
		"SetField":          setField,
		"Toggle":            toggle,
		"Any":               getField || setField || toggle,
		"SetDirtyMethod":    setDirtyMethod,
		"SetIdempotent":     setIdempotent,
		"ToggleDirtyMethod": toggleDirtyMethod,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
