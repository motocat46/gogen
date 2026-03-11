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

// nilableTmplStr 适用于指针、接口、func 类型：生成 Get/Set/Has 三个方法。
// Has 语义：字段是否已初始化（!= nil）。
const nilableTmplStr = `
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
	this.{{ .FieldName }} = {{ .FieldName }}
}
{{ end -}}
{{ if .HasField -}}
// Has{{ .FieldName }} 返回 {{ .FieldName }} 是否已初始化（非 nil）
func (this *{{ .ReceiverType }}) Has{{ .FieldName }}() bool {
	return this.{{ .FieldName }} != nil
}
{{ end }}`

var nilableTmpl = template.Must(template.New("nilable").Parse(nilableTmplStr))

// NilableGenerator 为指针、接口、func 类型字段生成 Get/Set/Has 方法。
type NilableGenerator struct{}

func (g *NilableGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	fn := f.Name
	r, w := f.IsReadable(), f.IsWritable()
	getField := r && s.CanGenerateMethod("Get"+fn)
	setField := w && s.CanGenerateMethod("Set"+fn)
	hasField := r && s.CanGenerateMethod("Has"+fn)
	var buf bytes.Buffer
	err := nilableTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"FieldName":    fn,
		"TypeStr":      f.Type.TypeStr,
		"Doc":          formatDoc(f.Doc),
		"GetField":     getField,
		"SetField":     setField,
		"HasField":     hasField,
		"Any":          getField || setField || hasField,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
