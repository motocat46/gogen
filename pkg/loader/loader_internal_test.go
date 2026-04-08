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

// 内部测试文件：覆盖包私有函数 packageDir 和 isExcludedPath。
package loader

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"
)

// ─── packageDir ───────────────────────────────────────────────────────────────

func TestPackageDir(t *testing.T) {
	tests := []struct {
		name    string
		goFiles []string
		want    string
	}{
		{
			name:    "有 GoFiles，返回第一个文件所在目录",
			goFiles: []string{filepath.Join("/project", "pkg", "foo", "foo.go")},
			want:    filepath.Join("/project", "pkg", "foo"),
		},
		{
			name:    "GoFiles 为空，返回空字符串",
			goFiles: nil,
			want:    "",
		},
		{
			name:    "多个文件，取第一个文件所在目录",
			goFiles: []string{filepath.Join("/a", "b.go"), filepath.Join("/a", "c.go")},
			want:    "/a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &packages.Package{GoFiles: tt.goFiles}
			assert.Equal(t, tt.want, packageDir(pkg))
		})
	}
}

// ─── isExcludedPath ───────────────────────────────────────────────────────────

func TestIsExcludedPath(t *testing.T) {
	sep := string(filepath.Separator)

	tests := []struct {
		name     string
		path     string
		excludes []string
		want     bool
	}{
		{
			name:     "空排除列表",
			path:     sep + filepath.Join("project", "pkg", "foo.go"),
			excludes: nil,
			want:     false,
		},
		{
			name:     "纯目录名匹配路径中某段",
			path:     sep + filepath.Join("project", "mock", "service.go"),
			excludes: []string{"mock"},
			want:     true,
		},
		{
			name:     "纯目录名不匹配",
			path:     sep + filepath.Join("project", "service", "logic.go"),
			excludes: []string{"mock"},
			want:     false,
		},
		{
			name:     "含分隔符的规则：前缀匹配命中",
			path:     sep + filepath.Join("project", "testdata", "foo.go"),
			excludes: []string{sep + filepath.Join("project", "testdata")},
			want:     true,
		},
		{
			name:     "含分隔符的规则：前缀不匹配",
			path:     sep + filepath.Join("project", "pkg", "foo.go"),
			excludes: []string{sep + filepath.Join("project", "testdata")},
			want:     false,
		},
		{
			name:     "多规则命中第二条",
			path:     sep + filepath.Join("project", "generated", "types.go"),
			excludes: []string{"mock", "generated"},
			want:     true,
		},
		{
			name:     "多规则均未命中",
			path:     sep + filepath.Join("project", "internal", "logic.go"),
			excludes: []string{"mock", "generated"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isExcludedPath(tt.path, tt.excludes))
		})
	}
}
