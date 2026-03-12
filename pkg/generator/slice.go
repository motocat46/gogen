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
{{ if .GetAt -}}
// Get{{ .MethodName }}At 获取切片 {{ .FieldName }} 中 index 位置的元素
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}At(index int) {{ .ElemType }} {
	return this.{{ .FieldName }}[index]
}
{{ end -}}
{{ if .GetLen -}}
// Get{{ .MethodName }}Len 获取切片 {{ .FieldName }} 的长度
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Len() int {
	return len(this.{{ .FieldName }})
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
{{ if .Has -}}
// Has{{ .MethodName }} 返回切片 {{ .FieldName }} 是否已初始化（非 nil）
func (this *{{ .ReceiverType }}) Has{{ .MethodName }}() bool {
	return this.{{ .FieldName }} != nil
}
{{ end -}}
{{ if .GetCopy -}}
// Get{{ .MethodName }}Copy 返回切片 {{ .FieldName }} 的浅拷贝
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Copy() {{ .SliceType }} {
	return slices.Clone(this.{{ .FieldName }})
}
{{ end -}}
{{ if .SetAt -}}
// Set{{ .MethodName }}At 设置切片 {{ .FieldName }} 中 index 位置的元素
func (this *{{ .ReceiverType }}) Set{{ .MethodName }}At(index int, elem {{ .ElemType }}) {
	this.{{ .FieldName }}[index] = elem
}
{{ end -}}
{{ if .Append -}}
// Append{{ .MethodName }} 向切片 {{ .FieldName }} 追加元素
func (this *{{ .ReceiverType }}) Append{{ .MethodName }}(elem {{ .ElemType }}) {
	this.{{ .FieldName }} = append(this.{{ .FieldName }}, elem)
}
{{ end -}}
{{ if .Delete -}}
// Delete{{ .MethodName }} 删除切片 {{ .FieldName }} 中 index 位置的元素，并清零释放的尾部槽位
// 注意：会改变被删除元素之后所有元素的下标
func (this *{{ .ReceiverType }}) Delete{{ .MethodName }}(index int) {
	this.{{ .FieldName }} = slices.Delete(this.{{ .FieldName }}, index, index+1)
}
{{ end }}`

var sliceTmpl = template.Must(template.New("slice").Parse(sliceTmplStr))

// SliceGenerator 为切片类型字段生成访问器方法。
type SliceGenerator struct{}

func (g *SliceGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	canGen := resolveCanGen(s, f)
	elemType := ""
	if f.Type.Elem != nil {
		elemType = f.Type.Elem.TypeStr
	}

	fn := f.Name
	r, w, plain := f.IsReadable(), f.IsWritable(), f.Config.Plain
	getAt := r && canGen("Get"+fn+"At")
	getLen := !plain && r && canGen("Get"+fn+"Len")
	rang := r && canGen("Range"+fn)
	has := !plain && r && canGen("Has"+fn)
	getCopy := !plain && r && canGen("Get"+fn+"Copy")
	setAt := w && canGen("Set"+fn+"At")
	appendFn := w && canGen("Append"+fn)
	deleteFn := w && canGen("Delete"+fn)
	var buf bytes.Buffer
	err := sliceTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"MethodName":   fn,
		"FieldName":    f.Name,
		"ElemType":     elemType,
		"SliceType":    f.Type.TypeStr,
		"Doc":          formatDoc(f.Doc),
		"GetAt":        getAt,
		"GetLen":       getLen,
		"Range":        rang,
		"Has":          has,
		"GetCopy":      getCopy,
		"SetAt":        setAt,
		"Append":       appendFn,
		"Delete":       deleteFn,
		"Any":          getAt || getLen || rang || has || getCopy || setAt || appendFn || deleteFn,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
