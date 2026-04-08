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

// 内部测试文件：覆盖包私有函数 isExcluded。
package analyzer

import (
	"path/filepath"
	"testing"
)

func TestIsExcluded(t *testing.T) {
	sep := string(filepath.Separator)

	tests := []struct {
		name         string
		filename     string
		excludePaths []string
		want         bool
	}{
		{
			name:         "空排除列表，不排除任何文件",
			filename:     "/project/pkg/foo/foo.go",
			excludePaths: nil,
			want:         false,
		},
		{
			name:         "纯目录名匹配路径中间段",
			filename:     sep + filepath.Join("project", "mock", "service.go"),
			excludePaths: []string{"mock"},
			want:         true,
		},
		{
			name:         "纯目录名匹配嵌套路径中的一段",
			filename:     sep + filepath.Join("project", "pkg", "mocks", "repo.go"),
			excludePaths: []string{"mocks"},
			want:         true,
		},
		{
			name:         "纯目录名不匹配路径中的不同段",
			filename:     sep + filepath.Join("project", "internal", "service.go"),
			excludePaths: []string{"mock"},
			want:         false,
		},
		{
			name:         "带路径分隔符的规则：前缀匹配",
			filename:     sep + filepath.Join("project", "testdata", "foo.go"),
			excludePaths: []string{sep + filepath.Join("project", "testdata")},
			want:         true,
		},
		{
			name:         "带路径分隔符的规则：前缀不匹配",
			filename:     sep + filepath.Join("project", "pkg", "foo.go"),
			excludePaths: []string{sep + filepath.Join("project", "testdata")},
			want:         false,
		},
		{
			name:         "多个排除规则，命中其中一个",
			filename:     sep + filepath.Join("project", "generated", "types.go"),
			excludePaths: []string{"mock", "generated"},
			want:         true,
		},
		{
			name:         "多个排除规则，均未命中",
			filename:     sep + filepath.Join("project", "service", "logic.go"),
			excludePaths: []string{"mock", "generated"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcluded(tt.filename, tt.excludePaths)
			if got != tt.want {
				t.Errorf("isExcluded(%q, %v) = %v, want %v",
					tt.filename, tt.excludePaths, got, tt.want)
			}
		})
	}
}
