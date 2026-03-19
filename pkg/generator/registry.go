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

// Package generator 实现代码生成层。
//
// 设计说明：
//   - Registry 模式：每种 TypeKind 对应一个独立的 MethodGenerator 实现
//   - 新增类型支持只需实现 MethodGenerator 接口并注册，不修改已有代码
//   - 每个生成器有自己独立的模板，互不干扰
package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/motocat46/gogen/pkg/model"
)

// resolveCanGen 根据字段是否标记 override，返回对应的方法生成判断函数。
//   - 普通字段：使用 CanGenerateMethod（三层检查，含提升方法检查）
//   - override 字段：使用 CanGenerateMethodOverride（两层检查，跳过提升方法）
func resolveCanGen(s *model.StructDef, f *model.FieldDef) func(string) bool {
	if f.Config.Override {
		return s.CanGenerateMethodOverride
	}
	return s.CanGenerateMethod
}

// formatDoc 将多行文档文本转换为合法的 Go 注释块。
// ast.CommentGroup.Text() 会剥离每行的 "//" 前缀，返回纯文本（含内嵌换行）。
// 若直接用 "// {{ .Doc }}" 渲染，多行文本中第二行起缺少 "//"，会导致语法错误。
// 此函数为每一行重新添加 "//" 前缀，返回可安全插入源文件的注释字符串。
func formatDoc(doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(doc, "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = "//"
		} else {
			lines[i] = "// " + line
		}
	}
	return strings.Join(lines, "\n")
}

// MethodGenerator 是代码生成器的核心接口，每种 TypeKind 对应一个实现。
type MethodGenerator interface {
	// Generate 为给定结构体的给定字段生成访问器方法代码。
	// 返回生成的 Go 源码片段（不包含 package 声明）。
	Generate(s *model.StructDef, f *model.FieldDef) ([]byte, error)
}

// StructGenerator 是结构体级方法生成器接口，与 MethodGenerator 并列注册。
// 生成阶段在字段级方法之后执行，结果追加至同一 *_access.go 文件。
type StructGenerator interface {
	Name() string                                // 生成器标识，如 "reset"
	Generate(s *model.StructDef) ([]byte, error) // 返回类型与 MethodGenerator 一致
}

// Registry 管理 TypeKind 到 MethodGenerator 的映射。
type Registry struct {
	generators       map[model.TypeKind]MethodGenerator
	structGenerators []StructGenerator
}

// NewRegistry 创建并返回已注册所有内置生成器的 Registry。
func NewRegistry() *Registry {
	r := &Registry{generators: make(map[model.TypeKind]MethodGenerator)}
	r.Register(model.KindBasic, &BasicGenerator{})       // string、TypeParam 等：Get/Set
	r.Register(model.KindBool, &BoolGenerator{})         // bool：Get/Set/Toggle
	r.Register(model.KindNumeric, &NumericGenerator{})   // int/float/uint/complex：Get/Set/Add/Sub
	r.Register(model.KindPointer, &NilableGenerator{})   // *T：Get/Set/Has
	r.Register(model.KindInterface, &NilableGenerator{}) // interface{}：Get/Set/Has
	r.Register(model.KindFunc, &NilableGenerator{})      // func：Get/Set/Has
	r.Register(model.KindStruct, &BasicGenerator{})      // 结构体字段：Get/Set
	r.Register(model.KindGeneric, &BasicGenerator{})     // 泛型类型：Get/Set
	r.Register(model.KindSlice, &SliceGenerator{})
	r.Register(model.KindArray, &ArrayGenerator{})
	r.Register(model.KindMap, &MapGenerator{})
	// KindUnsupported 不注册，自动跳过
	r.RegisterStruct(&ResetGenerator{}) // Phase 1: Reset
	return r
}

// Register 注册一个生成器，若已存在则覆盖。
func (r *Registry) Register(kind model.TypeKind, g MethodGenerator) {
	r.generators[kind] = g
}

// RegisterStruct 注册一个结构体级生成器，按注册顺序执行。
func (r *Registry) RegisterStruct(g StructGenerator) {
	r.structGenerators = append(r.structGenerators, g)
}

// GenerateStruct 为一个结构体生成完整的访问器文件内容（包含文件头）。
//
// 空结果判断规则（Phase 1 更新）：
//   - 字段级方法体为空 且 结构体级方法体也为空 → 返回 nil，不生成文件
//   - 即使所有字段被跳过，只要有结构体级方法（如 Reset），仍生成文件
func (r *Registry) GenerateStruct(s *model.StructDef) ([]byte, error) {
	// 生成字段级方法体
	var body bytes.Buffer
	for _, field := range s.ActiveFields() {
		g, ok := r.generators[field.Type.Kind]
		if !ok {
			continue
		}
		code, err := g.Generate(s, field)
		if err != nil {
			return nil, fmt.Errorf("生成字段 %s.%s 失败: %w", s.Name, field.Name, err)
		}
		body.Write(code)
	}

	// 生成结构体级方法体（如 Reset）
	var structBody bytes.Buffer
	for _, sg := range r.structGenerators {
		code, err := sg.Generate(s)
		if err != nil {
			return nil, fmt.Errorf("生成结构体 %s 的 %s 方法失败: %w", s.Name, sg.Name(), err)
		}
		structBody.Write(code)
	}

	// 两者均为空 → 不生成文件
	if len(bytes.TrimSpace(body.Bytes())) == 0 && len(bytes.TrimSpace(structBody.Bytes())) == 0 {
		return nil, nil
	}

	// 拼装完整文件：头部 + 字段方法体 + 结构体方法体
	header, err := renderFileHeader(s)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write(header)
	buf.Write(body.Bytes())
	buf.Write(structBody.Bytes())
	return buf.Bytes(), nil
}

// fileHeader 生成文件的固定头部模板。
// 遵循 Go 官方约定（https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source）：
// 第一行必须是 "// Code generated ... DO NOT EDIT."，不含时间戳。
// 去掉时间戳使生成内容完全确定——相同输入在任何环境、任何时间输出相同字节，
// 从而支持可靠的增量跳过和干净的 git diff。
var fileHeaderTmpl = template.Must(template.New("header").Parse(`// Code generated by gogen; DO NOT EDIT.

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

package {{ .PackageName }}
`))

func renderFileHeader(s *model.StructDef) ([]byte, error) {
	var buf bytes.Buffer
	err := fileHeaderTmpl.Execute(&buf, map[string]string{
		"PackageName": s.PackageName,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
