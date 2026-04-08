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

package model_test

import (
	"testing"

	"github.com/motocat46/gogen/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ParseFieldConfig ────────────────────────────────────────────────────────

func TestParseFieldConfig(t *testing.T) {
	tests := []struct {
		name     string
		rawTag   string
		wantSkip bool
		wantRO   bool
		wantWO   bool
	}{
		{
			name:   "空 tag，所有字段为零值",
			rawTag: "",
		},
		{
			name:     "gogen:\"-\" 标记跳过",
			rawTag:   `gogen:"-"`,
			wantSkip: true,
		},
		{
			name:   "gogen:\"readonly\"",
			rawTag: `gogen:"readonly"`,
			wantRO: true,
		},
		{
			name:   "gogen:\"writeonly\"",
			rawTag: `gogen:"writeonly"`,
			wantWO: true,
		},
		{
			name:     "多个选项组合：readonly + -",
			rawTag:   `gogen:"readonly,-"`,
			wantRO:   true,
			wantSkip: true,
		},
		{
			name:   "包含其他 tag：只解析 gogen 部分",
			rawTag: `json:"name,omitempty" gogen:"readonly"`,
			wantRO: true,
		},
		{
			name:   "选项有空格：应 trim",
			rawTag: `gogen:"readonly, writeonly"`,
			wantRO: true,
			wantWO: true,
		},
		{
			name:   "未知选项：忽略，不 panic",
			rawTag: `gogen:"unknown_option"`,
		},
		{
			name:   "无 gogen tag，有其他 tag",
			rawTag: `json:"id" yaml:"id"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := model.ParseFieldConfig(tt.rawTag)
			assert.Equal(t, tt.wantSkip, cfg.Skip)
			assert.Equal(t, tt.wantRO, cfg.Readonly)
			assert.Equal(t, tt.wantWO, cfg.WriteOnly)
		})
	}
}

func TestParseFieldConfigOverride(t *testing.T) {
	cfg, _ := model.ParseFieldConfig(`gogen:"override"`)
	assert.True(t, cfg.Override, "override tag 应设置 Override=true")
	assert.False(t, cfg.Skip || cfg.Readonly || cfg.WriteOnly || cfg.Plain, "override tag 不应影响其他字段")
}

func TestParseFieldConfig_unknownOptions(t *testing.T) {
	cases := []struct {
		name    string
		rawTag  string
		wantLen int
		wantOpt string
	}{
		{"无未知选项", `gogen:"readonly"`, 0, ""},
		{"单个未知选项", `gogen:"raedonly"`, 1, "raedonly"},
		{"多个未知选项", `gogen:"foo,bar"`, 2, "foo"},
		{"dirty= 空值返回哨兵", `gogen:"dirty="`, 1, "dirty="},
		{"双逗号产生的空选项被忽略", `gogen:"readonly,,writeonly"`, 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, unknown := model.ParseFieldConfig(tc.rawTag)
			assert.Len(t, unknown, tc.wantLen, "unknown 选项数量")
			if tc.wantOpt != "" {
				require.NotEmpty(t, unknown)
				assert.Equal(t, tc.wantOpt, unknown[0])
			}
		})
	}
}

// ─── IsReadable / IsWritable ──────────────────────────────────────────────────

func TestIsReadableIsWritable(t *testing.T) {
	tests := []struct {
		name     string
		cfg      model.FieldConfig
		readable bool
		writable bool
	}{
		{
			name:     "默认（无 tag）：读写均可",
			cfg:      model.FieldConfig{},
			readable: true,
			writable: true,
		},
		{
			name:     "Skip：读写均不可",
			cfg:      model.FieldConfig{Skip: true},
			readable: false,
			writable: false,
		},
		{
			name:     "Readonly：可读，不可写",
			cfg:      model.FieldConfig{Readonly: true},
			readable: true,
			writable: false,
		},
		{
			name:     "WriteOnly：不可读，可写",
			cfg:      model.FieldConfig{WriteOnly: true},
			readable: false,
			writable: true,
		},
		{
			name:     "Skip + Readonly：Skip 优先，读写均不可",
			cfg:      model.FieldConfig{Skip: true, Readonly: true},
			readable: false,
			writable: false,
		},
		{
			name:     "Readonly + WriteOnly 同时设置：读写均受限",
			cfg:      model.FieldConfig{Readonly: true, WriteOnly: true},
			readable: false,
			writable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &model.FieldDef{Name: "X", Config: tt.cfg}
			assert.Equal(t, tt.readable, f.IsReadable())
			assert.Equal(t, tt.writable, f.IsWritable())
		})
	}
}

// ─── StructDef.ReceiverType ───────────────────────────────────────────────────

func TestReceiverType(t *testing.T) {
	tests := []struct {
		name       string
		structName string
		typeParams string
		want       string
	}{
		{
			name:       "非泛型结构体",
			structName: "User",
			typeParams: "",
			want:       "User",
		},
		{
			name:       "单类型参数",
			structName: "Container",
			typeParams: "[T]",
			want:       "Container[T]",
		},
		{
			name:       "多类型参数",
			structName: "Pair",
			typeParams: "[K, V]",
			want:       "Pair[K, V]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &model.StructDef{Name: tt.structName, TypeParams: tt.typeParams}
			assert.Equal(t, tt.want, s.ReceiverType())
		})
	}
}

// ─── StructDef.CanGenerateMethod ──────────────────────────────────────────────

func TestCanGenerateMethod(t *testing.T) {
	sd := &model.StructDef{
		Name:            "Example",
		FieldNames:      map[string]bool{"Count": true, "name": true},
		ManualMethods:   map[string]bool{"GetID": true},
		PromotedMethods: map[string]bool{"GetBase": true},
	}

	tests := []struct {
		method string
		want   bool
		reason string
	}{
		// 层 1：方法名与字段名冲突
		{"Count", false, "与导出字段名相同"},
		{"name", false, "与非导出字段名相同"},
		// 层 2：手写方法冲突
		{"GetID", false, "手写文件已有同名方法"},
		// 层 3：提升方法冲突
		{"GetBase", false, "嵌入提升方法同名"},
		// 无冲突
		{"GetCount", true, "GetCount 无冲突（字段名是 Count，不是 GetCount）"},
		{"SetCount", true, "SetCount 无冲突"},
		{"SafeMethod", true, "完全自由的方法名"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			assert.Equal(t, tt.want, sd.CanGenerateMethod(tt.method), tt.reason)
		})
	}
}

// ─── StructDef.CanGenerateMethodOverride ─────────────────────────────────────

func TestCanGenerateMethodOverride(t *testing.T) {
	sd := &model.StructDef{
		Name:            "Example",
		FieldNames:      map[string]bool{"Count": true},
		ManualMethods:   map[string]bool{"GetID": true},
		PromotedMethods: map[string]bool{"GetBase": true},
	}

	tests := []struct {
		method string
		want   bool
		reason string
	}{
		// 层 1：方法名与字段名冲突
		{"Count", false, "与字段名相同"},
		// 层 2：手写方法冲突
		{"GetID", false, "手写文件已有同名方法"},
		// override 模式：提升方法不阻止生成
		{"GetBase", true, "提升方法在 override 模式下允许覆盖"},
		// 无冲突
		{"GetCount", true, "无冲突"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			assert.Equal(t, tt.want, sd.CanGenerateMethodOverride(tt.method), tt.reason)
		})
	}
}

// ─── StructDef.ActiveFields ───────────────────────────────────────────────────

func TestActiveFields(t *testing.T) {
	makeField := func(name string, skip bool) *model.FieldDef {
		return &model.FieldDef{Name: name, Config: model.FieldConfig{Skip: skip}}
	}

	tests := []struct {
		name      string
		fields    []*model.FieldDef
		wantNames []string
	}{
		{
			name:      "空字段列表",
			fields:    nil,
			wantNames: nil,
		},
		{
			name:      "全部活跃",
			fields:    []*model.FieldDef{makeField("A", false), makeField("B", false)},
			wantNames: []string{"A", "B"},
		},
		{
			name:      "全部跳过",
			fields:    []*model.FieldDef{makeField("A", true), makeField("B", true)},
			wantNames: nil,
		},
		{
			name:      "混合：跳过的不出现",
			fields:    []*model.FieldDef{makeField("A", false), makeField("B", true), makeField("C", false)},
			wantNames: []string{"A", "C"},
		},
		{
			name:      "单字段活跃",
			fields:    []*model.FieldDef{makeField("X", false)},
			wantNames: []string{"X"},
		},
		{
			name:      "单字段跳过",
			fields:    []*model.FieldDef{makeField("X", true)},
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &model.StructDef{Fields: tt.fields}
			got := sd.ActiveFields()
			require.Len(t, got, len(tt.wantNames),
				"ActiveFields() 长度不符（got %v, want %v）", fieldNames(got), tt.wantNames)
			for i, f := range got {
				assert.Equal(t, tt.wantNames[i], f.Name, "ActiveFields()[%d].Name", i)
			}
		})
	}
}

// fieldNames 提取字段名列表，用于测试失败信息
func fieldNames(fields []*model.FieldDef) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

// ─── TypeKind.String ──────────────────────────────────────────────────────────

func TestTypeKindString(t *testing.T) {
	tests := []struct {
		kind model.TypeKind
		want string
	}{
		{model.KindBasic, "basic"},
		{model.KindBool, "bool"},
		{model.KindNumeric, "numeric"},
		{model.KindPointer, "pointer"},
		{model.KindSlice, "slice"},
		{model.KindArray, "array"},
		{model.KindMap, "map"},
		{model.KindStruct, "struct"},
		{model.KindGeneric, "generic"},
		{model.KindInterface, "interface"},
		{model.KindFunc, "func"},
		{model.KindUnsupported, "unsupported"},
		{model.TypeKind(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kind.String())
		})
	}
}
