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

const sliceTmplStr = `
{{ if and .Any .Doc }}{{ .Doc }}
{{ end -}}
{{ if .GetElem -}}
// Get{{ .MethodName }}Elem 获取切片 {{ .FieldName }} 中 index 位置的元素
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Elem(index int) {{ .ElemType }} {
	return this.{{ .FieldName }}[index]
}
{{ end -}}
{{ if .GetLen -}}
// Get{{ .MethodName }}Len 获取切片 {{ .FieldName }} 的长度
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Len() int {
	return len(this.{{ .FieldName }})
}
{{ end -}}
{{ if .GetCap -}}
// Get{{ .MethodName }}Cap 获取切片 {{ .FieldName }} 的容量
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Cap() int {
	return cap(this.{{ .FieldName }})
}
{{ end -}}
{{ if .Range -}}
// Range{{ .MethodName }} 遍历切片 {{ .FieldName }}，fn 返回 false 时终止遍历
func (this *{{ .ReceiverType }}) Range{{ .MethodName }}(fn func(index int, value {{ .ElemType }}) bool) {
	for i, v := range this.{{ .FieldName }} {
		if !fn(i, v) {
			break
		}
	}
}
{{ end -}}
{{ if .SetElem -}}
// Set{{ .MethodName }}Elem 设置切片 {{ .FieldName }} 中 index 位置的元素
func (this *{{ .ReceiverType }}) Set{{ .MethodName }}Elem(index int, elem {{ .ElemType }}) {
	this.{{ .FieldName }}[index] = elem
}
{{ end -}}
{{ if .AddElem -}}
// Add{{ .MethodName }}Elem 向切片 {{ .FieldName }} 追加元素
func (this *{{ .ReceiverType }}) Add{{ .MethodName }}Elem(elem {{ .ElemType }}) {
	this.{{ .FieldName }} = append(this.{{ .FieldName }}, elem)
}
{{ end -}}
{{ if .DelElem -}}
// Del{{ .MethodName }}Elem 删除切片 {{ .FieldName }} 中 index 位置的元素
// 注意：会改变被删除元素之后所有元素的下标
func (this *{{ .ReceiverType }}) Del{{ .MethodName }}Elem(index int) {
	this.{{ .FieldName }} = append(this.{{ .FieldName }}[:index], this.{{ .FieldName }}[index+1:]...)
}
{{ end }}`

var sliceTmpl = template.Must(template.New("slice").Parse(sliceTmplStr))

// SliceGenerator 为切片类型字段生成访问器方法。
type SliceGenerator struct{}

func (g *SliceGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	elemType := ""
	if f.Type.Elem != nil {
		elemType = f.Type.Elem.TypeStr
	}

	fn := f.Name
	r, w := f.IsReadable(), f.IsWritable()
	getElem := r && s.CanGenerateMethod("Get"+fn+"Elem")
	getLen := r && s.CanGenerateMethod("Get"+fn+"Len")
	getCap := r && s.CanGenerateMethod("Get"+fn+"Cap")
	rang := r && s.CanGenerateMethod("Range"+fn)
	setElem := w && s.CanGenerateMethod("Set"+fn+"Elem")
	addElem := w && s.CanGenerateMethod("Add"+fn+"Elem")
	delElem := w && s.CanGenerateMethod("Del"+fn+"Elem")
	var buf bytes.Buffer
	err := sliceTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"MethodName":   fn,
		"FieldName":    f.Name,
		"ElemType":     elemType,
		"Doc":        formatDoc(f.Doc),
		"GetElem":    getElem,
		"GetLen":     getLen,
		"GetCap":     getCap,
		"Range":      rang,
		"SetElem":    setElem,
		"AddElem":    addElem,
		"DelElem":    delElem,
		"Any":        getElem || getLen || getCap || rang || setElem || addElem || delElem,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
