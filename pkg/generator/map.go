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

const mapTmplStr = `
{{ if and .Any .Doc }}{{ .Doc }}
{{ end -}}
{{ if .GetVal -}}
// Get{{ .MethodName }}Val 获取 {{ .FieldName }} 中指定 key 的值，ok 表示 key 是否存在
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Val(key {{ .KeyType }}) ({{ .ValueType }}, bool) {
	val, ok := this.{{ .FieldName }}[key]
	return val, ok
}
{{ end -}}
{{ if .Range -}}
// Range{{ .MethodName }} 遍历 {{ .FieldName }}，fn 返回 false 时终止遍历
func (this *{{ .ReceiverType }}) Range{{ .MethodName }}(fn func(key {{ .KeyType }}, value {{ .ValueType }}) bool) {
	for k, v := range this.{{ .FieldName }} {
		if !fn(k, v) {
			break
		}
	}
}
{{ end -}}
{{ if .SetKV -}}
// Set{{ .MethodName }}KV 设置 {{ .FieldName }} 中指定 key 的值
func (this *{{ .ReceiverType }}) Set{{ .MethodName }}KV(key {{ .KeyType }}, value {{ .ValueType }}) {
	this.{{ .FieldName }}[key] = value
}
{{ end -}}
{{ if .DelKV -}}
// Del{{ .MethodName }}KV 删除 {{ .FieldName }} 中指定 key
func (this *{{ .ReceiverType }}) Del{{ .MethodName }}KV(key {{ .KeyType }}) {
	delete(this.{{ .FieldName }}, key)
}
{{ end }}`

var mapTmpl = template.Must(template.New("map").Parse(mapTmplStr))

// MapGenerator 为 map 类型字段生成访问器方法。
type MapGenerator struct{}

func (g *MapGenerator) Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error) {
	keyType, valueType := "", ""
	if f.Type.Key != nil {
		keyType = f.Type.Key.TypeStr
	}
	if f.Type.Value != nil {
		valueType = f.Type.Value.TypeStr
	}

	fn := f.Name
	r, w := f.IsReadable(), f.IsWritable()
	getVal := r && s.CanGenerateMethod("Get"+fn+"Val")
	rang := r && s.CanGenerateMethod("Range"+fn)
	setKV := w && s.CanGenerateMethod("Set"+fn+"KV")
	delKV := w && s.CanGenerateMethod("Del"+fn+"KV")
	var buf bytes.Buffer
	err := mapTmpl.Execute(&buf, map[string]any{
		"ReceiverType": s.ReceiverType(),
		"MethodName":   fn,
		"FieldName":    f.Name,
		"KeyType":    keyType,
		"ValueType":  valueType,
		"Doc":        formatDoc(f.Doc),
		"GetVal":     getVal,
		"Range":      rang,
		"SetKV":      setKV,
		"DelKV":      delKV,
		"Any":        getVal || rang || setKV || delKV,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
