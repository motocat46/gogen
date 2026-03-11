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
{{ if .GetValOrDefault -}}
// Get{{ .MethodName }}ValOrDefault 获取 {{ .FieldName }} 中指定 key 的值，key 不存在时返回 def
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}ValOrDefault(key {{ .KeyType }}, def {{ .ValueType }}) {{ .ValueType }} {
	if val, ok := this.{{ .FieldName }}[key]; ok {
		return val
	}
	return def
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
{{ if .Has -}}
// Has{{ .MethodName }} 返回 {{ .FieldName }} 是否已初始化（非 nil）
func (this *{{ .ReceiverType }}) Has{{ .MethodName }}() bool {
	return this.{{ .FieldName }} != nil
}
{{ end -}}
{{ if .HasKey -}}
// Has{{ .MethodName }}Key 检查 {{ .FieldName }} 中指定 key 是否存在
func (this *{{ .ReceiverType }}) Has{{ .MethodName }}Key(key {{ .KeyType }}) bool {
	_, ok := this.{{ .FieldName }}[key]
	return ok
}
{{ end -}}
{{ if .GetLen -}}
// Get{{ .MethodName }}Len 获取 {{ .FieldName }} 的元素数量
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Len() int {
	return len(this.{{ .FieldName }})
}
{{ end -}}
{{ if .GetKeys -}}
// Get{{ .MethodName }}Keys 返回 {{ .FieldName }} 中所有 key 的切片（顺序不确定）
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Keys() []{{ .KeyType }} {
	keys := make([]{{ .KeyType }}, 0, len(this.{{ .FieldName }}))
	for k := range this.{{ .FieldName }} {
		keys = append(keys, k)
	}
	return keys
}
{{ end -}}
{{ if .GetCopy -}}
// Get{{ .MethodName }}Copy 返回 {{ .FieldName }} 的浅拷贝
func (this *{{ .ReceiverType }}) Get{{ .MethodName }}Copy() {{ .MapType }} {
	return maps.Clone(this.{{ .FieldName }})
}
{{ end -}}
{{ if .Ensure -}}
// Ensure{{ .MethodName }} 确保 {{ .FieldName }} 已初始化（nil 时自动创建空 map），返回字段引用
func (this *{{ .ReceiverType }}) Ensure{{ .MethodName }}() {{ .MapType }} {
	if this.{{ .FieldName }} == nil {
		this.{{ .FieldName }} = make({{ .MapType }})
	}
	return this.{{ .FieldName }}
}
{{ end -}}
{{ if .SetVal -}}
// Set{{ .MethodName }}Val 设置 {{ .FieldName }} 中指定 key 的值
func (this *{{ .ReceiverType }}) Set{{ .MethodName }}Val(key {{ .KeyType }}, value {{ .ValueType }}) {
	this.{{ .FieldName }}[key] = value
}
{{ end -}}
{{ if .DelKey -}}
// Del{{ .MethodName }}Key 删除 {{ .FieldName }} 中指定 key
func (this *{{ .ReceiverType }}) Del{{ .MethodName }}Key(key {{ .KeyType }}) {
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
	r, w, plain := f.IsReadable(), f.IsWritable(), f.Config.Plain
	getVal := r && s.CanGenerateMethod("Get"+fn+"Val")
	getValOrDefault := !plain && r && s.CanGenerateMethod("Get"+fn+"ValOrDefault")
	rang := r && s.CanGenerateMethod("Range"+fn)
	has := !plain && r && s.CanGenerateMethod("Has"+fn)
	hasKey := !plain && r && s.CanGenerateMethod("Has"+fn+"Key")
	getLen := !plain && r && s.CanGenerateMethod("Get"+fn+"Len")
	getKeys := !plain && r && s.CanGenerateMethod("Get"+fn+"Keys")
	getCopy := !plain && r && s.CanGenerateMethod("Get"+fn+"Copy")
	ensure := w && s.CanGenerateMethod("Ensure"+fn)
	setVal := w && s.CanGenerateMethod("Set"+fn+"Val")
	delKey := w && s.CanGenerateMethod("Del"+fn+"Key")
	var buf bytes.Buffer
	err := mapTmpl.Execute(&buf, map[string]any{
		"ReceiverType":    s.ReceiverType(),
		"MethodName":      fn,
		"FieldName":       f.Name,
		"KeyType":         keyType,
		"ValueType":       valueType,
		"MapType":         f.Type.TypeStr,
		"Doc":             formatDoc(f.Doc),
		"GetVal":          getVal,
		"GetValOrDefault": getValOrDefault,
		"Range":           rang,
		"Has":             has,
		"HasKey":          hasKey,
		"GetLen":          getLen,
		"GetKeys":         getKeys,
		"GetCopy":         getCopy,
		"Ensure":          ensure,
		"SetVal":          setVal,
		"DelKey":          delKey,
		"Any":             getVal || getValOrDefault || rang || has || hasKey || getLen || getKeys || getCopy || ensure || setVal || delKey,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
