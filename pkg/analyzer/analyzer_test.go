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
		// ── 第四类：override 覆盖提升方法 ──────────────────────────────
		{
			structName:        "OverrideEmbed",
			expectPromoted:    []string{"GetCount", "SetCount"}, // 提升集合仍包含（override 是生成层行为）
			expectNotPromoted: []string{"GetName", "SetName"},
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

// TestAnalyze_ExcludePaths 验证 ExcludePaths 排除后，对应包中的结构体不出现在结果中。
func TestAnalyze_ExcludePaths(t *testing.T) {
	dir := testdataDir(t)
	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		t.Fatalf("加载包失败: %v", err)
	}

	// 使用 testdata/examples 目录本身作为排除路径
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{
		ExcludePaths: []string{dir},
	})
	if err != nil {
		t.Fatalf("Analyze 失败: %v", err)
	}
	if len(structs) != 0 {
		names := make([]string, len(structs))
		for i, s := range structs {
			names[i] = s.Name
		}
		t.Errorf("排除目录后期望 0 个结构体，实际得到 %d 个: %v", len(structs), names)
	}
}

// TestAnalyze_FileFilter 验证 FileFilter 仅分析指定文件，其他文件中的结构体不出现。
func TestAnalyze_FileFilter(t *testing.T) {
	dir := testdataDir(t)

	// 仅分析 embed_cases.go——它只定义嵌入相关的结构体
	targetFile := filepath.Join(dir, "embed_cases.go")

	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		t.Fatalf("加载包失败: %v", err)
	}

	structs, err := analyzer.Analyze(pkgs, analyzer.Config{
		FileFilter: []string{targetFile},
	})
	if err != nil {
		t.Fatalf("Analyze 失败: %v", err)
	}

	// 验证：结果不为空，且所有结构体都来自 testdata/examples（Dir 字段相同）
	if len(structs) == 0 {
		t.Error("FileFilter 后结果为空，期望包含 embed_cases.go 中的结构体")
	}
	for _, s := range structs {
		if s.Dir != dir {
			t.Errorf("结构体 %q 的 Dir=%q 不是 testdata/examples，FilterSet 可能未生效", s.Name, s.Dir)
		}
	}

	// 验证：types.go 中定义的结构体（AllTypes）不出现在结果中
	for _, s := range structs {
		if s.Name == "AllTypes" {
			t.Errorf("AllTypes 定义在 types.go，FileFilter 过滤后不应出现")
		}
	}
}

// ─── analyzeFields 边界场景 ───────────────────────────────────────────────────

// edgeDir 返回 testdata/analyzer_edge 目录的绝对路径。
func edgeDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取当前文件路径")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "analyzer_edge")
}

// loadEdge 加载 testdata/analyzer_edge 包，返回结构体名 → StructDef 的映射。
func loadEdge(t *testing.T) map[string]*model.StructDef {
	t.Helper()
	dir := edgeDir(t)
	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		t.Fatalf("加载 testdata/analyzer_edge 失败: %v", err)
	}
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	if err != nil {
		t.Fatalf("分析 testdata/analyzer_edge 失败: %v", err)
	}
	m := make(map[string]*model.StructDef, len(structs))
	for _, s := range structs {
		m[s.Name] = s
	}
	return m
}

// TestAnalyzeFields_UnexportedFieldsSkipped 验证非导出字段不出现在 ActiveFields 中。
func TestAnalyzeFields_UnexportedFieldsSkipped(t *testing.T) {
	structs := loadEdge(t)

	sd, ok := structs["WithUnexported"]
	if !ok {
		t.Fatalf("未找到 WithUnexported 结构体，已加载: %v", structNames(structs))
	}

	active := sd.ActiveFields()
	for _, f := range active {
		if f.Name == "id" || f.Name == "secret" {
			t.Errorf("非导出字段 %q 不应出现在 ActiveFields 中", f.Name)
		}
	}

	exportedNames := make([]string, len(active))
	for i, f := range active {
		exportedNames[i] = f.Name
	}
	wantExported := map[string]bool{"Name": true, "Score": true}
	if len(active) != len(wantExported) {
		t.Errorf("ActiveFields 长度 = %d, want %d（got %v）",
			len(active), len(wantExported), exportedNames)
	}
	for _, f := range active {
		if !wantExported[f.Name] {
			t.Errorf("意外的字段 %q 出现在 ActiveFields 中", f.Name)
		}
	}
}

// TestAnalyzeFields_UnknownTagOptions 验证未知 gogen tag 选项不阻止正常字段收集。
// 未知选项会向 stderr 发出警告，但字段本身仍被收集。
func TestAnalyzeFields_UnknownTagOptions(t *testing.T) {
	structs := loadEdge(t)

	sd, ok := structs["UnknownTagOption"]
	if !ok {
		t.Fatalf("未找到 UnknownTagOption 结构体，已加载: %v", structNames(structs))
	}

	// 字段仍被收集（警告不阻止收集）
	active := sd.ActiveFields()
	if len(active) != 2 {
		names := make([]string, len(active))
		for i, f := range active {
			names[i] = f.Name
		}
		t.Errorf("UnknownTagOption 应有 2 个活跃字段，got %d: %v", len(active), names)
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
