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

// Package loader 负责使用 go/packages 加载 Go 包的完整类型信息。
//
// 设计说明：
//   - 以"包"为分析单元（而非单个文件），与 stringer 等官方工具保持一致
//   - 加载时同时获取 AST（用于读取注释和 struct tag）和类型信息（用于语义分析）
//   - 本层只负责加载，不做任何业务判断
package loader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// LoadMode 是加载包所需的最小模式集合：
//   - NeedName/NeedFiles：包名和文件路径
//   - NeedSyntax：AST，用于读取注释和 struct tag
//   - NeedTypes/NeedTypesInfo：完整类型信息，用于语义分析
//   - NeedImports：导入信息，用于跨包类型解析
const LoadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo |
	packages.NeedImports

// Config 是 Load 函数的配置选项。
type Config struct {
	// ExcludePaths 为排除规则，与 analyzer.Config.ExcludePaths 语义一致：
	//   - 绝对路径或含路径分隔符的路径：前缀匹配
	//   - 纯目录名（如 "mock"、"mocks"）：匹配路径中任意一段
	ExcludePaths []string

	// GeneratedSuffix 是 gogen 生成文件的名称后缀（不含下划线和 .go 扩展名）。
	// 例如 "access" 表示匹配 *_access.go；"gen" 表示匹配 *_gen.go。
	// 为空时使用默认值 "access"。
	GeneratedSuffix string
}

// Load 加载指定模式匹配的 Go 包，返回带有完整类型信息的包列表。
//
// patterns 支持以下格式：
//   - "."              当前目录的包
//   - "./..."          当前目录及所有子包
//   - "pkg/model"      指定包路径
//   - "./foo.go"       单个 .go 文件（自动转换为 file= 格式，加载整个包）
//   - "file=foo.go"    显式 file= 格式
//
// 内部采用两阶段加载，解决"旧生成文件有错误 → 包无法编译 → gogen 无法重新生成"的死锁：
//
//	阶段1：无 overlay 正常加载，收集所有错误
//	      ↓ 若发现某些 *_{suffix}.go 文件直接导致了错误
//	阶段2：仅对那些有问题的文件使用 overlay（替换为空包声明）再次加载
//	      → 其余正常的 *_{suffix}.go 保持原样，不影响依赖它们的代码
func Load(dir string, cfg Config, patterns ...string) ([]*packages.Package, error) {
	if cfg.GeneratedSuffix == "" {
		cfg.GeneratedSuffix = "access"
	}
	normalized := normalizePatterns(dir, patterns)

	// 阶段1：无 overlay 加载
	pkgs, err := rawLoad(dir, nil, normalized)
	if err != nil {
		return nil, err
	}
	pkgs = filterExcludedPackages(pkgs, cfg.ExcludePaths)

	// 分类错误：直接来自 *_{suffix}.go 的错误 vs 其他错误
	overlay, otherErrs := classifyErrors(pkgs, cfg)

	// 无任何错误 → 直接返回
	if len(overlay) == 0 && len(otherErrs) == 0 {
		return pkgs, nil
	}

	// 仅有用户代码错误（与 *_{suffix}.go 无关）→ 直接报错
	if len(overlay) == 0 {
		return nil, fmt.Errorf("包加载存在错误: %v", otherErrs)
	}

	// 阶段2：仅 overlay 有问题的 *_{suffix}.go，重新加载
	pkgs2, err := rawLoad(dir, overlay, normalized)
	if err != nil {
		return nil, err
	}
	pkgs2 = filterExcludedPackages(pkgs2, cfg.ExcludePaths)

	// 阶段2 错误收集：跳过使用了 overlay 的包（其中的 cascade 错误会在重新生成后消失）
	overlaidDirs := make(map[string]bool)
	for path := range overlay {
		overlaidDirs[filepath.Dir(path)] = true
	}

	var remainErrs []error
	packages.Visit(pkgs2, nil, func(pkg *packages.Package) {
		pkgDir := packageDir(pkg)
		if pkgDir != "" && (overlaidDirs[pkgDir] || isExcludedPath(pkgDir, cfg.ExcludePaths)) {
			return
		}
		for _, e := range pkg.Errors {
			if isNoGoFilesError(e) {
				continue
			}
			remainErrs = append(remainErrs, fmt.Errorf("包 %s: %s", pkg.PkgPath, e))
		}
	})
	if len(remainErrs) > 0 {
		return nil, fmt.Errorf("包加载存在错误: %v", remainErrs)
	}

	return pkgs2, nil
}

// rawLoad 执行一次 packages.Load，不做任何错误过滤。
func rawLoad(dir string, overlay map[string][]byte, normalizedPatterns []string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode:    LoadMode,
		Dir:     dir,
		Tests:   false,
		Overlay: overlay,
	}
	pkgs, err := packages.Load(cfg, normalizedPatterns...)
	if err != nil {
		return nil, fmt.Errorf("加载包失败: %w", err)
	}
	return pkgs, nil
}

// classifyErrors 遍历包错误，将直接来自 *_{suffix}.go 文件的错误分离出来，
// 为这些文件构建 overlay（替换为空包声明），其余错误作为真实用户错误返回。
func classifyErrors(pkgs []*packages.Package, cfg Config) (overlay map[string][]byte, otherErrs []error) {
	overlay = make(map[string][]byte)
	// 文件名后缀模式：_<suffix>.go
	fileSuffix := "_" + cfg.GeneratedSuffix + ".go"

	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		if pkgDir := packageDir(pkg); pkgDir != "" && isExcludedPath(pkgDir, cfg.ExcludePaths) {
			return
		}
		// 构建本包内 *_{suffix}.go 的 basename → 绝对路径映射
		generatedFiles := make(map[string]string)
		for _, f := range pkg.GoFiles {
			if strings.HasSuffix(f, fileSuffix) {
				generatedFiles[filepath.Base(f)] = f
			}
		}

		for _, e := range pkg.Errors {
			if isNoGoFilesError(e) {
				continue
			}
			// 检查此错误是否直接指向某个 *_{suffix}.go 文件
			if absPath := matchAccessFile(e, generatedFiles); absPath != "" {
				pkgName, isGogen := readGogenFilePkg(absPath)
				if isGogen && pkgName != "" {
					overlay[absPath] = []byte("package " + pkgName + "\n")
					continue
				}
			}
			otherErrs = append(otherErrs, fmt.Errorf("包 %s: %s", pkg.PkgPath, e))
		}
	})
	return overlay, otherErrs
}

// matchAccessFile 检查错误是否直接引用了某个 *_access.go 文件，
// 返回匹配的绝对路径；未找到返回空字符串。
func matchAccessFile(e packages.Error, accessFiles map[string]string) string {
	errStr := e.Error()
	for basename, absPath := range accessFiles {
		if strings.Contains(errStr, basename) {
			return absPath
		}
	}
	return ""
}

// filterExcludedPackages 从顶层包列表中移除位于排除路径中的包。
// 只过滤顶层（用户指定的）包，不影响它们的依赖关系解析。
func filterExcludedPackages(pkgs []*packages.Package, excludePaths []string) []*packages.Package {
	if len(excludePaths) == 0 {
		return pkgs
	}
	result := pkgs[:0:len(pkgs)]
	for _, pkg := range pkgs {
		dir := packageDir(pkg)
		if dir != "" && isExcludedPath(dir, excludePaths) {
			continue
		}
		result = append(result, pkg)
	}
	return result
}

// packageDir 返回包所在目录；若无法确定则返回空字符串。
func packageDir(pkg *packages.Package) string {
	if len(pkg.GoFiles) > 0 {
		return filepath.Dir(pkg.GoFiles[0])
	}
	return ""
}

// isExcludedPath 判断路径是否匹配排除规则。
//   - 含路径分隔符或绝对路径的规则：前缀匹配
//   - 纯目录名（如 "mock"、"mocks"）：匹配路径中任意一段
func isExcludedPath(path string, excludes []string) bool {
	for _, ex := range excludes {
		if strings.ContainsRune(ex, filepath.Separator) || filepath.IsAbs(ex) {
			if strings.HasPrefix(path, ex) {
				return true
			}
		} else {
			// 纯名称：检查路径的每一段
			for _, seg := range strings.Split(path, string(filepath.Separator)) {
				if seg == ex {
					return true
				}
			}
		}
	}
	return false
}

// normalizePatterns 将用户传入的 patterns 规范化为 go/packages 可识别的格式。
//
// 核心问题：直接传入 .go 文件路径（如 ./foo.go）时，go/packages 会将其作为
// 合成包 "command-line-arguments" 处理，只包含该单一文件，丢失完整包上下文，
// 导致跨文件/跨包引用（如嵌入的外部结构体）解析失败。
//
// 修复方式：将 .go 文件路径转换为 "file=<绝对路径>" 格式，
// go/packages 会以该文件所属的完整包为单位加载，包含所有文件和导入。
func normalizePatterns(dir string, patterns []string) []string {
	result := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if strings.HasSuffix(p, ".go") && !strings.HasPrefix(p, "file=") {
			abs := p
			if !filepath.IsAbs(p) {
				abs = filepath.Join(dir, p)
			}
			result = append(result, "file="+abs)
		} else {
			result = append(result, p)
		}
	}
	return result
}

// isNoGoFilesError 判断是否为 "no Go files" 类错误。
// 当用户传入不含 .go 文件的目录（如只有子目录的项目根目录）时，
// go/packages 会报此错误，属于正常情况，应静默跳过而非报错退出。
func isNoGoFilesError(e packages.Error) bool {
	return strings.Contains(e.Msg, "no Go files")
}

// ExtractFileFilter 从 patterns 中提取显式指定的 .go 文件绝对路径列表。
//
// 用途：当用户指定具体文件时（如 ./foo.go），加载必须扩展到整个包（保证类型解析正确），
// 但输出应只针对用户指定的文件。FileFilter 就是这个"只输出哪些文件的结构体"的过滤条件。
//
// 返回空列表表示未指定具体文件，应处理所有加载到的结构体。
func ExtractFileFilter(dir string, patterns []string) []string {
	var files []string
	for _, p := range patterns {
		switch {
		case strings.HasSuffix(p, ".go") && !strings.HasPrefix(p, "file="):
			abs := p
			if !filepath.IsAbs(p) {
				abs = filepath.Join(dir, p)
			}
			files = append(files, abs)
		case strings.HasPrefix(p, "file="):
			abs := strings.TrimPrefix(p, "file=")
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(dir, abs)
			}
			files = append(files, abs)
		}
	}
	return files
}

// readGogenFilePkg 读取文件头部，返回 package 名称和是否为 gogen 生成文件。
func readGogenFilePkg(path string) (pkgName string, isGogen bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Code generated") && strings.Contains(line, "DO NOT EDIT") {
			isGogen = true
		}
		if fields := strings.Fields(line); len(fields) >= 2 && fields[0] == "package" {
			pkgName = fields[1]
			break
		}
	}
	return pkgName, isGogen
}
