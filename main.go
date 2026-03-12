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
// 创建日期: 2025/7/31
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/motocat46/gogen/pkg/analyzer"
	"github.com/motocat46/gogen/pkg/config"
	"github.com/motocat46/gogen/pkg/generator"
	"github.com/motocat46/gogen/pkg/loader"
	"github.com/motocat46/gogen/pkg/writer"
)

// defaultExcludeNames 是默认跳过的目录名列表（纯名称，匹配路径中任意一段）。
// 这些目录通常包含第三方代码、生成代码或测试辅助文件，不应生成访问器。
// 使用 --no-default-excludes 可禁用此默认行为。
var defaultExcludeNames = []string{
	"vendor",   // Go modules vendor 目录（通常在模块根目录）
	"testdata", // 测试数据目录（go test 约定，./... 通常不展开）
	"mock",     // mock 目录（任意层级）
	"mocks",    // mock 目录复数形式（任意层级）
}

// Version 由构建时通过 -ldflags "-X main.Version=v1.x.x" 注入；
// 未注入时显示 "dev"（本地开发构建）。
var Version = "dev"

var (
	outputDir         string
	fileSuffix        string
	verbose           bool
	dryRun            bool
	excludePaths      []string
	noDefaultExcludes bool
)

var rootCmd = &cobra.Command{
	Use:   "gogen [patterns...]",
	Short: "Go 代码生成器 - 自动为结构体生成访问器方法",
	Long: `gogen 自动分析 Go 结构体定义，生成 getter/setter 等访问器方法。

支持的字段类型及生成方法：
  • bool 类型                   → Get / Set / Toggle
  • 数值类型（int/float 等）    → Get / Set / Add / Sub
  • string 等基础类型           → Get / Set
  • 指针 *T                     → Get / Set / Has
  • interface{} / any / 接口    → Get / Set / Has
  • func 类型                   → Get / Set / Has
  • 结构体 T / 泛型实例 List[T] → Get / Set
  • 切片 []T                    → GetAt / GetLen / Range / Has / GetCopy / SetAt / Append / Remove
  • 数组 [N]T                   → Get / GetAt / GetLen / Range / SetAt
  • map[K]V                     → GetVal / GetValOrDefault / Range / Has / HasKey / GetLen / GetKeys / GetCopy / Ensure / SetVal / DelKey

struct tag 控制（在目标结构体字段上添加）：
  gogen:"-"         跳过此字段，不生成任何方法
  gogen:"readonly"  只生成读方法（Get/Range/GetAt 等）
  gogen:"writeonly" 只生成写方法（Set/Append/SetVal 等）
  gogen:"plain"     简单模式：只保留核心访问器，跳过扩展方法
                    （bool 跳过 Toggle；数值跳过 Add/Sub；指针/接口跳过 Has；
                     切片跳过 GetLen/Has/GetCopy；map 跳过 Has/HasKey/GetLen/GetKeys/GetValOrDefault/GetCopy）

patterns 格式（同 go/packages）：
  .          当前目录
  ./...      当前目录及所有子包
  pkg/model  指定包路径

示例：
  gogen .                         # 处理当前目录
  gogen ./...                     # 处理当前目录及所有子包
  gogen --output ./gen ./...      # 指定输出目录
  gogen --dry-run ./...           # 预览，不实际生成
  gogen $GOFILE                   # go generate 中处理当前文件所在包

go:generate 集成（在需要生成的 .go 文件中添加）：
  //go:generate gogen $GOFILE
  //go:generate gogen ./...

孤儿文件清理：
  gogen 自动删除本次处理的目录中已失效的 *_<suffix>.go 文件
  （仅删除含 gogen 标记的文件，不影响手写代码）`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// 确定工作目录（当前目录）
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	// 加载配置文件（.gogen.yaml），CLI 显式传入的参数优先级更高
	fileCfg, err := config.Load(dir)
	if err != nil {
		return err
	}
	// 配置文件 → CLI 合并：只在 CLI 未显式设置时采用配置文件值
	if !cmd.Flags().Changed("suffix") && fileCfg.Suffix != "" {
		fileSuffix = fileCfg.Suffix
	}
	if !cmd.Flags().Changed("output") && fileCfg.Output != "" {
		outputDir = fileCfg.Output
	}
	if !cmd.Flags().Changed("no-default-excludes") && fileCfg.NoDefaultExcludes {
		noDefaultExcludes = fileCfg.NoDefaultExcludes
	}
	// excludes 是追加关系：配置文件排除 + CLI --exclude 合并
	if len(fileCfg.Excludes) > 0 {
		excludePaths = append(fileCfg.Excludes, excludePaths...)
	}
	if verbose && (fileCfg.Suffix != "" || fileCfg.Output != "" || len(fileCfg.Excludes) > 0 || fileCfg.NoDefaultExcludes) {
		fmt.Printf("已加载配置文件: %s\n", config.FileName)
	}

	if verbose {
		fmt.Printf("工作目录: %s\n", dir)
		fmt.Printf("处理模式: %v\n", args)
		if outputDir != "" {
			fmt.Printf("输出目录: %s\n", outputDir)
		}
	}

	// Step 1: 加载包
	if verbose {
		fmt.Println("正在加载包...")
	}
	// 构建最终排除规则：默认名称 + 用户自定义路径
	finalExcludes := buildExcludePaths(dir, excludePaths, noDefaultExcludes)
	if verbose && len(finalExcludes) > 0 {
		fmt.Printf("排除规则: %v\n", finalExcludes)
	}

	// 提取文件过滤列表：若用户指定了具体 .go 文件，只输出这些文件中的结构体；
	// 加载范围仍是整个包，保证跨文件类型引用能正确解析。
	fileFilter := loader.ExtractFileFilter(dir, args)

	pkgs, err := loader.Load(dir, loader.Config{
		ExcludePaths:    finalExcludes,
		GeneratedSuffix: fileSuffix,
	}, args...)
	if err != nil {
		return fmt.Errorf("加载失败: %w", err)
	}
	if verbose {
		fmt.Printf("已加载 %d 个包\n", len(pkgs))
		if len(fileFilter) > 0 {
			fmt.Printf("文件过滤: %v\n", fileFilter)
		}
	}

	// Step 2: 分析结构体
	structs, err := analyzer.Analyze(pkgs, analyzer.Config{
		FileFilter:   fileFilter,
		ExcludePaths: finalExcludes,
	})
	if err != nil {
		return fmt.Errorf("分析失败: %w", err)
	}
	if len(structs) == 0 {
		fmt.Println("未找到任何结构体。")
		fmt.Println("提示：若要处理整个项目，请使用 ./... 而非 ./")
		fmt.Println("  gogen ./...      # 当前目录及所有子包")
		fmt.Println("  gogen ./sub/...  # 指定子目录及其所有子包")
		return nil
	}
	if verbose {
		fmt.Printf("共找到 %d 个结构体\n", len(structs))
	}

	// Step 3+4: 并行生成代码并写入文件
	// Load 和 Analyze 阶段有数据依赖，必须串行；
	// Generate + Write 阶段每个结构体完全独立，并行安全。
	reg := generator.NewRegistry()
	writerCfg := writer.Config{
		OutputDir: outputDir,
		Suffix:    fileSuffix,
		DryRun:    dryRun,
		Verbose:   verbose,
	}

	// 统计信息（并发安全）
	type dirStat struct{ files, methods int }
	var (
		mu            sync.Mutex
		totalFiles    int   // 实际写入的文件数（内容有变化）
		skippedFiles  int   // 内容未变、增量跳过的文件数
		totalMethods  int
		dirStats      = make(map[string]dirStat)
		processedDirs = make(map[string]bool) // 本次运行实际写入的目录（用于孤儿文件扫描）
		validPaths    = make(map[string]bool) // 本次运行生成/保留的文件路径（不应删除）
	)

	// 并发数限制为 CPU 核心数，避免过多 goroutine 争抢 goimports 进程资源
	g, _ := errgroup.WithContext(cmd.Context())
	g.SetLimit(runtime.NumCPU())

	for _, s := range structs {
		g.Go(func() error {
			if verbose {
				fmt.Printf("生成: %s.%s\n", s.PackageName, s.Name)
			}

			// 计算本结构体的输出目录和文件路径（用于孤儿扫描）
			outDir := outputDir
			if outDir == "" {
				outDir = s.Dir
			}
			outPath := filepath.Join(outDir, writerCfg.OutputFilename(s.Name))

			// 记录本次运行的有效路径（即使内容未变也记录，防止被孤儿清理误删）
			mu.Lock()
			processedDirs[outDir] = true
			validPaths[outPath] = true
			mu.Unlock()

			code, err := reg.GenerateStruct(s)
			if err != nil {
				return fmt.Errorf("生成 %s 失败: %w", s.Name, err)
			}
			if code == nil {
				// 所有方法均已有手写实现或被跳过，无需生成文件；
				// 若上次已生成过 _access.go，须删除，否则旧文件与手写方法重复声明导致编译错误。
				return writer.Clean(s, writerCfg)
			}

			// 统计本文件生成的方法数：所有生成方法均使用 "func (this *" 接收者形式
			methodCount := bytes.Count(code, []byte("func (this *"))

			// 计算展示用的相对目录
			relDir, _ := filepath.Rel(dir, outDir)
			if relDir == "" {
				relDir = "."
			}
			fileName := writerCfg.OutputFilename(s.Name)

			written, err := writer.Write(s, code, writerCfg)
			if err != nil {
				return fmt.Errorf("写入 %s 失败: %w", s.Name, err)
			}
			if !written {
				if !dryRun {
					// 内容未变，增量跳过（dry-run 模式下不计入）
					mu.Lock()
					skippedFiles++
					mu.Unlock()
				}
				return nil
			}

			mu.Lock()
			totalFiles++
			totalMethods += methodCount
			ds := dirStats[relDir]
			ds.files++
			ds.methods += methodCount
			dirStats[relDir] = ds
			if !verbose && !dryRun {
				fmt.Printf("✅ %s/%s → %s  (%d 个方法)\n", s.PackageName, s.Name, fileName, methodCount)
			}
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Step 5: 孤儿文件清理
	// 扫描本次处理的目录，删除 *_{suffix}.go 中含 gogen 标记但不在 validPaths 的文件。
	// 安全原则：只扫描本次实际处理的目录，只删除 gogen 生成文件（含标记），绝不误删手写文件。
	if err := cleanOrphans(processedDirs, validPaths, writerCfg, dir, verbose, dryRun); err != nil {
		return err
	}

	// 打印汇总
	if totalFiles == 0 {
		if skippedFiles > 0 {
			fmt.Printf("🎉 代码生成完成！（%d 个文件内容未变，已跳过）\n", skippedFiles)
		} else {
			fmt.Println("🎉 代码生成完成！（无新文件生成）")
		}
		return nil
	}
	// 先打印目录分布（若有多个目录），最后输出总计，让总数作为视觉终点
	if len(dirStats) > 1 {
		dirs := make([]string, 0, len(dirStats))
		for d := range dirStats {
			dirs = append(dirs, d)
		}
		sort.Strings(dirs)
		fmt.Println("按目录分布：")
		for _, d := range dirs {
			ds := dirStats[d]
			fmt.Printf("  %-40s %d 个文件，%d 个方法\n", d+"/", ds.files, ds.methods)
		}
	}
	fmt.Printf("\n🎉 代码生成完成！共生成 %d 个文件，%d 个方法\n", totalFiles, totalMethods)
	return nil
}

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "指定输出目录（默认与源文件同目录）")
	rootCmd.Flags().StringVar(&fileSuffix, "suffix", writer.DefaultSuffix, "生成文件名后缀（如 gen → user_gen.go，access → user_access.go）")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示详细输出")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "预览模式，不实际生成文件")
	rootCmd.Flags().StringArrayVar(&excludePaths, "exclude", nil, "额外排除指定路径（支持目录或文件前缀，可多次指定）")
	rootCmd.Flags().BoolVar(&noDefaultExcludes, "no-default-excludes", false, "禁用默认排除（允许处理 vendor、testdata 等目录）")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(checkCmd)

	// 覆盖 cobra 自动生成的英文描述
	rootCmd.InitDefaultHelpFlag()
	rootCmd.Flags().Lookup("help").Usage = "显示帮助信息"
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "查看命令帮助信息",
		Long:  "查看任意命令的帮助信息。",
		RunE: func(c *cobra.Command, args []string) error {
			cmd, _, err := c.Root().Find(args)
			if cmd == nil || err != nil {
				return fmt.Errorf("未知命令 %q，运行 'gogen help' 查看可用命令", strings.Join(args, " "))
			}
			return cmd.Help()
		},
	})
	// 汉化 completion 子命令描述（cobra 自动注册，通过 Find 取到后修改）
	if compCmd, _, err := rootCmd.Find([]string{"completion"}); err == nil && compCmd != nil && compCmd.Use != rootCmd.Use {
		compCmd.Short = "生成 shell 自动补全脚本（bash/zsh/fish/powershell）"
	}

	checkCmd.Flags().StringVarP(&outputDir, "output", "o", "", "输出目录（需与生成时保持一致）")
	checkCmd.Flags().StringVar(&fileSuffix, "suffix", writer.DefaultSuffix, "生成文件名后缀（需与生成时保持一致）")
	checkCmd.Flags().StringArrayVar(&excludePaths, "exclude", nil, "额外排除路径（可多次指定）")
	checkCmd.Flags().BoolVar(&noDefaultExcludes, "no-default-excludes", false, "禁用默认排除（vendor、testdata、mock、mocks）")
}

// versionCmd 打印版本号后退出。
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本号",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gogen %s\n", Version)
	},
}

// initCmd 在当前目录生成 .gogen.yaml 配置文件模板。
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "在当前目录生成 .gogen.yaml 配置文件模板",
	RunE:  runInit,
}

func runInit(_ *cobra.Command, _ []string) error {
	const configTemplate = `# gogen 配置文件
# 所有字段均可选，未填写时使用内置默认值。
# CLI 参数优先级高于此文件。

# suffix: 生成文件名后缀，默认 "access"（即 user_access.go）
# suffix: access

# output: 统一输出目录，默认空（与源文件同目录）
# output: ""

# excludes: 额外排除的路径，支持纯目录名（任意层级匹配）或路径前缀
# excludes:
#   - mock
#   - proto
#   - internal/generated

# no-default-excludes: 是否禁用内置默认排除（vendor、testdata、mock、mocks）
# no-default-excludes: false
`
	cfgPath := config.FileName
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("文件 %s 已存在，跳过创建", cfgPath)
	}
	if err := os.WriteFile(cfgPath, []byte(configTemplate), 0o644); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	fmt.Printf("已创建 %s，请按需修改后运行 gogen .\n", cfgPath)
	return nil
}

// buildExcludePaths 构建最终排除规则列表。
//   - 默认排除：纯目录名（如 "mock"），由 isExcluded 做路径段匹配，覆盖任意层级
//   - 用户 --exclude：相对路径解析为绝对路径，由 isExcluded 做前缀匹配
func buildExcludePaths(dir string, userExcludes []string, noDefaults bool) []string {
	var result []string
	if !noDefaults {
		result = append(result, defaultExcludeNames...)
	}
	for _, ex := range userExcludes {
		if !filepath.IsAbs(ex) {
			ex = filepath.Join(dir, ex)
		}
		result = append(result, ex)
	}
	return result
}

// cleanOrphans 扫描本次处理的目录，删除已失效的 gogen 孤儿文件。
//
// 安全规则（缺一不可）：
//  1. 仅扫描 processedDirs 中的目录（本次运行实际写入的目录）
//  2. 仅删除文件名匹配 *_{suffix}.go 模式的文件
//  3. 仅删除文件头含 "Code generated ... DO NOT EDIT" 的文件（gogen 标记）
//  4. 不删除在 validPaths 中的文件（本次运行生成/保留的文件）
//
// 满足以上全部条件，才能认定文件是孤儿：结构体已被重命名或删除，旧文件未清理。
func cleanOrphans(processedDirs, validPaths map[string]bool, cfg writer.Config, workDir string, verbose, dryRun bool) error {
	suffix := cfg.Suffix
	if suffix == "" {
		suffix = writer.DefaultSuffix
	}
	// 文件名后缀模式：_<suffix>.go
	fileSuffix := "_" + suffix + ".go"

	var cleaned int
	for scanDir := range processedDirs {
		entries, err := os.ReadDir(scanDir)
		if err != nil {
			// 目录不存在或无权限，跳过（不作为错误）
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			// 规则2：文件名必须以 _{suffix}.go 结尾
			if !strings.HasSuffix(name, fileSuffix) {
				continue
			}
			fullPath := filepath.Join(scanDir, name)
			// 规则4：本次生成/保留的文件不删
			if validPaths[fullPath] {
				continue
			}
			// 规则3：只删除含 gogen 标记的文件
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			if !writer.IsGogenGenerated(content) {
				continue
			}

			// 满足所有安全规则，确认为孤儿文件
			relPath, _ := filepath.Rel(workDir, fullPath)
			if relPath == "" {
				relPath = fullPath
			}

			if dryRun {
				fmt.Printf("[dry-run] 将删除孤儿文件: %s\n", relPath)
				continue
			}
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("删除孤儿文件失败 %s: %w", fullPath, err)
			}
			cleaned++
			if verbose {
				fmt.Printf("🗑️  已删除孤儿文件: %s\n", relPath)
			} else {
				fmt.Printf("🗑️  %s（结构体已删除或重命名）\n", relPath)
			}
		}
	}
	if cleaned > 0 {
		fmt.Printf("共清理 %d 个孤儿文件\n", cleaned)
	}
	return nil
}


// checkCmd 验证生成文件是否最新，不写入任何内容，适用于 CI 和 pre-commit hook。
var checkCmd = &cobra.Command{
	Use:          "check [patterns...]",
	Short:        "验证生成文件是否最新，不写入任何内容（适用于 CI）",
	SilenceUsage:  true, // "文件过期"是业务错误，不是用法错误，不打印 Usage
	SilenceErrors: true, // 错误由 runCheck 自行打印，避免 cobra 重复输出
	Long: `check 运行与 gogen 相同的生成流程，但不写入任何文件。
若存在需要创建、更新或删除的生成文件，打印文件列表并以非零状态码退出。

适用场景：
  • CI 流水线：验证提交的生成文件与源码同步
  • pre-commit hook：防止过期的生成文件被提交

示例：
  gogen check ./...          # 检查当前目录及所有子包
  gogen check ./pkg/model    # 检查指定包`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	// 加载配置文件，CLI 参数优先
	fileCfg, err := config.Load(dir)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("suffix") && fileCfg.Suffix != "" {
		fileSuffix = fileCfg.Suffix
	}
	if !cmd.Flags().Changed("output") && fileCfg.Output != "" {
		outputDir = fileCfg.Output
	}
	if !cmd.Flags().Changed("no-default-excludes") && fileCfg.NoDefaultExcludes {
		noDefaultExcludes = fileCfg.NoDefaultExcludes
	}
	if len(fileCfg.Excludes) > 0 {
		excludePaths = append(fileCfg.Excludes, excludePaths...)
	}

	finalExcludes := buildExcludePaths(dir, excludePaths, noDefaultExcludes)
	fileFilter := loader.ExtractFileFilter(dir, args)

	pkgs, err := loader.Load(dir, loader.Config{
		ExcludePaths:    finalExcludes,
		GeneratedSuffix: fileSuffix,
	}, args...)
	if err != nil {
		return fmt.Errorf("加载失败: %w", err)
	}

	structs, err := analyzer.Analyze(pkgs, analyzer.Config{
		FileFilter:   fileFilter,
		ExcludePaths: finalExcludes,
	})
	if err != nil {
		return fmt.Errorf("分析失败: %w", err)
	}

	reg := generator.NewRegistry()
	writerCfg := writer.Config{
		OutputDir: outputDir,
		Suffix:    fileSuffix,
	}

	var (
		mu       sync.Mutex
		outdated []string
	)

	g, _ := errgroup.WithContext(cmd.Context())
	g.SetLimit(runtime.NumCPU())

	for _, s := range structs {
		g.Go(func() error {
			code, err := reg.GenerateStruct(s)
			if err != nil {
				return fmt.Errorf("生成 %s 失败: %w", s.Name, err)
			}

			upToDate, err := writer.Check(s, code, writerCfg)
			if err != nil {
				return err
			}
			if !upToDate {
				outDir := writerCfg.OutputDir
				if outDir == "" {
					outDir = s.Dir
				}
				relPath, _ := filepath.Rel(dir, filepath.Join(outDir, writerCfg.OutputFilename(s.Name)))
				mu.Lock()
				outdated = append(outdated, relPath)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if len(outdated) == 0 {
		fmt.Println("✅ 所有生成文件均为最新。")
		return nil
	}

	sort.Strings(outdated)
	fmt.Fprintf(os.Stderr, "❌ 以下文件需要更新（请运行 gogen %s 重新生成）：\n", strings.Join(args, " "))
	for _, p := range outdated {
		fmt.Fprintf(os.Stderr, "   %s\n", p)
	}
	return fmt.Errorf("%d 个生成文件已过期", len(outdated))
}

func main() {
	// cobra 懒注册 completion 命令，需在 Execute 前手动初始化后汉化所有描述
	rootCmd.InitDefaultCompletionCmd()
	if compCmd, _, _ := rootCmd.Find([]string{"completion"}); compCmd != nil && compCmd != rootCmd {
		compCmd.Short = "生成 shell 自动补全脚本（bash/zsh/fish/powershell）"
		compCmd.Long = "为指定的 shell 生成 gogen 的自动补全脚本。\n各 shell 的使用方法请参考对应子命令的帮助信息。"
		compCmd.InitDefaultHelpFlag()
		compCmd.Flags().Lookup("help").Usage = "显示帮助信息"
		shellDesc := map[string]string{
			"bash":       "生成 bash 自动补全脚本",
			"zsh":        "生成 zsh 自动补全脚本",
			"fish":       "生成 fish 自动补全脚本",
			"powershell": "生成 powershell 自动补全脚本",
		}
		for _, sub := range compCmd.Commands() {
			if zh, ok := shellDesc[sub.Name()]; ok {
				sub.Short = zh
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
