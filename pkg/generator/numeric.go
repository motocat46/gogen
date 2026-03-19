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

// numericTmplStr 适用于 int/float/uint/complex 等数值类型（含以数值为底层类型的具名类型）：
// 生成 Get/Set/Add/Sub 四个方法。
const numericTmplStr = `
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
{{ if .AddField -}}
// Add{{ .FieldName }} 将 {{ .FieldName }} 增加 delta
func (this *{{ .ReceiverType }}) Add{{ .FieldName }}(delta {{ .TypeStr }}) {
	this.{{ .FieldName }} += delta
{{- if .AddDirtyMethod }}
	this.{{ .AddDirtyMethod }}() // 需业务层实现此方法
{{- end }}
}
{{ end -}}
{{ if .SubField -}}
// Sub{{ .FieldName }} 将 {{ .FieldName }} 减少 delta
func (this *{{ .ReceiverType }}) Sub{{ .FieldName }}(delta {{ .TypeStr }}) {
	this.{{ .FieldName }} -= delta
{{- if .SubDirtyMethod }}
	this.{{ .SubDirtyMethod }}() // 需业务层实现此方法
{{- end }}
}
{{ end }}`

var numericTmpl = template.Must(template.New("numeric").Parse(numericTmplStr))

// NumericGenerator 为数值类型字段生成 Get/Set/Add/Sub 方法。
type NumericGenerator struct{}

func (g *NumericGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	canGen := resolveCanGen(s, f)
	fn := f.Name
	r, w, plain := f.IsReadable(), f.IsWritable(), f.Config.Plain
	getField := r && canGen("Get"+fn)
	setField := w && canGen("Set"+fn)
	addField := !plain && w && canGen("Add"+fn)
	subField := !plain && w && canGen("Sub"+fn)

	effectiveDM := model.EffectiveDirtyMethod(f, s)
	setDirtyMethod, addDirtyMethod, subDirtyMethod := "", "", ""
	setIdempotent := false
	if setField {
		setDirtyMethod = effectiveDM
		setIdempotent = setDirtyMethod != "" && f.Type.IsComparable
	}
	if addField {
		addDirtyMethod = effectiveDM
	}
	if subField {
		subDirtyMethod = effectiveDM
	}

	var buf bytes.Buffer
	err := numericTmpl.Execute(&buf, map[string]any{
		"ReceiverType":   s.ReceiverType(),
		"FieldName":      fn,
		"TypeStr":        f.Type.TypeStr,
		"Doc":            formatDoc(f.Doc),
		"GetField":       getField,
		"SetField":       setField,
		"AddField":       addField,
		"SubField":       subField,
		"Any":            getField || setField || addField || subField,
		"SetDirtyMethod": setDirtyMethod,
		"SetIdempotent":  setIdempotent,
		"AddDirtyMethod": addDirtyMethod,
		"SubDirtyMethod": subDirtyMethod,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
