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

package analyzer_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/motocat46/gogen/pkg/analyzer"
	"github.com/motocat46/gogen/pkg/loader"
	"github.com/motocat46/gogen/pkg/model"
)

// testdataDir 返回 testdata/examples 目录的绝对路径。
func testdataDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取当前文件路径")
	}
	// thisFile = .../gogen/pkg/analyzer/analyzer_test.go
	// 向上两级 → .../gogen，再拼接 testdata/examples
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "examples")
}

// loadExamples 加载 testdata/examples 包，返回结构体名 → StructDef 的映射。
func loadExamples(t *testing.T) map[string]*model.StructDef {
	t.Helper()
	dir := testdataDir(t)

	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		t.Fatalf("加载 testdata/examples 失败: %v", err)
	}

	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	if err != nil {
		t.Fatalf("分析 testdata/examples 失败: %v", err)
	}

	m := make(map[string]*model.StructDef, len(structs))
	for _, s := range structs {
		m[s.Name] = s
	}
	return m
}

// TestPromotedMethods 验证 collectPromotedMethods 和 CanGenerateMethod 在各嵌入场景下的正确性。
func TestPromotedMethods(t *testing.T) {
	structs := loadExamples(t)

	cases := []struct {
		structName        string
		expectPromoted    []string // PromotedMethods 中应包含（生成应被阻止）
		expectNotPromoted []string // PromotedMethods 中不应包含（可正常生成）
	}{
		// ── 第一类：非冲突字段不受影响 ──────────────────────────────
		{
			structName:        "EmbedByValue",
			expectPromoted:    []string{"GetCount", "SetCount"},
			expectNotPromoted: []string{"GetName", "SetName"},
		},
		{
			structName:        "EmbedByPointer",
			expectPromoted:    []string{"GetCount", "SetCount"},
			expectNotPromoted: []string{"GetName", "SetName"},
		},
		{
			structName:        "EmbedDeep",
			expectPromoted:    []string{"GetCount", "SetCount"},
			expectNotPromoted: []string{"GetID", "SetID"},
		},
		{
			structName:     "MultipleEmbeds",
			expectPromoted: []string{"GetCount", "SetCount", "GetVal"},
		},
		// ── 第二类：冲突字段不生成，非冲突字段正常 ──────────────────────
		{
			structName:        "FieldSameAsPromoted",
			expectPromoted:    []string{"GetCount", "SetCount"},
			expectNotPromoted: []string{"GetOtherField", "SetOtherField"},
		},
		// ── 第三类：接口保护场景 ────────────────────────────────────────
		{
			structName:     "EmbedWithInterface",
			expectPromoted: []string{"GetSpeed", "SetSpeed"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.structName, func(t *testing.T) {
			sd, ok := structs[tc.structName]
			if !ok {
				t.Fatalf("未找到结构体 %s（已加载的结构体: %v）", tc.structName, structNames(structs))
			}

			// 验证提升方法集合包含预期项
			for _, method := range tc.expectPromoted {
				if !sd.PromotedMethods[method] {
					t.Errorf("%s.PromotedMethods 应包含 %q，实际内容: %v",
						tc.structName, method, sd.PromotedMethods)
				}
				// 生成应被阻止
				if sd.CanGenerateMethod(method) {
					t.Errorf("%s.CanGenerateMethod(%q) 应返回 false（方法已被提升），实际返回 true",
						tc.structName, method)
				}
			}

			// 验证非冲突方法不在提升集合中，且可正常生成
			for _, method := range tc.expectNotPromoted {
				if sd.PromotedMethods[method] {
					t.Errorf("%s.PromotedMethods 不应包含 %q（误报），实际内容: %v",
						tc.structName, method, sd.PromotedMethods)
				}
				// 非冲突方法应该可以生成（除非与字段名或手写方法冲突，此处测试用例无此情况）
				if !sd.CanGenerateMethod(method) {
					t.Errorf("%s.CanGenerateMethod(%q) 应返回 true，实际返回 false（ManualMethods=%v, FieldNames=%v）",
						tc.structName, method, sd.ManualMethods, sd.FieldNames)
				}
			}
		})
	}
}

// TestCanGenerateMethodTripleCheck 验证 CanGenerateMethod 的三层检查逻辑。
func TestCanGenerateMethodTripleCheck(t *testing.T) {
	// 构造一个手动 StructDef，无需加载包，直接测试三层逻辑
	sd := &model.StructDef{
		Name:            "TestStruct",
		FieldNames:      map[string]bool{"MyField": true},
		ManualMethods:   map[string]bool{"ManualGet": true},
		PromotedMethods: map[string]bool{"PromotedGet": true},
	}

	tests := []struct {
		method string
		want   bool
		reason string
	}{
		{"MyField", false, "方法名与字段名相同，不能生成"},
		{"ManualGet", false, "手写文件已有同名方法，不能生成"},
		{"PromotedGet", false, "嵌入提升方法同名，不能生成"},
		{"SafeMethod", true, "无任何冲突，可以生成"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := sd.CanGenerateMethod(tt.method)
			if got != tt.want {
				t.Errorf("CanGenerateMethod(%q) = %v, want %v（%s）",
					tt.method, got, tt.want, tt.reason)
			}
		})
	}
}

// structNames 返回映射中所有结构体名，用于诊断输出。
func structNames(m map[string]*model.StructDef) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	return names
}
