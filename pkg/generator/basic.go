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

// basicTmplStr 适用于基础类型、指针、结构体字段、泛型类型：
// 生成 Get/Set 两个方法。
const basicTmplStr = `
{{ if and .Any .Doc }}{{ .Doc }}
{{ end -}}
{{ if .Readable -}}
// Get{{ .FieldName }} 获取 {{ .FieldName }}
func (this *{{ .ReceiverType }}) Get{{ .FieldName }}() {{ .TypeStr }} {
	return this.{{ .FieldName }}
}
{{ end -}}
{{ if .Writable -}}
// Set{{ .FieldName }} 设置 {{ .FieldName }}
func (this *{{ .ReceiverType }}) Set{{ .FieldName }}({{ .FieldName }} {{ .TypeStr }}) {
	this.{{ .FieldName }} = {{ .FieldName }}
}
{{ end }}`

var basicTmpl = template.Must(template.New("basic").Parse(basicTmplStr))

// BasicGenerator 为基础类型、指针、结构体字段、泛型实例生成 Get/Set 方法。
type BasicGenerator struct{}

func (g *BasicGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	canGen := resolveCanGen(s, f)
	var buf bytes.Buffer
	readable := f.IsReadable() && canGen("Get"+f.Name)
	writable := f.IsWritable() && canGen("Set"+f.Name)
	err := basicTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"FieldName":    f.Name,
		"TypeStr":      f.Type.TypeStr,
		"Doc":          formatDoc(f.Doc),
		"Readable":     readable,
		"Writable":     writable,
		"Any":          readable || writable,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
