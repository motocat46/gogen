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
	"sync"
	"testing"

	"github.com/motocat46/gogen/pkg/analyzer"
	"github.com/motocat46/gogen/pkg/generator"
	"github.com/motocat46/gogen/pkg/loader"
	"github.com/motocat46/gogen/pkg/model"
)

// BenchmarkGenerateAll 测量生成 testdata/examples 全部结构体的吞吐量。
// 每次迭代生成所有 ~30 个结构体，模拟真实项目的单次 gogen 运行。
// 场景：串行生成（main.go 单线程等价）。
func BenchmarkGenerateAll(b *testing.B) {
	dir := goldenDir(b)
	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		b.Fatalf("加载失败: %v", err)
	}
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	if err != nil {
		b.Fatalf("分析失败: %v", err)
	}

	reg := generator.NewRegistry()
	noop := func(string) {}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, s := range structs {
			if _, err := reg.GenerateStruct(s, noop); err != nil {
				b.Fatalf("生成 %s 失败: %v", s.Name, err)
			}
		}
	}
}

// BenchmarkGenerateAllParallel 测量并发生成时的吞吐量。
// 所有结构体并发生成（模拟 main.go 的 errgroup 并发模式）。
// 与 BenchmarkGenerateAll 对比可量化并发收益。
//
// 操作比例：全部为生成操作，无 I/O，纯 CPU 密集。
func BenchmarkGenerateAllParallel(b *testing.B) {
	dir := goldenDir(b)
	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		b.Fatalf("加载失败: %v", err)
	}
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	if err != nil {
		b.Fatalf("分析失败: %v", err)
	}

	reg := generator.NewRegistry()
	noop := func(string) {}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, s := range structs {
			s := s
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := reg.GenerateStruct(s, noop); err != nil {
					b.Errorf("并发生成 %s 失败: %v", s.Name, err)
				}
			}()
		}
		wg.Wait()
	}
}

// BenchmarkGenerateSingle 测量生成单个复杂结构体（AllTypes）的耗时。
// AllTypes 涵盖所有 TypeKind，是字段种类最丰富的结构体，代表最坏情况。
func BenchmarkGenerateSingle(b *testing.B) {
	dir := goldenDir(b)
	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		b.Fatalf("加载失败: %v", err)
	}
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	if err != nil {
		b.Fatalf("分析失败: %v", err)
	}

	var allTypes *model.StructDef
	for _, s := range structs {
		if s.Name == "AllTypes" {
			allTypes = s
			break
		}
	}
	if allTypes == nil {
		b.Fatal("未找到 AllTypes 结构体")
	}

	reg := generator.NewRegistry()
	noop := func(string) {}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := reg.GenerateStruct(allTypes, noop); err != nil {
			b.Fatalf("生成失败: %v", err)
		}
	}
}
