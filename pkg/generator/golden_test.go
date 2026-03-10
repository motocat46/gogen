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

package generator_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/motocat46/gogen/pkg/analyzer"
	"github.com/motocat46/gogen/pkg/generator"
	"github.com/motocat46/gogen/pkg/loader"
	"github.com/motocat46/gogen/pkg/writer"
)

var (
	// reImportSingle 匹配单行 import 语句，如 import "time"
	reImportSingle = regexp.MustCompile(`\nimport "[^"]*"\n`)
	// reImportMulti 匹配 import 块，如 import ( "a" "b" )
	reImportMulti = regexp.MustCompile(`(?s)\nimport \([^)]*\)\n`)
	// reBlankLines 匹配两个及以上连续换行符。
	// 黄金文件经过 imports.Process（内部含 gofmt），在 top-level 声明之间统一加一个空行；
	// 原始模板输出不加空行。规范化时折叠为单个 \n，使比对聚焦于方法内容。
	reBlankLines = regexp.MustCompile(`\n{2,}`)
)

// normalizeForCompare 在内容比对前规范化以下差异（来自 imports.Process 后处理，不影响代码语义）：
//  1. import 语句（imports.Process 自动推断，内存生成不产生）
//  2. 连续空行（gofmt 在声明之间加空行，模板不加）
//
// 注：时间戳已从生成文件中移除，无需再处理。
func normalizeForCompare(b []byte) []byte {
	b = reImportSingle.ReplaceAll(b, []byte("\n"))
	b = reImportMulti.ReplaceAll(b, []byte("\n"))
	b = reBlankLines.ReplaceAll(b, []byte("\n"))
	return bytes.TrimSpace(b)
}

// goldenDir 返回 testdata/examples 目录的绝对路径。
func goldenDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取当前文件路径")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "examples")
}

// TestGoldenFiles 是全链路黄金文件测试：
//
//  1. 加载 testdata/examples 包（loader）
//  2. 分析所有结构体（analyzer）
//  3. 为每个结构体生成代码（generator）
//  4. 与 testdata/examples 中已提交的 *_access.go 黄金文件对比
//
// 时间戳行在比对前 normalize，其余内容必须逐字节相同。
//
// 维护说明：
//   - 修改了 generator 模板/逻辑后，重新运行 go run . 生成新黄金文件，再提交。
//   - 新增 testdata/examples 中的结构体后，先运行 go run . 生成黄金文件，再提交。
func TestGoldenFiles(t *testing.T) {
	dir := goldenDir(t)

	pkgs, err := loader.Load(dir, loader.Config{}, ".")
	if err != nil {
		t.Fatalf("加载 testdata/examples 失败: %v", err)
	}

	structs, err := analyzer.Analyze(pkgs, analyzer.Config{})
	if err != nil {
		t.Fatalf("分析 testdata/examples 失败: %v", err)
	}

	reg := generator.NewRegistry()
	writerCfg := writer.Config{} // 使用默认后缀 "access"

	for _, s := range structs {
		s := s
		t.Run(s.Name, func(t *testing.T) {
			got, err := reg.GenerateStruct(s)
			if err != nil {
				t.Fatalf("生成 %s 失败: %v", s.Name, err)
			}

			goldenPath := filepath.Join(dir, writerCfg.OutputFilename(s.Name))

			if got == nil {
				// 此结构体所有方法均被手写或跳过，不应生成文件。
				// 若对应黄金文件存在且带有 gogen 标记，说明 Clean 没有被触发，测试警告。
				if content, err := os.ReadFile(goldenPath); err == nil && isGogenContent(content) {
					t.Errorf("%s：生成结果为 nil（无需生成），但黄金文件 %s 仍存在且含 gogen 标记",
						s.Name, filepath.Base(goldenPath))
				}
				return
			}

			// 读取黄金文件
			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Errorf("%s：黄金文件 %s 不存在。\n"+
						"请先运行 go run . --no-default-excludes ./testdata/examples 生成，然后提交。\n"+
						"生成内容预览（前 500 字节）:\n%s",
						s.Name, filepath.Base(goldenPath), contextAround(got, 0, 500))
				} else {
					t.Errorf("%s：读取黄金文件失败: %v", s.Name, err)
				}
				return
			}

			gotNorm := normalizeForCompare(got)
			goldenNorm := normalizeForCompare(golden)
			if !bytes.Equal(gotNorm, goldenNorm) {
				// 找到第一个不同的字节位置，辅助定位差异
				diffPos := firstDiff(goldenNorm, gotNorm)
				t.Errorf("%s：生成内容与黄金文件不符（首个差异位置: %d，黄金文件长度: %d，生成长度: %d）。\n"+
					"若改动是预期的，重新运行 go run . --no-default-excludes ./testdata/examples 更新黄金文件。\n"+
					"--- 黄金文件（差异上下文）---\n%s\n"+
					"+++ 实际生成（差异上下文）+++\n%s",
					s.Name, diffPos, len(goldenNorm), len(gotNorm),
					contextAround(goldenNorm, diffPos, 120),
					contextAround(gotNorm, diffPos, 120))
			}
		})
	}
}

// isGogenContent 判断文件内容是否含有 gogen 生成标记。
func isGogenContent(content []byte) bool {
	return bytes.Contains(content, []byte("Code generated")) &&
		bytes.Contains(content, []byte("DO NOT EDIT"))
}

// firstDiff 返回两个字节切片首个不同字节的位置；完全相同则返回 min(len(a),len(b))。
func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// contextAround 返回以 pos 为中心、前后各取 half 字节的内容，用于错误信息展示。
func contextAround(b []byte, pos, half int) []byte {
	start := pos - half
	if start < 0 {
		start = 0
	}
	end := pos + half
	if end > len(b) {
		end = len(b)
	}
	return b[start:end]
}
