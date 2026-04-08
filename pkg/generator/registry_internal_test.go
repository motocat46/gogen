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

// 内部测试文件：覆盖包私有函数 formatDoc。
package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDoc(t *testing.T) {
	tests := []struct {
		name      string
		doc       string
		wantEmpty bool
		wantLines []string
	}{
		{
			name:      "空文档，返回空字符串",
			doc:       "",
			wantEmpty: true,
		},
		{
			name:      "单行文档",
			doc:       "GetName 返回名称。",
			wantLines: []string{"// GetName 返回名称。"},
		},
		{
			name: "多行文档，每行加前缀",
			doc:  "GetName 返回名称。\n参数说明：无。",
			wantLines: []string{
				"// GetName 返回名称。",
				"// 参数说明：无。",
			},
		},
		{
			name: "文档中含空行，空行输出为 //",
			doc:  "第一段。\n\n第二段。",
			wantLines: []string{
				"// 第一段。",
				"//",
				"// 第二段。",
			},
		},
		{
			name: "末尾换行被裁剪，不产生多余空行",
			doc:  "只有这行。\n",
			wantLines: []string{"// 只有这行。"},
		},
		{
			name: "纯空白行变为 //",
			doc:  "前行。\n   \n后行。",
			wantLines: []string{
				"// 前行。",
				"//",
				"// 后行。",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDoc(tt.doc)
			if tt.wantEmpty {
				assert.Empty(t, got)
				return
			}
			gotLines := strings.Split(got, "\n")
			require.Len(t, gotLines, len(tt.wantLines),
				"formatDoc(%q) 行数不符\ngot:  %q\nwant: %q", tt.doc, gotLines, tt.wantLines)
			for i, wantLine := range tt.wantLines {
				assert.Equal(t, wantLine, gotLines[i], "第 %d 行", i)
			}
		})
	}
}
