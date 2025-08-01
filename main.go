// Package gogen - Go代码生成器工具
//
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
// 创建日期:2025/7/31
package main

import (
	"fmt"
	"os"
	
	"github.com/spf13/cobra"
	"gogen/pkg/gen"
)

var (
	// 全局标志变量
	outputDir   string
	packageName string
	verbose     bool
	dryRun      bool
)

// rootCmd 代表在没有任何子命令时调用的基础命令
var rootCmd = &cobra.Command{
	Use:   "gogen [files...]",
	Short: "Go代码生成器 - 自动生成结构体访问器方法",
	Long: `gogen 是一个强大的 Go 代码生成工具，可以自动分析 Go 结构体定义并生成相应的访问器方法。

支持的功能：
• 基础类型的 Get/Set 方法
• 切片类型的元素访问、添加、删除操作
• 数组类型的元素访问和设置
• 映射类型的键值访问和操作
• 嵌套结构体的访问方法

示例：
  gogen user.go                    # 为单个文件生成代码
  gogen *.go                       # 为当前目录所有 Go 文件生成代码
  gogen --output ./gen *.go        # 指定输出目录
  gogen --package models *.go      # 指定生成代码的包名
  gogen --dry-run user.go          # 预览生成内容，不实际写入文件`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGenerate,
}

// generateCmd 代表生成命令
var generateCmd = &cobra.Command{
	Use:   "generate [files...]",
	Short: "生成结构体访问器方法",
	Long:  `分析指定的 Go 文件并生成相应的访问器方法。`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runGenerate,
}

// runGenerate 执行代码生成逻辑
func runGenerate(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Printf("开始处理文件: %v\n", args)
		if outputDir != "" {
			fmt.Printf("输出目录: %s\n", outputDir)
		}
		if packageName != "" {
			fmt.Printf("包名: %s\n", packageName)
		}
	}

	// 处理每个输入文件
	for _, file := range args {
		if verbose {
			fmt.Printf("处理文件: %s\n", file)
		}

		if dryRun {
			fmt.Printf("DRY-RUN: 将要处理文件 %s\n", file)
			continue
		}

		// 调用现有的生成逻辑
		gen.GenerateCodes(file)

		if verbose {
			fmt.Printf("已完成文件: %s\n", file)
		}
	}

	return nil
}

func init() {
	// 为根命令添加标志
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "指定输出目录")
	rootCmd.PersistentFlags().StringVarP(&packageName, "package", "p", "", "指定生成代码的包名")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "显示详细输出")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "预览模式，不实际生成文件")

	// 添加子命令
	rootCmd.AddCommand(generateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "执行命令时出错: %v\n", err)
		os.Exit(1)
	}
}