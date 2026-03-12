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
			cfg := model.ParseFieldConfig(tt.rawTag)
			if cfg.Skip != tt.wantSkip {
				t.Errorf("Skip = %v, want %v", cfg.Skip, tt.wantSkip)
			}
			if cfg.Readonly != tt.wantRO {
				t.Errorf("Readonly = %v, want %v", cfg.Readonly, tt.wantRO)
			}
			if cfg.WriteOnly != tt.wantWO {
				t.Errorf("WriteOnly = %v, want %v", cfg.WriteOnly, tt.wantWO)
			}
		})
	}
}

func TestParseFieldConfigOverride(t *testing.T) {
	cfg := model.ParseFieldConfig(`gogen:"override"`)
	if !cfg.Override {
		t.Error("override tag 应设置 Override=true")
	}
	if cfg.Skip || cfg.Readonly || cfg.WriteOnly || cfg.Plain {
		t.Error("override tag 不应影响其他字段")
	}
}

// ─── IsReadable / IsWritable ──────────────────────────────────────────────────

func TestIsReadableIsWritable(t *testing.T) {
	tests := []struct {
		name      string
		cfg       model.FieldConfig
		readable  bool
		writable  bool
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
			if got := f.IsReadable(); got != tt.readable {
				t.Errorf("IsReadable() = %v, want %v", got, tt.readable)
			}
			if got := f.IsWritable(); got != tt.writable {
				t.Errorf("IsWritable() = %v, want %v", got, tt.writable)
			}
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
			if got := s.ReceiverType(); got != tt.want {
				t.Errorf("ReceiverType() = %q, want %q", got, tt.want)
			}
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
			if got := sd.CanGenerateMethod(tt.method); got != tt.want {
				t.Errorf("CanGenerateMethod(%q) = %v, want %v（%s）",
					tt.method, got, tt.want, tt.reason)
			}
		})
	}
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
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("TypeKind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
			}
		})
	}
}
