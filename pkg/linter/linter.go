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
// 创建日期: 2026/3/20

// Package linter 对 gogen struct tag 和注解进行静态检查，
// 捕获拼写错误、矛盾组合和 dirty 方法引用错误。
package linter

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/motocat46/gogen/pkg/loader"
)

// Severity 表示 Issue 的严重程度。
type Severity int

const (
	// Error 表示会导致生成代码无法编译或语义错误的问题（如 dirty 方法不存在）。
	Error Severity = iota
	// Warning 表示无害但可能不符合预期的配置（如 readonly 字段添加了 dirty tag）。
	Warning
)

// String 返回 Severity 的字符串表示。
func (s Severity) String() string {
	if s == Error {
		return "error"
	}
	return "warning"
}

// Issue 表示一个 lint 问题。
type Issue struct {
	Pos      token.Position
	Severity Severity
	Message  string
}

// String 返回 go vet 风格的问题描述：file:line:col: [severity] message
func (i Issue) String() string {
	return fmt.Sprintf("%s: [%s] %s", i.Pos, i.Severity, i.Message)
}

// Config 控制 Lint 行为。
// 注意：调用方负责在传入前用 buildExcludePaths() 将默认排除和用户排除合并展开，
// Config 只接收已处理好的绝对路径或纯目录名列表。
type Config struct {
	ExcludePaths []string
}

// Lint 加载 patterns 指定的包，对所有 struct 做 gogen tag 检查。
// 返回发现的所有问题（按文件位置排序）和加载/类型检查错误。
func Lint(dir string, cfg Config, patterns ...string) ([]Issue, error) {
	pkgs, err := loader.Load(dir, loader.Config{
		ExcludePaths: cfg.ExcludePaths,
	}, patterns...)
	if err != nil {
		return nil, fmt.Errorf("加载失败: %w", err)
	}

	var issues []Issue
	for _, pkg := range pkgs {
		issues = append(issues, lintPackage(pkg)...)
	}

	sort.Slice(issues, func(i, j int) bool {
		pi, pj := issues[i].Pos, issues[j].Pos
		if pi.Filename != pj.Filename {
			return pi.Filename < pj.Filename
		}
		if pi.Line != pj.Line {
			return pi.Line < pj.Line
		}
		return pi.Column < pj.Column
	})
	return issues, nil
}

// lintPackage 对单个包中的所有 struct 声明做检查。
func lintPackage(pkg *packages.Package) []Issue {
	var issues []Issue
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if _, ok = typeSpec.Type.(*ast.StructType); !ok {
				return true
			}
			// 获取 *types.Named（用于 dirty 方法集检查）
			obj, ok := pkg.TypesInfo.Defs[typeSpec.Name]
			if !ok || obj == nil {
				return true
			}
			named, ok := obj.Type().(*types.Named)
			if !ok {
				return true
			}
			issues = append(issues, checkStruct(pkg.Fset, typeSpec, named)...)
			return true
		})
	}
	return issues
}
