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
	"go/token"
	"go/types"
	"testing"

	"github.com/motocat46/gogen/pkg/annotations"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tc.want, got)
		})
	}
}

// ─── MethodSetContains ────────────────────────────────────────────────────────

// makeNamedWithMethod 构造一个带指定方法的 *types.Named，用于测试。
// sig 为 nil 时方法签名为零参无返回值（标准 dirty 方法形式）。
func makeNamedWithMethod(methodName string, sig *types.Signature) *types.Named {
	pkg := types.NewPackage("test", "test")
	obj := types.NewTypeName(token.NoPos, pkg, "MyStruct", nil)
	named := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
	if sig == nil {
		recv := types.NewVar(token.NoPos, pkg, "s", types.NewPointer(named))
		sig = types.NewSignatureType(recv, nil, nil, types.NewTuple(), types.NewTuple(), false)
	}
	method := types.NewFunc(token.NoPos, pkg, methodName, sig)
	named.AddMethod(method)
	return named
}

// makeNamedNoMethods 构造无任何方法的 *types.Named。
func makeNamedNoMethods() *types.Named {
	pkg := types.NewPackage("test", "test")
	obj := types.NewTypeName(token.NoPos, pkg, "Empty", nil)
	return types.NewNamed(obj, types.NewStruct(nil, nil), nil)
}

func TestMethodSetContains(t *testing.T) {
	pkg := types.NewPackage("test", "test")

	// 带参数的方法：func(x int)
	withParam := func(named *types.Named) *types.Signature {
		recv := types.NewVar(token.NoPos, pkg, "s", types.NewPointer(named))
		param := types.NewVar(token.NoPos, pkg, "x", types.Typ[types.Int])
		return types.NewSignatureType(recv, nil, nil, types.NewTuple(param), types.NewTuple(), false)
	}

	// 带返回值的方法：func() error
	withResult := func(named *types.Named) *types.Signature {
		recv := types.NewVar(token.NoPos, pkg, "s", types.NewPointer(named))
		result := types.NewVar(token.NoPos, pkg, "", types.Universe.Lookup("error").Type())
		return types.NewSignatureType(recv, nil, nil, types.NewTuple(), types.NewTuple(result), false)
	}

	t.Run("包含匹配的零参无返回值方法", func(t *testing.T) {
		named := makeNamedWithMethod("MakeDirty", nil)
		assert.True(t, annotations.MethodSetContains(named, "MakeDirty"), "MakeDirty() 存在，应返回 true")
	})

	t.Run("不包含指定方法名", func(t *testing.T) {
		named := makeNamedWithMethod("MakeDirty", nil)
		assert.False(t, annotations.MethodSetContains(named, "MarkDirty"), "MarkDirty 不存在，应返回 false")
	})

	t.Run("无任何方法", func(t *testing.T) {
		named := makeNamedNoMethods()
		assert.False(t, annotations.MethodSetContains(named, "MakeDirty"), "无方法，应返回 false")
	})

	t.Run("方法存在但有参数", func(t *testing.T) {
		obj2 := types.NewTypeName(token.NoPos, pkg, "WithParam", nil)
		named2 := types.NewNamed(obj2, types.NewStruct(nil, nil), nil)
		named2.AddMethod(types.NewFunc(token.NoPos, pkg, "SetDirty", withParam(named2)))
		assert.False(t, annotations.MethodSetContains(named2, "SetDirty"), "SetDirty(int) 有参数，应返回 false")
	})

	t.Run("方法存在但有返回值", func(t *testing.T) {
		obj3 := types.NewTypeName(token.NoPos, pkg, "WithResult", nil)
		named3 := types.NewNamed(obj3, types.NewStruct(nil, nil), nil)
		named3.AddMethod(types.NewFunc(token.NoPos, pkg, "MakeDirty", withResult(named3)))
		assert.False(t, annotations.MethodSetContains(named3, "MakeDirty"), "MakeDirty() error 有返回值，应返回 false")
	})

	t.Run("自定义方法名", func(t *testing.T) {
		named := makeNamedWithMethod("MarkChanged", nil)
		assert.True(t, annotations.MethodSetContains(named, "MarkChanged"), "MarkChanged() 存在，应返回 true")
		assert.False(t, annotations.MethodSetContains(named, "MakeDirty"), "MakeDirty 不存在，应返回 false")
	})
}
