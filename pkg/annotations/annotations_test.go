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
// 创建日期: 2026/3/25

package annotations_test

import (
	"testing"

	"github.com/motocat46/gogen/pkg/annotations"
)

func TestParseStructAnnotations(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want annotations.StructAnnotations
	}{
		{
			name: "空文档",
			doc:  "",
			want: annotations.StructAnnotations{},
		},
		{
			name: "gogen:plain",
			doc:  "gogen:plain",
			want: annotations.StructAnnotations{Plain: true},
		},
		{
			name: "gogen:nodirty",
			doc:  "gogen:nodirty",
			want: annotations.StructAnnotations{NoDirty: true},
		},
		{
			name: "gogen:dirty 使用默认方法名 MakeDirty",
			doc:  "gogen:dirty",
			want: annotations.StructAnnotations{DirtyMethod: "MakeDirty"},
		},
		{
			name: "gogen:dirty=CustomDirty",
			doc:  "gogen:dirty=CustomDirty",
			want: annotations.StructAnnotations{DirtyMethod: "CustomDirty"},
		},
		{
			name: "gogen:dirty= 空值不生效",
			doc:  "gogen:dirty=",
			want: annotations.StructAnnotations{},
		},
		{
			name: "gogen:modify=Apply",
			doc:  "gogen:modify=Apply",
			want: annotations.StructAnnotations{ModifyMethod: "Apply"},
		},
		{
			name: "gogen:modify= 空值不生效",
			doc:  "gogen:modify=",
			want: annotations.StructAnnotations{},
		},
		{
			name: "多注解组合",
			doc:  "gogen:plain\ngogen:dirty=MarkChanged\ngogen:modify=Apply",
			want: annotations.StructAnnotations{Plain: true, DirtyMethod: "MarkChanged", ModifyMethod: "Apply"},
		},
		{
			name: "忽略无关行",
			doc:  "这是普通注释\ngogen:plain\n其他内容",
			want: annotations.StructAnnotations{Plain: true},
		},
		{
			name: "行首尾空格被裁剪",
			doc:  "  gogen:plain  ",
			want: annotations.StructAnnotations{Plain: true},
		},
		{
			name: "gogen:dirty 后置 gogen:dirty=XXX 以后者为准（后者覆盖前者）",
			doc:  "gogen:dirty\ngogen:dirty=MarkChanged",
			want: annotations.StructAnnotations{DirtyMethod: "MarkChanged"},
		},
		{
			name: "nodirty 与 dirty 共存（nodirty 最高优先级由调用方处理，解析层两者都记录）",
			doc:  "gogen:nodirty\ngogen:dirty",
			want: annotations.StructAnnotations{NoDirty: true, DirtyMethod: "MakeDirty"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := annotations.ParseStructAnnotations(tc.doc)
			if got != tc.want {
				t.Errorf("ParseStructAnnotations(%q)\n  got  %+v\n  want %+v", tc.doc, got, tc.want)
			}
		})
	}
}
