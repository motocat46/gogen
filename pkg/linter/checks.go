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

package linter

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"
)

// knownOptions 是合法的 gogen tag 选项集合（不含 dirty=XXX 前缀形式）。
var knownOptions = map[string]bool{
	"-":        true,
	"readonly":  true,
	"writeonly": true,
	"plain":     true,
	"override":  true,
	"dirty":     true,
}

// checkStruct 对一个 struct 的所有字段做所有 lint 检查，返回发现的问题列表。
func checkStruct(fset *token.FileSet, typeSpec *ast.TypeSpec, named *types.Named) []Issue {
	structType := typeSpec.Type.(*ast.StructType)

	// 解析结构体文档注释中的 gogen 注解（优先使用 Doc，其次 Comment）
	docText := ""
	if typeSpec.Comment != nil {
		docText = typeSpec.Comment.Text()
	}
	if typeSpec.Doc != nil {
		docText = typeSpec.Doc.Text()
	}
	ann := parseDocAnnotations(docText)

	var issues []Issue
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // 嵌入字段，跳过
		}
		if field.Tag == nil {
			continue // 无 tag
		}

		rawTag := strings.Trim(field.Tag.Value, "`")
		tagVal := reflect.StructTag(rawTag).Get("gogen")
		if tagVal == "" {
			continue
		}

		pos := fset.Position(field.Pos())
		fieldName := field.Names[0].Name

		issues = append(issues, checkTagOptions(pos, fieldName, tagVal)...)
		issues = append(issues, checkTagCombinations(pos, fieldName, tagVal, ann)...)
		issues = append(issues, checkDirtyRef(pos, fieldName, tagVal, named)...)
	}

	// 检查结构体级 dirty 方法引用
	issues = append(issues, checkStructDirtyRef(fset.Position(typeSpec.Pos()), typeSpec.Name.Name, ann, named)...)

	return issues
}

// checkTagOptions 检查 gogen tag 中是否有拼写错误（未知选项）。
func checkTagOptions(pos token.Position, fieldName, tagVal string) []Issue {
	var issues []Issue
	for _, opt := range splitOptions(tagVal) {
		if knownOptions[opt] {
			continue
		}
		if strings.HasPrefix(opt, "dirty=") {
			if len(opt) == len("dirty=") {
				// dirty= 有等号但无方法名：ParseFieldConfig 静默忽略，lint 明确告知
				issues = append(issues, Issue{
					Pos:      pos,
					Severity: Warning,
					Message:  fmt.Sprintf("字段 %s：`dirty=` 方法名为空，将被静默忽略（请用 `dirty` 或 `dirty=方法名`）", fieldName),
				})
			}
			continue // dirty=XXX 形式（含空名）均跳过未知选项检查
		}
		// 尝试给出"你是否指的是"提示
		msg := fmt.Sprintf("字段 %s：未知的 gogen tag 选项 %q", fieldName, opt)
		if suggestion := suggest(opt); suggestion != "" {
			msg += fmt.Sprintf("（是否指的是 %q？）", suggestion)
		}
		issues = append(issues, Issue{Pos: pos, Severity: Error, Message: msg})
	}
	return issues
}

// checkTagCombinations 检查 gogen tag 选项组合是否有矛盾。
func checkTagCombinations(pos token.Position, fieldName, tagVal string, ann docAnnotations) []Issue {
	opts := map[string]bool{}
	dirtyMethod := ""
	for _, opt := range splitOptions(tagVal) {
		opts[opt] = true
		if opt == "dirty" {
			dirtyMethod = "MakeDirty"
		} else if name, ok := strings.CutPrefix(opt, "dirty="); ok && name != "" {
			dirtyMethod = name
		}
	}

	var issues []Issue

	// `-` 与其他选项共存
	if opts["-"] && len(opts) > 1 {
		issues = append(issues, Issue{
			Pos:      pos,
			Severity: Error,
			Message:  fmt.Sprintf("字段 %s：`-` 表示跳过此字段，不能与其他选项组合", fieldName),
		})
	}

	// readonly + writeonly 互斥
	if opts["readonly"] && opts["writeonly"] {
		issues = append(issues, Issue{
			Pos:      pos,
			Severity: Error,
			Message:  fmt.Sprintf("字段 %s：`readonly` 与 `writeonly` 互斥，组合后不会生成任何方法", fieldName),
		})
	}

	// readonly + dirty 无效（dirty 注入在 setter 中，readonly 无 setter）
	if opts["readonly"] && dirtyMethod != "" {
		issues = append(issues, Issue{
			Pos:      pos,
			Severity: Warning,
			Message:  fmt.Sprintf("字段 %s：`readonly` 字段不生成 setter，`dirty` tag 无法生效", fieldName),
		})
	}

	// 字段级 dirty=XXX 但结构体标注了 gogen:nodirty → 字段级 dirty 被压制，无效
	if dirtyMethod != "" && ann.NoDirty {
		issues = append(issues, Issue{
			Pos:      pos,
			Severity: Warning,
			Message:  fmt.Sprintf("字段 %s：结构体标注了 gogen:nodirty，字段级 dirty tag 无效", fieldName),
		})
	}

	return issues
}

// checkDirtyRef 检查字段级 dirty=XXX 指定的方法是否存在于结构体方法集中。
func checkDirtyRef(pos token.Position, fieldName, tagVal string, named *types.Named) []Issue {
	dirtyMethod := ""
	for _, opt := range splitOptions(tagVal) {
		if opt == "dirty" {
			dirtyMethod = "MakeDirty"
		} else if name, ok := strings.CutPrefix(opt, "dirty="); ok && name != "" {
			dirtyMethod = name
		}
	}
	if dirtyMethod == "" {
		return nil
	}
	if methodSetContains(named, dirtyMethod) {
		return nil
	}
	return []Issue{{
		Pos:      pos,
		Severity: Error,
		Message:  fmt.Sprintf("字段 %s：dirty 方法 %q 在类型 *%s 的方法集中不存在，生成代码将无法编译", fieldName, dirtyMethod, named.Obj().Name()),
	}}
}

// checkStructDirtyRef 检查结构体级 gogen:dirty=XXX 注解指定的方法是否存在。
func checkStructDirtyRef(pos token.Position, structName string, ann docAnnotations, named *types.Named) []Issue {
	if ann.DirtyMethod == "" || ann.NoDirty {
		return nil
	}
	if methodSetContains(named, ann.DirtyMethod) {
		return nil
	}
	return []Issue{{
		Pos:      pos,
		Severity: Error,
		Message:  fmt.Sprintf("结构体 %s：gogen:dirty 指定的方法 %q 在 *%s 的方法集中不存在，生成代码将无法编译", structName, ann.DirtyMethod, structName),
	}}
}

// docAnnotations 保存从结构体文档注释解析出的 gogen 注解。
type docAnnotations struct {
	Plain       bool
	DirtyMethod string // "" 表示未指定
	NoDirty     bool
}

// parseDocAnnotations 从结构体文档注释中解析 gogen 注解。
func parseDocAnnotations(doc string) docAnnotations {
	var ann docAnnotations
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "gogen:plain":
			ann.Plain = true
		case line == "gogen:nodirty":
			ann.NoDirty = true
		case line == "gogen:dirty":
			if ann.DirtyMethod == "" {
				ann.DirtyMethod = "MakeDirty"
			}
		case strings.HasPrefix(line, "gogen:dirty="):
			if name, _ := strings.CutPrefix(line, "gogen:dirty="); name != "" {
				ann.DirtyMethod = name
			}
		}
	}
	return ann
}

// methodSetContains 检查 *named 的方法集是否包含名为 methodName 的零参无返回值方法。
func methodSetContains(named *types.Named, methodName string) bool {
	mset := types.NewMethodSet(types.NewPointer(named))
	sel := mset.Lookup(nil, methodName)
	if sel == nil {
		return false
	}
	sig, ok := sel.Type().(*types.Signature)
	if !ok {
		return false
	}
	return sig.Params().Len() == 0 && sig.Results().Len() == 0
}

// splitOptions 将 tag 值按逗号分割，返回清理空格后的选项列表。
func splitOptions(tagVal string) []string {
	var opts []string
	for _, p := range strings.Split(tagVal, ",") {
		if p = strings.TrimSpace(p); p != "" {
			opts = append(opts, p)
		}
	}
	return opts
}

// suggest 为未知选项给出"你是否指的是"建议（Levenshtein 距离 ≤ 3）。
func suggest(unknown string) string {
	candidates := []string{"readonly", "writeonly", "plain", "override", "dirty", "-"}
	best, bestDist := "", 4
	for _, c := range candidates {
		if d := editDistance(unknown, c); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

// editDistance 计算两个字符串之间的 Levenshtein 距离。
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	m, n := len(ra), len(rb)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
		dp[i][0] = i
	}
	for j := range dp[0] {
		dp[0][j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if ra[i-1] == rb[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = 1 + min(dp[i-1][j], min(dp[i][j-1], dp[i-1][j-1]))
			}
		}
	}
	return dp[m][n]
}
