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
// 创建日期: 2026/3/24

package generator_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/motocat46/gogen/pkg/analyzer"
	"github.com/motocat46/gogen/pkg/generator"
	"github.com/motocat46/gogen/pkg/loader"
	"github.com/motocat46/gogen/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────────────────────────────
// 命题一：并发安全
//
// Registry 初始化后为只读；GenerateStruct 是纯函数（相同输入必然产生相同输出）。
// 命题：多个 goroutine 同时调用 GenerateStruct，结果与串行基线完全一致，
// 且 go test -race 下无任何数据竞争。
// ──────────────────────────────────────────────────────────────────────────────

func TestGenerateStructConcurrentSafety(t *testing.T) {
	dir := goldenDir(t)
	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	require.NoError(t, err, "加载 testdata/examples 失败")
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	require.NoError(t, err, "分析 testdata/examples 失败")
	require.NotEmpty(t, structs, "未找到任何结构体，测试无意义")

	reg := generator.NewRegistry()
	noop := func(string) {}

	// 串行基线：为每个结构体建立期望输出
	sequential := make([][]byte, len(structs))
	for i, s := range structs {
		code, err := reg.GenerateStruct(s, noop)
		require.NoError(t, err, "串行生成 %s 失败", s.Name)
		sequential[i] = code
	}

	// 并发执行 20 轮：所有结构体在同一轮中完全并发生成
	// 重复多轮给 -race 检测器充分的交错机会
	const rounds = 20
	for round := 0; round < rounds; round++ {
		results := make([][]byte, len(structs))
		var wg sync.WaitGroup
		for i, s := range structs {
			i, s := i, s
			wg.Add(1)
			go func() {
				defer wg.Done()
				code, err := reg.GenerateStruct(s, noop)
				if err != nil {
					t.Errorf("round %d: 并发生成 %s 失败: %v", round, s.Name, err)
					return
				}
				results[i] = code // 每个 goroutine 写入唯一下标，无竞争
			}()
		}
		wg.Wait()

		for i, s := range structs {
			assert.True(t, bytes.Equal(results[i], sequential[i]),
				"round %d: %s 并发结果与串行基线不一致（并发 %d 字节，串行 %d 字节）",
				round, s.Name, len(results[i]), len(sequential[i]))
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// 命题二：nil 结果语义
//
// GenerateStruct 返回 nil 当且仅当"字段体为空 AND 结构体级方法体也为空"。
//
// 关键设计：ResetGenerator 会为所有结构体生成 Reset()，除非：
//   a) 手写 Reset() 存在（ManualMethods["Reset"]）
//   b) 字段名与 "Reset" 冲突（FieldNames["Reset"]）
//
// 因此：
//   - 零字段结构体 → 字段体空，但 Reset() 仍生成 → 非 nil（Reset 有意义）
//   - 全跳过字段   → 字段体空，但 Reset() 仍生成 → 非 nil
//   - 手写 Reset + 全跳过字段 → 两部分都空 → nil
//   - 字段名 "Reset" + 全跳过其他字段 → Reset 无法生成 → nil
// ──────────────────────────────────────────────────────────────────────────────

// minimalStruct 构造用于边界测试的最小化 StructDef。
func minimalStruct(name string) *model.StructDef {
	return &model.StructDef{
		Name:            name,
		PackageName:     "test",
		ManualMethods:   map[string]bool{},
		PromotedMethods: map[string]bool{},
		FieldNames:      map[string]bool{},
	}
}

// TestGenerateStruct_ResetGenerated_NoFields 验证零字段结构体仍生成 Reset()：
// Reset() 对零字段结构体是合法的（*this = T{} 是空操作），不应跳过。
func TestGenerateStruct_ResetGenerated_NoFields(t *testing.T) {
	s := minimalStruct("NoFields")
	reg := generator.NewRegistry()
	code, err := reg.GenerateStruct(s, func(string) {})
	require.NoError(t, err, "生成失败")
	require.NotNil(t, code, "零字段结构体：期望生成 Reset()，但 GenerateStruct 返回 nil")
	assert.Contains(t, string(code), "func (this *NoFields) Reset()", "期望生成 Reset() 方法")
}

// TestGenerateStruct_NilResult_ManualResetAllSkipped 验证手写 Reset + 全字段跳过时返回 nil：
// 两部分均空（无字段方法 + 手写 Reset 阻止生成）→ 不生成文件。
func TestGenerateStruct_NilResult_ManualResetAllSkipped(t *testing.T) {
	s := minimalStruct("ManualResetAllSkipped")
	s.ManualMethods = map[string]bool{"Reset": true}
	s.Fields = []*model.FieldDef{
		{Name: "A", Type: &model.TypeInfo{Kind: model.KindBasic, TypeStr: "string"}, Config: model.FieldConfig{Skip: true}},
	}
	s.FieldNames = map[string]bool{"A": true}

	reg := generator.NewRegistry()
	var msgs []string
	code, err := reg.GenerateStruct(s, func(msg string) { msgs = append(msgs, msg) })
	require.NoError(t, err, "生成失败")
	assert.Nil(t, code, "手写 Reset + 全字段跳过：期望返回 nil，got %d 字节", len(code))
	// 副作用：应通过 log 传递 [Info] 消息
	assert.NotEmpty(t, msgs, "手写 Reset 时：期望 log 收到 [Info] 消息")
}

// TestGenerateStruct_NilResult_ResetFieldNameConflict 验证字段名 Reset + 全字段跳过时返回 nil：
// FieldNames["Reset"] 阻止 Reset() 生成，字段又全部跳过 → nil。
func TestGenerateStruct_NilResult_ResetFieldNameConflict(t *testing.T) {
	s := minimalStruct("ResetConflict")
	// 唯一字段名为 "Reset"，且该字段被跳过
	s.Fields = []*model.FieldDef{
		{Name: "Reset", Type: &model.TypeInfo{Kind: model.KindBool, TypeStr: "bool"}, Config: model.FieldConfig{Skip: true}},
	}
	s.FieldNames = map[string]bool{"Reset": true}

	reg := generator.NewRegistry()
	code, err := reg.GenerateStruct(s, func(string) {})
	require.NoError(t, err, "生成失败")
	assert.Nil(t, code, "字段名 Reset + 字段全跳过：期望返回 nil，got %d 字节", len(code))
}

// ──────────────────────────────────────────────────────────────────────────────
// 命题三：log 回调语义
//
// 生成器通过 log func(string) 回调传递诊断消息，不直接写 os.Stderr。
// 这是并发输出顺序正确性的基础——调用方在 mutex 保护下统一刷出。
//
// 命题：
//   1. 手写 Reset() 时，ResetGenerator 通过 log 传递含 [Info] 的消息
//   2. 正常生成场景（无手写 Reset），log 不应被调用
// ──────────────────────────────────────────────────────────────────────────────

func TestGenerateStruct_LogCallback_ManualReset(t *testing.T) {
	// 仅有手写 Reset，无其他字段
	s := minimalStruct("ManualReset")
	s.ManualMethods = map[string]bool{"Reset": true}

	reg := generator.NewRegistry()
	var msgs []string
	code, err := reg.GenerateStruct(s, func(msg string) { msgs = append(msgs, msg) })
	require.NoError(t, err, "生成失败")
	assert.Nil(t, code, "手写 Reset 且无字段时：期望返回 nil，got %d 字节", len(code))
	require.NotEmpty(t, msgs, "手写 Reset() 时：期望 log 回调至少收到 1 条 [Info] 消息")
	combined := strings.Join(msgs, "\n")
	assert.Contains(t, combined, "[Info]", "期望消息包含 [Info]")
	assert.Contains(t, combined, "Reset", "期望消息提及 Reset")
}

func TestGenerateStruct_LogCallback_NoMessages(t *testing.T) {
	// 普通结构体，无手写 Reset：log 不应被调用
	s := minimalStruct("Normal")
	s.Fields = []*model.FieldDef{
		{Name: "Name", Type: &model.TypeInfo{Kind: model.KindBasic, TypeStr: "string"}},
	}
	s.FieldNames = map[string]bool{"Name": true}

	reg := generator.NewRegistry()
	var msgs []string
	_, err := reg.GenerateStruct(s, func(msg string) { msgs = append(msgs, msg) })
	require.NoError(t, err, "生成失败")
	assert.Empty(t, msgs, "正常生成场景：log 不应被调用，got %d 条消息: %v", len(msgs), msgs)
}
