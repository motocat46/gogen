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

// 内部测试文件：覆盖包私有函数 extractDocText。
// Severity.String() 和 Issue.String() 也在此文件测试，避免拆包。
package linter

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── Severity.String ──────────────────────────────────────────────────────────

func TestSeverityString(t *testing.T) {
	tests := []struct {
		s    Severity
		want string
	}{
		{Error, "error"},
		{Warning, "warning"},
		{Severity(99), "warning"}, // 未知值走 default 分支，返回 "warning"
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.s.String())
	}
}

// ─── Issue.String ─────────────────────────────────────────────────────────────

func TestIssueString(t *testing.T) {
	pos := token.Position{Filename: "foo.go", Line: 10, Column: 5}
	iss := Issue{Pos: pos, Severity: Error, Message: "bad tag"}
	assert.Equal(t, "foo.go:10:5: [error] bad tag", iss.String())

	iss2 := Issue{Pos: pos, Severity: Warning, Message: "unused annotation"}
	assert.Equal(t, "foo.go:10:5: [warning] unused annotation", iss2.String())
}

// ─── extractDocText ───────────────────────────────────────────────────────────

func makeCommentGroup(text string) *ast.CommentGroup {
	return &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// " + text}},
	}
}

func TestExtractDocText(t *testing.T) {
	tests := []struct {
		name        string
		genDoc      *ast.CommentGroup
		specComment *ast.CommentGroup
		wantEmpty   bool
		wantContain string
	}{
		{
			name:        "两者均为 nil，返回空字符串",
			genDoc:      nil,
			specComment: nil,
			wantEmpty:   true,
		},
		{
			name:        "genDoc 非空，优先返回 genDoc",
			genDoc:      makeCommentGroup("gogen:plain"),
			specComment: makeCommentGroup("other"),
			wantContain: "gogen:plain",
		},
		{
			name:        "genDoc 为 nil，回退到 specComment",
			genDoc:      nil,
			specComment: makeCommentGroup("gogen:dirty"),
			wantContain: "gogen:dirty",
		},
		{
			name:        "genDoc 非空，specComment 为 nil",
			genDoc:      makeCommentGroup("hello"),
			specComment: nil,
			wantContain: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDocText(tt.genDoc, tt.specComment)
			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assert.Contains(t, got, tt.wantContain)
			}
		})
	}
}
