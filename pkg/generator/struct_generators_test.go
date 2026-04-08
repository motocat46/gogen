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
// 创建日期: 2026/4/8

package generator_test

import (
	"strings"
	"testing"

	"github.com/motocat46/gogen/pkg/generator"
	"github.com/motocat46/gogen/pkg/model"
)

// nopLog 丢弃所有日志，用于不关心日志输出的测试用例。
func nopLog(string) {}

// ─── ModifyGenerator.Generate ────────────────────────────────────────────────

func TestModifyGenerator_Generate(t *testing.T) {
	g := &generator.ModifyGenerator{}

	t.Run("ModifyMethod 为空，返回 nil", func(t *testing.T) {
		s := &model.StructDef{
			Name:         "Player",
			PackageName:  "game",
			ModifyMethod: "",
		}
		out, err := g.Generate(s, nopLog)
		if err != nil || out != nil {
			t.Errorf("ModifyMethod=空 时 Generate() 应返回 nil,nil；got %q, %v", out, err)
		}
	})

	t.Run("ModifyMethod 与字段名冲突，CanGenerateMethodOverride=false，返回 nil", func(t *testing.T) {
		s := &model.StructDef{
			Name:         "Player",
			PackageName:  "game",
			ModifyMethod: "Apply",
			DirtyMethod:  "MakeDirty",
			// "Apply" 与字段名冲突，触发 CanGenerateMethodOverride 返回 false
			FieldNames:    map[string]bool{"Apply": true},
			ManualMethods: map[string]bool{},
		}
		out, err := g.Generate(s, nopLog)
		if err != nil || out != nil {
			t.Errorf("字段名冲突时 Generate() 应返回 nil,nil；got %q, %v", out, err)
		}
	})

	t.Run("ModifyMethod 与手写方法冲突，返回 nil", func(t *testing.T) {
		s := &model.StructDef{
			Name:          "Player",
			PackageName:   "game",
			ModifyMethod:  "Modify",
			DirtyMethod:   "MakeDirty",
			FieldNames:    map[string]bool{},
			ManualMethods: map[string]bool{"Modify": true},
		}
		out, err := g.Generate(s, nopLog)
		if err != nil || out != nil {
			t.Errorf("手写方法冲突时 Generate() 应返回 nil,nil；got %q, %v", out, err)
		}
	})

	t.Run("正常生成：输出包含方法名和 dirty 调用", func(t *testing.T) {
		s := &model.StructDef{
			Name:          "Player",
			PackageName:   "game",
			ModifyMethod:  "Modify",
			DirtyMethod:   "MakeDirty",
			FieldNames:    map[string]bool{},
			ManualMethods: map[string]bool{},
		}
		out, err := g.Generate(s, nopLog)
		if err != nil {
			t.Fatalf("Generate() 返回错误: %v", err)
		}
		code := string(out)
		if !strings.Contains(code, "func (this *Player) Modify(") {
			t.Errorf("生成代码缺少方法签名，got:\n%s", code)
		}
		if !strings.Contains(code, "this.MakeDirty()") {
			t.Errorf("生成代码缺少 dirty 调用，got:\n%s", code)
		}
	})

	t.Run("自定义方法名", func(t *testing.T) {
		s := &model.StructDef{
			Name:          "Config",
			PackageName:   "cfg",
			ModifyMethod:  "Apply",
			DirtyMethod:   "MarkChanged",
			FieldNames:    map[string]bool{},
			ManualMethods: map[string]bool{},
		}
		out, err := g.Generate(s, nopLog)
		if err != nil {
			t.Fatalf("Generate() 返回错误: %v", err)
		}
		code := string(out)
		if !strings.Contains(code, "func (this *Config) Apply(") {
			t.Errorf("期望自定义方法名 Apply，got:\n%s", code)
		}
		if !strings.Contains(code, "this.MarkChanged()") {
			t.Errorf("期望自定义 dirty 方法 MarkChanged，got:\n%s", code)
		}
	})
}

// ─── ResetGenerator.Generate ─────────────────────────────────────────────────

func TestResetGenerator_Generate(t *testing.T) {
	g := &generator.ResetGenerator{}

	t.Run("手写 Reset()，打印 Info 并返回 nil", func(t *testing.T) {
		var logged []string
		s := &model.StructDef{
			Name:          "Player",
			PackageName:   "game",
			PackagePath:   "game/player",
			ManualMethods: map[string]bool{"Reset": true},
			FieldNames:    map[string]bool{},
		}
		out, err := g.Generate(s, func(msg string) { logged = append(logged, msg) })
		if err != nil || out != nil {
			t.Errorf("手写 Reset 时 Generate() 应返回 nil,nil；got %q, %v", out, err)
		}
		if len(logged) == 0 || !strings.Contains(logged[0], "[Info]") {
			t.Errorf("期望 [Info] 日志，got %v", logged)
		}
	})

	t.Run("字段名为 Reset，CanGenerateMethodOverride=false，返回 nil", func(t *testing.T) {
		s := &model.StructDef{
			Name:          "Weird",
			PackageName:   "pkg",
			PackagePath:   "pkg/weird",
			ManualMethods: map[string]bool{},
			// 字段名与 Reset 冲突
			FieldNames: map[string]bool{"Reset": true},
		}
		out, err := g.Generate(s, nopLog)
		if err != nil || out != nil {
			t.Errorf("字段名冲突时 Generate() 应返回 nil,nil；got %q, %v", out, err)
		}
	})

	t.Run("正常生成，无 dirty", func(t *testing.T) {
		s := &model.StructDef{
			Name:          "Config",
			PackageName:   "cfg",
			PackagePath:   "cfg",
			ManualMethods: map[string]bool{},
			FieldNames:    map[string]bool{},
			DirtyMethod:   "",
		}
		out, err := g.Generate(s, nopLog)
		if err != nil {
			t.Fatalf("Generate() 返回错误: %v", err)
		}
		code := string(out)
		if !strings.Contains(code, "func (this *Config) Reset()") {
			t.Errorf("生成代码缺少方法签名，got:\n%s", code)
		}
		if strings.Contains(code, "this.") {
			// 无 dirty 时不应有 this.XXX() 调用
			t.Errorf("无 dirty 时不应生成 dirty 调用，got:\n%s", code)
		}
	})

	t.Run("NoDirty=true，不生成 dirty 调用（即使 DirtyMethod 非空）", func(t *testing.T) {
		s := &model.StructDef{
			Name:          "Player",
			PackageName:   "game",
			PackagePath:   "game",
			ManualMethods: map[string]bool{},
			FieldNames:    map[string]bool{},
			DirtyMethod:   "MakeDirty",
			NoDirty:       true,
		}
		out, err := g.Generate(s, nopLog)
		if err != nil {
			t.Fatalf("Generate() 返回错误: %v", err)
		}
		if strings.Contains(string(out), "MakeDirty") {
			t.Errorf("NoDirty=true 时不应调用 MakeDirty，got:\n%s", out)
		}
	})

	t.Run("有 dirty 方法时生成 dirty 调用", func(t *testing.T) {
		s := &model.StructDef{
			Name:          "Player",
			PackageName:   "game",
			PackagePath:   "game",
			ManualMethods: map[string]bool{},
			FieldNames:    map[string]bool{},
			DirtyMethod:   "MakeDirty",
		}
		out, err := g.Generate(s, nopLog)
		if err != nil {
			t.Fatalf("Generate() 返回错误: %v", err)
		}
		if !strings.Contains(string(out), "this.MakeDirty()") {
			t.Errorf("期望 dirty 调用 MakeDirty，got:\n%s", out)
		}
	})
}
