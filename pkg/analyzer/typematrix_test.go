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

// 类型矩阵测试：直接对 buildTypeInfo 产出的 TypeInfo 做字段级断言，
// 独立于黄金文件测试，明确验证每种 TypeKind 的 Kind/TypeStr/Elem/Key/Value 字段。
package analyzer_test

import (
	"testing"

	"github.com/motocat46/gogen/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// structField 从已加载的 structs 中提取指定结构体的指定字段的 TypeInfo。
func structField(t *testing.T, structs map[string]*model.StructDef, structName, fieldName string) *model.TypeInfo {
	t.Helper()
	sd, ok := structs[structName]
	require.True(t, ok, "未找到结构体 %s", structName)
	for _, f := range sd.Fields {
		if f.Name == fieldName {
			require.NotNil(t, f.Type, "字段 %s.%s 的 TypeInfo 为 nil", structName, fieldName)
			return f.Type
		}
	}
	names := make([]string, len(sd.Fields))
	for i, f := range sd.Fields {
		names[i] = f.Name
	}
	t.Fatalf("未找到字段 %s.%s（Fields: %v）", structName, fieldName, names)
	return nil
}

// TestTypeMatrix_Basic 验证基础类型（string）被识别为 KindBasic。
func TestTypeMatrix_Basic(t *testing.T) {
	structs := loadExamples(t)
	ti := structField(t, structs, "AllTypes", "FieldString")

	assert.Equal(t, model.KindBasic, ti.Kind, "string 应为 KindBasic，got %s", ti.Kind)
	assert.Equal(t, "string", ti.TypeStr)
	assert.Nil(t, ti.Elem, "基础类型无 Elem")
}

// TestTypeMatrix_Bool 验证 bool 被识别为 KindBool。
func TestTypeMatrix_Bool(t *testing.T) {
	structs := loadExamples(t)
	ti := structField(t, structs, "AllTypes", "FieldBool")

	assert.Equal(t, model.KindBool, ti.Kind, "bool 应为 KindBool，got %s", ti.Kind)
	assert.Equal(t, "bool", ti.TypeStr)
}

// TestTypeMatrix_Numeric 验证各数值类型（int、int64、float64、uint32）被识别为 KindNumeric。
func TestTypeMatrix_Numeric(t *testing.T) {
	structs := loadExamples(t)

	cases := []struct {
		field   string
		typeStr string
	}{
		{"FieldInt", "int"},
		{"FieldInt64", "int64"},
		{"FieldFloat64", "float64"},
		{"FieldUint32", "uint32"},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			ti := structField(t, structs, "AllTypes", tc.field)
			assert.Equal(t, model.KindNumeric, ti.Kind,
				"%s 应为 KindNumeric，got %s", tc.field, ti.Kind)
			assert.Equal(t, tc.typeStr, ti.TypeStr, "%s 的 TypeStr 不符", tc.field)
		})
	}
}

// TestTypeMatrix_Pointer 验证指针类型被识别为 KindPointer，且 Elem 正确填充。
func TestTypeMatrix_Pointer(t *testing.T) {
	structs := loadExamples(t)

	t.Run("*int", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldPtrInt")
		assert.Equal(t, model.KindPointer, ti.Kind, "got %s", ti.Kind)
		assert.Equal(t, "*int", ti.TypeStr)
		require.NotNil(t, ti.Elem, "*int 的 Elem 不应为 nil")
		assert.Equal(t, model.KindNumeric, ti.Elem.Kind, "Elem 应为 KindNumeric，got %s", ti.Elem.Kind)
		assert.Equal(t, "int", ti.Elem.TypeStr)
	})

	t.Run("*struct", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldPtrStruct")
		assert.Equal(t, model.KindPointer, ti.Kind, "got %s", ti.Kind)
		require.NotNil(t, ti.Elem, "*BaseInfo 的 Elem 不应为 nil")
		assert.Equal(t, model.KindStruct, ti.Elem.Kind, "Elem 应为 KindStruct，got %s", ti.Elem.Kind)
	})
}

// TestTypeMatrix_Slice 验证切片类型被识别为 KindSlice，且 Elem 正确。
func TestTypeMatrix_Slice(t *testing.T) {
	structs := loadExamples(t)

	t.Run("[]int", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldSliceInt")
		assert.Equal(t, model.KindSlice, ti.Kind, "got %s", ti.Kind)
		assert.Equal(t, "[]int", ti.TypeStr)
		require.NotNil(t, ti.Elem)
		assert.Equal(t, model.KindNumeric, ti.Elem.Kind)
	})

	t.Run("[]*struct", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldSliceStruct")
		assert.Equal(t, model.KindSlice, ti.Kind, "got %s", ti.Kind)
		require.NotNil(t, ti.Elem)
		assert.Equal(t, model.KindPointer, ti.Elem.Kind, "Elem 应为 KindPointer，got %s", ti.Elem.Kind)
	})
}

// TestTypeMatrix_Array 验证数组类型被识别为 KindArray，且 Elem 和 ArrayLen 正确。
func TestTypeMatrix_Array(t *testing.T) {
	structs := loadExamples(t)

	ti := structField(t, structs, "AllTypes", "FieldArray8")
	assert.Equal(t, model.KindArray, ti.Kind, "got %s", ti.Kind)
	assert.Equal(t, "[8]int", ti.TypeStr)
	assert.Equal(t, "8", ti.ArrayLen, "ArrayLen 应为 \"8\"")
	require.NotNil(t, ti.Elem)
	assert.Equal(t, model.KindNumeric, ti.Elem.Kind)
}

// TestTypeMatrix_Map 验证 map 类型被识别为 KindMap，Key 和 Value 正确填充。
func TestTypeMatrix_Map(t *testing.T) {
	structs := loadExamples(t)

	ti := structField(t, structs, "AllTypes", "FieldMapStrInt")
	assert.Equal(t, model.KindMap, ti.Kind, "got %s", ti.Kind)
	assert.Equal(t, "map[string]int", ti.TypeStr)
	require.NotNil(t, ti.Key, "map 的 Key 不应为 nil")
	require.NotNil(t, ti.Value, "map 的 Value 不应为 nil")
	assert.Equal(t, model.KindBasic, ti.Key.Kind, "Key 应为 KindBasic（string），got %s", ti.Key.Kind)
	assert.Equal(t, model.KindNumeric, ti.Value.Kind, "Value 应为 KindNumeric（int），got %s", ti.Value.Kind)
}

// TestTypeMatrix_Struct 验证具名结构体类型（time.Time）被识别为 KindStruct。
func TestTypeMatrix_Struct(t *testing.T) {
	structs := loadExamples(t)
	ti := structField(t, structs, "AllTypes", "FieldTime")

	assert.Equal(t, model.KindStruct, ti.Kind, "time.Time 应为 KindStruct，got %s", ti.Kind)
	assert.Equal(t, "time.Time", ti.TypeStr, "跨包结构体 TypeStr 应含包名前缀")
}

// TestTypeMatrix_Interface 验证 interface{}/any 被识别为 KindInterface。
func TestTypeMatrix_Interface(t *testing.T) {
	structs := loadExamples(t)

	t.Run("interface{}", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldInterface")
		assert.Equal(t, model.KindInterface, ti.Kind, "interface{} 应为 KindInterface，got %s", ti.Kind)
	})

	t.Run("any", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldAny")
		// any 是 interface{} 的别名，Kind 应相同
		assert.Equal(t, model.KindInterface, ti.Kind, "any 应为 KindInterface，got %s", ti.Kind)
	})
}

// TestTypeMatrix_Func 验证 func 类型被识别为 KindFunc。
func TestTypeMatrix_Func(t *testing.T) {
	structs := loadExamples(t)
	ti := structField(t, structs, "AllTypes", "FieldFunc")

	assert.Equal(t, model.KindFunc, ti.Kind, "func 类型应为 KindFunc，got %s", ti.Kind)
	assert.Equal(t, "func(int) string", ti.TypeStr)
}

// TestTypeMatrix_Chan 验证 chan 类型被识别为 KindUnsupported（生成层会跳过此字段）。
func TestTypeMatrix_Chan(t *testing.T) {
	structs := loadExamples(t)
	sd := structs["AllTypes"]
	require.NotNil(t, sd)

	var chanField *model.TypeInfo
	for _, f := range sd.Fields {
		if f.Name == "FieldChan" {
			chanField = f.Type
			break
		}
	}
	require.NotNil(t, chanField, "FieldChan 应存在于 Fields 中")
	assert.Equal(t, model.KindUnsupported, chanField.Kind, "chan 应为 KindUnsupported，got %s", chanField.Kind)
}

// TestTypeMatrix_NamedTypeUnderlying 验证具名类型底层解析：TypeStr 保留具名名称，Kind 由底层类型决定。
//   - type UserID int64 → KindNumeric，TypeStr = "UserID"
//   - type Status string → KindBasic，TypeStr = "Status"
//   - type Tags []string → KindSlice，TypeStr = "Tags"
//   - type Metadata map[string]string → KindMap，TypeStr = "Metadata"
func TestTypeMatrix_NamedTypeUnderlying(t *testing.T) {
	structs := loadExamples(t)

	cases := []struct {
		field    string
		wantKind model.TypeKind
		wantStr  string
	}{
		{"FieldUserID", model.KindNumeric, "UserID"},
		{"FieldStatus", model.KindBasic, "Status"},
		{"FieldTags", model.KindSlice, "Tags"},
		{"FieldMetadata", model.KindMap, "Metadata"},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			ti := structField(t, structs, "AllTypes", tc.field)
			assert.Equal(t, tc.wantKind, ti.Kind,
				"%s 应为 %s，got %s", tc.field, tc.wantKind, ti.Kind)
			assert.Equal(t, tc.wantStr, ti.TypeStr,
				"%s 的 TypeStr 应保留具名名称 %q，got %q", tc.field, tc.wantStr, ti.TypeStr)
		})
	}
}

// TestTypeMatrix_TypeAlias 验证类型别名（type MyTime = time.Time）：
// IsAlias=true，Kind 由底层类型决定（time.Time 是 struct → KindStruct）。
func TestTypeMatrix_TypeAlias(t *testing.T) {
	structs := loadExamples(t)
	ti := structField(t, structs, "AllTypes", "FieldMyTime")

	assert.Equal(t, model.KindStruct, ti.Kind,
		"MyTime 别名底层是 time.Time（struct），应为 KindStruct，got %s", ti.Kind)
	assert.True(t, ti.IsAlias, "MyTime 是类型别名，IsAlias 应为 true")
}

// TestTypeMatrix_GenericTypeParam 验证泛型类型参数（T、K、V）被识别为 KindBasic。
func TestTypeMatrix_GenericTypeParam(t *testing.T) {
	structs := loadExamples(t)

	t.Run("单类型参数 Item T", func(t *testing.T) {
		ti := structField(t, structs, "Container", "Item")
		assert.Equal(t, model.KindBasic, ti.Kind, "TypeParam T 应为 KindBasic，got %s", ti.Kind)
		assert.Equal(t, "T", ti.TypeStr)
	})

	t.Run("多类型参数 Key K", func(t *testing.T) {
		ti := structField(t, structs, "Pair", "Key")
		assert.Equal(t, model.KindBasic, ti.Kind, "TypeParam K 应为 KindBasic，got %s", ti.Kind)
		assert.Equal(t, "K", ti.TypeStr)
	})

	t.Run("多类型参数 Value V", func(t *testing.T) {
		ti := structField(t, structs, "Pair", "Value")
		assert.Equal(t, model.KindBasic, ti.Kind, "TypeParam V 应为 KindBasic，got %s", ti.Kind)
		assert.Equal(t, "V", ti.TypeStr)
	})
}

// TestTypeMatrix_NestedComposite 验证嵌套复合类型的 Elem/Key/Value 递归解析。
func TestTypeMatrix_NestedComposite(t *testing.T) {
	structs := loadExamples(t)

	t.Run("map[string][]string（Value 是 slice）", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldMapSlice")
		assert.Equal(t, model.KindMap, ti.Kind, "got %s", ti.Kind)
		require.NotNil(t, ti.Value)
		assert.Equal(t, model.KindSlice, ti.Value.Kind,
			"Value 应为 KindSlice，got %s", ti.Value.Kind)
		require.NotNil(t, ti.Value.Elem)
		assert.Equal(t, model.KindBasic, ti.Value.Elem.Kind,
			"Value.Elem 应为 KindBasic（string），got %s", ti.Value.Elem.Kind)
	})

	t.Run("[]map[string]int（Elem 是 map）", func(t *testing.T) {
		ti := structField(t, structs, "AllTypes", "FieldSliceMap")
		assert.Equal(t, model.KindSlice, ti.Kind, "got %s", ti.Kind)
		require.NotNil(t, ti.Elem)
		assert.Equal(t, model.KindMap, ti.Elem.Kind,
			"Elem 应为 KindMap，got %s", ti.Elem.Kind)
	})
}
