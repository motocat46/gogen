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

// Package writer 负责将生成的代码格式化并写入文件。
package writer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"

	"github.com/motocat46/gogen/pkg/model"
)

// DefaultSuffix 是生成文件的默认后缀（不含下划线和 .go）。
const DefaultSuffix = "access"

// Config 控制写入行为。
type Config struct {
	// OutputDir 指定输出目录；为空时输出到结构体源文件的同级目录。
	OutputDir string
	// Suffix 是生成文件名中下划线后的部分，默认为 "access"（即 user_access.go）。
	// 自定义示例：Suffix="gen" → user_gen.go；Suffix="gogen" → user_gogen.go。
	Suffix string
	// DryRun 为 true 时只打印将要生成的文件路径，不实际写入。
	DryRun bool `gogen:"plain"`
	// Verbose 为 true 时输出详细日志。
	Verbose bool `gogen:"plain"`
}

// OutputFilename 返回结构体对应的生成文件名（含 .go 扩展名）。
func (c Config) OutputFilename(structName string) string {
	suffix := c.Suffix
	if suffix == "" {
		suffix = DefaultSuffix
	}
	return strings.ToLower(structName) + "_" + suffix + ".go"
}

// Write 格式化并写入生成代码，返回文件是否实际发生了写入。
//
// 格式化使用 golang.org/x/tools/imports 在进程内完成（等价于 goimports），
// 不依赖外部命令，任何环境输出一致。
//
// 增量跳过：格式化后与磁盘文件逐字节对比，内容相同则返回 false（跳过写入）。
// 因生成内容不含时间戳，相同输入在任何环境产生相同字节，跳过逻辑完全可靠。
func Write(s *model.StructDef, code []byte, cfg Config) (written bool, err error) {
	outDir := cfg.OutputDir
	if outDir == "" {
		outDir = s.Dir
	}

	filename := cfg.OutputFilename(s.Name)
	outputPath := filepath.Join(outDir, filename)

	if cfg.DryRun {
		fmt.Printf("[dry-run] 将生成文件: %s\n", outputPath)
		return false, nil
	}

	// 在进程内格式化（imports.Process 等价于 goimports，自动处理 import 和代码风格）
	formatted, err := imports.Process(outputPath, code, nil)
	if err != nil {
		return false, fmt.Errorf("格式化失败 %s: %w", outputPath, err)
	}

	// 读取已有文件（一次读取，同时用于安全检查和增量对比）
	existing, readErr := os.ReadFile(outputPath)
	if readErr == nil {
		// 安全检查：手写文件（无 "Code generated" 标记）不得覆盖
		if !IsGogenGenerated(existing) {
			return false, fmt.Errorf(
				"文件 %s 已存在，但不含 'Code generated ... DO NOT EDIT' 标记，"+
					"判定为手写文件，拒绝覆盖以保护其中的业务逻辑。\n"+
					"请选择以下方式之一解决冲突：\n"+
					"  1. 将该文件中需要保留的方法移至其他文件，然后删除 %s\n"+
					"  2. 使用 --output 指定其他输出目录，避免路径冲突",
				outputPath, filepath.Base(outputPath),
			)
		}
		// 增量跳过：内容完全一致则无需写入（格式化结果确定，对比可靠）
		if bytes.Equal(formatted, existing) {
			return false, nil
		}
	}

	// 确保输出目录存在
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return false, fmt.Errorf("创建输出目录失败 %s: %w", outDir, err)
	}

	if err := os.WriteFile(outputPath, formatted, 0o644); err != nil {
		return false, fmt.Errorf("写入文件失败 %s: %w", outputPath, err)
	}

	if cfg.Verbose {
		fmt.Printf("✅ 已生成: %s\n", outputPath)
	}
	return true, nil
}

// Check 检查生成代码是否与磁盘文件一致，不写入任何内容。
//
// 返回 true 表示文件已是最新（无需操作），false 表示需要创建、更新或删除。
// code 为 nil 时表示该结构体无需生成文件（所有方法均已手写），此时检查对应文件是否应被删除。
func Check(s *model.StructDef, code []byte, cfg Config) (upToDate bool, err error) {
	outDir := cfg.OutputDir
	if outDir == "" {
		outDir = s.Dir
	}
	outputPath := filepath.Join(outDir, cfg.OutputFilename(s.Name))

	if code == nil {
		// 无需生成：若对应文件存在且含 gogen 标记，说明需要删除 → 不是最新
		content, readErr := os.ReadFile(outputPath)
		if readErr != nil {
			return true, nil // 文件不存在，无需操作
		}
		return !IsGogenGenerated(content), nil
	}

	formatted, err := imports.Process(outputPath, code, nil)
	if err != nil {
		return false, fmt.Errorf("格式化失败 %s: %w", outputPath, err)
	}

	existing, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		return false, nil // 文件不存在，需要创建
	}
	return bytes.Equal(formatted, existing), nil
}

// Clean 删除结构体对应的生成文件（若存在）。
//
// 当结构体所有字段的方法均已有手写实现时，gogen 不再生成新文件。
// 但若上次已生成过 *_access.go，该文件与手写方法会产生重复声明编译错误。
// 此函数负责清理该旧文件，保持包的编译正确性。
func Clean(s *model.StructDef, cfg Config) error {
	outDir := cfg.OutputDir
	if outDir == "" {
		outDir = s.Dir
	}
	outputPath := filepath.Join(outDir, cfg.OutputFilename(s.Name))

	if cfg.DryRun {
		if _, err := os.Stat(outputPath); err == nil {
			fmt.Printf("[dry-run] 将删除文件: %s\n", outputPath)
		}
		return nil
	}

	err := os.Remove(outputPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除旧文件失败 %s: %w", outputPath, err)
	}
	if err == nil && cfg.Verbose {
		fmt.Printf("🗑️  已删除: %s（所有方法均已有手写实现）\n", outputPath)
	}
	return nil
}

// IsGogenGenerated 判断文件内容是否为 gogen（或其他工具）生成的文件。
//
// 依据 Go 官方约定，生成文件必须包含 "// Code generated ... DO NOT EDIT." 注释。
// 只检查文件前 1 KB，避免读取整个大文件。
func IsGogenGenerated(content []byte) bool {
	const headerBytes = 1024
	header := content
	if len(header) > headerBytes {
		header = header[:headerBytes]
	}
	return bytes.Contains(header, []byte("Code generated")) &&
		bytes.Contains(header, []byte("DO NOT EDIT"))
}
