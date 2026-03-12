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

// arrayTmplStr 数组类型不支持 append/delete，只提供元素访问和遍历。
// GetField 返回数组值拷贝（数组是值类型，赋值即拷贝，无需 Copy 后缀）。
const arrayTmplStr = `
{{ if and .Any .Doc }}{{ .Doc }}
{{ end -}}
{{ if .GetField -}}
// Get{{ .MethodName }} 获取数组 {{ .FieldName }} 的完整副本
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}() {{ .ArrayType }} {
	return this.{{ .FieldName }}
}
{{ end -}}
{{ if .GetAt -}}
// Get{{ .MethodName }}At 获取数组 {{ .FieldName }} 中 index 位置的元素
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}At(index int) {{ .ElemType }} {
	return this.{{ .FieldName }}[index]
}
{{ end -}}
{{ if .GetLen -}}
// Get{{ .MethodName }}Len 获取数组 {{ .FieldName }} 的长度
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Len() int {
	return len(this.{{ .FieldName }})
}
{{ end -}}
{{ if .Range -}}
// Range{{ .MethodName }} 遍历数组 {{ .FieldName }}，fn 返回 false 时终止遍历
func (this *{{ .ReceiverType }}) Range{{ .MethodName }}(fn func(index int, value {{ .ElemType }}) bool) {
	for i, v := range this.{{ .FieldName }} {
		if !fn(i, v) {
			break
		}
	}
}
{{ end -}}
{{ if .SetAt -}}
// Set{{ .MethodName }}At 设置数组 {{ .FieldName }} 中 index 位置的元素
func (this *{{ .ReceiverType }}) Set{{ .MethodName }}At(index int, elem {{ .ElemType }}) {
	this.{{ .FieldName }}[index] = elem
}
{{ end }}`

var arrayTmpl = template.Must(template.New("array").Parse(arrayTmplStr))

// ArrayGenerator 为数组类型字段生成访问器方法。
// 数组长度固定，不支持 Append/Remove 操作。
type ArrayGenerator struct{}

func (g *ArrayGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	canGen := resolveCanGen(s, f)
	elemType := ""
	if f.Type.Elem != nil {
		elemType = f.Type.Elem.TypeStr
	}

	fn := f.Name
	r, w, plain := f.IsReadable(), f.IsWritable(), f.Config.Plain
	getField := r && canGen("Get"+fn)
	getAt := r && canGen("Get"+fn+"At")
	getLen := !plain && r && canGen("Get"+fn+"Len")
	rang := r && canGen("Range"+fn)
	setAt := w && canGen("Set"+fn+"At")
	var buf bytes.Buffer
	err := arrayTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"MethodName":   fn,
		"FieldName":    f.Name,
		"ElemType":     elemType,
		"ArrayType":    f.Type.TypeStr,
		"Doc":          formatDoc(f.Doc),
		"GetField":     getField,
		"GetAt":        getAt,
		"GetLen":       getLen,
		"Range":        rang,
		"SetAt":        setAt,
		"Any":          getField || getAt || getLen || rang || setAt,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
